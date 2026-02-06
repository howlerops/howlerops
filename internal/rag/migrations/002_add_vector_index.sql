-- Migration: Add HNSW vector index using sqlite-vec
-- Note: Extension must be loaded before running this migration

-- Create virtual table for vector embeddings
CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
    document_id TEXT PRIMARY KEY,
    embedding float[768]  -- Adjust dimension based on model (768 for nomic-embed-text)
);

-- Create indexes for efficient queries
-- HNSW index for approximate nearest neighbor search
CREATE INDEX IF NOT EXISTS idx_vec_hnsw
ON vec_embeddings(embedding)
USING hnsw (
    m=16,                -- Number of connections per layer (higher = better recall, slower build)
    ef_construction=200, -- Quality during index build (higher = better quality, slower build)
    metric='cosine'      -- Distance metric (cosine, euclidean, or dot_product)
);

-- Keep embeddings table for backward compatibility
-- Triggers will keep vec_embeddings and embeddings in sync

-- Trigger: Insert into vec_embeddings when embeddings is updated
CREATE TRIGGER IF NOT EXISTS sync_vec_embeddings_insert
AFTER INSERT ON embeddings
BEGIN
    INSERT OR REPLACE INTO vec_embeddings (document_id, embedding)
    VALUES (NEW.document_id, NEW.embedding);
END;

-- Trigger: Update vec_embeddings when embeddings is updated
CREATE TRIGGER IF NOT EXISTS sync_vec_embeddings_update
AFTER UPDATE ON embeddings
BEGIN
    INSERT OR REPLACE INTO vec_embeddings (document_id, embedding)
    VALUES (NEW.document_id, NEW.embedding);
END;

-- Trigger: Delete from vec_embeddings when embeddings is deleted
CREATE TRIGGER IF NOT EXISTS sync_vec_embeddings_delete
AFTER DELETE ON embeddings
BEGIN
    DELETE FROM vec_embeddings WHERE document_id = OLD.document_id;
END;

-- Add metadata to track migration
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

INSERT OR IGNORE INTO schema_migrations (version, description)
VALUES (2, 'Add HNSW vector index with sqlite-vec');
