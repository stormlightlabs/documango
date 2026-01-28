package github

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/shared"
)

type Options struct {
	Owner  string
	Repo   string
	Branch string
	DB     *db.Store
	Cache  *cache.FilesystemCache
}

type repoMetadata struct {
	DefaultBranch string `json:"default_branch"`
}

type treeResponse struct {
	Sha       string `json:"sha"`
	Truncated bool   `json:"truncated"`
	Tree      []treeEntry
}

type treeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	Sha  string `json:"sha"`
	Size int    `json:"size,omitempty"`
}

type frontMatter struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type httpClient struct {
	client    *http.Client
	remaining int
	resetAt   time.Time
}

const (
	maxRetries = 3
	retryDelay = 1 * time.Second
)

func IngestRepository(ctx context.Context, opts Options) error {
	if opts.Owner == "" {
		return errors.New("owner is required")
	}
	if opts.Repo == "" {
		return errors.New("repo is required")
	}
	if opts.DB == nil {
		return errors.New("db store is required")
	}

	log.Info("github repository ingest starting", "owner", opts.Owner, "repo", opts.Repo)

	httpClient := &httpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	metadata, err := fetchRepoMetadata(ctx, httpClient, opts.Owner, opts.Repo)
	if err != nil {
		return err
	}

	branch := opts.Branch
	if branch == "" {
		branch = metadata.DefaultBranch
	}

	tree, truncated, err := fetchTree(ctx, httpClient, opts.Owner, opts.Repo, branch)
	if err != nil {
		return err
	}

	var markdownFiles []string

	if truncated {
		log.Info("tree truncated, falling back to clone", "repo", fmt.Sprintf("%s/%s", opts.Owner, opts.Repo))

		tmpDir, cleanup, err := cloneRepository(ctx, opts.Owner, opts.Repo, branch, opts.Cache)
		if err != nil {
			return err
		}
		defer cleanup()

		markdownFiles, err = walkMarkdownFiles(tmpDir)
		if err != nil {
			return err
		}

		if len(markdownFiles) == 0 {
			return fmt.Errorf("no markdown files found in repository")
		}

		return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
			return processMarkdownFiles(ctx, tx, tmpDir, markdownFiles, fmt.Sprintf("%s/%s", opts.Owner, opts.Repo))
		})
	}

	for _, entry := range tree {
		if entry.Type == "blob" && isMarkdownFile(entry.Path) {
			markdownFiles = append(markdownFiles, entry.Path)
		}
	}

	if len(markdownFiles) == 0 {
		return fmt.Errorf("no markdown files found in repository")
	}

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		return processMarkdownFromAPI(ctx, tx, httpClient, opts.Owner, opts.Repo, branch, markdownFiles)
	})
}

func fetchRepoMetadata(ctx context.Context, client *httpClient, owner, repo string) (*repoMetadata, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "documango (https://github.com/stormlightlabs/documango)")
	req.Header.Set("Accept", "application/vnd.github+json")

	var metadata repoMetadata
	if err := client.do(ctx, req, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func fetchTree(ctx context.Context, client *httpClient, owner, repo, ref string) ([]treeEntry, bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "documango (https://github.com/stormlightlabs/documango)")
	req.Header.Set("Accept", "application/vnd.github+json")

	var tree treeResponse
	if err := client.do(ctx, req, &tree); err != nil {
		return nil, false, err
	}

	return tree.Tree, tree.Truncated, nil
}

func fetchRawContent(ctx context.Context, client *httpClient, owner, repo, ref, path string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "documango (https://github.com/stormlightlabs/documango)")

	resp, err := client.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch raw content: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (c *httpClient) do(ctx context.Context, req *http.Request, result any) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		c.updateRateLimits(resp)

		if resp.StatusCode == http.StatusForbidden && c.remaining <= 0 {
			resp.Body.Close()
			waitUntil := time.Until(c.resetAt)
			if waitUntil > 0 {
				log.Info("rate limit exceeded, waiting", "until", c.resetAt, "wait_seconds", waitUntil.Seconds())
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(waitUntil):
				}
			}
			continue
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("github api error: %s: %s", resp.Status, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}

		resp.Body.Close()
		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *httpClient) updateRateLimits(resp *http.Response) {
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if n, err := strconv.Atoi(remaining); err == nil {
			c.remaining = n
		}
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if sec, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.resetAt = time.Unix(sec, 0)
		}
	}
}

func cloneRepository(ctx context.Context, owner, repo, branch string, c *cache.FilesystemCache) (string, func(), error) {
	cacheKey := cache.GithubRepoKey(owner, repo, branch)
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	if c != nil {
		cachedDir := filepath.Join(c.Dir(), cacheKey)
		if _, err := os.Stat(cachedDir); err == nil {
			log.Info("using cached repository", "owner", owner, "repo", repo)
			return cachedDir, func() {}, nil
		}
	}

	tmpDir, err := os.MkdirTemp("", "documango-github-clone-")
	if err != nil {
		return "", nil, err
	}

	args := []string{"clone", "--depth", "1", "--single-branch"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repoURL, tmpDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, fmt.Errorf("git clone failed: %w", err)
	}

	if c != nil {
		cachedDir := filepath.Join(c.Dir(), cacheKey)
		if err := os.MkdirAll(filepath.Dir(cachedDir), 0o755); err == nil {
			if err := os.Rename(tmpDir, cachedDir); err == nil {
				log.Info("cached repository", "owner", owner, "repo", repo, "path", cacheKey)
				return cachedDir, func() {}, nil
			}
		}
		log.Warn("failed to cache repository, using temp dir", "owner", owner, "repo", repo)
	}

	return tmpDir, func() { _ = os.RemoveAll(tmpDir) }, nil
}

func walkMarkdownFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if isMarkdownFile(relPath) {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func isMarkdownFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".md") || strings.HasSuffix(strings.ToLower(path), ".markdown")
}

func processMarkdownFromAPI(ctx context.Context, tx *sql.Tx, client *httpClient, owner, repo, branch string, paths []string) error {
	repoPrefix := fmt.Sprintf("github/%s/%s", owner, repo)

	for _, path := range paths {
		content, err := fetchRawContent(ctx, client, owner, repo, branch, path)
		if err != nil {
			log.Warn("failed to fetch content", "path", path, "err", err)
			continue
		}

		if err := processMarkdownContent(ctx, tx, content, repoPrefix, path); err != nil {
			log.Warn("failed to process markdown", "path", path, "err", err)
			continue
		}
	}

	return nil
}

func processMarkdownFiles(ctx context.Context, tx *sql.Tx, rootDir string, paths []string, repoName string) error {
	repoPrefix := fmt.Sprintf("github/%s", repoName)

	for _, path := range paths {
		fullPath := filepath.Join(rootDir, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Warn("failed to read file", "path", path, "err", err)
			continue
		}

		if err := processMarkdownContent(ctx, tx, string(content), repoPrefix, path); err != nil {
			log.Warn("failed to process markdown", "path", path, "err", err)
			continue
		}
	}

	return nil
}

func processMarkdownContent(ctx context.Context, tx *sql.Tx, content, repoPrefix, docPath string) error {
	title, contentWithoutFrontMatter := extractTitleAndContent(content)

	docID, err := db.InsertDocumentTx(ctx, tx, db.Document{
		Path:   repoPrefix + "/" + docPath,
		Format: "markdown",
		Body:   shared.Compress(contentWithoutFrontMatter),
		Hash:   db.HashBytes([]byte(contentWithoutFrontMatter)),
	})
	if err != nil {
		return err
	}

	if title == "" {
		title = titleFromPath(docPath)
	}

	if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
		Name:  title,
		Type:  "Document",
		Body:  title + " " + shared.FirstLine(contentWithoutFrontMatter),
		DocID: docID,
	}); err != nil {
		return err
	}

	return nil
}

func extractTitleAndContent(content string) (string, string) {
	content = shared.NormalizeLineEndings(content)

	lines := strings.Split(content, "\n")

	if len(lines) >= 3 && strings.TrimSpace(lines[0]) == "---" {
		endIdx := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIdx = i
				break
			}
		}

		if endIdx > 0 {
			frontMatterText := strings.Join(lines[1:endIdx], "\n")
			var fm frontMatter
			if err := yaml.Unmarshal([]byte(frontMatterText), &fm); err == nil && fm.Title != "" {
				remaining := strings.Join(lines[endIdx+1:], "\n")
				return fm.Title, strings.TrimSpace(remaining)
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			return title, content
		}
	}

	return "", content
}

func titleFromPath(path string) string {
	base := filepath.Base(path)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	title = cases.Title(language.Und).String(title)
	return title
}
