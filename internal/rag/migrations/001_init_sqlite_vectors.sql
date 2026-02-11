-- SQLite Vector Store Schema
-- Migration: 001 - Initialize SQLite vector store tables

-- Documents table (core document storage)
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    type TEXT NOT NULL,  -- 'schema', 'query', 'plan', 'result', 'business', 'performance'
    content TEXT NOT NULL,
    metadata TEXT,  -- JSON
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    access_count INTEGER DEFAULT 0,
    last_accessed INTEGER
);

-- Vector embeddings table (stores document embeddings)
-- Note: This will use sqlite-vec extension when available
-- For now, we store as BLOB and implement search in Go
CREATE TABLE IF NOT EXISTS embeddings (
    document_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,  -- Serialized float32 array
    dimension INTEGER NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Full-text search index
CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
    document_id UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);

-- Trigger to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
    INSERT INTO documents_fts(document_id, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
    UPDATE documents_fts SET content = new.content WHERE document_id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
    DELETE FROM documents_fts WHERE document_id = old.id;
END;

-- Collections metadata (tracks different document collections)
CREATE TABLE IF NOT EXISTS collections (
    name TEXT PRIMARY KEY,
    vector_size INTEGER NOT NULL,
    distance TEXT NOT NULL,  -- 'cosine', 'euclidean', 'dot'
    document_count INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_documents_connection ON documents(connection_id);
CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(type);
CREATE INDEX IF NOT EXISTS idx_documents_created ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_documents_accessed ON documents(last_accessed);
CREATE INDEX IF NOT EXISTS idx_embeddings_dimension ON embeddings(dimension);

-- Initialize default collections
INSERT OR IGNORE INTO collections (name, vector_size, distance, document_count, created_at, updated_at) VALUES
    ('schemas', 1536, 'cosine', 0, unixepoch(), unixepoch()),
    ('queries', 1536, 'cosine', 0, unixepoch(), unixepoch()),
    ('performance', 1536, 'euclidean', 0, unixepoch(), unixepoch()),
    ('business', 1536, 'cosine', 0, unixepoch(), unixepoch());

