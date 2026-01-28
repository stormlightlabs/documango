package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Cache defines the interface for cache operations.
type Cache interface {
	// Get retrieves a cached item by key, returning the file path and metadata.
	Get(key string) (string, *CacheEntry, error)

	// Put stores an item in the cache with the given key and metadata.
	Put(key string, source string, reader io.Reader, ttl time.Duration) (*CacheEntry, error)

	// Has checks if a key exists in the cache and is not expired.
	Has(key string) bool

	// Delete removes a cached item by key.
	Delete(key string) error

	// Validate verifies a cache entry's checksum and returns true if valid.
	Validate(key string) (bool, error)

	// Refresh updates the fetched_at time for a cache entry.
	Refresh(key string) error

	// Size returns the total size of all cache entries in bytes.
	Size() int64

	// Prune removes expired entries and optionally those exceeding maxAge.
	Prune(maxAge time.Duration) (int, error)

	// Clear removes all cache entries.
	Clear() error

	// List returns all cache entries with keys matching the prefix.
	List(prefix string) []*CacheEntry
}

// FilesystemCache implements the Cache interface using the filesystem.
type FilesystemCache struct {
	dir      string
	manifest *CacheManifest
}

// New creates a new filesystem cache at the specified directory.
func New(dir string) (*FilesystemCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		return nil, err
	}

	return &FilesystemCache{
		dir:      dir,
		manifest: manifest,
	}, nil
}

// Dir returns the cache directory path.
func (c *FilesystemCache) Dir() string {
	return c.dir
}

// Get retrieves a cached item by key.
func (c *FilesystemCache) Get(key string) (string, *CacheEntry, error) {
	entry, ok := c.manifest.Get(key)
	if !ok {
		return "", nil, fmt.Errorf("cache key not found: %s", key)
	}

	if entry.IsExpired() {
		_ = c.Delete(key)
		return "", nil, fmt.Errorf("cache entry expired: %s", key)
	}

	path := filepath.Join(c.dir, entry.Path)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_ = c.Delete(key)
			return "", nil, fmt.Errorf("cache file missing: %s", key)
		}
		return "", nil, err
	}

	return path, entry, nil
}

// Put stores an item in the cache.
func (c *FilesystemCache) Put(key string, source string, reader io.Reader, ttl time.Duration) (*CacheEntry, error) {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return nil, err
	}

	entry := &CacheEntry{
		Source:    source,
		FetchedAt: time.Now(),
	}

	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	finalPath := filepath.Join(c.dir, key)
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return nil, err
	}

	hash := sha256.New()
	tmpFile, err := os.CreateTemp(c.dir, ".cache_tmp_*")
	if err != nil {
		return nil, err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	multiWriter := io.MultiWriter(tmpFile, hash)
	size, err := io.Copy(multiWriter, reader)
	if err != nil {
		tmpFile.Close()
		return nil, err
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return nil, err
	}
	tmpFile.Close()

	entry.Size = size
	entry.Checksum = hex.EncodeToString(hash.Sum(nil))
	entry.Path = key

	if err := os.Rename(tmpName, finalPath); err != nil {
		return nil, err
	}

	c.manifest.Add(key, entry)
	if err := c.manifest.Save(c.dir); err != nil {
		_ = os.Remove(finalPath)
		return nil, err
	}

	return entry, nil
}

// Has checks if a key exists and is not expired.
func (c *FilesystemCache) Has(key string) bool {
	entry, ok := c.manifest.Get(key)
	if !ok {
		return false
	}
	if entry.IsExpired() {
		_ = c.Delete(key)
		return false
	}

	// For normal entries, check if the file exists.
	// We skip this for _git_meta which is a virtual entry stored in the manifest.
	if key != "_git_meta" {
		path := filepath.Join(c.dir, entry.Path)
		if _, err := os.Stat(path); err != nil {
			_ = c.Delete(key)
			return false
		}
	}

	return true
}

// Delete removes a cached item.
func (c *FilesystemCache) Delete(key string) error {
	entry, ok := c.manifest.Get(key)
	if !ok {
		return nil
	}

	path := filepath.Join(c.dir, entry.Path)
	_ = os.Remove(path)

	c.manifest.Delete(key)
	return c.manifest.Save(c.dir)
}

// Validate verifies a cache entry's checksum.
func (c *FilesystemCache) Validate(key string) (bool, error) {
	path, entry, err := c.Get(key)
	if err != nil {
		return false, err
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	valid := checksum == entry.Checksum
	return valid, nil
}

// Refresh updates the fetched_at time for a cache entry.
func (c *FilesystemCache) Refresh(key string) error {
	entry, ok := c.manifest.Get(key)
	if !ok {
		return fmt.Errorf("cache key not found: %s", key)
	}

	entry.FetchedAt = time.Now()
	return c.manifest.Save(c.dir)
}

// Size returns the total size of all cache entries.
func (c *FilesystemCache) Size() int64 {
	return c.manifest.TotalSize()
}

// Prune removes expired entries and optionally those exceeding maxAge.
func (c *FilesystemCache) Prune(maxAge time.Duration) (int, error) {
	var keysToDelete []string
	cutoff := time.Now().Add(-maxAge)

	for key, entry := range c.manifest.Entries {
		if entry.IsExpired() {
			keysToDelete = append(keysToDelete, key)
			continue
		}
		if maxAge > 0 && entry.FetchedAt.Before(cutoff) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	count := 0
	for _, key := range keysToDelete {
		if err := c.Delete(key); err == nil {
			count++
		}
	}

	return count, nil
}

// Clear removes all cache entries.
func (c *FilesystemCache) Clear() error {
	for key := range c.manifest.Entries {
		entry := c.manifest.Entries[key]
		path := filepath.Join(c.dir, entry.Path)
		_ = os.Remove(path)
	}

	c.manifest = NewCacheManifest()
	return c.manifest.Save(c.dir)
}

// List returns all cache entries with keys matching the prefix.
func (c *FilesystemCache) List(prefix string) []*CacheEntry {
	var entries []*CacheEntry
	for key, entry := range c.manifest.Entries {
		if prefix == "" || matchPrefix(key, prefix) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func matchPrefix(key, prefix string) bool {
	if len(prefix) > len(key) {
		return false
	}
	return key[:len(prefix)] == prefix
}

// ModuleKey returns the cache key for a Go module at version.
// Format: go/modules/{module}@{version}
func ModuleKey(module, version string) string {
	return fmt.Sprintf("go/modules/%s@%s", module, version)
}

// StdlibKey returns the cache key for a Go stdlib package at version.
// Format: go/stdlib/{version}/{package}
func StdlibKey(version, pkg string) string {
	return fmt.Sprintf("go/stdlib/%s/%s", version, pkg)
}

// AtprotoKey returns the cache key for an AT Protocol repository.
// Format: atproto/{repo}
func AtprotoKey(repo string) string {
	return fmt.Sprintf("atproto/%s", repo)
}
