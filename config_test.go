package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestInitConfigCreatesParentDirectoryAndDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go-to-kindle", "config.toml")

	if err := initConfig(path); err != nil {
		t.Fatalf("initConfig() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if config != Conf {
		t.Errorf("config = %+v, want %+v", config, Conf)
	}
}
