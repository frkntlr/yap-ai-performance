package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the structure of the personal configuration.
type Config struct {
	DefaultModel string            `json:"default_model"`
	LogLevel     string            `json:"log_level"`
	Aliases      map[string]string `json:"aliases"`
}

// NewDefault returns a configuration initialized with system default values.
func NewDefault() *Config {
	return &Config{
		DefaultModel: "gemini-1.5-flash",
		LogLevel:     "INFO",
		Aliases:      make(map[string]string),
	}
}

// Load loads settings from global and local .yaprc files, overriding them with environment variables.
func Load(homeDir, projectDir string) (*Config, error) {
	cfg := NewDefault()

	// 1. Load Global Config (~/.yaprc)
	globalPath := filepath.Join(homeDir, ".yaprc")
	_ = mergeFromFile(globalPath, cfg)

	// Fallback to ~/.config/yap/config.json if ~/.yaprc doesn't exist
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		globalConfigJSON := filepath.Join(homeDir, ".config", "yap", "config.json")
		_ = mergeFromFile(globalConfigJSON, cfg)
	}

	// 2. Load Local Config (./.yaprc)
	if projectDir != "" {
		localPath := filepath.Join(projectDir, ".yaprc")
		_ = mergeFromFile(localPath, cfg)
	}

	// 3. Override with Environment Variables
	if val := os.Getenv("YAP_DEFAULT_MODEL"); val != "" {
		cfg.DefaultModel = val
	}
	if val := os.Getenv("YAP_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	return cfg, nil
}

func mergeFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var temp struct {
		DefaultModel *string           `json:"default_model"`
		LogLevel     *string           `json:"log_level"`
		Aliases      map[string]string `json:"aliases"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	if temp.DefaultModel != nil {
		cfg.DefaultModel = *temp.DefaultModel
	}
	if temp.LogLevel != nil {
		cfg.LogLevel = *temp.LogLevel
	}
	if temp.Aliases != nil {
		if cfg.Aliases == nil {
			cfg.Aliases = make(map[string]string)
		}
		for k, v := range temp.Aliases {
			cfg.Aliases[k] = v
		}
	}

	return nil
}
