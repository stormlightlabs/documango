package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const manifestVersion = 1

// CacheEntry represents metadata for a single cached item.
type CacheEntry struct {
	Path      string    `json:"path"`       // Relative path within cache directory
	Source    string    `json:"source"`     // Source identifier (e.g., URL, module path)
	ETag      string    `json:"etag"`       // ETag for validation (optional)
	FetchedAt time.Time `json:"fetched_at"` // When the entry was fetched
	ExpiresAt time.Time `json:"expires_at"` // When the entry expires (zero = never)
	Size      int64     `json:"size"`       // Size in bytes
	Checksum  string    `json:"checksum"`   // SHA256 checksum
}

// IsExpired returns true if the cache entry has expired.
func (e *CacheEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}

// CacheManifest represents the cache manifest file.
type CacheManifest struct {
	Version int                    `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"` // Key is cache key
}

// NewCacheManifest creates a new empty cache manifest.
func NewCacheManifest() *CacheManifest {
	return &CacheManifest{
		Version: manifestVersion,
		Entries: make(map[string]*CacheEntry),
	}
}

// LoadManifest reads the cache manifest from the cache directory.
func LoadManifest(cacheDir string) (*CacheManifest, error) {
	manifestPath := filepath.Join(cacheDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewCacheManifest(), nil
		}
		return nil, err
	}

	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	if manifest.Entries == nil {
		manifest.Entries = make(map[string]*CacheEntry)
	}

	return &manifest, nil
}

// Save writes the cache manifest to the cache directory.
func (m *CacheManifest) Save(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	manifestPath := filepath.Join(cacheDir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, data, 0o644)
}

// Add adds a new cache entry to the manifest.
func (m *CacheManifest) Add(key string, entry *CacheEntry) {
	m.Entries[key] = entry
}

// Get retrieves a cache entry by key.
func (m *CacheManifest) Get(key string) (*CacheEntry, bool) {
	entry, ok := m.Entries[key]
	return entry, ok
}

// Delete removes a cache entry from the manifest.
func (m *CacheManifest) Delete(key string) {
	delete(m.Entries, key)
}

// Keys returns all cache keys.
func (m *CacheManifest) Keys() []string {
	keys := make([]string, 0, len(m.Entries))
	for key := range m.Entries {
		keys = append(keys, key)
	}
	return keys
}

// Prune removes expired entries and returns the list of removed keys.
func (m *CacheManifest) Prune() []string {
	var pruned []string
	for key, entry := range m.Entries {
		if entry.IsExpired() {
			pruned = append(pruned, key)
			delete(m.Entries, key)
		}
	}
	return pruned
}

// TotalSize returns the total size of all cache entries in bytes.
func (m *CacheManifest) TotalSize() int64 {
	var total int64
	for _, entry := range m.Entries {
		total += entry.Size
	}
	return total
}

// Count returns the number of cache entries.
func (m *CacheManifest) Count() int {
	return len(m.Entries)
}
