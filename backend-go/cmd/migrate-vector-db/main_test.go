package main

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
)

type fakeStatsStore struct {
	collections    []string
	collectionsErr error
	stats          map[string]*rag.CollectionStats
	statsErr       map[string]error
	listCalled     bool
	statsCalls     map[string]int
}

func (f *fakeStatsStore) ListCollections(_ context.Context) ([]string, error) {
	f.listCalled = true
	if f.collectionsErr != nil {
		return nil, f.collectionsErr
	}
	return append([]string(nil), f.collections...), nil
}

func (f *fakeStatsStore) GetCollectionStats(_ context.Context, collection string) (*rag.CollectionStats, error) {
	if f.statsCalls == nil {
		f.statsCalls = make(map[string]int)
	}
	f.statsCalls[collection]++
	if err, ok := f.statsErr[collection]; ok {
		return nil, err
	}
	if stat, ok := f.stats[collection]; ok {
		statCopy := *stat
		return &statCopy, nil
	}
	return nil, nil
}

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestMigrateVectorStoreDryRun(t *testing.T) {
	source := &fakeStatsStore{
		collections: []string{"alpha", "beta"},
		stats: map[string]*rag.CollectionStats{
			"alpha": {DocumentCount: 10, Dimension: 1536},
			"beta":  {DocumentCount: 5, Dimension: 1536},
		},
	}

	if err := migrateVectorStore(context.Background(), source, source, 50, true, newTestLogger()); err != nil {
		t.Fatalf("migrateVectorStore returned error: %v", err)
	}

	if !source.listCalled {
		t.Fatalf("expected ListCollections to be called")
	}
	if source.statsCalls["alpha"] != 1 || source.statsCalls["beta"] != 1 {
		t.Fatalf("expected stats to be requested for each collection, got %#v", source.statsCalls)
	}
}

func TestMigrateVectorStoreListCollectionsError(t *testing.T) {
	source := &fakeStatsStore{
		collectionsErr: errors.New("boom"),
	}

	err := migrateVectorStore(context.Background(), source, source, 50, true, newTestLogger())
	if err == nil {
		t.Fatalf("expected error when list collections fails")
	}
}

func TestMigrateVectorStoreSkipsCollectionOnStatsError(t *testing.T) {
	source := &fakeStatsStore{
		collections: []string{"alpha", "beta"},
		stats: map[string]*rag.CollectionStats{
			"alpha": {DocumentCount: 10},
			"beta":  {DocumentCount: 0},
		},
		statsErr: map[string]error{
			"beta": errors.New("stats unavailable"),
		},
	}

	if err := migrateVectorStore(context.Background(), source, source, 50, true, newTestLogger()); err != nil {
		t.Fatalf("migrateVectorStore returned error: %v", err)
	}

	if source.statsCalls["beta"] != 1 {
		t.Fatalf("expected GetCollectionStats to be called for beta")
	}
}
