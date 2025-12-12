package main

import (
	"log"
	"net"
	"regexp"
	"strings"
)

// checker matches an input string against preloaded rules.
type checker interface {
	Match(string) bool
}

type literalChecker struct {
	entries map[string]struct{}
}

// newLiteralChecker stores exact strings for quick membership tests.
func newLiteralChecker(values []string) *literalChecker {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return &literalChecker{entries: set}
}

func (c *literalChecker) Match(input string) bool {
	_, ok := c.entries[input]
	return ok
}

type regexChecker struct {
	patterns []*regexp.Regexp
}

// newRegexChecker compiles case-insensitive regex patterns, skipping invalid ones.
func newRegexChecker(patterns []string) *regexChecker {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			log.Printf("skipping invalid regex %q: %v", p, err)
			continue
		}
		compiled = append(compiled, re)
	}
	return &regexChecker{patterns: compiled}
}

func (c *regexChecker) Match(input string) bool {
	for _, re := range c.patterns {
		if re.MatchString(input) {
			return true
		}
	}
	return false
}

type netChecker struct {
	exact map[string]struct{}
	nets  []*net.IPNet
}

// newNetChecker parses IPs and CIDRs, ignoring malformed entries.
func newNetChecker(entries []string) *netChecker {
	exact := make(map[string]struct{})
	var nets []*net.IPNet

	for _, raw := range entries {
		if strings.Contains(raw, "/") {
			_, cidr, err := net.ParseCIDR(raw)
			if err != nil {
				log.Printf("skipping invalid CIDR %q: %v", raw, err)
				continue
			}
			nets = append(nets, cidr)
			continue
		}

		ip := net.ParseIP(raw)
		if ip == nil {
			log.Printf("skipping invalid IP %q", raw)
			continue
		}
		exact[ip.String()] = struct{}{}
	}

	return &netChecker{exact: exact, nets: nets}
}

func (c *netChecker) Match(input string) bool {
	ip := net.ParseIP(strings.TrimSpace(input))
	if ip == nil {
		return false
	}

	if _, ok := c.exact[ip.String()]; ok {
		return true
	}

	for _, n := range c.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
