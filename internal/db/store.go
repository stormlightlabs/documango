package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Document struct {
	Path    string
	Format  string
	Body    []byte
	RawHTML []byte
	Hash    string
}

type SearchEntry struct {
	Name  string
	Type  string
	Body  string
	DocID int64
}

type AgentContext struct {
	DocID     int64
	Symbol    string
	Signature string
	Summary   string
}

type SearchResult struct {
	Name  string
	Type  string
	DocID int64
	Score float64
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("db path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Init(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, Schema)
	return err
}

func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *Store) InsertDocument(ctx context.Context, doc Document) (int64, error) {
	if doc.Hash == "" {
		doc.Hash = HashBytes(doc.Body)
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO documents (path, format, body, raw_html, hash) VALUES (?, ?, ?, ?, ?)`,
		doc.Path, doc.Format, doc.Body, doc.RawHTML, doc.Hash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) InsertSearchEntry(ctx context.Context, entry SearchEntry) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO search_index (name, type, body, doc_id) VALUES (?, ?, ?, ?)`,
		entry.Name, entry.Type, entry.Body, entry.DocID,
	)
	return err
}

func (s *Store) InsertAgentContext(ctx context.Context, entry AgentContext) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agent_context (doc_id, symbol, signature, summary) VALUES (?, ?, ?, ?)`,
		entry.DocID, entry.Symbol, entry.Signature, entry.Summary,
	)
	return err
}

func (s *Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func InsertDocumentTx(ctx context.Context, tx *sql.Tx, doc Document) (int64, error) {
	if doc.Hash == "" {
		doc.Hash = HashBytes(doc.Body)
	}
	res, err := tx.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO documents (path, format, body, raw_html, hash) VALUES (?, ?, ?, ?, ?)`,
		doc.Path, doc.Format, doc.Body, doc.RawHTML, doc.Hash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func InsertSearchEntryTx(ctx context.Context, tx *sql.Tx, entry SearchEntry) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO search_index (name, type, body, doc_id) VALUES (?, ?, ?, ?)`,
		entry.Name, entry.Type, entry.Body, entry.DocID,
	)
	return err
}

func InsertAgentContextTx(ctx context.Context, tx *sql.Tx, entry AgentContext) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO agent_context (doc_id, symbol, signature, summary) VALUES (?, ?, ?, ?)`,
		entry.DocID, entry.Symbol, entry.Signature, entry.Summary,
	)
	return err
}

func (s *Store) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return s.SearchPackage(ctx, query, "", limit)
}

var namespaces = []string{"atproto", "go", "rust", "hex", "github"}

// SearchPackage searches for documents matching the given query and optional package prefix.
//
// It contains implicit namespace detection logic to handle queries like "rust/serde/Serialize".
// If the query starts with a known namespace and contains a slash, it uses the namespace as the
// package prefix and the rest of the query as the symbol to search for.
//
//   - For ATProto, it also handles special cases like "lexicon/", "docs/", and "spec/".
//   - For Go/Hex, the first part after the namespace is usually the package name.
//   - rust/crate/item -> rust/crate/%/item
//   - rust/crate -> rust/crate/index or rust/crate/% (for crate root)
func (s *Store) SearchPackage(ctx context.Context, query, packagePrefix string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	if packagePrefix == "" && strings.Contains(query, "/") {
		parts := strings.Split(query, "/")
		if len(parts) >= 2 {
			ns := parts[0]
			if slices.Contains(namespaces, ns) {
				packagePrefix = ns + "/"
				remaining := parts[1:]

				if ns == "atproto" {
					for _, sub := range []string{"lexicon", "docs", "spec"} {
						if len(remaining) >= 2 && remaining[0] == sub {
							packagePrefix += sub + "/"
							remaining = remaining[1:]
							break
						}
					}
				} else if len(remaining) >= 2 {
					packagePrefix += remaining[0] + "/"
					remaining = remaining[1:]
				}
				query = strings.Join(remaining, " ")
			}
		}
	}

	query = SanitizeQuery(query)

	var sqlQuery string
	var args []any

	if packagePrefix != "" {
		sqlQuery = `SELECT name, type, search_index.doc_id, (CASE WHEN name = ? THEN 100 ELSE 0 END) - bm25(search_index, 5.0, 1.0, 1.0) AS score
			FROM search_index
			JOIN documents ON search_index.doc_id = documents.id
			WHERE search_index MATCH ? AND documents.path LIKE ?
			ORDER BY score DESC
			LIMIT ?`
		args = []any{query, query, packagePrefix + "%", limit}
	} else {
		sqlQuery = `SELECT name, type, doc_id, (CASE WHEN name = ? THEN 100 ELSE 0 END) - bm25(search_index, 5.0, 1.0, 1.0) AS score
			FROM search_index
			WHERE search_index MATCH ?
			ORDER BY score DESC
			LIMIT ?`
		args = []any{query, query, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var res SearchResult
		if err := rows.Scan(&res.Name, &res.Type, &res.DocID, &res.Score); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// ReadDocument reads a document from the database by its path.
//
// Rust specific stuff:
//   - rust/crate/sub/path -> rust/crate/%/sub/path
//   - rust/crate -> rust/crate/index or rust/crate/% (for crate root)
func (s *Store) ReadDocument(ctx context.Context, path string) (Document, error) {
	var doc Document
	err := s.db.QueryRowContext(
		ctx,
		`SELECT path, format, body, raw_html, hash FROM documents WHERE path = ?`,
		path,
	).Scan(&doc.Path, &doc.Format, &doc.Body, &doc.RawHTML, &doc.Hash)

	if err == nil {
		return doc, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return Document{}, err
	}

	err = s.db.QueryRowContext(
		ctx,
		`SELECT path, format, body, raw_html, hash FROM documents WHERE path LIKE ? LIMIT 1`,
		path,
	).Scan(&doc.Path, &doc.Format, &doc.Body, &doc.RawHTML, &doc.Hash)

	if err == nil {
		return doc, nil
	}

	parts := strings.Split(path, "/")
	if len(parts) >= 2 && parts[0] == "rust" {
		var fallbackPath string
		if len(parts) >= 3 {
			remaining := strings.Join(parts[2:], "/")
			fallbackPath = fmt.Sprintf("rust/%s/%%/%s", parts[1], remaining)
		} else if len(parts) == 2 {
			fallbackPath = fmt.Sprintf("rust/%s/%%", parts[1])
		}

		if fallbackPath != "" {
			err = s.db.QueryRowContext(
				ctx,
				`SELECT path, format, body, raw_html, hash FROM documents WHERE path LIKE ? LIMIT 1`,
				fallbackPath,
			).Scan(&doc.Path, &doc.Format, &doc.Body, &doc.RawHTML, &doc.Hash)

			if err == nil {
				return doc, nil
			}
		}
	}

	return Document{}, err
}

func (s *Store) ReadDocumentByID(ctx context.Context, id int64) (Document, error) {
	var doc Document
	err := s.db.QueryRowContext(
		ctx,
		`SELECT path, format, body, raw_html, hash FROM documents WHERE id = ?`,
		id,
	).Scan(&doc.Path, &doc.Format, &doc.Body, &doc.RawHTML, &doc.Hash)
	if err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Store) CountDocuments(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) CountSearchEntries(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM search_index`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) CountAgentEntries(ctx context.Context) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_context`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) Vacuum(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	if err := s.Init(ctx); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *Store) GetSymbolContext(ctx context.Context, symbol string) (AgentContext, error) {
	var entry AgentContext
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT doc_id, symbol, signature, summary FROM agent_context WHERE symbol = ?`,
		symbol,
	).Scan(&entry.DocID, &entry.Symbol, &entry.Signature, &entry.Summary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentContext{}, fmt.Errorf("symbol not found: %s", symbol)
		}
		return AgentContext{}, err
	}
	return entry, nil
}

// DB returns the underlying SQL database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// PackageInfo represents a package with its document count.
type PackageInfo struct {
	Name          string
	Language      string
	DocumentCount int
}

// ListPackages returns all packages grouped by language with document counts.
func (s *Store) ListPackages(ctx context.Context) ([]PackageInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN path LIKE '%/%' THEN SUBSTR(path, 1, INSTR(path, '/') - 1)
				ELSE path
			END as language,
			CASE
				WHEN path LIKE '%/%/%' THEN SUBSTR(path, 1, INSTR(SUBSTR(path, INSTR(path, '/') + 1), '/') + INSTR(path, '/') - 1)
				WHEN path LIKE '%/%' THEN SUBSTR(path, 1, INSTR(path, '/') - 1)
				ELSE path
			END as package,
			COUNT(*) as doc_count
		FROM documents
		GROUP BY language, package
		ORDER BY language, package
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []PackageInfo
	for rows.Next() {
		var p PackageInfo
		if err := rows.Scan(&p.Language, &p.Name, &p.DocumentCount); err != nil {
			return nil, err
		}
		packages = append(packages, p)
	}

	return packages, rows.Err()
}

// SanitizeQuery wraps the query in double quotes if it contains characters
// that might break FTS5 (like slashes) and isn't already quoted.
// It preserves column filters like "type:Func".
func SanitizeQuery(q string) string {
	if q == "" {
		return q
	}

	if strings.HasPrefix(q, "\"") && strings.HasSuffix(q, "\"") {
		return q
	}

	terms := strings.Fields(q)
	var sanitized []string
	for _, term := range terms {

		if strings.HasPrefix(term, "\"") && strings.HasSuffix(term, "\"") {
			sanitized = append(sanitized, term)
			continue
		}

		lower := strings.ToLower(term)
		if strings.HasPrefix(lower, "name:") || strings.HasPrefix(lower, "type:") || strings.HasPrefix(lower, "body:") {
			sanitized = append(sanitized, term)
			continue
		}

		if strings.ContainsAny(term, "/-.*():\"") {
			escaped := strings.ReplaceAll(term, "\"", "\"\"")
			sanitized = append(sanitized, "\""+escaped+"\"")
		} else {
			sanitized = append(sanitized, term)
		}
	}

	return strings.Join(sanitized, " ")
}
