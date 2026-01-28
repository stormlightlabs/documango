package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration.
type Config struct {
	Database DatabaseConfig `toml:"database"`
	Cache    CacheConfig    `toml:"cache"`
	Search   SearchConfig   `toml:"search"`
	Display  DisplayConfig  `toml:"display"`
}

// DatabaseConfig holds database-related settings.
type DatabaseConfig struct {
	Default string `toml:"default"` // Default database name or path
}

// CacheConfig holds cache-related settings.
type CacheConfig struct {
	MaxSizeBytes int64 `toml:"max_size_bytes"` // Maximum cache size in bytes
	MaxAgeDays   int   `toml:"max_age_days"`   // Maximum age of cache entries in days
	TTLSeconds   int   `toml:"ttl_seconds"`    // Default TTL for cache entries in seconds
}

// SearchConfig holds search-related settings.
type SearchConfig struct {
	DefaultLimit int `toml:"default_limit"` // Default number of search results
}

// DisplayConfig holds display-related settings.
type DisplayConfig struct {
	Width          int   `toml:"width"`           // Default output width
	UsePager       bool  `toml:"use_pager"`       // Enable pager by default
	RenderMarkdown bool  `toml:"render_markdown"` // Render markdown by default
	ColorOutput    *bool `toml:"color_output"`    // Enable colored output (nil = auto)
}

// Load reads the configuration from the XDG config path or uses defaults.
func Load() (*Config, error) {
	configPath, err := configFilePath()
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()

	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to the XDG config path.
func (cfg *Config) Save() error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o644)
}

func configFilePath() (string, error) {
	if path := os.Getenv("DOCUMANGO_CONFIG"); path != "" {
		return path, nil
	}

	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.toml"), nil
}
