package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/rag"
)

type collectionStatsStore interface {
	ListCollections(ctx context.Context) ([]string, error)
	GetCollectionStats(ctx context.Context, collection string) (*rag.CollectionStats, error)
}

// Migration tool to migrate vector data from Qdrant to SQLite
//
// Usage:
//   go run backend-go/cmd/migrate-vector-db/main.go \
//     --from=qdrant --to=sqlite \
//     --qdrant-host=localhost --qdrant-port=6333 \
//     --sqlite-path=~/.howlerops/vectors.db \
//     --batch-size=100

func main() {
	// Parse flags
	from := flag.String("from", "sqlite", "Source vector store (only sqlite supported)")
	to := flag.String("to", "sqlite", "Target vector store (only sqlite supported)")
	sqlitePath := flag.String("sqlite-path", "~/.howlerops/vectors.db", "SQLite database path")
	batchSize := flag.Int("batch-size", 100, "Batch size for migration")
	dryRun := flag.Bool("dry-run", false, "Dry run - don't write to target")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	ctx := context.Background()

	logger.WithFields(logrus.Fields{
		"from":       *from,
		"to":         *to,
		"batch_size": *batchSize,
		"dry_run":    *dryRun,
	}).Info("Starting vector database migration")

	// Initialize source and target stores (both SQLite)
	var sourceStore rag.VectorStore
	var targetStore rag.VectorStore
	var err error

	// Only SQLite is supported
	sourceStore, err = initSQLiteStore(*sqlitePath, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize source store")
	}

	targetStore, err = initSQLiteStore(*sqlitePath, logger)

	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize target store")
	}

	// Perform migration
	startTime := time.Now()

	if err := migrateVectorStore(ctx, sourceStore, targetStore, *batchSize, *dryRun, logger); err != nil {
		logger.WithError(err).Fatal("Migration failed")
	}

	duration := time.Since(startTime)
	logger.WithField("duration", duration).Info("✓ Migration completed successfully")
}

func initSQLiteStore(path string, logger *logrus.Logger) (rag.VectorStore, error) {
	// Expand home directory
	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = home + path[1:]
	}

	config := &rag.SQLiteVectorConfig{
		Path:        path,
		Extension:   "sqlite-vec",
		VectorSize:  1536,
		CacheSizeMB: 128,
		MMapSizeMB:  256,
		WALEnabled:  true,
		Timeout:     10 * time.Second,
	}

	store, err := rag.NewSQLiteVectorStore(config, logger)
	if err != nil {
		return nil, err
	}

	if err := store.Initialize(context.Background()); err != nil {
		return nil, err
	}

	return store, nil
}

func migrateVectorStore(ctx context.Context, source, _ collectionStatsStore, _ int, dryRun bool, logger *logrus.Logger) error {
	// Get collections from source
	collections, err := source.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list source collections: %w", err)
	}

	logger.WithField("collections", len(collections)).Info("Found collections in source")

	totalDocuments := 0

	// Migrate each collection
	for _, collectionName := range collections {
		logger.WithField("collection", collectionName).Info("Migrating collection...")

		// Get stats
		stats, err := source.GetCollectionStats(ctx, collectionName)
		if err != nil {
			logger.WithError(err).Warnf("Failed to get stats for collection %s, skipping", collectionName)
			continue
		}

		logger.WithFields(logrus.Fields{
			"collection": collectionName,
			"documents":  stats.DocumentCount,
			"dimension":  stats.Dimension,
		}).Info("Collection stats")

		// For now, just log the stats
		// Full implementation would:
		// 1. Fetch documents in batches from source
		// 2. Write to target in batches
		// 3. Verify counts match

		if !dryRun {
			logger.Warn("Actual migration not yet implemented")
			logger.Info("To migrate, you would:")
			logger.Info("  1. Export documents from source")
			logger.Info("  2. Transform to target format")
			logger.Info("  3. Batch import to target")
			logger.Info("  4. Verify document counts")
		}

		totalDocuments += int(stats.DocumentCount)
	}

	logger.WithFields(logrus.Fields{
		"collections": len(collections),
		"documents":   totalDocuments,
	}).Info("Migration summary")

	return nil
}
