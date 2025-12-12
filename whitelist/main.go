package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	// Log to stderr so stdout remains reserved for the rewrite map protocol.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[rewrite-map] ")

	name, confPath := deriveNameAndPath()
	mode := modeForName(name)

	// Create config manager that loads and hot-reloads the matching rules.
	manager := newConfigManager(name, confPath, mode)

	scanner := bufio.NewScanner(os.Stdin)
	// Allow longer lines by bumping the scanner buffer.
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
