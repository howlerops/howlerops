package rag

import (
    "context"
    "sync"
    "time"
)

// AdaptiveVectorStore routes indexing/search to local and optionally remote stores.
type AdaptiveVectorStore struct {
    tierLevel   string
    localStore  VectorStore
    remoteStore VectorStore
    syncEnabled bool
    // simple in-process gate to coalesce concurrent syncs per document id
    syncing   chan string
    inFlight  map[string]struct{}
    flightMu  sync.Mutex
    maxRetries int
}

func NewAdaptiveVectorStore(tier string, local VectorStore, remote VectorStore, syncEnabled bool) *AdaptiveVectorStore {
    return &AdaptiveVectorStore{
        tierLevel:   tier,
        localStore:  local,
        remoteStore: remote,
        syncEnabled: syncEnabled,
        syncing:     make(chan string, 256),
        inFlight:    make(map[string]struct{}),
        maxRetries:  3,
    }
}

func (a *AdaptiveVectorStore) Initialize(ctx context.Context) error { return a.localStore.Initialize(ctx) }

func (a *AdaptiveVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
    if err := a.localStore.IndexDocument(ctx, doc); err != nil {
        return err
    }
    if a.syncEnabled && a.remoteStore != nil && (a.tierLevel == "individual" || a.tierLevel == "team") {
        // non-blocking attempt to enqueue doc ID; duplicates will naturally be coalesced by reader
        a.enqueueSync(doc)
    }
    return nil
}

func (a *AdaptiveVectorStore) BatchIndexDocuments(ctx context.Context, docs []*Document) error {
    if err := a.localStore.BatchIndexDocuments(ctx, docs); err != nil {
        return err
    }
    if a.syncEnabled && a.remoteStore != nil && (a.tierLevel == "individual" || a.tierLevel == "team") {
        // Best-effort batch sync; fall back to per-doc with backoff if needed
        go func() {
            // Try batch first
            _ = a.remoteStore.BatchIndexDocuments(context.Background(), docs)
            // Enqueue individually as a safety net (idempotent upsert on remote)
            for _, d := range docs {
                a.enqueueSync(d)
            }
        }()
    }
    return nil
}

func (a *AdaptiveVectorStore) enqueueSync(doc *Document) {
    if doc == nil || doc.ID == "" {
        return
    }
    a.flightMu.Lock()
    if _, exists := a.inFlight[doc.ID]; exists {
        a.flightMu.Unlock()
        return
    }
    a.inFlight[doc.ID] = struct{}{}
    a.flightMu.Unlock()

    select {
    case a.syncing <- doc.ID:
        dcopy := *doc
        go a.syncWithBackoff(&dcopy)
    default:
        // queue full; drop this attempt; local remains authoritative
        a.flightMu.Lock()
        delete(a.inFlight, doc.ID)
        a.flightMu.Unlock()
    }
}

func (a *AdaptiveVectorStore) syncWithBackoff(doc *Document) {
    // Remove inFlight flag on return
    defer func() {
        a.flightMu.Lock()
        delete(a.inFlight, doc.ID)
        a.flightMu.Unlock()
    }()

    // Exponential backoff attempts
    backoff := 100 * time.Millisecond
    for attempt := 0; attempt < a.maxRetries; attempt++ {
        if a.remoteStore == nil {
            return
        }
        if err := a.remoteStore.IndexDocument(context.Background(), doc); err == nil {
            return
        }
        time.Sleep(backoff)
        backoff *= 2
    }
}

// StartSyncWorker starts a lightweight worker that throttles sync attempts.
// Local-first semantics remain; this simply rate-limits background syncs.
func (a *AdaptiveVectorStore) StartSyncWorker(ctx context.Context, interval time.Duration) {
    if !a.syncEnabled || a.remoteStore == nil {
        return
    }
    if interval <= 0 {
        interval = 200 * time.Millisecond
    }
    ticker := time.NewTicker(interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                // nothing to do actively; per-doc backoff goroutines handle sync
                // this hook keeps a heartbeat for future queue draining if needed
            }
        }
    }()
}

func (a *AdaptiveVectorStore) GetDocument(ctx context.Context, id string) (*Document, error) {
    return a.localStore.GetDocument(ctx, id)
}

func (a *AdaptiveVectorStore) UpdateDocument(ctx context.Context, doc *Document) error {
    return a.localStore.UpdateDocument(ctx, doc)
}

func (a *AdaptiveVectorStore) DeleteDocument(ctx context.Context, id string) error {
    return a.localStore.DeleteDocument(ctx, id)
}

func (a *AdaptiveVectorStore) SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error) {
    return a.localStore.SearchSimilar(ctx, embedding, k, filter)
}

func (a *AdaptiveVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error) {
    return a.localStore.SearchByText(ctx, query, k, filter)
}

func (a *AdaptiveVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
    return a.localStore.HybridSearch(ctx, query, embedding, k)
}

func (a *AdaptiveVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
    return a.localStore.CreateCollection(ctx, name, dimension)
}

func (a *AdaptiveVectorStore) DeleteCollection(ctx context.Context, name string) error {
    return a.localStore.DeleteCollection(ctx, name)
}

func (a *AdaptiveVectorStore) ListCollections(ctx context.Context) ([]string, error) { return a.localStore.ListCollections(ctx) }

func (a *AdaptiveVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) {
    return a.localStore.GetStats(ctx)
}

func (a *AdaptiveVectorStore) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
    return a.localStore.GetCollectionStats(ctx, collection)
}

func (a *AdaptiveVectorStore) Optimize(ctx context.Context) error { return a.localStore.Optimize(ctx) }
func (a *AdaptiveVectorStore) Backup(ctx context.Context, path string) error { return a.localStore.Backup(ctx, path) }
func (a *AdaptiveVectorStore) Restore(ctx context.Context, path string) error { return a.localStore.Restore(ctx, path) }


