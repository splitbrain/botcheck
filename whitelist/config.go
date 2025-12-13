package main

import (
	"bufio"
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
