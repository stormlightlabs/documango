package atproto

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
)

type Options struct {
	DB    *db.Store
	Cache *cache.FilesystemCache
}

func IngestAtproto(ctx context.Context, opts Options) error {
	tmpDir, err := os.MkdirTemp("", "documango-atproto-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	repos := map[string]string{
		"atproto":         "https://github.com/bluesky-social/atproto",
		"atproto-website": "https://github.com/bluesky-social/atproto-website",
		"bsky-docs":       "https://github.com/bluesky-social/bsky-docs",
	}

	for name, url := range repos {
		log.Info("fetching repository", "repo", name)
		dest := filepath.Join(tmpDir, name)

		if opts.Cache != nil {
			cacheKey := cache.AtprotoKey(name)
			gitCache, err := cache.NewGitCache(opts.Cache.Dir())
			if err == nil {
				if commitSHA, ok := gitCache.GetCommit(cacheKey); ok {
					log.Info("using cached commit", "repo", name, "commit", commitSHA)
					if err := cache.ShallowClone(url, commitSHA, dest); err == nil {
						continue
					}
					log.Warn("shallow clone failed, falling back to full clone", "repo", name, "err", err)
				}
			}

			if err := gitClone(ctx, url, dest); err != nil {
				return fmt.Errorf("failed to clone %s: %w", name, err)
			}

			commitSHA, err := cache.GetRepoCommit(dest)
			if err == nil {
				log.Info("caching commit SHA", "repo", name, "commit", commitSHA)
				_ = gitCache.PutCommit(cacheKey, url, commitSHA, 0)
			} else {
				log.Warn("failed to get commit SHA", "repo", name, "err", err)
			}
		} else {
			if err := gitClone(ctx, url, dest); err != nil {
				return fmt.Errorf("failed to clone %s: %w", name, err)
			}
		}
	}

	return opts.DB.WithTx(ctx, func(tx *sql.Tx) error {
		lexiconDir := filepath.Join(tmpDir, "atproto", "lexicons")
		if err := ingestLexicons(ctx, tx, lexiconDir); err != nil {
			return err
		}

		specDir := filepath.Join(tmpDir, "atproto-website", "src", "app", "[locale]")
		if err := ingestSpecs(ctx, tx, specDir); err != nil {
			return err
		}

		docsDir := filepath.Join(tmpDir, "bsky-docs", "docs")
		if err := ingestDocs(ctx, tx, docsDir); err != nil {
			return err
		}

		return nil
	})
}

func gitClone(ctx context.Context, url, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, dest)
	return cmd.Run()
}

func ingestLexicons(ctx context.Context, tx *sql.Tx, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var lex Lexicon
		if err := json.Unmarshal(data, &lex); err != nil {
			log.Warn("failed to parse lexicon", "path", path, "err", err)
			return nil
		}

		md := LexiconToMarkdown(&lex)
		docPath := "atproto/lexicon/" + lex.ID

		docID, err := insertDoc(ctx, tx, docPath, md)
		if err != nil {
			return err
		}

		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name: lex.ID, Type: "Lexicon", Body: lex.ID, DocID: docID,
		}); err != nil {
			return err
		}

		if err := db.InsertAgentContextTx(ctx, tx, db.AgentContext{
			DocID:     docID,
			Symbol:    lex.ID,
			Signature: "lexicon " + lex.ID,
			Summary:   "Lexicon definition for " + lex.ID,
		}); err != nil {
			return err
		}

		return nil
	})
}

func insertDoc(ctx context.Context, tx *sql.Tx, path, body string) (int64, error) {
	compressed, err := codec.Compress([]byte(body))
	if err != nil {
		return 0, err
	}
	return db.InsertDocumentTx(ctx, tx, db.Document{
		Path:   path,
		Format: "markdown",
		Body:   compressed,
		Hash:   db.HashBytes([]byte(body)),
	})
}

func ingestSpecs(ctx context.Context, tx *sql.Tx, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || (!strings.HasSuffix(d.Name(), ".md") && !strings.HasSuffix(d.Name(), ".mdx")) {
			return nil
		}

		if strings.Contains(path, "[locale]") && !strings.Contains(path, "/en/") && !strings.HasSuffix(path, "/en.mdx") && !strings.HasSuffix(path, "/en.md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		body := transformMDX(string(data))
		rel, _ := filepath.Rel(root, path)

		docPath := "atproto/spec/" + rel
		docPath = strings.Replace(docPath, "[locale]/", "", 1)
		docPath = strings.Replace(docPath, "/en.mdx", "", 1)
		docPath = strings.Replace(docPath, "/en.md", "", 1)
		docPath = strings.Replace(docPath, "/page.mdx", "", 1)
		docPath = strings.TrimSuffix(docPath, filepath.Ext(docPath))

		docID, err := insertDoc(ctx, tx, docPath, body)
		if err != nil {
			return err
		}

		name := filepath.Base(docPath)
		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  name,
			Type:  "Spec",
			Body:  name,
			DocID: docID,
		}); err != nil {
			return err
		}

		return nil
	})
}

func ingestDocs(ctx context.Context, tx *sql.Tx, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || (!strings.HasSuffix(d.Name(), ".md") && !strings.HasSuffix(d.Name(), ".mdx")) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		body := transformMDX(string(data))
		rel, _ := filepath.Rel(root, path)
		docPath := "atproto/docs/" + strings.TrimSuffix(rel, filepath.Ext(rel))

		docID, err := insertDoc(ctx, tx, docPath, body)
		if err != nil {
			return err
		}

		name := filepath.Base(docPath)
		if err := db.InsertSearchEntryTx(ctx, tx, db.SearchEntry{
			Name:  name,
			Type:  "Doc",
			Body:  name,
			DocID: docID,
		}); err != nil {
			return err
		}

		return nil
	})
}

func transformMDX(input string) string {
	if strings.HasPrefix(input, "---") {
		parts := strings.SplitN(input, "---", 3)
		if len(parts) == 3 {
			input = parts[2]
		}
	}

	lines := strings.Split(input, "\n")
	var out []string
	inExport := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "export const") || strings.HasPrefix(trimmed, "import ") {
			if strings.Contains(line, "{") && !strings.Contains(line, "}") {
				inExport = true
			}
			continue
		}
		if inExport {
			if strings.Contains(line, "}") {
				inExport = false
			}
			continue
		}
		out = append(out, line)
	}
	input = strings.Join(out, "\n")

	re := regexp.MustCompile(`\{\{.*?\}\}`)
	input = re.ReplaceAllString(input, "")

	tagRegex := regexp.MustCompile(`<(Tabs|TabItem|Admonition|video|img|br|hr|p|div|section)[^>]*>`)
	input = tagRegex.ReplaceAllString(input, "")

	closeTagRegex := regexp.MustCompile(`</(Tabs|TabItem|Admonition|video|img|br|hr|p|div|section)>`)
	input = closeTagRegex.ReplaceAllString(input, "")

	return strings.TrimSpace(input)
}
