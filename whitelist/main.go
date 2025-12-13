package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// main runs the rewrite map helper that selects configs dynamically.
func main() {
	// Log to stderr so stdout remains reserved for the rewrite map protocol.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[rewrite-map] ")

	exeDir, err := executableDir()
	if err != nil {
		log.Fatalf("cannot determine executable dir: %v", err)
	}
	cache := newConfigCache(exeDir)

	scanner := bufio.NewScanner(os.Stdin)
	// Allow longer lines by bumping the scanner buffer.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		configName, lookup, err := parseInput(scanner.Text())
		if err != nil {
			log.Printf("invalid input %q: %v", scanner.Text(), err)
			fmt.Println("NULL")
			continue
		}

		manager, err := cache.managerFor(configName)
		if err != nil {
			log.Printf("rejecting config %q: %v", configName, err)
			fmt.Println("NULL")
			continue
		}

		manager.reloadIfNeeded()
		if manager.match(lookup) {
			fmt.Println("FOUND")
		} else {
			fmt.Println("NULL")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("error reading stdin: %v", err)
	}
}

// parseInput expects "<configFilename> <lookupValue>" and preserves spaces in the lookup value.
func parseInput(input string) (string, string, error) {
	line := strings.TrimSpace(input)
	if line == "" {
		return "", "", fmt.Errorf("empty input")
	}

	sep := strings.IndexFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t'
	})
	if sep == -1 {
		return "", "", fmt.Errorf("missing lookup value")
	}

	config := strings.TrimSpace(line[:sep])
	lookup := strings.TrimSpace(line[sep+1:])
	if config == "" || lookup == "" {
		return "", "", fmt.Errorf("missing config or lookup value")
	}
	return config, lookup, nil
}

// executableDir returns the directory of the running binary.
func executableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}
