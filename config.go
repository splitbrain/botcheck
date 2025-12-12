package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type configManager struct {
	name    string
	path    string
	mode    string
	mu      sync.RWMutex
	modTime time.Time
	checker checker
}

// newConfigManager loads configuration immediately and prepares for reloads.
func newConfigManager(name, path, mode string) *configManager {
	m := &configManager{name: name, path: path, mode: mode}
	m.reloadIfNeeded()
	return m
}

// reloadIfNeeded watches mod time and rebuilds the matcher when the file changes.
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

func modeForName(name string) string {
	switch name {
	case "useragents":
		return "regex"
	case "ips":
		return "nets"
	default:
		return "literal"
	}
}
