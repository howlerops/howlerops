package rag

import "context"

// TursoVectorSync is a placeholder for syncing embeddings to a remote store.
type TursoVectorSync struct{}

func NewTursoVectorSync() *TursoVectorSync { return &TursoVectorSync{} }

func (t *TursoVectorSync) Sync(ctx context.Context, doc *Document) error { return nil }


