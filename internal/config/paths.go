package config

import (
	"os"
	"path/filepath"

	"github.com/stormlightlabs/documango/internal/cache"
)

// ConfigDir returns the XDG configuration directory.
func ConfigDir() (string, error) {
	return cache.ConfigDir()
}

// ResolveDatabasePath converts a database name or path to an absolute path.
// If the input is "default", returns the default database path from config.
// If the input is already an absolute path, returns it as-is.
// If the input is a relative path or basename, resolves it in the data directory.
func ResolveDatabasePath(nameOrPath string) (string, error) {
	if nameOrPath == "" || nameOrPath == "default" {
		cfg, err := Load()
		if err != nil {
			return "", err
		}
		return cfg.Database.Default, nil
	}

	if filepath.IsAbs(nameOrPath) {
		return nameOrPath, nil
	}

	dataDir, err := cache.DataDir()
	if err != nil {
		return "", err
	}

	if filepath.Ext(nameOrPath) == "" {
		nameOrPath += ".usde"
	}

	return filepath.Join(dataDir, nameOrPath), nil
}

// GetDefaultDatabase returns the path to the default database.
func GetDefaultDatabase() (string, error) {
	return ResolveDatabasePath("default")
}

// IsDefaultDatabase checks if the given path is the default database.
func IsDefaultDatabase(path string) bool {
	defaultPath, err := GetDefaultDatabase()
	if err != nil {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absDefault, err := filepath.Abs(defaultPath)
	if err != nil {
		return false
	}

	return absPath == absDefault
}

// EnsureDatabaseDir ensures the directory for a database file exists.
func EnsureDatabaseDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
