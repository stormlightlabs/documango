// package golang contains the ingestion pipeline for Go docs.
//
// TODO: Add a Go-specific filter option to exclude unexported symbols from the
// search index. I think Go devs know the difference between exported (uppercase)
// and unexported (lowercase) symbols, so including both is useful for context.
// However, for a more focused public API view, we could add a flag to filter
// out unexported symbols.
package golang

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/princjef/gomarkdoc"
	"github.com/princjef/gomarkdoc/lang"
	"github.com/princjef/gomarkdoc/logger"
	"golang.org/x/mod/module"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
)

type Options struct {
	Module  string
	Version string
	DB      *db.Store
	Cache   *cache.FilesystemCache
}

type latestResponse struct {
	Version string `json:"Version"`
}

func IngestModule(ctx context.Context, opts Options) error {
	if opts.Module == "" {
		return errors.New("module is required")
	}
	if opts.DB == nil {
		return errors.New("db store is required")
	}

	version := opts.Version
	if version == "" {
		var err error
		version, err = fetchLatestVersion(ctx, opts.Module)
		if err != nil {
			return err
		}
	}

	root, cleanup, err := downloadModuleZip(ctx, opts.Module, version, opts.Cache)
	if err != nil {
		return err
	}
	defer cleanup()

	packages, err := discoverPackages(root)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		return fmt.Errorf("no packages found in %s@%s", opts.Module, version)
	}
	log.Info("go module ingest starting", "module", opts.Module, "version", version, "packages", len(packages))

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		for _, pkgDir := range packages {
			if err := ingestPackage(ctx, tx, opts.Module, root, pkgDir); err != nil {
				return err
			}
		}
		return nil
	})
}

func fetchLatestVersion(ctx context.Context, modulePath string) (string, error) {
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@latest", escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("module proxy error: %s", resp.Status)
	}
	var payload latestResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Version == "" {
		return "", errors.New("module proxy response missing version")
	}
	return payload.Version, nil
}

func downloadModuleZip(ctx context.Context, modulePath, version string, c *cache.FilesystemCache) (string, func(), error) {
	cacheKey := cache.ModuleKey(modulePath, version)

	if c != nil {
		if cachedPath, entry, err := c.Get(cacheKey); err == nil {
			log.Info("using cached module", "module", modulePath, "version", version, "path", cachedPath)
			extractDir, err := os.MkdirTemp("", "documango-module-")
			if err != nil {
				return "", nil, err
			}
			if err := unzip(cachedPath, extractDir); err != nil {
				_ = os.RemoveAll(extractDir)
				return "", nil, err
			}
			root := extractDir
			moduleDir := filepath.Join(extractDir, filepath.Base(entry.Path))
			if info, err := os.Stat(moduleDir); err == nil && info.IsDir() {
				root = moduleDir
			} else {
				entries, err := os.ReadDir(extractDir)
				if err == nil && len(entries) == 1 && entries[0].IsDir() {
					root = filepath.Join(extractDir, entries[0].Name())
				}
			}
			cleanup := func() {
				_ = os.RemoveAll(extractDir)
			}
			return root, cleanup, nil
		}
	}

	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return "", nil, err
	}
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.zip", escaped, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("module proxy error: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "documango-module-*.zip")
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", nil, err
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", nil, err
	}

	if c != nil {
		_, err := c.Put(cacheKey, url, tmpFile, 0)
		_ = tmpFile.Close()
		if err != nil {
			log.Warn("failed to cache module", "module", modulePath, "err", err)
		}
		if cachedPath, _, err := c.Get(cacheKey); err == nil {
			_ = os.Remove(tmpFile.Name())
			tmpFile, err = os.Open(cachedPath)
			if err != nil {
				return "", nil, err
			}
			defer tmpFile.Close()
		}
	} else {
		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpFile.Name())
			return "", nil, err
		}
	}

	zipPath := tmpFile.Name()
	if c == nil {
		defer os.Remove(zipPath)
	}

	extractDir, err := os.MkdirTemp("", "documango-module-")
	if err != nil {
		if c == nil {
			_ = os.Remove(zipPath)
		}
		return "", nil, err
	}

	if err := unzip(zipPath, extractDir); err != nil {
		_ = os.RemoveAll(extractDir)
		if c == nil {
			_ = os.Remove(zipPath)
		}
		return "", nil, err
	}
	if c == nil {
		_ = os.Remove(zipPath)
	}

	root := extractDir
	moduleDir := filepath.Join(extractDir, escaped+"@"+version)
	if info, err := os.Stat(moduleDir); err == nil && info.IsDir() {
		root = moduleDir
	} else {
		entries, err := os.ReadDir(extractDir)
		if err == nil && len(entries) == 1 && entries[0].IsDir() {
			root = filepath.Join(extractDir, entries[0].Name())
		}
	}

	cleanup := func() {
		_ = os.RemoveAll(extractDir)
	}
	return root, cleanup, nil
}

func unzip(zipPath, dest string) error {
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
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			_ = out.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			return err
		}
		_ = in.Close()
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func discoverPackages(root string) ([]string, error) {
	var packages []string
	seen := map[string]struct{}{}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		if _, ok := seen[dir]; ok {
			return nil
		}
		seen[dir] = struct{}{}
		packages = append(packages, dir)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(packages)
	return packages, nil
}

func buildImportPath(modulePath, moduleRoot, pkgDir string) string {
	rel, err := filepath.Rel(moduleRoot, pkgDir)
	if err != nil || rel == "." {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(rel)
}

func generateMarkdown(pkg *doc.Package, workDir, pkgDir string) (string, error) {
	log := logger.New(logger.ErrorLevel)
	cfg, err := lang.NewConfig(log, workDir, pkgDir)
	if err != nil {
		return "", err
	}
	cfg.Pkg = pkg
	cfg.Symbols = lang.PackageSymbols(pkg)
	examples := doc.Examples(cfg.Files...)

	langPkg := lang.NewPackage(cfg, examples)
	renderer, err := gomarkdoc.NewRenderer()
	if err != nil {
		return "", err
	}
	output, err := renderer.Package(langPkg)
	if err != nil {
		return "", err
	}
	return output, nil
}

func signatureForDecl(fset *token.FileSet, node ast.Node) string {
	var buf strings.Builder
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}

func collectSymbols(pkg *doc.Package, fset *token.FileSet) ([]symbolEntry, []agentEntry) {
	var symbols []symbolEntry
	var agents []agentEntry

	if pkg.Doc != "" {
		summary := doc.Synopsis(pkg.Doc)
		symbols = append(symbols, symbolEntry{
			Name: pkg.Name,
			Type: "Package",
			Body: summaryText(pkg.Doc),
		})
		agents = append(agents, agentEntry{
			Symbol:    pkg.Name,
			Signature: fmt.Sprintf("package %s", pkg.Name),
			Summary:   summary,
		})
	}

	for _, fn := range pkg.Funcs {
		symbols = append(symbols, symbolEntry{
			Name: fn.Name,
			Type: "Func",
			Body: summaryText(fn.Doc),
		})
		agents = append(agents, agentEntry{
			Symbol:    fn.Name,
			Signature: signatureForDecl(fset, fn.Decl),
			Summary:   doc.Synopsis(fn.Doc),
		})
	}

	for _, typ := range pkg.Types {
		symbols = append(symbols, symbolEntry{
			Name: typ.Name,
			Type: "Type",
			Body: summaryText(typ.Doc),
		})
		agents = append(agents, agentEntry{
			Symbol:    typ.Name,
			Signature: signatureForDecl(fset, typ.Decl),
			Summary:   doc.Synopsis(typ.Doc),
		})

		for _, method := range typ.Methods {
			name := typ.Name + "." + method.Name
			symbols = append(symbols, symbolEntry{
				Name: name,
				Type: "Method",
				Body: summaryText(method.Doc),
			})
			agents = append(agents, agentEntry{
				Symbol:    name,
				Signature: signatureForDecl(fset, method.Decl),
				Summary:   doc.Synopsis(method.Doc),
			})
		}
	}

	for _, val := range pkg.Vars {
		for _, name := range val.Names {
			symbols = append(symbols, symbolEntry{
				Name: name,
				Type: "Var",
				Body: summaryText(val.Doc),
			})
			agents = append(agents, agentEntry{
				Symbol:    name,
				Signature: signatureForDecl(fset, val.Decl),
				Summary:   doc.Synopsis(val.Doc),
			})
		}
	}

	for _, val := range pkg.Consts {
		for _, name := range val.Names {
			symbols = append(symbols, symbolEntry{
				Name: name,
				Type: "Const",
				Body: summaryText(val.Doc),
			})
			agents = append(agents, agentEntry{
				Symbol:    name,
				Signature: signatureForDecl(fset, val.Decl),
				Summary:   doc.Synopsis(val.Doc),
			})
		}
	}

	return symbols, agents
}

type symbolEntry struct {
	Name string
	Type string
	Body string
}

type agentEntry struct {
	Symbol    string
	Signature string
	Summary   string
}

func summaryText(text string) string {
	if text == "" {
		return ""
	}
	summary := doc.Synopsis(text)
	if summary == "" {
		return text
	}
	return summary + "\n\n" + text
}

func injectAnchors(markdown string, symbols []symbolEntry) string {
	anchorMap := make(map[string]string, len(symbols))
	for _, sym := range symbols {
		anchorMap[sym.Name] = sym.Name
	}

	lines := strings.Split(markdown, "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "#### ") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			symbol := symbolFromHeading(heading)
			if symbol != "" {
				if anchor, ok := anchorMap[symbol]; ok {
					out = append(out, fmt.Sprintf("<a name=\"%s\"></a>", anchor))
					delete(anchorMap, symbol)
				}
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func symbolFromHeading(heading string) string {
	heading = strings.TrimSpace(heading)
	switch {
	case strings.HasPrefix(heading, "type "):
		fields := strings.Fields(strings.TrimPrefix(heading, "type "))
		if len(fields) > 0 {
			return fields[0]
		}
	case strings.HasPrefix(heading, "func "):
		rest := strings.TrimPrefix(heading, "func ")
		if strings.HasPrefix(rest, "(") {
			idx := strings.Index(rest, ")")
			if idx != -1 {
				receiver := strings.TrimSpace(rest[1:idx])
				recvFields := strings.Fields(receiver)
				recvType := receiver
				if len(recvFields) > 0 {
					recvType = recvFields[len(recvFields)-1]
				}
				recvType = strings.TrimLeft(recvType, "*")
				rest = strings.TrimSpace(rest[idx+1:])
				nameFields := strings.Fields(rest)
				if len(nameFields) > 0 {
					return recvType + "." + nameFields[0]
				}
				return ""
			}
		}
		nameFields := strings.Fields(rest)
		if len(nameFields) > 0 {
			return nameFields[0]
		}
	case strings.HasPrefix(heading, "var "):
		fields := strings.Fields(strings.TrimPrefix(heading, "var "))
		if len(fields) > 0 {
			return fields[0]
		}
	case strings.HasPrefix(heading, "const "):
		fields := strings.Fields(strings.TrimPrefix(heading, "const "))
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

func ingestPackage(ctx context.Context, tx *sql.Tx, modulePath, moduleRoot, pkgDir string) error {
	importPath := buildImportPath(modulePath, moduleRoot, pkgDir)
	docPath := "go/" + importPath
	return IngestPackageDir(ctx, tx, importPath, moduleRoot, pkgDir, docPath)
}

func IngestPackageDir(ctx context.Context, tx *sql.Tx, importPath, workDir, pkgDir, docPath string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, func(info os.FileInfo) bool {
		name := info.Name()
		return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return nil
	}

	pkgNames := make([]string, 0, len(pkgs))
	for name := range pkgs {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)
	astPkg := pkgs[pkgNames[0]]

	pkgDoc := doc.New(astPkg, importPath, doc.AllDecls)

	md, err := generateMarkdown(pkgDoc, workDir, pkgDir)
	if err != nil {
		return err
	}
	symbols, agents := collectSymbols(pkgDoc, fset)
	md = injectAnchors(md, symbols)
	compressed, err := codec.Compress([]byte(md))
	if err != nil {
		return err
	}
	docID, err := db.InsertDocumentTx(ctx, tx, db.Document{
		Path:   docPath,
		Format: "markdown",
		Body:   compressed,
		Hash:   db.HashBytes([]byte(md)),
	})
	if err != nil {
		return err
	}

	for _, sym := range symbols {
		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  sym.Name,
			Type:  sym.Type,
			Body:  sym.Body,
			DocID: docID,
		}); err != nil {
			return err
		}
	}

	for _, agent := range agents {
		if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
			DocID:     docID,
			Symbol:    agent.Symbol,
			Signature: agent.Signature,
			Summary:   agent.Summary,
		}); err != nil {
			return err
		}
	}

	return nil
}
