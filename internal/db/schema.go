package db

const Schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS documents (
	id INTEGER PRIMARY KEY,
	path TEXT NOT NULL UNIQUE,
	format TEXT NOT NULL,
	body BLOB NOT NULL,
	raw_html BLOB,
	hash TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS search_index USING fts5(
	name,
	type,
	body,
	doc_id UNINDEXED,
	tokenize = 'trigram'
);

CREATE TABLE IF NOT EXISTS agent_context (
	doc_id INTEGER NOT NULL,
	symbol TEXT NOT NULL,
	signature TEXT,
	summary TEXT,
	FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_context_symbol ON agent_context(symbol);
CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);
`
