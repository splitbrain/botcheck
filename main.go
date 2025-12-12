package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type checker interface {
	Match(string) bool
}

type literalChecker struct {
	entries map[string]struct{}
}

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

type configManager struct {
	name    string
	path    string
	mode    string
	mu      sync.RWMutex
	modTime time.Time
	checker checker
}

func newConfigManager(name, path, mode string) *configManager {
	m := &configManager{name: name, path: path, mode: mode}
	m.reloadIfNeeded()
	return m
}

func (m *configManager) reloadIfNeeded() {
	info, err := os.Stat(m.path)
	var modTime time.Time
	if err == nil {
		modTime = info.ModTime()
	}

	m.mu.RLock()
	currentMod := m.modTime
	m.mu.RUnlock()

	shouldReload := false
	switch {
	case err == nil && (currentMod.IsZero() || modTime.After(currentMod)):
		shouldReload = true
	case os.IsNotExist(err) && (!currentMod.IsZero() || m.checker == nil):
		shouldReload = true
	case err != nil && !os.IsNotExist(err):
		log.Printf("unable to stat config %q: %v", m.path, err)
	}

	if !shouldReload {
		return
	}

	chk, err := loadChecker(m.mode, m.path)
	if err != nil {
		log.Printf("failed to load config %q: %v", m.path, err)
		return
	}

	m.mu.Lock()
	m.checker = chk
	m.modTime = modTime
	m.mu.Unlock()

	log.Printf("loaded configuration for %q from %s", m.name, m.path)
}

func (m *configManager) match(input string) bool {
	m.mu.RLock()
	chk := m.checker
	m.mu.RUnlock()

	if chk == nil {
		return false
	}
	return chk.Match(input)
}

func loadChecker(mode, path string) (checker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyChecker(mode), nil
		}
		return nil, err
	}

	lines := parseConfigLines(string(data))

	switch mode {
	case "regex":
		return newRegexChecker(lines), nil
	case "nets":
		return newNetChecker(lines), nil
	default:
		return newLiteralChecker(lines), nil
	}
}

func emptyChecker(mode string) checker {
	switch mode {
	case "regex":
		return newRegexChecker(nil)
	case "nets":
		return newNetChecker(nil)
	default:
		return newLiteralChecker(nil)
	}
}

func parseConfigLines(contents string) []string {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func deriveNameAndPath() (string, string) {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("cannot determine executable path: %v", err)
	}
	base := filepath.Base(exePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	confPath := filepath.Join(filepath.Dir(exePath), name+".conf")

	return name, confPath
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[rewrite-map] ")

	name, confPath := deriveNameAndPath()

	mode := "literal"
	switch name {
	case "useragents":
		mode = "regex"
	case "ips":
		mode = "nets"
	}

	manager := newConfigManager(name, confPath, mode)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		input := scanner.Text()
		manager.reloadIfNeeded()
		if manager.match(input) {
			fmt.Println("FOUND")
		} else {
			fmt.Println("NULL")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("error reading stdin: %v", err)
	}
}
