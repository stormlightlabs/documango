package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/stormlightlabs/documango/internal/cache"
)

// DatabaseRegistry tracks all known databases.
type DatabaseRegistry struct {
	Version   int                       `json:"version"`
	Databases map[string]*DatabaseEntry `json:"databases"`
	Default   string                    `json:"default"`
}

// DatabaseEntry represents a database in the registry.
type DatabaseEntry struct {
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoadRegistry reads the database registry from the data directory.
func LoadRegistry() (*DatabaseRegistry, error) {
	registryPath, err := registryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &DatabaseRegistry{
				Version:   1,
				Databases: make(map[string]*DatabaseEntry),
			}, nil
		}
		return nil, err
	}

	var registry DatabaseRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	if registry.Databases == nil {
		registry.Databases = make(map[string]*DatabaseEntry)
	}

	return &registry, nil
}

// Save writes the database registry to the data directory.
func (r *DatabaseRegistry) Save() error {
	registryPath, err := registryPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(registryPath, data, 0o644)
}

// Add registers a new database.
func (r *DatabaseRegistry) Add(name, path string) error {
	if entry, exists := r.Databases[name]; exists {
		entry.UpdatedAt = time.Now()
		return nil
	}

	r.Databases[name] = &DatabaseEntry{
		Path:      path,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if r.Default == "" {
		r.Default = name
	}

	return nil
}

// Remove removes a database from the registry.
func (r *DatabaseRegistry) Remove(name string) {
	delete(r.Databases, name)
	if r.Default == name {
		r.Default = ""
		for k := range r.Databases {
			r.Default = k
			break
		}
	}
}

// GetPath returns the path for a database name.
func (r *DatabaseRegistry) GetPath(name string) (string, bool) {
	entry, ok := r.Databases[name]
	if !ok {
		return "", false
	}
	return entry.Path, true
}

// SetDefault sets the default database.
func (r *DatabaseRegistry) SetDefault(name string) {
	if _, ok := r.Databases[name]; ok {
		r.Default = name
	}
}

// List returns all registered database names.
func (r *DatabaseRegistry) List() []string {
	names := make([]string, 0, len(r.Databases))
	for name := range r.Databases {
		names = append(names, name)
	}
	return names
}

func registryPath() (string, error) {
	dataDir, err := cache.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "databases.json"), nil
}
