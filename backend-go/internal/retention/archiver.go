package retention

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Archiver handles archiving data to storage
type Archiver interface {
	Archive(ctx context.Context, resourceType string, data []map[string]interface{}) (string, error)
	Restore(ctx context.Context, location string) ([]map[string]interface{}, error)
	GetArchiveLocation(orgID, resourceType string, timestamp time.Time) string
}

// LocalArchiver archives data to local filesystem
type LocalArchiver struct {
	basePath string
	logger   *logrus.Logger
}

// NewLocalArchiver creates a new local archiver
func NewLocalArchiver(basePath string, logger *logrus.Logger) *LocalArchiver {
	return &LocalArchiver{
		basePath: basePath,
		logger:   logger,
	}
}

func (a *LocalArchiver) Archive(ctx context.Context, resourceType string, data []map[string]interface{}) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("no data to archive")
	}

	// Create archive directory
	if err := os.MkdirAll(a.basePath, 0755); err != nil {
		return "", fmt.Errorf("create archive directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.json.gz", resourceType, timestamp)
	location := filepath.Join(a.basePath, filename)

	// Create file
	file, err := os.Create(location)
	if err != nil {
		return "", fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Create archive data structure
	archiveData := ArchiveData{
		ResourceType: resourceType,
		Records:      data,
		Metadata: map[string]interface{}{
			"archived_at":   time.Now().Unix(),
			"record_count":  len(data),
			"archive_version": "1.0",
		},
	}

	// Encode to JSON
	encoder := json.NewEncoder(gzipWriter)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(archiveData); err != nil {
		return "", fmt.Errorf("encode archive data: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"resource_type": resourceType,
		"records":       len(data),
		"location":      location,
	}).Info("Archived data")

	return location, nil
}

func (a *LocalArchiver) Restore(ctx context.Context, location string) ([]map[string]interface{}, error) {
	// Open file
	file, err := os.Open(location)
	if err != nil {
		return nil, fmt.Errorf("open archive file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Decode JSON
	var archiveData ArchiveData
	decoder := json.NewDecoder(gzipReader)
	if err := decoder.Decode(&archiveData); err != nil {
		return nil, fmt.Errorf("decode archive data: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"resource_type": archiveData.ResourceType,
		"records":       len(archiveData.Records),
		"location":      location,
	}).Info("Restored archived data")

	return archiveData.Records, nil
}

func (a *LocalArchiver) GetArchiveLocation(orgID, resourceType string, timestamp time.Time) string {
	dateStr := timestamp.Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.json.gz", orgID, resourceType, dateStr)
	return filepath.Join(a.basePath, filename)
}

// S3Archiver would handle archiving to S3
// type S3Archiver struct {
//     s3Client *s3.Client
//     bucket   string
//     logger   *logrus.Logger
// }
// Implementation would use AWS SDK to upload/download from S3
