package cache

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

const gitMetadataKey = "_git_meta"

// GitCache provides caching for git repositories using commit SHAs.
type GitCache struct {
	cache *FilesystemCache
}

type gitMetadata struct {
	Commits map[string]string `json:"commits"`
}

// NewGitCache creates a new git-aware cache wrapper.
func NewGitCache(dir string) (*GitCache, error) {
	c, err := New(dir)
	if err != nil {
		return nil, err
	}
	return &GitCache{cache: c}, nil
}

// GetCommit retrieves a cached commit SHA for a repository.
func (gc *GitCache) GetCommit(key string) (string, bool) {
	meta, err := gc.loadMetadata()
	if err != nil {
		return "", false
	}

	commit, ok := meta.Commits[key]
	if !ok {
		return "", false
	}

	return commit, true
}

// PutCommit stores a commit SHA for a repository.
func (gc *GitCache) PutCommit(key, source, commitSHA string, ttl time.Duration) error {
	meta, err := gc.loadMetadata()
	if err != nil {
		meta = &gitMetadata{Commits: make(map[string]string)}
	}

	meta.Commits[key] = commitSHA
	return gc.saveMetadata(meta)
}

// loadMetadata loads git metadata from the cache manifest.
func (gc *GitCache) loadMetadata() (*gitMetadata, error) {
	entry, ok := gc.cache.manifest.Get(gitMetadataKey)
	if !ok {
		return &gitMetadata{Commits: make(map[string]string)}, nil
	}

	var meta gitMetadata
	if err := json.Unmarshal([]byte(entry.Source), &meta); err != nil {
		return &gitMetadata{Commits: make(map[string]string)}, nil
	}

	if meta.Commits == nil {
		meta.Commits = make(map[string]string)
	}

	return &meta, nil
}

// saveMetadata saves git metadata to the cache manifest.
func (gc *GitCache) saveMetadata(meta *gitMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	gc.cache.manifest.Add(gitMetadataKey, &CacheEntry{
		Path:      gitMetadataKey,
		Source:    string(data),
		FetchedAt: time.Now(),
	})

	return gc.cache.manifest.Save(gc.cache.Dir())
}

// Count returns the number of cached git repositories.
func (gc *GitCache) Count() int {
	meta, err := gc.loadMetadata()
	if err != nil {
		return 0
	}
	return len(meta.Commits)
}

// ListCommits returns all cached commit SHAs indexed by key.
func (gc *GitCache) ListCommits() map[string]string {
	meta, err := gc.loadMetadata()
	if err != nil {
		return make(map[string]string)
	}
	return meta.Commits
}

// GetRepoCommit fetches the current commit SHA of a git repository.
func GetRepoCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ShallowClone clones a repository to a specific commit SHA.
func ShallowClone(url, commit, dest string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, dest)
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "checkout", commit)
	cmd.Dir = dest
	return cmd.Run()
}
