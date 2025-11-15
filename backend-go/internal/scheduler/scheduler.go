package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/internal/templates"
	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// QueryExecutor defines the interface for executing queries
// This allows us to inject different executors (real DB, mock, etc.)
type QueryExecutor interface {
	ExecuteQuery(ctx context.Context, sql string, connectionID string) (*QueryResult, error)
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Rows         []map[string]interface{} `json:"rows"`
	RowsReturned int                      `json:"rows_returned"`
	DurationMs   int64                    `json:"duration_ms"`
	Error        string                   `json:"error,omitempty"`
}

// Scheduler manages scheduled query execution
type Scheduler struct {
	scheduleStore   *turso.ScheduleStore
	templateStore   *turso.TemplateStore
	templateService *templates.Service
	queryExecutor   QueryExecutor
	logger          *logrus.Logger

	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool

	// Configuration
	interval      time.Duration
	maxConcurrent int
	timeout       time.Duration
}

// Config holds scheduler configuration
type Config struct {
	Interval      time.Duration // How often to check for due schedules (default: 1 minute)
	MaxConcurrent int           // Max concurrent executions (default: 10)
	Timeout       time.Duration // Execution timeout (default: 5 minutes)
}

// NewScheduler creates a new scheduler
func NewScheduler(
	scheduleStore *turso.ScheduleStore,
	templateStore *turso.TemplateStore,
	templateService *templates.Service,
	queryExecutor QueryExecutor,
	logger *logrus.Logger,
	config Config,
) *Scheduler {
	// Set defaults
	if config.Interval == 0 {
		config.Interval = 1 * time.Minute
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	return &Scheduler{
		scheduleStore:   scheduleStore,
		templateStore:   templateStore,
		templateService: templateService,
		queryExecutor:   queryExecutor,
		logger:          logger,
		stopChan:        make(chan struct{}),
		interval:        config.Interval,
		maxConcurrent:   config.MaxConcurrent,
		timeout:         config.Timeout,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true
	s.ticker = time.NewTicker(s.interval)

	s.wg.Add(1)
	go s.run()

	s.logger.WithFields(logrus.Fields{
		"interval":       s.interval,
		"max_concurrent": s.maxConcurrent,
		"timeout":        s.timeout,
	}).Info("Scheduler started")

	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	s.ticker.Stop()
	s.wg.Wait()

	s.logger.Info("Scheduler stopped")
	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	defer s.wg.Done()

	// Run immediately on start
	s.checkAndExecuteSchedules()

	for {
		select {
		case <-s.ticker.C:
			s.checkAndExecuteSchedules()
		case <-s.stopChan:
			return
		}
	}
}

// checkAndExecuteSchedules checks for due schedules and executes them
func (s *Scheduler) checkAndExecuteSchedules() {
	ctx := context.Background()

	// Get all schedules that need to be executed
	schedules, err := s.scheduleStore.GetDueSchedules(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get due schedules")
		return
	}

	if len(schedules) == 0 {
		s.logger.Debug("No schedules due for execution")
		return
	}

	s.logger.WithField("count", len(schedules)).Info("Found schedules due for execution")

	// Use a semaphore to limit concurrent executions
	sem := make(chan struct{}, s.maxConcurrent)

	for _, schedule := range schedules {
		select {
		case <-s.stopChan:
			return
		case sem <- struct{}{}:
			s.wg.Add(1)
			go func(sched *turso.QuerySchedule) {
				defer s.wg.Done()
				defer func() { <-sem }()

				s.executeSchedule(context.Background(), sched)
			}(schedule)
		}
	}
}

// executeSchedule executes a single schedule
func (s *Scheduler) executeSchedule(ctx context.Context, schedule *turso.QuerySchedule) {
	startTime := time.Now()

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"name":        schedule.Name,
		"template_id": schedule.TemplateID,
	}).Info("Executing schedule")

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Get template
	template, err := s.templateStore.GetByID(execCtx, schedule.TemplateID)
	if err != nil {
		s.recordFailure(ctx, schedule, startTime, fmt.Errorf("failed to get template: %w", err))
		return
	}

	// Substitute parameters
	sql, err := templates.SubstituteParameters(template.SQLTemplate, schedule.Parameters, template.Parameters)
	if err != nil {
		s.recordFailure(ctx, schedule, startTime, fmt.Errorf("parameter substitution failed: %w", err))
		return
	}

	// Execute query
	result, err := s.queryExecutor.ExecuteQuery(execCtx, sql, "")
	if err != nil {
		s.recordFailure(ctx, schedule, startTime, fmt.Errorf("query execution failed: %w", err))
		return
	}

	// Check if execution was cancelled
	if execCtx.Err() == context.DeadlineExceeded {
		s.recordTimeout(ctx, schedule, startTime)
		return
	}

	// Record success
	s.recordSuccess(ctx, schedule, startTime, result)

	// Update next run time
	s.updateNextRun(ctx, schedule)

	s.logger.WithFields(logrus.Fields{
		"schedule_id":   schedule.ID,
		"rows_returned": result.RowsReturned,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}).Info("Schedule executed successfully")
}

// recordSuccess records a successful execution
func (s *Scheduler) recordSuccess(ctx context.Context, schedule *turso.QuerySchedule, startTime time.Time, result *QueryResult) {
	duration := time.Since(startTime)

	// Create result preview (first 10 rows)
	preview := s.createResultPreview(result.Rows)

	execution := &turso.ScheduleExecution{
		ScheduleID:    schedule.ID,
		ExecutedAt:    startTime,
		Status:        "success",
		DurationMs:    int(duration.Milliseconds()),
		RowsReturned:  result.RowsReturned,
		ResultPreview: preview,
	}

	if err := s.scheduleStore.RecordExecution(ctx, execution); err != nil {
		s.logger.WithError(err).Error("Failed to record execution")
	}

	// Send notification if configured
	if schedule.NotificationEmail != "" {
		s.sendSuccessNotification(schedule, execution)
	}
}

// recordFailure records a failed execution
func (s *Scheduler) recordFailure(ctx context.Context, schedule *turso.QuerySchedule, startTime time.Time, err error) {
	duration := time.Since(startTime)

	execution := &turso.ScheduleExecution{
		ScheduleID:   schedule.ID,
		ExecutedAt:   startTime,
		Status:       "failed",
		DurationMs:   int(duration.Milliseconds()),
		ErrorMessage: err.Error(),
	}

	if execErr := s.scheduleStore.RecordExecution(ctx, execution); execErr != nil {
		s.logger.WithError(execErr).Error("Failed to record execution")
	}

	// Update schedule status to failed
	if updateErr := s.scheduleStore.UpdateStatus(ctx, schedule.ID, "failed"); updateErr != nil {
		s.logger.WithError(updateErr).Error("Failed to update schedule status")
	}

	// Send failure notification
	if schedule.NotificationEmail != "" {
		s.sendFailureNotification(schedule, execution)
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"error":       err.Error(),
	}).Error("Schedule execution failed")
}

// recordTimeout records a timeout
func (s *Scheduler) recordTimeout(ctx context.Context, schedule *turso.QuerySchedule, startTime time.Time) {
	duration := time.Since(startTime)

	execution := &turso.ScheduleExecution{
		ScheduleID:   schedule.ID,
		ExecutedAt:   startTime,
		Status:       "timeout",
		DurationMs:   int(duration.Milliseconds()),
		ErrorMessage: fmt.Sprintf("execution exceeded timeout of %v", s.timeout),
	}

	if err := s.scheduleStore.RecordExecution(ctx, execution); err != nil {
		s.logger.WithError(err).Error("Failed to record execution")
	}

	// Update schedule status to failed
	if err := s.scheduleStore.UpdateStatus(ctx, schedule.ID, "failed"); err != nil {
		s.logger.WithError(err).Error("Failed to update schedule status")
	}

	// Send timeout notification
	if schedule.NotificationEmail != "" {
		s.sendFailureNotification(schedule, execution)
	}

	s.logger.WithField("schedule_id", schedule.ID).Warn("Schedule execution timed out")
}

// updateNextRun calculates and updates the next run time
func (s *Scheduler) updateNextRun(ctx context.Context, schedule *turso.QuerySchedule) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	cronSchedule, err := parser.Parse(schedule.Frequency)
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse cron expression")
		return
	}

	now := time.Now()
	nextRun := cronSchedule.Next(now)

	if err := s.scheduleStore.UpdateNextRun(ctx, schedule.ID, now, nextRun); err != nil {
		s.logger.WithError(err).Error("Failed to update next run time")
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"next_run":    nextRun.Format(time.RFC3339),
	}).Debug("Updated next run time")
}

// createResultPreview creates a JSON preview of the first 10 rows
func (s *Scheduler) createResultPreview(rows []map[string]interface{}) string {
	if len(rows) == 0 {
		return ""
	}

	// Take first 10 rows
	preview := rows
	if len(preview) > 10 {
		preview = preview[:10]
	}

	// Marshal to JSON
	data, err := json.Marshal(preview)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to create result preview")
		return ""
	}

	// Limit size to 10KB
	if len(data) > 10240 {
		return string(data[:10240]) + "..."
	}

	return string(data)
}

// sendSuccessNotification sends a success notification email
func (s *Scheduler) sendSuccessNotification(schedule *turso.QuerySchedule, execution *turso.ScheduleExecution) {
	// TODO: Implement email notification
	// This would integrate with an email service (SendGrid, AWS SES, etc.)
	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"email":       schedule.NotificationEmail,
		"rows":        execution.RowsReturned,
	}).Info("Would send success notification")
}

// sendFailureNotification sends a failure notification email
func (s *Scheduler) sendFailureNotification(schedule *turso.QuerySchedule, execution *turso.ScheduleExecution) {
	// TODO: Implement email notification
	// This would integrate with an email service (SendGrid, AWS SES, etc.)
	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"email":       schedule.NotificationEmail,
		"error":       execution.ErrorMessage,
	}).Warn("Would send failure notification")
}

// ExecuteNow manually executes a schedule immediately (for testing or manual triggers)
func (s *Scheduler) ExecuteNow(ctx context.Context, scheduleID string) error {
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	if schedule.Status != "active" {
		return fmt.Errorf("schedule is not active")
	}

	go s.executeSchedule(context.Background(), schedule)

	s.logger.WithField("schedule_id", scheduleID).Info("Manual execution triggered")
	return nil
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Count active schedules
	activeFilters := turso.ScheduleFilters{
		Status: stringPtr("active"),
		Limit:  1000,
	}
	activeSchedules, err := s.scheduleStore.List(ctx, activeFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to count active schedules: %w", err)
	}

	// Count paused schedules
	pausedFilters := turso.ScheduleFilters{
		Status: stringPtr("paused"),
		Limit:  1000,
	}
	pausedSchedules, err := s.scheduleStore.List(ctx, pausedFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to count paused schedules: %w", err)
	}

	// Count failed schedules
	failedFilters := turso.ScheduleFilters{
		Status: stringPtr("failed"),
		Limit:  1000,
	}
	failedSchedules, err := s.scheduleStore.List(ctx, failedFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to count failed schedules: %w", err)
	}

	stats := map[string]interface{}{
		"running":          s.IsRunning(),
		"interval_seconds": s.interval.Seconds(),
		"max_concurrent":   s.maxConcurrent,
		"timeout_seconds":  s.timeout.Seconds(),
		"active_schedules": len(activeSchedules),
		"paused_schedules": len(pausedSchedules),
		"failed_schedules": len(failedSchedules),
	}

	return stats, nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
