package hexpm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/shared"
)

type Options struct {
	Package string
	Version string
	DB      *db.Store
	Cache   *cache.FilesystemCache
}

// Gleam package-interface.json structures
type GleamInterface struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version"`
	Modules map[string]GleamModule `json:"modules"`
}

type DocString []string

func (d *DocString) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*d = []string{s}
		return nil
	}
	var sli []string
	if err := json.Unmarshal(data, &sli); err != nil {
		return err
	}
	*d = sli
	return nil
}

func (d DocString) String() string {
	return strings.Join(d, "")
}

type GleamModule struct {
	Documentation DocString                `json:"documentation"`
	Types         map[string]GleamTypeDef  `json:"types"`
	TypeAliases   map[string]GleamAlias    `json:"type-aliases"`
	Functions     map[string]GleamFunction `json:"functions"`
	Constants     map[string]GleamConstant `json:"constants"`
}

type GleamTypeDef struct {
	Documentation DocString          `json:"documentation"`
	Deprecation   *string            `json:"deprecation"`
	Parameters    int                `json:"parameters"`
	Constructors  []GleamConstructor `json:"constructors"`
}

type GleamConstructor struct {
	Name       string       `json:"name"`
	Parameters []GleamParam `json:"parameters"`
}

type GleamAlias struct {
	Documentation DocString     `json:"documentation"`
	Deprecation   *string       `json:"deprecation"`
	Parameters    int           `json:"parameters"`
	Alias         GleamTypeExpr `json:"alias"`
}

type GleamConstant struct {
	Documentation DocString     `json:"documentation"`
	Deprecation   *string       `json:"deprecation"`
	Type          GleamTypeExpr `json:"type"`
}

type GleamFunction struct {
	Documentation DocString     `json:"documentation"`
	Deprecation   *string       `json:"deprecation"`
	Parameters    []GleamParam  `json:"parameters"`
	Return        GleamTypeExpr `json:"return"`
}

type GleamParam struct {
	Label *string       `json:"label"`
	Type  GleamTypeExpr `json:"type"`
}

type GleamTypeExpr struct {
	Kind       string          `json:"kind"`
	Name       string          `json:"name,omitempty"`
	Module     string          `json:"module,omitempty"`
	Package    string          `json:"package,omitempty"`
	Parameters []GleamTypeExpr `json:"parameters,omitempty"`
	Elements   []GleamTypeExpr `json:"elements,omitempty"`
	ID         int             `json:"id,omitempty"`
	Return     *GleamTypeExpr  `json:"return,omitempty"`
}

// Elixir structures
type SearchData struct {
	Items []SearchItem `json:"items"`
}

type SearchItem struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Doc   string `json:"doc"`
	Ref   string `json:"ref"`
}

func IngestPackage(ctx context.Context, opts Options) error {
	if opts.Package == "" {
		return errors.New("package name is required")
	}
	if opts.DB == nil {
		return errors.New("db store is required")
	}

	version := opts.Version
	if version == "" {
		var err error
		version, err = fetchLatestVersion(ctx, opts.Package)
		if err != nil {
			return err
		}
	}

	tmpDir, cleanup, err := downloadDocs(ctx, opts.Package, version, opts.Cache)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Info("hex package ingest starting", "package", opts.Package, "version", version)

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		interfacePath := filepath.Join(tmpDir, "package-interface.json")
		if _, err := os.Stat(interfacePath); err == nil {
			return ingestGleam(ctx, tx, opts.Package, interfacePath)
		}

		return ingestElixir(ctx, tx, opts.Package, tmpDir)
	})
}

func fetchLatestVersion(ctx context.Context, pkg string) (string, error) {
	url := fmt.Sprintf("https://hex.pm/api/packages/%s", pkg)
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
		return "", fmt.Errorf("hex api error: %s", resp.Status)
	}

	var payload struct {
		Releases []struct {
			Version string `json:"version"`
		} `json:"releases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if len(payload.Releases) == 0 {
		return "", fmt.Errorf("no releases found for package %s", pkg)
	}
	return payload.Releases[0].Version, nil
}

func downloadDocs(ctx context.Context, pkg, version string, c *cache.FilesystemCache) (string, func(), error) {
	cacheKey := cache.HexKey(pkg, version)
	var tarPath string

	if c != nil {
		if cached, _, err := c.Get(cacheKey); err == nil {
			tarPath = cached
		}
	}

	if tarPath == "" {
		url := fmt.Sprintf("https://repo.hex.pm/docs/%s-%s.tar.gz", pkg, version)
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
			return "", nil, fmt.Errorf("hex repo error: %s", resp.Status)
		}

		if c != nil {
			entry, err := c.Put(cacheKey, url, resp.Body, 0)
			if err != nil {
				return "", nil, err
			}
			tarPath = filepath.Join(c.Dir(), entry.Path)
		} else {
			f, err := os.CreateTemp("", "documango-hex-*.tar.gz")
			if err != nil {
				return "", nil, err
			}
			defer f.Close()
			if _, err := io.Copy(f, resp.Body); err != nil {
				return "", nil, err
			}
			tarPath = f.Name()
		}
	}

	tmpDir, err := os.MkdirTemp("", "documango-hex-extract-")
	if err != nil {
		return "", nil, err
	}

	if err := untar(tarPath, tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return "", nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
		if c == nil {
			_ = os.Remove(tarPath)
		}
	}

	return tmpDir, cleanup, nil
}

func untar(tarPath, dest string) error {
	f, err := os.Open(tarPath)
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
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			out, err := os.Create(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

func ingestGleam(ctx context.Context, tx *sql.Tx, pkgName string, interfacePath string) error {
	data, err := os.ReadFile(interfacePath)
	if err != nil {
		return err
	}

	var iface GleamInterface
	if err := json.Unmarshal(data, &iface); err != nil {
		return err
	}

	for modName, mod := range iface.Modules {
		docPath := "hex/" + pkgName + "/" + modName

		var docBuilder strings.Builder
		docBuilder.WriteString("# " + modName + "\n\n")
		if modDoc := mod.Documentation.String(); modDoc != "" {
			docBuilder.WriteString(modDoc + "\n\n")
		}

		if len(mod.Types) > 0 {
			docBuilder.WriteString("## Types\n\n")
			for typeName, td := range mod.Types {
				sig := renderGleamTypeDef(typeName, td)
				docBuilder.WriteString("### " + typeName + "\n\n")
				docBuilder.WriteString("```gleam\n" + sig + "\n```\n\n")
				if typeDoc := td.Documentation.String(); typeDoc != "" {
					docBuilder.WriteString(typeDoc + "\n\n")
				}
			}
		}

		if len(mod.TypeAliases) > 0 {
			docBuilder.WriteString("## Type Aliases\n\n")
			for aliasName, ta := range mod.TypeAliases {
				vars := make(map[int]string)
				aliasType := renderGleamType(ta.Alias, vars)
				docBuilder.WriteString("### " + aliasName + "\n\n")
				docBuilder.WriteString("```gleam\ntype " + aliasName + " = " + aliasType + "\n```\n\n")
				if aliasDoc := ta.Documentation.String(); aliasDoc != "" {
					docBuilder.WriteString(aliasDoc + "\n\n")
				}
			}
		}

		if len(mod.Functions) > 0 {
			docBuilder.WriteString("## Functions\n\n")
			for fnName, fn := range mod.Functions {
				sig := renderGleamSignature(fnName, fn)
				docBuilder.WriteString("### " + fnName + "\n\n")
				docBuilder.WriteString("```gleam\n" + sig + "\n```\n\n")
				if fnDoc := fn.Documentation.String(); fnDoc != "" {
					docBuilder.WriteString(fnDoc + "\n\n")
				}
			}
		}

		md := docBuilder.String()
		docID, err := db.InsertDocumentTx(ctx, tx, db.Document{
			Path:   docPath,
			Format: "markdown",
			Body:   compress(md),
			Hash:   db.HashBytes([]byte(md)),
		})
		if err != nil {
			return err
		}

		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  modName,
			Type:  "Module",
			Body:  modName + " " + mod.Documentation.String(),
			DocID: docID,
		}); err != nil {
			return err
		}

		for fnName, fn := range mod.Functions {
			symbol := modName + "." + fnName
			sig := renderGleamSignature(fnName, fn)
			fnDoc := fn.Documentation.String()
			if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
				Name:  symbol,
				Type:  "Function",
				Body:  symbol + " " + sig + " " + fnDoc,
				DocID: docID,
			}); err != nil {
				return err
			}

			if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
				DocID:     docID,
				Symbol:    symbol,
				Signature: sig,
				Summary:   shared.FirstLine(fnDoc),
			}); err != nil {
				return err
			}
		}

		for typeName, td := range mod.Types {
			symbol := modName + "." + typeName
			sig := renderGleamTypeDef(typeName, td)
			typeDoc := td.Documentation.String()
			if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
				Name:  symbol,
				Type:  "Type",
				Body:  symbol + " " + sig + " " + typeDoc,
				DocID: docID,
			}); err != nil {
				return err
			}

			if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
				DocID:     docID,
				Symbol:    symbol,
				Signature: sig,
				Summary:   shared.FirstLine(typeDoc),
			}); err != nil {
				return err
			}
		}

		for aliasName, ta := range mod.TypeAliases {
			symbol := modName + "." + aliasName
			vars := make(map[int]string)
			sig := "type " + aliasName + " = " + renderGleamType(ta.Alias, vars)
			aliasDoc := ta.Documentation.String()
			if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
				Name:  symbol,
				Type:  "TypeAlias",
				Body:  symbol + " " + sig + " " + aliasDoc,
				DocID: docID,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func ingestElixir(ctx context.Context, tx *sql.Tx, pkgName string, tmpDir string) error {
	matches, err := filepath.Glob(filepath.Join(tmpDir, "dist", "search_data-*.js"))
	if err != nil || len(matches) == 0 {
		return errors.New("could not find search_data in doc tarball")
	}
	searchDataPath := matches[0]

	data, err := os.ReadFile(searchDataPath)
	if err != nil {
		return err
	}

	re := regexp.MustCompile(`searchData\s*=\s*({.*})`)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return errors.New("could not parse searchData JS")
	}

	var searchData SearchData
	if err := json.Unmarshal(match[1], &searchData); err != nil {
		return err
	}

	pages := make(map[string][]SearchItem)
	for _, item := range searchData.Items {
		ref := strings.Split(item.Ref, "#")[0]
		pages[ref] = append(pages[ref], item)
	}

	for ref, items := range pages {
		docPath := "hex/" + pkgName + "/" + strings.TrimSuffix(ref, ".html")

		var pageDoc string
		for _, item := range items {
			if !strings.Contains(item.Ref, "#") {
				pageDoc = item.Doc
				break
			}
		}

		if pageDoc == "" && len(items) > 0 {
			pageDoc = items[0].Doc
		}

		docID, err := db.InsertDocumentTx(ctx, tx, db.Document{
			Path:   docPath,
			Format: "markdown",
			Body:   compress(pageDoc),
			Hash:   db.HashBytes([]byte(pageDoc)),
		})
		if err != nil {
			return err
		}

		for _, item := range items {
			name := item.Title
			if item.Type == "task" {
				name = "mix " + name
			}

			if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
				Name:  name,
				Type:  shared.Capitalize(item.Type),
				Body:  name + " " + item.Doc,
				DocID: docID,
			}); err != nil {
				return err
			}

			if item.Type != "module" && item.Type != "extras" {
				if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
					DocID:     docID,
					Symbol:    name,
					Signature: name,
					Summary:   shared.FirstLine(item.Doc),
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func compress(body string) []byte {
	compressed, _ := codec.Compress([]byte(body))
	return compressed
}

// renderGleamType converts a GleamTypeExpr to Gleam type syntax.
func renderGleamType(t GleamTypeExpr, vars map[int]string) string {
	switch t.Kind {
	case "named":
		base := t.Name
		if len(t.Parameters) > 0 {
			params := make([]string, len(t.Parameters))
			for i, p := range t.Parameters {
				params[i] = renderGleamType(p, vars)
			}
			base += "(" + strings.Join(params, ", ") + ")"
		}
		return base
	case "variable":
		if name, ok := vars[t.ID]; ok {
			return name
		}
		varNames := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		if t.ID < len(varNames) {
			return varNames[t.ID]
		}
		return fmt.Sprintf("t%d", t.ID)
	case "fn":
		params := make([]string, len(t.Parameters))
		for i, p := range t.Parameters {
			params[i] = renderGleamType(p, vars)
		}
		ret := "Nil"
		if t.Return != nil {
			ret = renderGleamType(*t.Return, vars)
		}
		return "fn(" + strings.Join(params, ", ") + ") -> " + ret
	case "tuple":
		elems := make([]string, len(t.Elements))
		for i, e := range t.Elements {
			elems[i] = renderGleamType(e, vars)
		}
		return "#(" + strings.Join(elems, ", ") + ")"
	default:
		return "?"
	}
}

// renderGleamSignature builds a Gleam function signature string.
func renderGleamSignature(name string, fn GleamFunction) string {
	vars := make(map[int]string)
	params := make([]string, len(fn.Parameters))
	for i, p := range fn.Parameters {
		typeStr := renderGleamType(p.Type, vars)
		if p.Label != nil && *p.Label != "" {
			params[i] = *p.Label + " " + typeStr
		} else {
			params[i] = typeStr
		}
	}
	ret := renderGleamType(fn.Return, vars)
	return "fn " + name + "(" + strings.Join(params, ", ") + ") -> " + ret
}

// renderGleamTypeDef builds a type definition string with constructors.
func renderGleamTypeDef(name string, td GleamTypeDef) string {
	vars := make(map[int]string)
	var sb strings.Builder
	sb.WriteString("type ")
	sb.WriteString(name)
	if td.Parameters > 0 {
		typeParams := make([]string, td.Parameters)
		varNames := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		for i := 0; i < td.Parameters; i++ {
			if i < len(varNames) {
				typeParams[i] = varNames[i]
				vars[i] = varNames[i]
			} else {
				typeParams[i] = fmt.Sprintf("t%d", i)
				vars[i] = typeParams[i]
			}
		}
		sb.WriteString("(" + strings.Join(typeParams, ", ") + ")")
	}
	if len(td.Constructors) > 0 {
		sb.WriteString(" {\n")
		for _, c := range td.Constructors {
			sb.WriteString("  " + c.Name)
			if len(c.Parameters) > 0 {
				cParams := make([]string, len(c.Parameters))
				for i, p := range c.Parameters {
					typeStr := renderGleamType(p.Type, vars)
					if p.Label != nil && *p.Label != "" {
						cParams[i] = *p.Label + ": " + typeStr
					} else {
						cParams[i] = typeStr
					}
				}
				sb.WriteString("(" + strings.Join(cParams, ", ") + ")")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("}")
	}
	return sb.String()
}
