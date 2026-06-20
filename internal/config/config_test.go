package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	homeDir := t.TempDir()
	projDir := t.TempDir()

	// Write global .yaprc
	globalJSON := `{
  "default_model": "global-model",
  "log_level": "WARN",
  "aliases": {
    "review": "global review"
  }
}`
	err := os.WriteFile(filepath.Join(homeDir, ".yaprc"), []byte(globalJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write global .yaprc: %v", err)
	}

	// Write local .yaprc
	localJSON := `{
  "default_model": "local-model",
  "aliases": {
    "explain": "local explain"
  }
}`
	err = os.WriteFile(filepath.Join(projDir, ".yaprc"), []byte(localJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write local .yaprc: %v", err)
	}

	// Set Env override
	os.Setenv("YAP_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("YAP_LOG_LEVEL")

	cfg, err := Load(homeDir, projDir)
	if err != nil {
		t.Fatalf("Load config failed: %v", err)
	}

	// default_model should be "local-model" (local overrides global)
	if cfg.DefaultModel != "local-model" {
		t.Errorf("expected DefaultModel 'local-model', got %q", cfg.DefaultModel)
	}

	// log_level should be "DEBUG" (env overrides local/global)
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel 'DEBUG', got %q", cfg.LogLevel)
	}

	// Aliases should contain both "review" (from global) and "explain" (from local)
	if cfg.Aliases["review"] != "global review" {
		t.Errorf("expected alias review='global review', got %q", cfg.Aliases["review"])
	}
	if cfg.Aliases["explain"] != "local explain" {
		t.Errorf("expected alias explain='local explain', got %q", cfg.Aliases["explain"])
	}
}
