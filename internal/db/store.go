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

func (s *Store) SearchPackage(ctx context.Context, query, packagePrefix string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

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

func (s *Store) ReadDocument(ctx context.Context, path string) (Document, error) {
	var doc Document
	err := s.db.QueryRowContext(
		ctx,
		`SELECT path, format, body, raw_html, hash FROM documents WHERE path = ?`,
		path,
	).Scan(&doc.Path, &doc.Format, &doc.Body, &doc.RawHTML, &doc.Hash)
	if err != nil {
		return Document{}, err
	}
	return doc, nil
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

// DB returns the underlying SQL database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}
