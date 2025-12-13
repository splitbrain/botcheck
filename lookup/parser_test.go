package main

import (
	"reflect"
	"testing"
)

func TestParseConfigLines(t *testing.T) {
	input := `
# comment
 value1

value2
  # another comment
value3   
`
	want := []string{"value1", "value2", "value3"}
	got := parseConfigLines(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseConfigLines() = %#v, want %#v", got, want)
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCfg string
		wantVal string
		wantErr bool
	}{
		{
			name:    "spaces preserved",
			input:   "ua.list;Mozilla/5.0 (Mac OS)",
			wantCfg: "ua.list",
			wantVal: "Mozilla/5.0 (Mac OS)",
		},
		{
			name:    "trims whitespace",
			input:   "\tusers.re.list ; Alice",
			wantCfg: "users.re.list",
			wantVal: "Alice",
		},
		{
			name:    "missing lookup",
			input:   "users.list",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "missing config name",
			input:   "   \t valueOnly",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg, val, err := parseInput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseInput(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseInput(%q) unexpected error: %v", tt.input, err)
			}
			if cfg != tt.wantCfg || val != tt.wantVal {
				t.Fatalf("parseInput(%q) = (%q, %q), want (%q, %q)", tt.input, cfg, val, tt.wantCfg, tt.wantVal)
			}
		})
	}
}
