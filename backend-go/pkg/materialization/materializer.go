package materialization

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
)

// Snapshot represents a materialized report view
type Snapshot struct {
	ID           string                 `json:"id"`
	ReportID     string                 `json:"reportId"`
	CreatedAt    time.Time              `json:"createdAt"`
	ExpiresAt    time.Time              `json:"expiresAt"`
	FilterValues map[string]interface{} `json:"filterValues"`
	Results      []byte                 `json:"-"` // Compressed results (not exposed in JSON)
	Metadata     map[string]string      `json:"metadata"`
	SizeBytes    int64                  `json:"sizeBytes"`
}

// SnapshotResult contains decompressed snapshot data
type SnapshotResult struct {
	Snapshot *Snapshot
	Results  []ReportComponentResult
}

// ReportComponentResult mirrors the execution result structure
type ReportComponentResult struct {
	ComponentID string                      `json:"componentId"`
	Type        storage.ReportComponentType `json:"type"`
	Columns     []string                    `json:"columns,omitempty"`
	Rows        [][]interface{}             `json:"rows,omitempty"`
	RowCount    int64                       `json:"rowCount,omitempty"`
	DurationMS  int64                       `json:"durationMs,omitempty"`
	Content     string                      `json:"content,omitempty"`
	Metadata    map[string]any              `json:"metadata,omitempty"`
	Error       string                      `json:"error,omitempty"`
	CacheHit    bool                        `json:"cacheHit,omitempty"`
	TotalRows   int64                       `json:"totalRows,omitempty"`
	LimitedRows int                         `json:"limitedRows,omitempty"`
}

// Materializer manages report snapshot materialization
type Materializer struct {
	db        *sql.DB
	logger    *logrus.Logger
	scheduler *cron.Cron
	entries   map[string]cron.EntryID
	mu        sync.RWMutex

	// Default TTL for snapshots
	defaultTTL time.Duration

	// Maximum snapshots to keep per report
	maxSnapshotsPerReport int

	// Callback to run report for materialization
	runReport func(reportID string, filters map[string]interface{}) ([]ReportComponentResult, error)
}

// NewMaterializer creates a new materializer
func NewMaterializer(db *sql.DB, logger *logrus.Logger) *Materializer {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronScheduler := cron.New(cron.WithParser(parser), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronScheduler.Start()

	return &Materializer{
		db:                    db,
		logger:                logger,
		scheduler:             cronScheduler,
		entries:               make(map[string]cron.EntryID),
		defaultTTL:            24 * time.Hour,
		maxSnapshotsPerReport: 10,
	}
}

// SetRunReportCallback sets the callback for running reports
func (m *Materializer) SetRunReportCallback(callback func(reportID string, filters map[string]interface{}) ([]ReportComponentResult, error)) {
	m.runReport = callback
}

// EnsureSchema creates required tables
func (m *Materializer) EnsureSchema() error {
	statement := `
CREATE TABLE IF NOT EXISTS report_snapshots (
	id TEXT PRIMARY KEY,
	report_id TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at DATETIME,
	filter_values TEXT,
	results BLOB NOT NULL,
	metadata TEXT,
	size_bytes INTEGER,
	FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_snapshots_report_time ON report_snapshots(report_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_expiry ON report_snapshots(expires_at);
`

	if _, err := m.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure materialization schema: %w", err)
	}

	return nil
}

// MaterializeReport creates a snapshot of a report's results
func (m *Materializer) MaterializeReport(reportID string, filterValues map[string]interface{}, ttl time.Duration) (*Snapshot, error) {
	if m.runReport == nil {
		return nil, fmt.Errorf("run report callback not set")
	}

	// Run the report
	results, err := m.runReport(reportID, filterValues)
	if err != nil {
		return nil, fmt.Errorf("failed to run report: %w", err)
	}

	// Serialize results
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal results: %w", err)
	}

	// Compress results
	compressed, err := compress(resultsJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to compress results: %w", err)
	}

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = m.defaultTTL
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	filterJSON, _ := json.Marshal(filterValues)
	metadata := map[string]string{
		"compression": "gzip",
		"format":      "json",
	}
	metadataJSON, _ := json.Marshal(metadata)

	snapshot := &Snapshot{
		ID:           uuid.NewString(),
		ReportID:     reportID,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
		FilterValues: filterValues,
		Results:      compressed,
		Metadata:     metadata,
		SizeBytes:    int64(len(compressed)),
	}

	// Save snapshot
	_, err = m.db.Exec(`
		INSERT INTO report_snapshots (id, report_id, created_at, expires_at, filter_values, results, metadata, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, snapshot.ID, snapshot.ReportID, snapshot.CreatedAt, snapshot.ExpiresAt,
		string(filterJSON), compressed, string(metadataJSON), snapshot.SizeBytes)

	if err != nil {
		return nil, fmt.Errorf("failed to save snapshot: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"snapshot_id":     snapshot.ID,
		"report_id":       reportID,
		"size_bytes":      snapshot.SizeBytes,
		"compression_pct": fmt.Sprintf("%.1f%%", float64(snapshot.SizeBytes)/float64(len(resultsJSON))*100),
		"ttl_hours":       ttl.Hours(),
	}).Info("Report snapshot materialized")

	return snapshot, nil
}

// GetLatestSnapshot retrieves the most recent snapshot for a report
func (m *Materializer) GetLatestSnapshot(reportID string, filterValues map[string]interface{}, maxAge time.Duration) (*SnapshotResult, error) {
	// Build filter matching query
	filterJSON, _ := json.Marshal(filterValues)

	var snapshot Snapshot
	var filterValuesJSON, metadataJSON string
	var results []byte

	// Query for latest non-expired snapshot matching filters
	err := m.db.QueryRow(`
		SELECT id, report_id, created_at, expires_at, filter_values, results, metadata, size_bytes
		FROM report_snapshots
		WHERE report_id = ? AND filter_values = ? AND expires_at > ?
		ORDER BY created_at DESC
		LIMIT 1
	`, reportID, string(filterJSON), time.Now().UTC()).Scan(
		&snapshot.ID, &snapshot.ReportID, &snapshot.CreatedAt, &snapshot.ExpiresAt,
		&filterValuesJSON, &results, &metadataJSON, &snapshot.SizeBytes)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No snapshot found
		}
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Check max age if specified
	if maxAge > 0 && time.Since(snapshot.CreatedAt) > maxAge {
		return nil, nil // Snapshot too old
	}

	// Parse filter values
	json.Unmarshal([]byte(filterValuesJSON), &snapshot.FilterValues)
	json.Unmarshal([]byte(metadataJSON), &snapshot.Metadata)
	snapshot.Results = results

	// Decompress results
	decompressed, err := decompress(results)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress results: %w", err)
	}

	// Unmarshal results
	var componentResults []ReportComponentResult
	if err := json.Unmarshal(decompressed, &componentResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return &SnapshotResult{
		Snapshot: &snapshot,
		Results:  componentResults,
	}, nil
}

// GetSnapshot retrieves a specific snapshot by ID
func (m *Materializer) GetSnapshot(snapshotID string) (*SnapshotResult, error) {
	var snapshot Snapshot
	var filterValuesJSON, metadataJSON string
	var results []byte

	err := m.db.QueryRow(`
		SELECT id, report_id, created_at, expires_at, filter_values, results, metadata, size_bytes
		FROM report_snapshots WHERE id = ?
	`, snapshotID).Scan(
		&snapshot.ID, &snapshot.ReportID, &snapshot.CreatedAt, &snapshot.ExpiresAt,
		&filterValuesJSON, &results, &metadataJSON, &snapshot.SizeBytes)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	json.Unmarshal([]byte(filterValuesJSON), &snapshot.FilterValues)
	json.Unmarshal([]byte(metadataJSON), &snapshot.Metadata)
	snapshot.Results = results

	// Check if expired
	if time.Now().UTC().After(snapshot.ExpiresAt) {
		return nil, fmt.Errorf("snapshot expired")
	}

	// Decompress results
	decompressed, err := decompress(results)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress results: %w", err)
	}

	// Unmarshal results
	var componentResults []ReportComponentResult
	if err := json.Unmarshal(decompressed, &componentResults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	return &SnapshotResult{
		Snapshot: &snapshot,
		Results:  componentResults,
	}, nil
}

// ListSnapshots lists snapshots for a report
func (m *Materializer) ListSnapshots(reportID string, limit int) ([]*Snapshot, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := m.db.Query(`
		SELECT id, report_id, created_at, expires_at, filter_values, metadata, size_bytes
		FROM report_snapshots
		WHERE report_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, reportID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*Snapshot
	for rows.Next() {
		var snapshot Snapshot
		var filterValuesJSON, metadataJSON string

		if err := rows.Scan(&snapshot.ID, &snapshot.ReportID, &snapshot.CreatedAt, &snapshot.ExpiresAt,
			&filterValuesJSON, &metadataJSON, &snapshot.SizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}

		json.Unmarshal([]byte(filterValuesJSON), &snapshot.FilterValues)
		json.Unmarshal([]byte(metadataJSON), &snapshot.Metadata)

		snapshots = append(snapshots, &snapshot)
	}

	return snapshots, nil
}

// InvalidateSnapshots removes all snapshots for a report
func (m *Materializer) InvalidateSnapshots(reportID string) error {
	_, err := m.db.Exec("DELETE FROM report_snapshots WHERE report_id = ?", reportID)
	if err != nil {
		return fmt.Errorf("failed to invalidate snapshots: %w", err)
	}

	m.logger.WithField("report_id", reportID).Info("Snapshots invalidated")
	return nil
}

// CleanupExpiredSnapshots removes expired snapshots
func (m *Materializer) CleanupExpiredSnapshots() error {
	// Keep last N snapshots per report, delete older expired ones
	result, err := m.db.Exec(`
		DELETE FROM report_snapshots
		WHERE expires_at < ?
		AND id NOT IN (
			SELECT id FROM (
				SELECT id, report_id,
					ROW_NUMBER() OVER (PARTITION BY report_id ORDER BY created_at DESC) as rn
				FROM report_snapshots
			) WHERE rn <= ?
		)
	`, time.Now().UTC(), m.maxSnapshotsPerReport)

	if err != nil {
		return fmt.Errorf("failed to cleanup snapshots: %w", err)
	}

	deleted, _ := result.RowsAffected()
	if deleted > 0 {
		m.logger.WithField("deleted_count", deleted).Info("Cleaned up expired snapshots")
	}

	return nil
}

// ScheduleMaterialization schedules periodic materialization for a report
func (m *Materializer) ScheduleMaterialization(reportID, schedule string, filters map[string]interface{}, ttl time.Duration) error {
	if schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing schedule
	scheduleKey := fmt.Sprintf("materialize_%s", reportID)
	if entryID, ok := m.entries[scheduleKey]; ok {
		m.scheduler.Remove(entryID)
		delete(m.entries, scheduleKey)
	}

	// Add new schedule
	entryID, err := m.scheduler.AddFunc(schedule, func() {
		if _, err := m.MaterializeReport(reportID, filters, ttl); err != nil {
			m.logger.WithError(err).WithField("report_id", reportID).Warn("Scheduled materialization failed")
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule materialization: %w", err)
	}

	m.entries[scheduleKey] = entryID
	m.logger.WithFields(logrus.Fields{
		"report_id": reportID,
		"schedule":  schedule,
	}).Info("Scheduled report materialization")

	return nil
}

// UnscheduleMaterialization removes scheduled materialization
func (m *Materializer) UnscheduleMaterialization(reportID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	scheduleKey := fmt.Sprintf("materialize_%s", reportID)
	if entryID, ok := m.entries[scheduleKey]; ok {
		m.scheduler.Remove(entryID)
		delete(m.entries, scheduleKey)
	}
}

// GetStats returns materialization statistics
func (m *Materializer) GetStats() (map[string]interface{}, error) {
	var totalSnapshots, totalSize int64
	var avgSize float64

	err := m.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(size_bytes), 0), COALESCE(AVG(size_bytes), 0)
		FROM report_snapshots
		WHERE expires_at > ?
	`, time.Now().UTC()).Scan(&totalSnapshots, &totalSize, &avgSize)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]interface{}{
		"total_snapshots": totalSnapshots,
		"total_size_mb":   float64(totalSize) / (1024 * 1024),
		"avg_size_kb":     avgSize / 1024,
	}, nil
}

// Shutdown stops the materializer scheduler
func (m *Materializer) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.scheduler != nil {
		m.scheduler.Stop()
		m.scheduler = nil
	}
	m.entries = make(map[string]cron.EntryID)
}

// Compression helpers

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// IsExpired checks if a snapshot has expired
func (s *Snapshot) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}
