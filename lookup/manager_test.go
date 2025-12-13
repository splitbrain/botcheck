package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestModeForFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     matchMode
	}{
		{"names.ri.list", modeRegexCI},
		{"names.re.list", modeRegex},
		{"names.net.list", modeNets},
		{"names.list", modeLiteral},
		{"names.other.list", modeLiteral},
	}

	for _, tt := range tests {
		if got := modeForFilename(tt.filename); got != tt.want {
			t.Fatalf("modeForFilename(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestSanitizeConfigFilename(t *testing.T) {
	valid := []string{"abc.list", "abc123.ri.list", "XYZ.re.list", "name.net.list"}
	for _, v := range valid {
		if got, err := sanitizeConfigFilename(v); err != nil || got != v {
			t.Fatalf("sanitizeConfigFilename(%q) = (%q, %v), want (%q, nil)", v, got, err, v)
		}
	}

	invalid := []string{
		"",
		"../evil.list",
		"with space.list",
		"bad!.list",
		"subdir/file.list",
		"name.txt",
	}
	for _, v := range invalid {
		if got, err := sanitizeConfigFilename(v); err == nil {
			t.Fatalf("sanitizeConfigFilename(%q) = %q, expected error", v, got)
		}
	}
}

func TestLoadCheckerMissingFile(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "does-not-exist.list")
	chk, err := loadChecker(modeLiteral, absent)
	if err != nil {
		t.Fatalf("loadChecker unexpected error: %v", err)
	}
	if _, ok := chk.(noMatchChecker); !ok {
		t.Fatalf("loadChecker missing file = %T, want noMatchChecker", chk)
	}
	if chk.Match("anything") {
		t.Fatal("noMatchChecker should never match")
	}
}

func TestConfigManagerReloadIfNeeded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.list")

	writeFile := func(contents string, modTime time.Time) {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}

	initialTime := time.Now().Add(-time.Minute)
	writeFile("first", initialTime)

	mgr := newConfigManager("test.list", path, modeLiteral)
	if !mgr.match("first") {
		t.Fatal("expected initial config to match")
	}
	if mgr.match("second") {
		t.Fatal("unexpected match before reload")
	}

	newTime := time.Now().Add(time.Minute)
	writeFile("second", newTime)
	mgr.reloadIfNeeded()
	if mgr.match("first") {
		t.Fatal("old entry should not match after reload")
	}
	if !mgr.match("second") {
		t.Fatal("expected new entry to match after reload")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	mgr.reloadIfNeeded()
	if mgr.match("second") {
		t.Fatal("expected no matches after config removal")
	}
}

func TestConfigCacheManagerFor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.list")
	if err := os.WriteFile(path, []byte("allowed"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cache := newConfigCache(dir)
	mgr1, err := cache.managerFor("cache.list")
	if err != nil {
		t.Fatalf("managerFor valid name: %v", err)
	}
	if mgr1.mode != modeLiteral {
		t.Fatalf("manager mode = %s, want %s", mgr1.mode, modeLiteral)
	}
	mgr2, err := cache.managerFor("cache.list")
	if err != nil {
		t.Fatalf("managerFor cached name: %v", err)
	}
	if mgr1 != mgr2 {
		t.Fatal("expected cached manager to be reused")
	}

	if mgr1.match("allowed") == false {
		t.Fatal("expected loaded manager to match entry")
	}
	if _, err := cache.managerFor("../escape.list"); err == nil {
		t.Fatal("expected error for invalid filename")
	}
}
