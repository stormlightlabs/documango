package cache

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the XDG configuration directory for documango.
// Uses $XDG_CONFIG_HOME/documango or ~/.config/documango on Unix.
// On macOS, uses ~/Library/Application Support/documango.
func ConfigDir() (string, error) {
	if homeOverride := os.Getenv("DOCUMANGO_HOME"); homeOverride != "" {
		return filepath.Join(homeOverride, "config"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "documango"), nil
	}

	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "documango"), nil
	}

	return filepath.Join(home, ".config", "documango"), nil
}

// DataDir returns the XDG data directory for documango.
// Uses $XDG_DATA_HOME/documango or ~/.local/share/documango on Unix.
// On macOS, uses ~/Library/Application Support/documango.
func DataDir() (string, error) {
	if homeOverride := os.Getenv("DOCUMANGO_HOME"); homeOverride != "" {
		return filepath.Join(homeOverride, "data"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "documango"), nil
	}

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "documango"), nil
	}

	return filepath.Join(home, ".local", "share", "documango"), nil
}

// CacheDir returns the XDG cache directory for documango.
// Uses $XDG_CACHE_HOME/documango or ~/.cache/documango on Unix.
// On macOS, uses ~/Library/Caches/documango.
func CacheDir() (string, error) {
	if homeOverride := os.Getenv("DOCUMANGO_HOME"); homeOverride != "" {
		return filepath.Join(homeOverride, "cache"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Caches", "documango"), nil
	}

	if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
		return filepath.Join(cacheHome, "documango"), nil
	}

	return filepath.Join(home, ".cache", "documango"), nil
}
