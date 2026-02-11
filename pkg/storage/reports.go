package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Report represents a drag-and-drop report definition with layout and components.
type Report struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Folder        string                 `json:"folder"`
	Tags          []string               `json:"tags"`
	Definition    ReportDefinition       `json:"definition"`
	Filter        ReportFilterDefinition `json:"filter"`
	SyncOptions   ReportSyncOptions      `json:"syncOptions"`
	Starred       bool                   `json:"starred"`
	StarredAt     *time.Time             `json:"starredAt,omitempty"`
	LastRunAt     *time.Time             `json:"lastRunAt"`
	LastRunStatus string                 `json:"lastRunStatus"`
	Metadata      map[string]string      `json:"metadata"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// ReportDefinition stores canvas layout metadata and embedded components.
type ReportDefinition struct {
	Layout     []ReportLayoutSlot `json:"layout"`
	Components []ReportComponent  `json:"components"`
}

// ReportLayoutSlot defines a single area on the report canvas.
type ReportLayoutSlot struct {
	ComponentID string `json:"componentId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	W           int    `json:"w"`
	H           int    `json:"h"`
}

// ReportComponentType enumerates supported component types.
type ReportComponentType string

const (
	ReportComponentChart  ReportComponentType = "chart"
	ReportComponentMetric ReportComponentType = "metric"
	ReportComponentTable  ReportComponentType = "table"
	ReportComponentCombo  ReportComponentType = "combo"
	ReportComponentLLM    ReportComponentType = "llm"
)

// ReportComponent defines a configurable block on a report.
type ReportComponent struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Type        ReportComponentType  `json:"type"`
	Size        ReportComponentSize  `json:"size"`
	Query       ReportQueryConfig    `json:"query"`
	Chart       *ReportChartSettings `json:"chart,omitempty"`
	LLM         *ReportLLMSettings   `json:"llm,omitempty"`
	Options     map[string]any       `json:"options,omitempty"`
}

// ReportComponentSize drives the rendered footprint of a component.
type ReportComponentSize struct {
	MinW int `json:"minW"`
	MinH int `json:"minH"`
	MaxW int `json:"maxW"`
	MaxH int `json:"maxH"`
}

// ReportQueryMode indicates how data is sourced for a component.
type ReportQueryMode string

const (
	ReportQueryModeSQL     ReportQueryMode = "sql"
	ReportQueryModeBuilder ReportQueryMode = "builder"
)

// ReportQueryConfig encapsulates SQL or builder state for component queries.
type ReportQueryConfig struct {
	Mode           ReportQueryMode `json:"mode"`
	ConnectionID   string          `json:"connectionId"`
	SQL            string          `json:"sql"`
	BuilderState   json.RawMessage `json:"builderState"`
	QueryIR        json.RawMessage `json:"queryIr"`
	UseFederation  bool            `json:"useFederation"`
	Limit          *int            `json:"limit"`
	CacheSeconds   int             `json:"cacheSeconds"`
	TopLevelFilter []string        `json:"topLevelFilter"`
	Parameters     map[string]any  `json:"parameters"`
}

// ReportChartSettings customises graph rendering.
type ReportChartSettings struct {
	Variant    string            `json:"variant"`
	XField     string            `json:"xField"`
	YField     string            `json:"yField"`
	Series     []string          `json:"series"`
	Options    map[string]string `json:"options"`
	Comparison *ChartComparison  `json:"comparison,omitempty"`
	Transform  *ChartTransform   `json:"transform,omitempty"`
}

// ChartComparison config for multi-series overlays.
type ChartComparison struct {
	BaselineComponentID string `json:"baselineComponentId"`
	Type                string `json:"type"`
}

// ChartTransform enables custom pipelines (resample, moving average, etc.).
type ChartTransform struct {
	Kind   string                 `json:"kind"`
	Config map[string]interface{} `json:"config"`
}

// ReportLLMSettings encapsulates prompt + provider metadata.
type ReportLLMSettings struct {
	Provider          string            `json:"provider"`
	Model             string            `json:"model"`
	PromptTemplate    string            `json:"promptTemplate"`
	ContextComponents []string          `json:"contextComponents"`
	Temperature       float64           `json:"temperature"`
	MaxTokens         int               `json:"maxTokens"`
	Metadata          map[string]string `json:"metadata"`
}

// ReportFilterDefinition defines a reusable top-level filter bar.
type ReportFilterDefinition struct {
	Fields []ReportFilterField `json:"fields"`
}

// ReportFilterField represents a single interactive filter control.
type ReportFilterField struct {
	Key          string      `json:"key"`
	Label        string      `json:"label"`
	Type         string      `json:"type"`
	DefaultValue interface{} `json:"defaultValue"`
	Required     bool        `json:"required"`
	Choices      []string    `json:"choices"`
}

// ReportSyncOptions configures background refresh & sync.
type ReportSyncOptions struct {
	Enabled bool   `json:"enabled"`
	Cadence string `json:"cadence"`
	Target  string `json:"target"`
}

// ReportSummary contains lightweight metadata for listing views.
type ReportSummary struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Folder        string     `json:"folder"`
	Tags          []string   `json:"tags"`
	Starred       bool       `json:"starred"`
	StarredAt     *time.Time `json:"starredAt,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	LastRunAt     *time.Time `json:"lastRunAt"`
	LastRunStatus string     `json:"lastRunStatus"`
}

// ReportStorage persists report definitions to SQLite/Turso.
type ReportStorage struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewReportStorage creates a new storage helper.
func NewReportStorage(db *sql.DB, logger *logrus.Logger) *ReportStorage {
	return &ReportStorage{db: db, logger: logger}
}

// EnsureSchema creates required tables and indexes.
func (s *ReportStorage) EnsureSchema() error {
	if s.db == nil {
		return errors.New("report storage database not available")
	}

	statement := `
CREATE TABLE IF NOT EXISTS reports (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	folder TEXT,
	tags TEXT,
	definition TEXT NOT NULL,
	filter TEXT,
	sync_options TEXT,
	starred BOOLEAN DEFAULT FALSE,
	starred_at DATETIME,
	last_run_at DATETIME,
	last_run_status TEXT,
	metadata TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reports_updated_at ON reports(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_folder ON reports(folder);
CREATE INDEX IF NOT EXISTS idx_reports_starred ON reports(starred DESC, starred_at DESC);

-- Alert rules table
CREATE TABLE IF NOT EXISTS alert_rules (
	id TEXT PRIMARY KEY,
	report_id TEXT NOT NULL,
	component_id TEXT NOT NULL,
	name TEXT NOT NULL,
	description TEXT,
	condition TEXT NOT NULL,
	actions TEXT NOT NULL,
	schedule TEXT,
	enabled BOOLEAN NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (report_id) REFERENCES reports(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_report ON alert_rules(report_id);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);

-- Alert history table
CREATE TABLE IF NOT EXISTS alert_history (
	id TEXT PRIMARY KEY,
	rule_id TEXT NOT NULL,
	triggered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	actual_value REAL NOT NULL,
	message TEXT,
	resolved BOOLEAN NOT NULL DEFAULT 0,
	resolved_at DATETIME,
	FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_alert_history_rule ON alert_history(rule_id, triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_history_unresolved ON alert_history(resolved, triggered_at DESC);

-- Report snapshots table
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

	if _, err := s.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure report schema: %w", err)
	}

	return nil
}

// SaveReport inserts or updates a report definition.
func (s *ReportStorage) SaveReport(report *Report) error {
	if s.db == nil {
		return errors.New("report storage database not available")
	}
	if report == nil {
		return errors.New("report is nil")
	}

	if report.ID == "" {
		report.ID = uuid.NewString()
	}
	if report.Metadata == nil {
		report.Metadata = map[string]string{}
	}

	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	report.UpdatedAt = time.Now().UTC()

	definitionJSON, err := json.Marshal(report.Definition)
	if err != nil {
		return fmt.Errorf("failed to marshal report definition: %w", err)
	}
	tagsJSON, err := json.Marshal(report.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}
	filterJSON, err := json.Marshal(report.Filter)
	if err != nil {
		return fmt.Errorf("failed to marshal filter: %w", err)
	}
	syncJSON, err := json.Marshal(report.SyncOptions)
	if err != nil {
		return fmt.Errorf("failed to marshal sync options: %w", err)
	}
	metadataJSON, err := json.Marshal(report.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var lastRun, starredAt interface{}
	if report.LastRunAt != nil {
		lastRun = report.LastRunAt.UTC()
	}
	if report.StarredAt != nil {
		starredAt = report.StarredAt.UTC()
	}

	query := `
INSERT INTO reports (
	id, name, description, folder, tags, definition, filter, sync_options, starred, starred_at, last_run_at, last_run_status, metadata, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	description = excluded.description,
	folder = excluded.folder,
	tags = excluded.tags,
	definition = excluded.definition,
	filter = excluded.filter,
	sync_options = excluded.sync_options,
	starred = excluded.starred,
	starred_at = excluded.starred_at,
	last_run_at = excluded.last_run_at,
	last_run_status = excluded.last_run_status,
	metadata = excluded.metadata,
	updated_at = excluded.updated_at;
`

	_, err = s.db.Exec(
		query,
		report.ID,
		report.Name,
		report.Description,
		report.Folder,
		string(tagsJSON),
		string(definitionJSON),
		string(filterJSON),
		string(syncJSON),
		report.Starred,
		starredAt,
		lastRun,
		report.LastRunStatus,
		string(metadataJSON),
		report.CreatedAt,
		report.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

// ListReports returns report summaries sorted by update time.
func (s *ReportStorage) ListReports() ([]ReportSummary, error) {
	if s.db == nil {
		return nil, errors.New("report storage database not available")
	}

	rows, err := s.db.Query(`
SELECT id, name, description, folder, tags, starred, starred_at, updated_at, last_run_at, last_run_status
FROM reports
ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	defer rows.Close()

	var summaries []ReportSummary
	for rows.Next() {
		var (
			id, name, description, folder, tagsJSON, status string
			starred                                         bool
			updatedAt                                       time.Time
			lastRun, starredAt                              sql.NullTime
		)
		if err := rows.Scan(&id, &name, &description, &folder, &tagsJSON, &starred, &starredAt, &updatedAt, &lastRun, &status); err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}
		var tags []string
		if tagsJSON != "" {
			_ = json.Unmarshal([]byte(tagsJSON), &tags)
		}
		var lastRunPtr, starredAtPtr *time.Time
		if lastRun.Valid {
			t := lastRun.Time
			lastRunPtr = &t
		}
		if starredAt.Valid {
			t := starredAt.Time
			starredAtPtr = &t
		}
		summaries = append(summaries, ReportSummary{
			ID:            id,
			Name:          name,
			Description:   description,
			Folder:        folder,
			Tags:          tags,
			Starred:       starred,
			StarredAt:     starredAtPtr,
			UpdatedAt:     updatedAt,
			LastRunAt:     lastRunPtr,
			LastRunStatus: status,
		})
	}

	return summaries, nil
}

// GetReport loads a full report definition by ID.
func (s *ReportStorage) GetReport(id string) (*Report, error) {
	if s.db == nil {
		return nil, errors.New("report storage database not available")
	}

	row := s.db.QueryRow(`
SELECT id, name, description, folder, tags, definition, filter, sync_options, starred, starred_at, last_run_at, last_run_status, metadata, created_at, updated_at
FROM reports
WHERE id = ?`, id)

	var (
		report                                                       Report
		tagsJSON, definitionJSON, filterJSON, syncJSON, metadataJSON string
		lastRun, starredAt                                           sql.NullTime
	)
	if err := row.Scan(
		&report.ID,
		&report.Name,
		&report.Description,
		&report.Folder,
		&tagsJSON,
		&definitionJSON,
		&filterJSON,
		&syncJSON,
		&report.Starred,
		&starredAt,
		&lastRun,
		&report.LastRunStatus,
		&metadataJSON,
		&report.CreatedAt,
		&report.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("report not found: %s", id)
		}
		return nil, fmt.Errorf("failed to load report: %w", err)
	}

	if tagsJSON != "" {
		_ = json.Unmarshal([]byte(tagsJSON), &report.Tags)
	}
	if definitionJSON != "" {
		_ = json.Unmarshal([]byte(definitionJSON), &report.Definition)
	}
	if filterJSON != "" {
		_ = json.Unmarshal([]byte(filterJSON), &report.Filter)
	}
	if syncJSON != "" {
		_ = json.Unmarshal([]byte(syncJSON), &report.SyncOptions)
	}
	if metadataJSON != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &report.Metadata)
	}
	if lastRun.Valid {
		report.LastRunAt = &lastRun.Time
	}
	if starredAt.Valid {
		report.StarredAt = &starredAt.Time
	}

	return &report, nil
}

// DeleteReport removes a report definition.
func (s *ReportStorage) DeleteReport(id string) error {
	if s.db == nil {
		return errors.New("report storage database not available")
	}

	if _, err := s.db.Exec(`DELETE FROM reports WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return nil
}

// UpdateRunState updates execution metadata (last run time/status).
func (s *ReportStorage) UpdateRunState(id, status string, runAt time.Time) error {
	if s.db == nil {
		return errors.New("report storage database not available")
	}

	_, err := s.db.Exec(`
UPDATE reports SET last_run_at = ?, last_run_status = ?, updated_at = MAX(updated_at, ?)
WHERE id = ?
`, runAt.UTC(), status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update report run state: %w", err)
	}

	return nil
}

// ToggleStarred toggles the starred status of a report.
func (s *ReportStorage) ToggleStarred(id string) (bool, error) {
	if s.db == nil {
		return false, errors.New("report storage database not available")
	}

	// Get current starred status
	var currentStarred bool
	err := s.db.QueryRow(`SELECT starred FROM reports WHERE id = ?`, id).Scan(&currentStarred)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("report not found: %s", id)
		}
		return false, fmt.Errorf("failed to get starred status: %w", err)
	}

	newStarred := !currentStarred
	var starredAt interface{}
	if newStarred {
		starredAt = time.Now().UTC()
	}

	_, err = s.db.Exec(`
UPDATE reports SET starred = ?, starred_at = ?, updated_at = ?
WHERE id = ?
`, newStarred, starredAt, time.Now().UTC(), id)
	if err != nil {
		return false, fmt.Errorf("failed to toggle starred: %w", err)
	}

	return newStarred, nil
}

// ListStarredReports returns starred reports ordered by starred date.
func (s *ReportStorage) ListStarredReports() ([]ReportSummary, error) {
	if s.db == nil {
		return nil, errors.New("report storage database not available")
	}

	rows, err := s.db.Query(`
SELECT id, name, description, folder, tags, starred, starred_at, updated_at, last_run_at, last_run_status
FROM reports
WHERE starred = TRUE
ORDER BY starred_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list starred reports: %w", err)
	}
	defer rows.Close()

	var summaries []ReportSummary
	for rows.Next() {
		var (
			id, name, description, folder, tagsJSON, status string
			starred                                         bool
			updatedAt                                       time.Time
			lastRun, starredAt                              sql.NullTime
		)
		if err := rows.Scan(&id, &name, &description, &folder, &tagsJSON, &starred, &starredAt, &updatedAt, &lastRun, &status); err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}
		var tags []string
		if tagsJSON != "" {
			_ = json.Unmarshal([]byte(tagsJSON), &tags)
		}
		var lastRunPtr, starredAtPtr *time.Time
		if lastRun.Valid {
			t := lastRun.Time
			lastRunPtr = &t
		}
		if starredAt.Valid {
			t := starredAt.Time
			starredAtPtr = &t
		}
		summaries = append(summaries, ReportSummary{
			ID:            id,
			Name:          name,
			Description:   description,
			Folder:        folder,
			Tags:          tags,
			Starred:       starred,
			StarredAt:     starredAtPtr,
			UpdatedAt:     updatedAt,
			LastRunAt:     lastRunPtr,
			LastRunStatus: status,
		})
	}

	return summaries, nil
}
