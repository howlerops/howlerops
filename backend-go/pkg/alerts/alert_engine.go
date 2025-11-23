package alerts

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
)

// AlertRule defines a monitoring rule for report metrics
type AlertRule struct {
	ID          string         `json:"id"`
	ReportID    string         `json:"reportId"`
	ComponentID string         `json:"componentId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Condition   AlertCondition `json:"condition"`
	Actions     []AlertAction  `json:"actions"`
	Schedule    string         `json:"schedule"` // Cron expression
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// AlertCondition defines when an alert should trigger
type AlertCondition struct {
	Metric     string  `json:"metric"`     // Column name or aggregation
	Operator   string  `json:"operator"`   // ">", "<", ">=", "<=", "=", "!="
	Threshold  float64 `json:"threshold"`  // Threshold value
	Lookback   string  `json:"lookback"`   // "1h", "24h", "7d" (for future use)
	Comparison string  `json:"comparison"` // Optional: "previous_period", "baseline"
}

// AlertAction defines what happens when an alert triggers
type AlertAction struct {
	Type    string `json:"type"`    // "email", "slack", "webhook"
	Target  string `json:"target"`  // Email address, Slack channel, URL
	Message string `json:"message"` // Template with variables
}

// AlertResult represents the outcome of evaluating an alert
type AlertResult struct {
	RuleID      string    `json:"ruleId"`
	Triggered   bool      `json:"triggered"`
	ActualValue float64   `json:"actualValue"`
	Threshold   float64   `json:"threshold"`
	Message     string    `json:"message"`
	EvaluatedAt time.Time `json:"evaluatedAt"`
}

// AlertHistory records alert trigger events
type AlertHistory struct {
	ID          string     `json:"id"`
	RuleID      string     `json:"ruleId"`
	TriggeredAt time.Time  `json:"triggeredAt"`
	ActualValue float64    `json:"actualValue"`
	Message     string     `json:"message"`
	Resolved    bool       `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
}

// AlertEngine manages alert evaluation and notification
type AlertEngine struct {
	db        *sql.DB
	logger    *logrus.Logger
	scheduler *cron.Cron
	entries   map[string]cron.EntryID
	mu        sync.RWMutex

	// Callback to run report components for evaluation
	evaluateComponent func(reportID, componentID string, filters map[string]interface{}) (*ComponentResult, error)
}

// ComponentResult mirrors the report execution result
type ComponentResult struct {
	ComponentID string
	Type        storage.ReportComponentType
	Columns     []string
	Rows        [][]interface{}
	RowCount    int64
	Error       string
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(db *sql.DB, logger *logrus.Logger) *AlertEngine {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronScheduler := cron.New(cron.WithParser(parser), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronScheduler.Start()

	return &AlertEngine{
		db:        db,
		logger:    logger,
		scheduler: cronScheduler,
		entries:   make(map[string]cron.EntryID),
	}
}

// SetEvaluateComponentCallback sets the callback for evaluating report components
func (e *AlertEngine) SetEvaluateComponentCallback(callback func(reportID, componentID string, filters map[string]interface{}) (*ComponentResult, error)) {
	e.evaluateComponent = callback
}

// EnsureSchema creates required tables
func (e *AlertEngine) EnsureSchema() error {
	statement := `
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
`

	if _, err := e.db.Exec(statement); err != nil {
		return fmt.Errorf("failed to ensure alert schema: %w", err)
	}

	return nil
}

// SaveRule persists an alert rule
func (e *AlertEngine) SaveRule(rule *AlertRule) error {
	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	if rule.ID == "" {
		rule.ID = uuid.NewString()
	}

	now := time.Now().UTC()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	conditionJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return fmt.Errorf("failed to marshal condition: %w", err)
	}

	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("failed to marshal actions: %w", err)
	}

	query := `
INSERT INTO alert_rules (id, report_id, component_id, name, description, condition, actions, schedule, enabled, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	report_id = excluded.report_id,
	component_id = excluded.component_id,
	name = excluded.name,
	description = excluded.description,
	condition = excluded.condition,
	actions = excluded.actions,
	schedule = excluded.schedule,
	enabled = excluded.enabled,
	updated_at = excluded.updated_at
`

	_, err = e.db.Exec(query,
		rule.ID, rule.ReportID, rule.ComponentID, rule.Name, rule.Description,
		string(conditionJSON), string(actionsJSON), rule.Schedule, rule.Enabled,
		rule.CreatedAt, rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save alert rule: %w", err)
	}

	// Update schedule
	if rule.Enabled && rule.Schedule != "" {
		e.scheduleRule(rule)
	} else {
		e.unscheduleRule(rule.ID)
	}

	return nil
}

// GetRule retrieves an alert rule by ID
func (e *AlertEngine) GetRule(id string) (*AlertRule, error) {
	var rule AlertRule
	var conditionJSON, actionsJSON string

	err := e.db.QueryRow(`
		SELECT id, report_id, component_id, name, description, condition, actions, schedule, enabled, created_at, updated_at
		FROM alert_rules WHERE id = ?`, id).Scan(
		&rule.ID, &rule.ReportID, &rule.ComponentID, &rule.Name, &rule.Description,
		&conditionJSON, &actionsJSON, &rule.Schedule, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert rule: %w", err)
	}

	if err := json.Unmarshal([]byte(conditionJSON), &rule.Condition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal condition: %w", err)
	}

	if err := json.Unmarshal([]byte(actionsJSON), &rule.Actions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actions: %w", err)
	}

	return &rule, nil
}

// ListRulesByReport retrieves all rules for a report
func (e *AlertEngine) ListRulesByReport(reportID string) ([]*AlertRule, error) {
	rows, err := e.db.Query(`
		SELECT id, report_id, component_id, name, description, condition, actions, schedule, enabled, created_at, updated_at
		FROM alert_rules WHERE report_id = ? ORDER BY created_at DESC`, reportID)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert rules: %w", err)
	}
	defer rows.Close()

	var rules []*AlertRule
	for rows.Next() {
		var rule AlertRule
		var conditionJSON, actionsJSON string

		if err := rows.Scan(&rule.ID, &rule.ReportID, &rule.ComponentID, &rule.Name, &rule.Description,
			&conditionJSON, &actionsJSON, &rule.Schedule, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}

		if err := json.Unmarshal([]byte(conditionJSON), &rule.Condition); err != nil {
			e.logger.WithError(err).Warn("Failed to unmarshal condition")
			continue
		}

		if err := json.Unmarshal([]byte(actionsJSON), &rule.Actions); err != nil {
			e.logger.WithError(err).Warn("Failed to unmarshal actions")
			continue
		}

		rules = append(rules, &rule)
	}

	return rules, nil
}

// DeleteRule removes an alert rule
func (e *AlertEngine) DeleteRule(id string) error {
	e.unscheduleRule(id)

	_, err := e.db.Exec("DELETE FROM alert_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	return nil
}

// EvaluateAlert evaluates a single alert rule
func (e *AlertEngine) EvaluateAlert(rule *AlertRule) (*AlertResult, error) {
	if e.evaluateComponent == nil {
		return nil, fmt.Errorf("evaluate component callback not set")
	}

	// Run the component to get current data
	result, err := e.evaluateComponent(rule.ReportID, rule.ComponentID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate component: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("component execution error: %s", result.Error)
	}

	// Extract metric value from result
	metricValue, err := e.extractMetricValue(result, rule.Condition.Metric)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metric: %w", err)
	}

	// Evaluate condition
	triggered := e.evaluateCondition(metricValue, rule.Condition)

	alertResult := &AlertResult{
		RuleID:      rule.ID,
		Triggered:   triggered,
		ActualValue: metricValue,
		Threshold:   rule.Condition.Threshold,
		EvaluatedAt: time.Now(),
		Message:     e.formatAlertMessage(rule, metricValue, triggered),
	}

	// Record history if triggered
	if triggered {
		if err := e.recordAlert(rule.ID, metricValue, alertResult.Message); err != nil {
			e.logger.WithError(err).Warn("Failed to record alert history")
		}

		// Send notifications
		for _, action := range rule.Actions {
			if err := e.sendNotification(&action, alertResult); err != nil {
				e.logger.WithError(err).WithField("action_type", action.Type).Warn("Failed to send notification")
			}
		}
	}

	return alertResult, nil
}

// extractMetricValue extracts the metric value from component results
func (e *AlertEngine) extractMetricValue(result *ComponentResult, metric string) (float64, error) {
	if len(result.Rows) == 0 {
		return 0, fmt.Errorf("no data returned from component")
	}

	// Find column index
	colIdx := -1
	for i, col := range result.Columns {
		if col == metric {
			colIdx = i
			break
		}
	}

	if colIdx == -1 {
		// If metric is not a column name, try to use first row first column
		if len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
			colIdx = 0
		} else {
			return 0, fmt.Errorf("metric column not found: %s", metric)
		}
	}

	// Get value from first row
	if len(result.Rows[0]) <= colIdx {
		return 0, fmt.Errorf("column index out of range")
	}

	value := result.Rows[0][colIdx]

	// Convert to float64
	return toFloat64(value)
}

// evaluateCondition checks if the condition is met
func (e *AlertEngine) evaluateCondition(value float64, condition AlertCondition) bool {
	switch condition.Operator {
	case ">":
		return value > condition.Threshold
	case "<":
		return value < condition.Threshold
	case ">=":
		return value >= condition.Threshold
	case "<=":
		return value <= condition.Threshold
	case "=":
		return value == condition.Threshold
	case "!=":
		return value != condition.Threshold
	default:
		e.logger.WithField("operator", condition.Operator).Warn("Unknown operator")
		return false
	}
}

// formatAlertMessage formats the alert notification message
func (e *AlertEngine) formatAlertMessage(rule *AlertRule, value float64, triggered bool) string {
	if triggered {
		return fmt.Sprintf("Alert '%s' triggered: %s is %.2f %s %.2f (threshold)",
			rule.Name, rule.Condition.Metric, value, rule.Condition.Operator, rule.Condition.Threshold)
	}
	return fmt.Sprintf("Alert '%s' not triggered: %s is %.2f (within threshold)", rule.Name, rule.Condition.Metric, value)
}

// recordAlert records an alert trigger in history
func (e *AlertEngine) recordAlert(ruleID string, value float64, message string) error {
	// Check for recent alert (deduplication - max 1 per hour)
	var lastAlertTime sql.NullTime
	err := e.db.QueryRow(`
		SELECT MAX(triggered_at) FROM alert_history
		WHERE rule_id = ? AND resolved = 0
	`, ruleID).Scan(&lastAlertTime)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check recent alerts: %w", err)
	}

	if lastAlertTime.Valid && time.Since(lastAlertTime.Time) < time.Hour {
		e.logger.WithField("rule_id", ruleID).Debug("Skipping duplicate alert (within 1 hour)")
		return nil
	}

	// Insert new alert
	_, err = e.db.Exec(`
		INSERT INTO alert_history (id, rule_id, triggered_at, actual_value, message, resolved)
		VALUES (?, ?, ?, ?, ?, 0)
	`, uuid.NewString(), ruleID, time.Now().UTC(), value, message)

	return err
}

// sendNotification sends alert notifications
func (e *AlertEngine) sendNotification(action *AlertAction, result *AlertResult) error {
	switch action.Type {
	case "email":
		return e.sendEmail(action.Target, result)
	case "slack":
		return e.sendSlack(action.Target, result)
	case "webhook":
		return e.sendWebhook(action.Target, result)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// sendEmail sends email notification (placeholder for now)
func (e *AlertEngine) sendEmail(to string, result *AlertResult) error {
	// TODO: Implement email sending (SMTP, SendGrid, etc.)
	e.logger.WithFields(logrus.Fields{
		"to":      to,
		"message": result.Message,
	}).Info("Would send email alert")
	return nil
}

// sendSlack sends Slack notification (placeholder for now)
func (e *AlertEngine) sendSlack(channel string, result *AlertResult) error {
	// TODO: Implement Slack webhook
	e.logger.WithFields(logrus.Fields{
		"channel": channel,
		"message": result.Message,
	}).Info("Would send Slack alert")
	return nil
}

// sendWebhook sends webhook notification (placeholder for now)
func (e *AlertEngine) sendWebhook(url string, result *AlertResult) error {
	// TODO: Implement HTTP POST webhook
	e.logger.WithFields(logrus.Fields{
		"url":     url,
		"message": result.Message,
	}).Info("Would send webhook alert")
	return nil
}

// GetAlertHistory retrieves alert history for a rule
func (e *AlertEngine) GetAlertHistory(ruleID string, limit int) ([]*AlertHistory, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := e.db.Query(`
		SELECT id, rule_id, triggered_at, actual_value, message, resolved, resolved_at
		FROM alert_history WHERE rule_id = ?
		ORDER BY triggered_at DESC LIMIT ?
	`, ruleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}
	defer rows.Close()

	var history []*AlertHistory
	for rows.Next() {
		var h AlertHistory
		var resolvedAt sql.NullTime

		if err := rows.Scan(&h.ID, &h.RuleID, &h.TriggeredAt, &h.ActualValue, &h.Message, &h.Resolved, &resolvedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert history: %w", err)
		}

		if resolvedAt.Valid {
			h.ResolvedAt = &resolvedAt.Time
		}

		history = append(history, &h)
	}

	return history, nil
}

// Scheduling

func (e *AlertEngine) scheduleRule(rule *AlertRule) {
	if rule.Schedule == "" {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove existing schedule
	if entryID, ok := e.entries[rule.ID]; ok {
		e.scheduler.Remove(entryID)
		delete(e.entries, rule.ID)
	}

	// Add new schedule
	entryID, err := e.scheduler.AddFunc(rule.Schedule, func() {
		if _, err := e.EvaluateAlert(rule); err != nil {
			e.logger.WithError(err).WithField("rule_id", rule.ID).Warn("Scheduled alert evaluation failed")
		}
	})

	if err != nil {
		e.logger.WithError(err).WithField("rule_id", rule.ID).Warn("Failed to schedule alert")
		return
	}

	e.entries[rule.ID] = entryID
}

func (e *AlertEngine) unscheduleRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if entryID, ok := e.entries[ruleID]; ok {
		e.scheduler.Remove(entryID)
		delete(e.entries, ruleID)
	}
}

// Shutdown stops the alert scheduler
func (e *AlertEngine) Shutdown() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.scheduler != nil {
		e.scheduler.Stop()
		e.scheduler = nil
	}
	e.entries = make(map[string]cron.EntryID)
}

// Helper functions

func toFloat64(v interface{}) (float64, error) {
	if v == nil {
		return 0, fmt.Errorf("value is nil")
	}

	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string to float: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}
