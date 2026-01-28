package config

import (
	"path/filepath"

	"github.com/stormlightlabs/documango/internal/cache"
)

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	defaultDBPath := defaultDatabasePath()

	return &Config{
		Database: DatabaseConfig{Default: defaultDBPath},
		Cache: CacheConfig{
			MaxSizeBytes: 5 * 1024 * 1024 * 1024, MaxAgeDays: 30, TTLSeconds: 86400,
		},
		Search: SearchConfig{DefaultLimit: 20},
		Display: DisplayConfig{
			Width: 80, UsePager: false, RenderMarkdown: false, ColorOutput: nil,
		},
	}
}

func defaultDatabasePath() string {
	dataDir, err := cache.DataDir()
	if err != nil {
		return "default.usde"
	}
	return filepath.Join(dataDir, "default.usde")
}
