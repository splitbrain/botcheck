package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// configManager owns a single config file and the compiled checker for it.
type configManager struct {
	name    string
	path    string
	mode    matchMode
	mu      sync.RWMutex
	modTime time.Time
	checker checker
}

// configCache keeps already-loaded configs keyed by filename.
type configCache struct {
	baseDir  string
	mu       sync.RWMutex
	managers map[string]*configManager
}

// matchMode controls how lookups are evaluated for a given config.
type matchMode string

const (
	modeLiteral matchMode = "literal"
	modeRegexCI matchMode = "regex-ci"
	modeRegex   matchMode = "regex"
	modeNets    matchMode = "nets"
)

var configNameRe = regexp.MustCompile(`^[A-Za-z0-9]+(?:\.(ri|re|net))?\.list$`)

// newConfigManager loads configuration immediately and prepares for reloads.
func newConfigManager(name, path string, mode matchMode) *configManager {
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

	log.Printf("loaded configuration for %q from %s (mode=%s)", m.name, m.path, m.mode)
}

// match runs the current checker against the provided input.
func (m *configManager) match(input string) bool {
	m.mu.RLock()
	chk := m.checker
	m.mu.RUnlock()

	if chk == nil {
		return false
	}
	return chk.Match(input)
}

// newConfigCache constructs a config cache rooted at baseDir.
func newConfigCache(baseDir string) *configCache {
	return &configCache{
		baseDir:  baseDir,
		managers: make(map[string]*configManager),
	}
}

// managerFor returns a cached config manager for the given filename, creating it if needed.
func (c *configCache) managerFor(filename string) (*configManager, error) {
	cleanName, err := sanitizeConfigFilename(filename)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	if mgr := c.managers[cleanName]; mgr != nil {
		c.mu.RUnlock()
		return mgr, nil
	}
	c.mu.RUnlock()

	path := filepath.Join(c.baseDir, cleanName)
	mode := modeForFilename(cleanName)
	mgr := newConfigManager(cleanName, path, mode)

	c.mu.Lock()
	defer c.mu.Unlock()
	// Another goroutine may have created it while we were loading; reuse existing if so.
	if existing := c.managers[cleanName]; existing != nil {
		return existing, nil
	}
	c.managers[cleanName] = mgr
	return mgr, nil
}

// sanitizeConfigFilename ensures the config filename is safe and matches the expected pattern:
// <name>[.<type>].list where name is alphanumeric and type is optional (ri, re, net).
func sanitizeConfigFilename(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("empty config filename")
	}
	if name != filepath.Base(name) {
		return "", errors.New("config filename must not contain path separators")
	}
	if !configNameRe.MatchString(name) {
		return "", errors.New("config filename must match name[.(ri|re|net)].list")
	}
	return name, nil
}

// modeForFilename infers the match mode from a validated config filename.
func modeForFilename(name string) matchMode {
	switch {
	case strings.HasSuffix(name, ".ri.list"):
		return modeRegexCI
	case strings.HasSuffix(name, ".re.list"):
		return modeRegex
	case strings.HasSuffix(name, ".net.list"):
		return modeNets
	case strings.HasSuffix(name, ".list"):
		return modeLiteral
	default:
		return modeLiteral
	}
}

// loadChecker builds a checker for the given mode and config path.
func loadChecker(mode matchMode, path string) (checker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// returns a checker that always returns false.
			return noMatchChecker{}, nil
		}
		return nil, err
	}

	lines := parseConfigLines(string(data))

	switch mode {
	case modeRegexCI:
		return newRegexChecker(lines, true), nil
	case modeRegex:
		return newRegexChecker(lines, false), nil
	case modeNets:
		return newNetChecker(lines), nil
	default:
		return newLiteralChecker(lines), nil
	}
}
