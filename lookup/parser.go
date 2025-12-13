package main

import (
	"bufio"
	"fmt"
	"strings"
)

// parseConfigLines splits config contents into cleaned, non-empty, non-comment lines.
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

// parseInput expects "<configFilename>;<lookupValue>" and preserves spaces in the lookup value.
func parseInput(input string) (string, string, error) {
	line := strings.TrimSpace(input)
	if line == "" {
		return "", "", fmt.Errorf("empty input")
	}

	sep := strings.IndexRune(line, ';')
	if sep == -1 {
		return "", "", fmt.Errorf("missing delimiter ; between config and lookup value")
	}

	config := strings.TrimSpace(line[:sep])
	lookup := strings.TrimSpace(line[sep+1:])
	if config == "" || lookup == "" {
		return "", "", fmt.Errorf("missing config or lookup value")
	}
	return config, lookup, nil
}
