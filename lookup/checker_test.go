package main

import "testing"

func TestLiteralChecker(t *testing.T) {
	c := newLiteralChecker([]string{"alpha", "beta"})
	if !c.Match("alpha") || !c.Match("beta") {
		t.Fatal("literal checker failed to match existing entries")
	}
	if c.Match("gamma") {
		t.Fatal("literal checker matched non-existent entry")
	}
}

func TestRegexChecker(t *testing.T) {
	cases := []struct {
		name    string
		c       *regexChecker
		match   string
		noMatch string
	}{
		{
			name:    "case insensitive",
			c:       newRegexChecker([]string{"hello"}, true),
			match:   "HeLLo, world",
			noMatch: "goodbye",
		},
		{
			name:    "case sensitive",
			c:       newRegexChecker([]string{"foo"}, false),
			match:   "foo fighters",
			noMatch: "FOO",
		},
		{
			name:    "skips invalid",
			c:       newRegexChecker([]string{"fo+", "("}, false),
			match:   "fooooo",
			noMatch: "bar",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if !tt.c.Match(tt.match) {
				t.Fatalf("expected %q to match in %s", tt.match, tt.name)
			}
			if tt.c.Match(tt.noMatch) {
				t.Fatalf("expected %q not to match in %s", tt.noMatch, tt.name)
			}
		})
	}
}

func TestNetChecker(t *testing.T) {
	c := newNetChecker([]string{
		"192.0.2.1",
		"198.51.100.0/24",
		"2001:db8::/48",
		"not-an-ip",
		"203.0.113.0/33", // invalid CIDR
	})

	if !c.Match("192.0.2.1") {
		t.Fatal("expected exact IP match")
	}
	if !c.Match("198.51.100.25") {
		t.Fatal("expected CIDR match")
	}
	if !c.Match("2001:db8::1234") {
		t.Fatal("expected IPv6 CIDR match")
	}
	if c.Match("203.0.113.12") {
		t.Fatal("unexpected match on invalid CIDR entry")
	}
	if c.Match("not-an-ip") {
		t.Fatal("unexpected match on invalid IP input")
	}
	if !c.Match(" 198.51.100.200 ") {
		t.Fatal("expected match with surrounding whitespace trimmed")
	}
}
