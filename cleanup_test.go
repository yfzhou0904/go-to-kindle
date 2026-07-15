package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFileWithAge(t *testing.T, path string, age time.Duration) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupArchiveRemovesOldKeepsNew(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.html")
	newFile := filepath.Join(dir, "new.html")
	writeFileWithAge(t, oldFile, 400*24*time.Hour)
	writeFileWithAge(t, newFile, 10*24*time.Hour)

	removed, err := cleanupArchive(dir, 365, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old file should have been removed")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Errorf("new file should have been kept: %v", err)
	}
}

func TestCleanupArchiveSkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o770); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-400 * 24 * time.Hour)
	if err := os.Chtimes(sub, old, old); err != nil {
		t.Fatal(err)
	}

	removed, err := cleanupArchive(dir, 365, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed, got %d", removed)
	}
	if _, err := os.Stat(sub); err != nil {
		t.Errorf("subdirectory should have been kept: %v", err)
	}
}

func TestCleanupArchiveMissingDir(t *testing.T) {
	removed, err := cleanupArchive(filepath.Join(t.TempDir(), "does-not-exist"), 365, time.Now())
	if err != nil {
		t.Fatalf("missing dir should not error: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed, got %d", removed)
	}
}
