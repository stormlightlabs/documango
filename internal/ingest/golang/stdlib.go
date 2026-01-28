package golang

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/db"
)

const (
	stdlibURL      = "https://pkg.go.dev/std"
	gitilesArchive = "https://go.googlesource.com/go/+archive/%s/src/%s.tar.gz"
)

type StdlibOptions struct {
	DB          *db.Store
	Version     string
	Start       string
	MaxPackages int
	Cache       *cache.FilesystemCache
}

func IngestStdlib(ctx context.Context, opts StdlibOptions) error {
	if opts.DB == nil {
		return errors.New("db store is required")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	fetch := &fetcher{
		client:       client,
		minInterval:  1 * time.Second,
		minRetryWait: 2 * time.Second,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	doc, err := fetchHTML(ctx, fetch, stdlibURL)
	if err != nil {
		return err
	}

	version := opts.Version
	if version == "" {
		version, err = extractStdlibVersion(doc)
		if err != nil {
			return err
		}
	}

	packages := extractStdlibPackages(doc)
	if len(packages) == 0 {
		return errors.New("no stdlib packages found")
	}
	packages = filterStdlibPackages(packages, opts.Start, opts.MaxPackages)
	if len(packages) == 0 {
		return errors.New("no stdlib packages selected")
	}
	log.Info("stdlib ingest starting", "version", version, "packages", len(packages), "start", opts.Start, "max", opts.MaxPackages)

	root, err := os.MkdirTemp("", "documango-stdlib-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		for _, pkg := range packages {
			log.Info("ingesting stdlib package", "path", pkg)
			pkgDir := filepath.Join(root, "src", filepath.FromSlash(pkg))
			if err := os.MkdirAll(pkgDir, 0o755); err != nil {
				return err
			}

			if err := fetchArchive(ctx, fetch, version, pkg, pkgDir, opts.Cache); err != nil {
				return fmt.Errorf("%s: %w", pkg, err)
			}

			docPath := "go/" + pkg
			if err := IngestPackageDir(ctx, tx, pkg, root, pkgDir, docPath); err != nil {
				return fmt.Errorf("%s: %w", pkg, err)
			}
		}
		return nil
	})
}

func fetchHTML(ctx context.Context, fetch *fetcher, url string) (*goquery.Document, error) {
	resp, err := fetch.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return goquery.NewDocumentFromReader(resp.Body)
}

func extractStdlibVersion(doc *goquery.Document) (string, error) {
	var version string
	if sel := doc.Find(".js-canonicalURLPath"); sel.Length() > 0 {
		if path, ok := sel.Attr("data-canonical-url-path"); ok {
			if idx := strings.Index(path, "@"); idx != -1 {
				version = strings.TrimPrefix(path[idx+1:], "v")
			}
		}
	}
	if version == "" {
		if sel := doc.Find("[data-test-id='UnitHeader-breadcrumbCurrent']"); sel.Length() > 0 {
			if href, ok := sel.Attr("href"); ok {
				if idx := strings.Index(href, "@"); idx != -1 {
					version = strings.TrimPrefix(href[idx+1:], "v")
				}
			}
		}
	}
	doc.Find("a").EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		text := strings.TrimSpace(sel.Text())
		if strings.HasPrefix(text, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(text, "Version:"))
			return false
		}
		return true
	})
	if version == "" {
		text := doc.Text()
		re := regexp.MustCompile(`Version:\s*(go[0-9.]+)`)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			version = matches[1]
		}
	}
	if version == "" {
		return "", errors.New("unable to detect stdlib version")
	}
	return version, nil
}

func extractStdlibPackages(doc *goquery.Document) []string {
	packages := map[string]struct{}{}
	doc.Find("table.UnitDirectories-table tr[data-id]").Each(func(_ int, sel *goquery.Selection) {
		dataID, ok := sel.Attr("data-id")
		if !ok || dataID == "" {
			return
		}
		path := strings.ReplaceAll(dataID, "-", "/")
		path = strings.TrimPrefix(path, "/")
		if path == "" || path == "std" {
			return
		}
		packages[path] = struct{}{}
	})

	list := make([]string, 0, len(packages))
	for pkg := range packages {
		list = append(list, pkg)
	}
	sort.Strings(list)
	return list
}

func filterStdlibPackages(packages []string, start string, max int) []string {
	if start == "" && max <= 0 {
		return packages
	}
	filtered := packages
	if start != "" {
		found := false
		filtered = make([]string, 0, len(packages))
		for _, pkg := range packages {
			if pkg == start {
				found = true
			}
			if !found {
				continue
			}
			filtered = append(filtered, pkg)
		}
	}
	if max > 0 && len(filtered) > max {
		filtered = filtered[:max]
	}
	return filtered
}

func fetchArchive(ctx context.Context, fetch *fetcher, version, pkg, destDir string, c *cache.FilesystemCache) error {
	cacheKey := cache.StdlibKey(version, pkg)

	if c != nil {
		if cachedPath, _, err := c.Get(cacheKey); err == nil {
			log.Info("using cached stdlib package", "package", pkg, "version", version)
			return extractTarGz(cachedPath, destDir)
		}
	}

	url := fmt.Sprintf(gitilesArchive, version, pkg)
	resp, err := fetch.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "documango-stdlib-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if c != nil {
		_, err := c.Put(cacheKey, url, tmpFile, 0)
		_ = tmpFile.Close()
		if err != nil {
			log.Warn("failed to cache stdlib package", "package", pkg, "err", err)
		}
		if cachedPath, _, err := c.Get(cacheKey); err == nil {
			return extractTarGz(cachedPath, destDir)
		}
	} else {
		if err := tmpFile.Close(); err != nil {
			return err
		}
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return err
	}
	return extractTarGz(tmpFile.Name(), destDir)
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(header.Name, "/")
		if name == "" {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(name))
		if !strings.HasPrefix(target, destDir+string(os.PathSeparator)) {
			return fmt.Errorf("invalid archive path: %s", header.Name)
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
}

type fetcher struct {
	client       *http.Client
	minInterval  time.Duration
	minRetryWait time.Duration
	mu           sync.Mutex
	lastRequest  time.Time
	rand         *rand.Rand
}

func (f *fetcher) get(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		f.throttle()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
			f.sleepBackoff(attempt)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			retryAfter := retryAfterDelay(resp)
			log.Warn(
				"request throttled, retrying",
				"url", url,
				"status", resp.Status,
				"retry_after", retryAfter,
				"attempt", attempt+1,
			)
			_ = resp.Body.Close()
			f.sleepRetry(retryAfter, attempt)
			lastErr = fmt.Errorf("request failed: %s", resp.Status)
			continue
		}
		defer resp.Body.Close()
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("request failed")
}

func (f *fetcher) throttle() {
	if f.minInterval <= 0 {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	wait := f.minInterval - time.Since(f.lastRequest)
	if wait > 0 {
		time.Sleep(wait)
	}
	f.lastRequest = time.Now()
}

func (f *fetcher) sleepBackoff(attempt int) {
	base := time.Second * time.Duration(1<<attempt)
	if base < f.minRetryWait {
		base = f.minRetryWait
	}
	time.Sleep(f.jitter(base))
}

func (f *fetcher) sleepRetry(retryAfter time.Duration, attempt int) {
	if retryAfter > 0 {
		if retryAfter < f.minRetryWait {
			retryAfter = f.minRetryWait
		}
		time.Sleep(f.jitter(retryAfter))
		return
	}
	f.sleepBackoff(attempt)
}

func retryAfterDelay(resp *http.Response) time.Duration {
	retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if retryAfter == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}
	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}
	return 0
}

func (f *fetcher) jitter(d time.Duration) time.Duration {
	if f.rand == nil || d <= 0 {
		return d
	}
	factor := 0.8 + f.rand.Float64()*0.4
	return time.Duration(float64(d) * factor)
}
