package rust

import (
	"archive/zip"
	"compress/bzip2"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/log"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/shared"
)

type Options struct {
	Crate   string
	Version string
	DB      *db.Store
	Cache   *cache.FilesystemCache
}

type cratesioResponse struct {
	Crate struct {
		Name    string `json:"name"`
		Version string `json:"max_version"`
	} `json:"crate"`
}

type sidebarItems struct {
	Modules   []string `json:"mod"`
	Structs   []string `json:"struct"`
	Enums     []string `json:"enum"`
	Traits    []string `json:"trait"`
	Funcs     []string `json:"fn"`
	TypeDefs  []string `json:"type"`
	Constants []string `json:"constant"`
	Statics   []string `json:"static"`
}

var targetPreference = []string{
	"x86_64-unknown-linux-gnu",
	"x86_64-apple-darwin",
	"aarch64-unknown-linux-gnu",
}

func IngestCrate(ctx context.Context, opts Options) error {
	if opts.Crate == "" {
		return errors.New("crate name is required")
	}
	if opts.DB == nil {
		return errors.New("db store is required")
	}

	version := opts.Version
	if version == "" {
		var err error
		version, err = fetchLatestVersion(ctx, opts.Crate)
		if err != nil {
			return err
		}
	}

	tmpDir, cleanup, err := downloadDocs(ctx, opts.Crate, version, opts.Cache)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Info("rust crate ingest starting", "crate", opts.Crate, "version", version)

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		target, err := selectTarget(tmpDir)
		if err != nil {
			return err
		}

		crateName := strings.ReplaceAll(opts.Crate, "-", "_")
		crateDir := filepath.Join(tmpDir, target, crateName)
		if _, err := os.Stat(crateDir); os.IsNotExist(err) {
			return fmt.Errorf("crate directory not found: %s", crateDir)
		}

		return ingestCrateDir(ctx, tx, opts.Crate, version, crateDir)
	})
}

func fetchLatestVersion(ctx context.Context, crate string) (string, error) {
	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s", crate)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "documango (https://github.com/stormlightlabs/documango)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("crates.io api error: %s", resp.Status)
	}

	var payload cratesioResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.Crate.Version == "" {
		return "", fmt.Errorf("no version found for crate %s", crate)
	}

	return payload.Crate.Version, nil
}

func downloadDocs(ctx context.Context, crate, version string, c *cache.FilesystemCache) (string, func(), error) {
	cacheKey := cache.RustCrateKey(crate, version)
	var zipPath string

	if c != nil {
		if cached, _, err := c.Get(cacheKey); err == nil {
			zipPath = cached
		}
	}

	if zipPath == "" {
		url := fmt.Sprintf("https://docs.rs/crate/%s/%s/download", crate, version)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", nil, err
		}
		req.Header.Set("User-Agent", "documango (https://github.com/stormlightlabs/documango)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", nil, fmt.Errorf("docs.rs download error: %s", resp.Status)
		}

		tmpFile, err := os.CreateTemp("", "documango-rust-*.zip")
		if err != nil {
			return "", nil, err
		}
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			return "", nil, fmt.Errorf("failed to write zip: %w", err)
		}

		if c != nil {
			if _, err := tmpFile.Seek(0, 0); err != nil {
				return "", nil, err
			}
			entry, err := c.Put(cacheKey, url, tmpFile, 0)
			if err != nil {
				log.Warn("failed to cache rust crate", "crate", crate, "err", err)
			} else {
				zipPath = filepath.Join(c.Dir(), entry.Path)
			}
		}

		if zipPath == "" {
			zipPath = tmpFile.Name()
		}
	}

	tmpDir, err := os.MkdirTemp("", "documango-rust-extract-")
	if err != nil {
		if c == nil {
			_ = os.Remove(zipPath)
		}
		return "", nil, err
	}

	if err := unzipRustdoc(zipPath, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		if c == nil {
			_ = os.Remove(zipPath)
		}
		return "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
		if c == nil {
			_ = os.Remove(zipPath)
		}
	}

	return tmpDir, cleanup, nil
}

func init() {
	zip.RegisterDecompressor(12, func(r io.Reader) io.ReadCloser {
		return &bzip2ReadCloser{reader: bzip2.NewReader(r)}
	})
}

type bzip2ReadCloser struct {
	reader io.Reader
}

func (b *bzip2ReadCloser) Read(p []byte) (n int, err error) {
	return b.reader.Read(p)
}

func (b *bzip2ReadCloser) Close() error {
	return nil
}

func unzipRustdoc(zipPath, dest string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(dest, file.Name)
		if !strings.HasPrefix(path, dest+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip path: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer out.Close()

		written, err := io.Copy(out, rc)
		if err != nil {
			return err
		}
		if written == 0 && file.UncompressedSize64 > 0 {
			return fmt.Errorf("decompression failed: %s (method %d, size %d)", file.Name, file.Method, file.UncompressedSize64)
		}
	}
	return nil
}

func selectTarget(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}

	targets := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			targets = append(targets, entry.Name())
		}
	}

	for _, pref := range targetPreference {
		for _, t := range targets {
			if strings.HasPrefix(t, pref) {
				return t, nil
			}
		}
	}

	if len(targets) == 0 {
		return "", errors.New("no targets found in rustdoc archive")
	}

	return targets[0], nil
}

func ingestCrateDir(ctx context.Context, tx *sql.Tx, crate, version, crateDir string) error {
	sidebarPath := findSidebarItems(crateDir)
	if sidebarPath == "" {
		return fmt.Errorf("sidebar-items.js not found in %s", crateDir)
	}

	sidebarData, err := os.ReadFile(sidebarPath)
	if err != nil {
		return err
	}

	var items sidebarItems
	jsContent := strings.TrimPrefix(string(sidebarData), "window.SIDEBAR_ITEMS = ")
	jsContent = strings.TrimSuffix(jsContent, ";")
	if err := json.Unmarshal([]byte(jsContent), &items); err != nil {
		return fmt.Errorf("failed to parse sidebar items: %w", err)
	}

	var allItems []docItem
	for _, m := range items.Modules {
		allItems = append(allItems, docItem{Name: m, Type: "Module"})
	}
	for _, s := range items.Structs {
		allItems = append(allItems, docItem{Name: s, Type: "Struct"})
	}
	for _, e := range items.Enums {
		allItems = append(allItems, docItem{Name: e, Type: "Enum"})
	}
	for _, t := range items.Traits {
		allItems = append(allItems, docItem{Name: t, Type: "Trait"})
	}
	for _, f := range items.Funcs {
		allItems = append(allItems, docItem{Name: f, Type: "Function"})
	}
	for _, t := range items.TypeDefs {
		allItems = append(allItems, docItem{Name: t, Type: "Type"})
	}
	for _, c := range items.Constants {
		allItems = append(allItems, docItem{Name: c, Type: "Constant"})
	}
	for _, s := range items.Statics {
		allItems = append(allItems, docItem{Name: s, Type: "Static"})
	}

	crateIndexPath := filepath.Join(crateDir, "index.html")
	crateDoc, err := parseRustdocHTML(crateIndexPath)
	if err == nil && crateDoc != "" {
		log.Info("inserting crate index", "path", crateIndexPath, "doc_length", len(crateDoc))
		docID, err := insertDoc(ctx, tx, crate, version, "", crateDoc)
		if err != nil {
			log.Error("failed to insert crate index", "path", crateIndexPath, "err", err)
			return err
		}

		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  crate,
			Type:  "Crate",
			Body:  crate + " " + shared.FirstLine(crateDoc),
			DocID: docID,
		}); err != nil {
			return err
		}
	} else {
		log.Warn("failed to parse crate index", "err", err)
	}

	processedCount := 0
	for _, item := range allItems {
		var htmlPath string
		switch item.Type {
		case "Module":
			htmlPath = filepath.Join(crateDir, item.Name, "index.html")
		case "Struct":
			htmlPath = filepath.Join(crateDir, "struct."+item.Name+".html")
		case "Enum":
			htmlPath = filepath.Join(crateDir, "enum."+item.Name+".html")
		case "Trait":
			htmlPath = filepath.Join(crateDir, "trait."+item.Name+".html")
		case "Function":
			htmlPath = filepath.Join(crateDir, "fn."+item.Name+".html")
		case "Type":
			htmlPath = filepath.Join(crateDir, "type."+item.Name+".html")
		case "Constant":
			htmlPath = filepath.Join(crateDir, "constant."+item.Name+".html")
		case "Static":
			htmlPath = filepath.Join(crateDir, "static."+item.Name+".html")
		default:
			continue
		}

		if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
			continue
		}

		markdown, err := parseRustdocHTML(htmlPath)
		if err != nil {
			log.Warn("failed to parse rustdoc", "file", htmlPath, "err", err)
			continue
		}

		if markdown == "" {
			continue
		}

		processedCount++
		docPath := fmt.Sprintf("rust/%s/%s/%s", crate, item.Type, item.Name)
		docID, err := insertDoc(ctx, tx, crate, version, docPath, markdown)
		if err != nil {
			log.Error("failed to insert doc", "path", docPath, "err", err)
			return err
		}

		fullName := crate + "::" + item.Name
		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  fullName,
			Type:  item.Type,
			Body:  fullName + " " + shared.FirstLine(markdown),
			DocID: docID,
		}); err != nil {
			return err
		}

		signature := extractSignature(markdown)
		if signature != "" {
			if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
				DocID:     docID,
				Symbol:    fullName,
				Signature: signature,
				Summary:   shared.FirstLine(markdown),
			}); err != nil {
				return err
			}
		}
	}

	log.Info("ingestion complete", "processed", processedCount, "total", len(allItems))
	return nil
}

type docItem struct {
	Name string
	Type string
}

func findSidebarItems(crateDir string) string {
	matches, _ := filepath.Glob(filepath.Join(crateDir, "sidebar-items*.js"))
	if len(matches) > 0 {
		return matches[0]
	}

	entries, err := os.ReadDir(crateDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(crateDir, entry.Name())
			subMatches, _ := filepath.Glob(filepath.Join(subDir, "sidebar-items*.js"))
			if len(subMatches) > 0 {
				return subMatches[0]
			}
		}
	}

	return ""
}

func parseRustdocHTML(htmlPath string) (string, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}

	var lines []string

	doc.Find("main").Each(func(i int, s *goquery.Selection) {
		s.Find("h1").Each(func(i int, h *goquery.Selection) {
			title := h.Text()
			title = strings.TrimSpace(title)
			if title != "" {
				lines = append(lines, "# "+title)
			}
		})

		s.Find("pre.rust.item-decl").Each(func(i int, pre *goquery.Selection) {
			code := pre.Text()
			code = strings.TrimSpace(code)
			if code != "" {
				lines = append(lines, "```rust")
				lines = append(lines, code)
				lines = append(lines, "```")
			}
		})

		s.Find(".docblock").Each(func(i int, d *goquery.Selection) {
			text := d.Text()
			text = strings.TrimSpace(text)
			if text != "" {
				lines = append(lines, text)
			}
		})

		s.Find("h2").Each(func(i int, h *goquery.Selection) {
			title := h.Text()
			title = strings.TrimSpace(title)
			if title != "" {
				lines = append(lines, "")
				lines = append(lines, "## "+title)
			}
		})
	})

	result := strings.Join(lines, "\n")
	result = strings.TrimSpace(result)
	if result == "" {
		return "", nil
	}

	return result, nil
}

func extractSignature(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```rust") {
			continue
		}
		if strings.HasPrefix(trimmed, "pub ") || strings.HasPrefix(trimmed, "fn ") ||
			strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "enum ") ||
			strings.HasPrefix(trimmed, "trait ") || strings.HasPrefix(trimmed, "type ") {
			return strings.TrimSuffix(strings.TrimPrefix(trimmed, "pub "), ";")
		}
	}
	return ""
}

func insertDoc(ctx context.Context, tx *sql.Tx, crate, version, docPath, markdown string) (int64, error) {
	if docPath == "" {
		docPath = "rust/" + crate + "/index"
	}

	fullDoc := fmt.Sprintf("# %s\n\nVersion: %s\n\n%s", crate, version, markdown)
	compressed, err := codec.Compress([]byte(fullDoc))
	if err != nil {
		return 0, err
	}

	return db.InsertDocumentTx(ctx, tx, db.Document{
		Path:   docPath,
		Format: "markdown",
		Body:   compressed,
		Hash:   db.HashBytes([]byte(fullDoc)),
	})
}
