package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	cron "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/backend-go/pkg/ai"
	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage"
)

// ReportService coordinates report persistence and execution.
type ReportService struct {
	storage *storage.ReportStorage
	db      *DatabaseService
	ai      *ai.Service
	logger  *logrus.Logger
	ctx     context.Context
	mu      sync.RWMutex
	cron    *cron.Cron
	entries map[string]cron.EntryID
}

// ReportRunRequest represents a run invocation from the UI.
type ReportRunRequest struct {
	ReportID     string                 `json:"reportId"`
	ComponentIDs []string               `json:"componentIds"`
	FilterValues map[string]interface{} `json:"filters"`
	Force        bool                   `json:"force"`
}

// ReportRunResponse contains aggregated component data.
type ReportRunResponse struct {
	ReportID    string                  `json:"reportId"`
	StartedAt   time.Time               `json:"startedAt"`
	CompletedAt time.Time               `json:"completedAt"`
	Results     []ReportComponentResult `json:"results"`
}

// ReportComponentResult wraps query or LLM outputs.
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
}

// NewReportService builds a new report service.
func NewReportService(logger *logrus.Logger, db *DatabaseService) *ReportService {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronScheduler := cron.New(cron.WithParser(parser), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	cronScheduler.Start()

	return &ReportService{
		logger:  logger,
		db:      db,
		cron:    cronScheduler,
		entries: make(map[string]cron.EntryID),
	}
}

// SetContext wires the Wails context for downstream calls.
func (s *ReportService) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

// SetStorage injects the report storage implementation once available.
func (s *ReportService) SetStorage(storage *storage.ReportStorage) {
	s.mu.Lock()
	s.storage = storage
	s.mu.Unlock()

	go s.bootstrapSchedules()
}

// SetAIService wires the AI runtime for LLM blocks.
func (s *ReportService) SetAIService(aiService *ai.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ai = aiService
}

// Shutdown stops the background scheduler.
func (s *ReportService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}
	s.entries = make(map[string]cron.EntryID)
}

func (s *ReportService) withStorage() (*storage.ReportStorage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.storage == nil {
		return nil, fmt.Errorf("report storage not initialised")
	}
	return s.storage, nil
}

func (s *ReportService) withAI() *ai.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ai
}

func (s *ReportService) bootstrapSchedules() {
	store, err := s.withStorage()
	if err != nil {
		return
	}
	reports, err := store.ListReports()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to list reports for scheduling")
		return
	}

	for _, summary := range reports {
		report, err := store.GetReport(summary.ID)
		if err != nil {
			s.logger.WithError(err).WithField("report_id", summary.ID).Warn("Failed to load report for scheduling")
			continue
		}
		s.configureSchedule(report)
	}
}

func (s *ReportService) ensureCronLocked() {
	if s.cron == nil {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		s.cron = cron.New(cron.WithParser(parser), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
		s.cron.Start()
	}
	if s.entries == nil {
		s.entries = make(map[string]cron.EntryID)
	}
}

func (s *ReportService) configureSchedule(report *storage.Report) {
	if report == nil || report.ID == "" {
		return
	}

	// Skip local scheduling when the report is configured for a remote target
	if strings.EqualFold(strings.TrimSpace(report.SyncOptions.Target), "remote") {
		s.removeSchedule(report.ID)
		return
	}

	cadence := strings.TrimSpace(report.SyncOptions.Cadence)
	if !report.SyncOptions.Enabled || cadence == "" {
		s.removeSchedule(report.ID)
		return
	}

	normalized := normalizeCadence(cadence)
	reportID := report.ID

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCronLocked()

	if entry, ok := s.entries[reportID]; ok {
		s.cron.Remove(entry)
		delete(s.entries, reportID)
	}

	entryID, err := s.cron.AddFunc(normalized, func() {
		if _, err := s.RunReport(&ReportRunRequest{ReportID: reportID}); err != nil {
			s.logger.WithError(err).WithField("report_id", reportID).Warn("Scheduled report execution failed")
		}
	})
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"report_id": reportID,
			"cadence":   cadence,
		}).Warn("Failed to schedule report")
		return
	}

	s.entries[reportID] = entryID
}

func (s *ReportService) removeSchedule(reportID string) {
	if reportID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron == nil {
		return
	}
	if entry, ok := s.entries[reportID]; ok {
		s.cron.Remove(entry)
		delete(s.entries, reportID)
	}
}

// SaveReport persists a report definition.
func (s *ReportService) SaveReport(report *storage.Report) (*storage.Report, error) {
	store, err := s.withStorage()
	if err != nil {
		return nil, err
	}

	if err := store.SaveReport(report); err != nil {
		return nil, err
	}

	s.configureSchedule(report)

	return report, nil
}

// ListReports returns report summaries for navigation lists.
func (s *ReportService) ListReports() ([]storage.ReportSummary, error) {
	store, err := s.withStorage()
	if err != nil {
		return nil, err
	}
	return store.ListReports()
}

// GetReport returns a specific report definition.
func (s *ReportService) GetReport(id string) (*storage.Report, error) {
	store, err := s.withStorage()
	if err != nil {
		return nil, err
	}
	return store.GetReport(id)
}

// DeleteReport removes a report definition.
func (s *ReportService) DeleteReport(id string) error {
	store, err := s.withStorage()
	if err != nil {
		return err
	}
	if err := store.DeleteReport(id); err != nil {
		return err
	}
	s.removeSchedule(id)
	return nil
}

// RunReport executes configured components sequentially.
func (s *ReportService) RunReport(req *ReportRunRequest) (*ReportRunResponse, error) {
	store, err := s.withStorage()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.ReportID == "" {
		return nil, fmt.Errorf("report id is required")
	}

	report, err := store.GetReport(req.ReportID)
	if err != nil {
		return nil, err
	}

	componentFilter := map[string]struct{}{}
	for _, id := range req.ComponentIDs {
		componentFilter[id] = struct{}{}
	}

	results := make([]ReportComponentResult, 0, len(report.Definition.Components))
	resultIndex := make(map[string]ReportComponentResult)
	started := time.Now()

	var firstErr error
	for _, component := range report.Definition.Components {
		componentCopy := component
		if len(componentFilter) > 0 {
			if _, ok := componentFilter[componentCopy.ID]; !ok {
				continue
			}
		}

		res := s.runComponent(report, &componentCopy, req.FilterValues, resultIndex)
		if res.Error != "" && firstErr == nil {
			firstErr = fmt.Errorf(res.Error)
		}
		results = append(results, res)
		resultIndex[componentCopy.ID] = res
	}

	resp := &ReportRunResponse{
		ReportID:    report.ID,
		StartedAt:   started,
		CompletedAt: time.Now(),
		Results:     results,
	}

	status := "ok"
	if firstErr != nil {
		status = "error"
	}

	if err := store.UpdateRunState(report.ID, status, resp.CompletedAt); err != nil {
		s.logger.WithError(err).Warn("Failed to update report run state")
	}

	if firstErr != nil {
		return resp, firstErr
	}

	return resp, nil
}

func (s *ReportService) runComponent(report *storage.Report, component *storage.ReportComponent, filters map[string]interface{}, cache map[string]ReportComponentResult) ReportComponentResult {
	res := ReportComponentResult{
		ComponentID: component.ID,
		Type:        component.Type,
		Metadata:    map[string]any{},
	}

	switch component.Type {
	case storage.ReportComponentLLM:
		return s.runLLMComponent(report, component, filters, cache)
	default:
		return s.runQueryComponent(report, component, filters)
	}

	return res
}

func (s *ReportService) runQueryComponent(report *storage.Report, component *storage.ReportComponent, filters map[string]interface{}) ReportComponentResult {
	res := ReportComponentResult{
		ComponentID: component.ID,
		Type:        component.Type,
	}

	if s.db == nil {
		res.Error = "database service not available"
		return res
	}

	queryConfig := component.Query
	if queryConfig.ConnectionID == "" {
		res.Error = "component missing connection"
		return res
	}

	sqlText := strings.TrimSpace(queryConfig.SQL)
	if sqlText == "" {
		res.Error = "component SQL is empty"
		return res
	}

	sqlText = applyFilterPlaceholders(sqlText, component, filters)

	limit := 1000
	if queryConfig.Limit != nil && *queryConfig.Limit > 0 {
		limit = *queryConfig.Limit
	}

	opts := &database.QueryOptions{
		Timeout:  60 * time.Second,
		ReadOnly: true,
		Limit:    limit,
	}

	started := time.Now()
	result, err := s.db.ExecuteQuery(queryConfig.ConnectionID, sqlText, opts)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.DurationMS = int64(time.Since(started).Milliseconds())
	res.Columns = result.Columns
	res.Rows = result.Rows
	res.RowCount = result.RowCount

	return res
}

func (s *ReportService) runLLMComponent(report *storage.Report, component *storage.ReportComponent, filters map[string]interface{}, prior map[string]ReportComponentResult) ReportComponentResult {
	res := ReportComponentResult{
		ComponentID: component.ID,
		Type:        component.Type,
	}

	aiService := s.withAI()
	if aiService == nil {
		res.Error = "AI service not configured"
		return res
	}
	if component.LLM == nil {
		res.Error = "LLM settings missing"
		return res
	}

	contextBlob := buildContextPayload(component.LLM.ContextComponents, prior)
	prompt := injectPlaceholders(component.LLM.PromptTemplate, filters, contextBlob)

	chatReq := &ai.ChatRequest{
		Prompt:      prompt,
		Context:     contextBlob["context"],
		Provider:    component.LLM.Provider,
		Model:       component.LLM.Model,
		MaxTokens:   component.LLM.MaxTokens,
		Temperature: component.LLM.Temperature,
		Metadata: map[string]string{
			"report_id":    report.ID,
			"component_id": component.ID,
		},
	}
	for k, v := range component.LLM.Metadata {
		if chatReq.Metadata == nil {
			chatReq.Metadata = map[string]string{}
		}
		chatReq.Metadata[k] = v
	}

	start := time.Now()
	resp, err := aiService.Chat(s.ctx, chatReq)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Content = resp.Content
	res.Metadata = map[string]any{
		"provider":   resp.Provider,
		"model":      resp.Model,
		"tokens":     resp.TokensUsed,
		"durationMs": time.Since(start).Milliseconds(),
	}

	return res
}

func applyFilterPlaceholders(sql string, component *storage.ReportComponent, filters map[string]interface{}) string {
	if len(filters) == 0 || len(component.Query.TopLevelFilter) == 0 {
		return sql
	}
	replacements := map[string]string{}
	for _, key := range component.Query.TopLevelFilter {
		if val, ok := filters[key]; ok {
			replacements["{{"+key+"}}"] = formatSQLValue(val)
		}
	}
	return replaceTokens(sql, replacements)
}

func formatSQLValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case string:
		escaped := strings.ReplaceAll(val, "'", "''")
		return "'" + escaped + "'"
	case time.Time:
		return "'" + val.UTC().Format(time.RFC3339) + "'"
	case fmt.Stringer:
		return "'" + strings.ReplaceAll(val.String(), "'", "''") + "'"
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func buildContextPayload(componentIDs []string, prior map[string]ReportComponentResult) map[string]string {
	payload := map[string]string{}
	if len(componentIDs) == 0 {
		return payload
	}

	var builder strings.Builder
	for _, id := range componentIDs {
		result, ok := prior[id]
		if !ok {
			continue
		}
		fragment := map[string]interface{}{
			"componentId": id,
			"columns":     result.Columns,
			"rows":        result.Rows,
		}
		blob, err := json.Marshal(fragment)
		if err != nil {
			continue
		}
		payload["component."+id] = string(blob)
		builder.Write(blob)
		builder.WriteString("\n")
	}

	if builder.Len() > 0 {
		payload["context"] = builder.String()
	}

	return payload
}

func injectPlaceholders(template string, filters map[string]interface{}, context map[string]string) string {
	replacements := map[string]string{}
	for key, value := range filters {
		replacements["{{"+key+"}}"] = fmt.Sprintf("%v", value)
	}
	for key, value := range context {
		replacements["{{"+key+"}}"] = value
	}
	return replaceTokens(template, replacements)
}

func replaceTokens(input string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return input
	}
	// Deterministic ordering for tests/debugging
	keys := make([]string, 0, len(replacements))
	for k := range replacements {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := input
	for _, key := range keys {
		result = strings.ReplaceAll(result, key, replacements[key])
	}
	return result
}

func normalizeCadence(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "@every 5m"
	}
	if strings.HasPrefix(trimmed, "@every") {
		return trimmed
	}
	if _, err := time.ParseDuration(trimmed); err == nil {
		return "@every " + trimmed
	}
	return trimmed
}
