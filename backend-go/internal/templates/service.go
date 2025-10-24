package templates

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// Service handles business logic for query templates and schedules
type Service struct {
	templateStore *turso.TemplateStore
	scheduleStore *turso.ScheduleStore
	orgRepo       organization.Repository
	logger        *logrus.Logger
	cronParser    cron.Parser
}

// NewService creates a new templates service
func NewService(
	templateStore *turso.TemplateStore,
	scheduleStore *turso.ScheduleStore,
	orgRepo organization.Repository,
	logger *logrus.Logger,
) *Service {
	return &Service{
		templateStore: templateStore,
		scheduleStore: scheduleStore,
		orgRepo:       orgRepo,
		logger:        logger,
		cronParser:    cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	Name           string                      `json:"name" validate:"required,min=3,max=100"`
	Description    string                      `json:"description" validate:"max=500"`
	SQLTemplate    string                      `json:"sql_template" validate:"required"`
	Parameters     []turso.TemplateParameter   `json:"parameters"`
	Tags           []string                    `json:"tags"`
	Category       string                      `json:"category" validate:"required,oneof=reporting analytics maintenance custom"`
	OrganizationID *string                     `json:"organization_id"`
	IsPublic       bool                        `json:"is_public"`
}

// CreateTemplate creates a new query template
func (s *Service) CreateTemplate(ctx context.Context, input CreateTemplateInput, userID string) (*turso.QueryTemplate, error) {
	// Validate SQL template syntax
	if err := s.ValidateTemplate(ctx, input.SQLTemplate, input.Parameters); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Check organization permissions if template is org-scoped
	if input.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *input.OrganizationID, userID)
		if err != nil || member == nil {
			return nil, fmt.Errorf("user not member of organization")
		}

		// Require connections:create permission to create templates
		if !organization.HasPermission(member.Role, organization.PermCreateConnections) {
			s.createAuditLog(ctx, *input.OrganizationID, userID, "permission_denied", "template", nil, map[string]interface{}{
				"permission": string(organization.PermCreateConnections),
				"action":     "create_template",
			})
			return nil, fmt.Errorf("insufficient permissions to create templates")
		}
	}

	template := &turso.QueryTemplate{
		Name:           input.Name,
		Description:    input.Description,
		SQLTemplate:    input.SQLTemplate,
		Parameters:     input.Parameters,
		Tags:           input.Tags,
		Category:       input.Category,
		OrganizationID: input.OrganizationID,
		CreatedBy:      userID,
		IsPublic:       input.IsPublic,
	}

	if err := s.templateStore.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Create audit log if in organization
	if input.OrganizationID != nil {
		s.createAuditLog(ctx, *input.OrganizationID, userID, "create_template", "template", &template.ID, map[string]interface{}{
			"template_name": template.Name,
			"category":      template.Category,
			"is_public":     template.IsPublic,
		})
	}

	s.logger.WithFields(logrus.Fields{
		"template_id": template.ID,
		"name":        template.Name,
		"created_by":  userID,
	}).Info("Template created")

	return template, nil
}

// GetTemplate retrieves a template by ID with permission checks
func (s *Service) GetTemplate(ctx context.Context, templateID, userID string) (*turso.QueryTemplate, error) {
	template, err := s.templateStore.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Check access permissions
	if !s.canAccessTemplate(ctx, template, userID) {
		return nil, fmt.Errorf("access denied to template")
	}

	return template, nil
}

// ListTemplates retrieves templates accessible to the user
func (s *Service) ListTemplates(ctx context.Context, userID string, filters turso.TemplateFilters) ([]*turso.QueryTemplate, error) {
	// If organization filter is set, verify membership
	if filters.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *filters.OrganizationID, userID)
		if err != nil || member == nil {
			return nil, fmt.Errorf("user not member of organization")
		}

		// Check view permissions
		if !organization.HasPermission(member.Role, organization.PermViewConnections) {
			return nil, fmt.Errorf("insufficient permissions to view templates")
		}
	}

	templates, err := s.templateStore.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Filter templates based on access
	var accessible []*turso.QueryTemplate
	for _, template := range templates {
		if s.canAccessTemplate(ctx, template, userID) {
			accessible = append(accessible, template)
		}
	}

	return accessible, nil
}

// UpdateTemplate updates a template with permission checks
func (s *Service) UpdateTemplate(ctx context.Context, template *turso.QueryTemplate, userID string) error {
	// Get existing template
	existing, err := s.templateStore.GetByID(ctx, template.ID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Check update permissions
	if existing.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *existing.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Only creator or admin can update
		if !organization.CanUpdateResource(member.Role, existing.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to update this template")
		}
	} else if existing.CreatedBy != userID {
		return fmt.Errorf("cannot update another user's personal template")
	}

	// Validate if SQL changed
	if template.SQLTemplate != existing.SQLTemplate {
		if err := s.ValidateTemplate(ctx, template.SQLTemplate, template.Parameters); err != nil {
			return fmt.Errorf("invalid template: %w", err)
		}
	}

	// Keep immutable fields
	template.CreatedBy = existing.CreatedBy
	template.CreatedAt = existing.CreatedAt
	template.UsageCount = existing.UsageCount

	if err := s.templateStore.Update(ctx, template); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Create audit log if in organization
	if template.OrganizationID != nil {
		s.createAuditLog(ctx, *template.OrganizationID, userID, "update_template", "template", &template.ID, map[string]interface{}{
			"template_name": template.Name,
		})
	}

	return nil
}

// DeleteTemplate deletes a template with permission checks
func (s *Service) DeleteTemplate(ctx context.Context, templateID, userID string) error {
	template, err := s.templateStore.GetByID(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Check delete permissions
	if template.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *template.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Only creator or admin can delete
		if !organization.CanDeleteResource(member.Role, template.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to delete this template")
		}

		// Create audit log
		s.createAuditLog(ctx, *template.OrganizationID, userID, "delete_template", "template", &templateID, map[string]interface{}{
			"template_name": template.Name,
		})
	} else if template.CreatedBy != userID {
		return fmt.Errorf("cannot delete another user's personal template")
	}

	return s.templateStore.Delete(ctx, templateID)
}

// ExecuteTemplate executes a template with parameter substitution
func (s *Service) ExecuteTemplate(ctx context.Context, templateID string, params map[string]interface{}, userID string) (string, error) {
	// Get template and check access
	template, err := s.GetTemplate(ctx, templateID, userID)
	if err != nil {
		return "", err
	}

	// Substitute parameters
	sql, err := SubstituteParameters(template.SQLTemplate, params, template.Parameters)
	if err != nil {
		return "", fmt.Errorf("parameter substitution failed: %w", err)
	}

	// Increment usage counter (non-blocking)
	go func() {
		if err := s.templateStore.IncrementUsage(context.Background(), templateID); err != nil {
			s.logger.WithError(err).Warn("Failed to increment template usage")
		}
	}()

	s.logger.WithFields(logrus.Fields{
		"template_id": templateID,
		"user_id":     userID,
	}).Info("Template executed")

	return sql, nil
}

// ValidateTemplate validates SQL template syntax and parameter references
func (s *Service) ValidateTemplate(ctx context.Context, sqlTemplate string, params []turso.TemplateParameter) error {
	// Extract parameter references from template
	paramRefs := extractParameterReferences(sqlTemplate)

	// Create a map of defined parameters
	definedParams := make(map[string]bool)
	for _, param := range params {
		definedParams[param.Name] = true
	}

	// Check all referenced parameters are defined
	for _, ref := range paramRefs {
		if !definedParams[ref] {
			return fmt.Errorf("undefined parameter: %s", ref)
		}
	}

	// Check for SQL injection patterns (basic validation)
	if containsSQLInjectionPatterns(sqlTemplate) {
		return fmt.Errorf("template contains potentially unsafe SQL patterns")
	}

	// Validate parameter types
	for _, param := range params {
		if !isValidParameterType(param.Type) {
			return fmt.Errorf("invalid parameter type: %s", param.Type)
		}
	}

	return nil
}

// CreateScheduleInput represents input for creating a schedule
type CreateScheduleInput struct {
	TemplateID        string                 `json:"template_id" validate:"required"`
	Name              string                 `json:"name" validate:"required,min=3,max=100"`
	Description       string                 `json:"description" validate:"max=500"`
	Frequency         string                 `json:"frequency" validate:"required"`
	Parameters        map[string]interface{} `json:"parameters"`
	OrganizationID    *string                `json:"organization_id"`
	NotificationEmail string                 `json:"notification_email" validate:"omitempty,email"`
	ResultStorage     string                 `json:"result_storage" validate:"oneof=none s3 database"`
}

// CreateSchedule creates a new query schedule
func (s *Service) CreateSchedule(ctx context.Context, input CreateScheduleInput, userID string) (*turso.QuerySchedule, error) {
	// Validate template exists and user has access
	template, err := s.GetTemplate(ctx, input.TemplateID, userID)
	if err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}

	// Validate cron expression
	schedule, err := s.cronParser.Parse(input.Frequency)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time
	nextRun := schedule.Next(time.Now())

	// Check organization permissions if schedule is org-scoped
	if input.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *input.OrganizationID, userID)
		if err != nil || member == nil {
			return nil, fmt.Errorf("user not member of organization")
		}

		// Require connections:create permission to create schedules
		if !organization.HasPermission(member.Role, organization.PermCreateConnections) {
			return nil, fmt.Errorf("insufficient permissions to create schedules")
		}
	}

	// Validate parameters against template
	if err := s.validateScheduleParameters(template, input.Parameters); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	querySchedule := &turso.QuerySchedule{
		TemplateID:        input.TemplateID,
		Name:              input.Name,
		Description:       input.Description,
		Frequency:         input.Frequency,
		Parameters:        input.Parameters,
		NextRunAt:         &nextRun,
		Status:            "active",
		CreatedBy:         userID,
		OrganizationID:    input.OrganizationID,
		NotificationEmail: input.NotificationEmail,
		ResultStorage:     input.ResultStorage,
	}

	if err := s.scheduleStore.Create(ctx, querySchedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	// Create audit log if in organization
	if input.OrganizationID != nil {
		s.createAuditLog(ctx, *input.OrganizationID, userID, "create_schedule", "schedule", &querySchedule.ID, map[string]interface{}{
			"schedule_name": querySchedule.Name,
			"frequency":     querySchedule.Frequency,
			"template_id":   querySchedule.TemplateID,
		})
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": querySchedule.ID,
		"name":        querySchedule.Name,
		"frequency":   querySchedule.Frequency,
	}).Info("Schedule created")

	return querySchedule, nil
}

// PauseSchedule pauses a schedule
func (s *Service) PauseSchedule(ctx context.Context, scheduleID, userID string) error {
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Check permissions
	if err := s.checkSchedulePermission(ctx, schedule, userID); err != nil {
		return err
	}

	if err := s.scheduleStore.UpdateStatus(ctx, scheduleID, "paused"); err != nil {
		return fmt.Errorf("failed to pause schedule: %w", err)
	}

	if schedule.OrganizationID != nil {
		s.createAuditLog(ctx, *schedule.OrganizationID, userID, "pause_schedule", "schedule", &scheduleID, map[string]interface{}{
			"schedule_name": schedule.Name,
		})
	}

	return nil
}

// ResumeSchedule resumes a paused schedule
func (s *Service) ResumeSchedule(ctx context.Context, scheduleID, userID string) error {
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Check permissions
	if err := s.checkSchedulePermission(ctx, schedule, userID); err != nil {
		return err
	}

	// Recalculate next run time
	cronSchedule, err := s.cronParser.Parse(schedule.Frequency)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	nextRun := cronSchedule.Next(time.Now())
	if err := s.scheduleStore.UpdateNextRun(ctx, scheduleID, time.Now(), nextRun); err != nil {
		return fmt.Errorf("failed to update next run: %w", err)
	}

	if err := s.scheduleStore.UpdateStatus(ctx, scheduleID, "active"); err != nil {
		return fmt.Errorf("failed to resume schedule: %w", err)
	}

	if schedule.OrganizationID != nil {
		s.createAuditLog(ctx, *schedule.OrganizationID, userID, "resume_schedule", "schedule", &scheduleID, map[string]interface{}{
			"schedule_name": schedule.Name,
		})
	}

	return nil
}

// ListSchedules retrieves schedules accessible to the user
func (s *Service) ListSchedules(ctx context.Context, userID string, filters turso.ScheduleFilters) ([]*turso.QuerySchedule, error) {
	// If organization filter is set, verify membership
	if filters.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *filters.OrganizationID, userID)
		if err != nil || member == nil {
			return nil, fmt.Errorf("user not member of organization")
		}

		// Check view permissions
		if !organization.HasPermission(member.Role, organization.PermViewConnections) {
			return nil, fmt.Errorf("insufficient permissions to view schedules")
		}
	}

	schedules, err := s.scheduleStore.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Filter schedules based on access
	var accessible []*turso.QuerySchedule
	for _, schedule := range schedules {
		if s.canAccessSchedule(ctx, schedule, userID) {
			accessible = append(accessible, schedule)
		}
	}

	return accessible, nil
}

// GetSchedule retrieves a schedule by ID with permission checks
func (s *Service) GetSchedule(ctx context.Context, scheduleID, userID string) (*turso.QuerySchedule, error) {
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}

	// Check access permissions
	if !s.canAccessSchedule(ctx, schedule, userID) {
		return nil, fmt.Errorf("access denied to schedule")
	}

	return schedule, nil
}

// UpdateSchedule updates a schedule with permission checks
func (s *Service) UpdateSchedule(ctx context.Context, schedule *turso.QuerySchedule, userID string) error {
	// Get existing schedule
	existing, err := s.scheduleStore.GetByID(ctx, schedule.ID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Check update permissions
	if err := s.checkSchedulePermission(ctx, existing, userID); err != nil {
		return err
	}

	// If frequency changed, validate cron expression
	if schedule.Frequency != existing.Frequency {
		_, err := s.cronParser.Parse(schedule.Frequency)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// Validate parameters if template didn't change
	if schedule.TemplateID == existing.TemplateID {
		template, err := s.templateStore.GetByID(ctx, schedule.TemplateID)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}

		if err := s.validateScheduleParameters(template, schedule.Parameters); err != nil {
			return fmt.Errorf("invalid parameters: %w", err)
		}
	}

	// Keep immutable fields
	schedule.CreatedBy = existing.CreatedBy
	schedule.CreatedAt = existing.CreatedAt
	schedule.OrganizationID = existing.OrganizationID

	if err := s.scheduleStore.Update(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Create audit log if in organization
	if schedule.OrganizationID != nil {
		s.createAuditLog(ctx, *schedule.OrganizationID, userID, "update_schedule", "schedule", &schedule.ID, map[string]interface{}{
			"schedule_name": schedule.Name,
		})
	}

	return nil
}

// DeleteSchedule deletes a schedule with permission checks
func (s *Service) DeleteSchedule(ctx context.Context, scheduleID, userID string) error {
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Check delete permissions
	if err := s.checkSchedulePermission(ctx, schedule, userID); err != nil {
		return err
	}

	if err := s.scheduleStore.Delete(ctx, scheduleID); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	// Create audit log if in organization
	if schedule.OrganizationID != nil {
		s.createAuditLog(ctx, *schedule.OrganizationID, userID, "delete_schedule", "schedule", &scheduleID, map[string]interface{}{
			"schedule_name": schedule.Name,
		})
	}

	s.logger.WithField("schedule_id", scheduleID).Info("Schedule deleted")
	return nil
}

// GetExecutionHistory retrieves execution history for a schedule
func (s *Service) GetExecutionHistory(ctx context.Context, scheduleID, userID string, limit int) ([]*turso.ScheduleExecution, error) {
	// Get schedule and check access
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	// Check access permissions
	if !s.canAccessSchedule(ctx, schedule, userID) {
		return nil, fmt.Errorf("access denied to schedule")
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	return s.scheduleStore.GetExecutionHistory(ctx, scheduleID, limit)
}

// GetExecutionStats retrieves execution statistics for a schedule
func (s *Service) GetExecutionStats(ctx context.Context, scheduleID, userID string) (map[string]interface{}, error) {
	// Get schedule and check access
	schedule, err := s.scheduleStore.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	// Check access permissions
	if !s.canAccessSchedule(ctx, schedule, userID) {
		return nil, fmt.Errorf("access denied to schedule")
	}

	return s.scheduleStore.GetExecutionStats(ctx, scheduleID)
}

// GetDueSchedules returns all active schedules that need to be executed
func (s *Service) GetDueSchedules(ctx context.Context) ([]*turso.QuerySchedule, error) {
	return s.scheduleStore.GetDueSchedules(ctx)
}

// CalculateNextRun calculates the next run time from a cron expression
func (s *Service) CalculateNextRun(cronExpr string) (time.Time, error) {
	schedule, err := s.cronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return schedule.Next(time.Now()), nil
}

// canAccessSchedule checks if user can access a schedule
func (s *Service) canAccessSchedule(ctx context.Context, schedule *turso.QuerySchedule, userID string) bool {
	// Owner always has access
	if schedule.CreatedBy == userID {
		return true
	}

	// Check organization access
	if schedule.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *schedule.OrganizationID, userID)
		if err == nil && member != nil {
			return true
		}
	}

	return false
}

// canAccessTemplate checks if user can access a template
func (s *Service) canAccessTemplate(ctx context.Context, template *turso.QueryTemplate, userID string) bool {
	// Owner always has access
	if template.CreatedBy == userID {
		return true
	}

	// Check organization access
	if template.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *template.OrganizationID, userID)
		if err == nil && member != nil {
			// Public templates accessible to all org members
			if template.IsPublic {
				return true
			}
			// Private templates only to creator
			return template.CreatedBy == userID
		}
	}

	return false
}

// checkSchedulePermission checks if user can modify a schedule
func (s *Service) checkSchedulePermission(ctx context.Context, schedule *turso.QuerySchedule, userID string) error {
	if schedule.OrganizationID != nil {
		member, err := s.orgRepo.GetMember(ctx, *schedule.OrganizationID, userID)
		if err != nil || member == nil {
			return fmt.Errorf("user not member of organization")
		}

		// Only creator or admin can modify
		if !organization.CanUpdateResource(member.Role, schedule.CreatedBy, userID) {
			return fmt.Errorf("insufficient permissions to modify this schedule")
		}
	} else if schedule.CreatedBy != userID {
		return fmt.Errorf("cannot modify another user's personal schedule")
	}

	return nil
}

// validateScheduleParameters validates parameters against template definition
func (s *Service) validateScheduleParameters(template *turso.QueryTemplate, params map[string]interface{}) error {
	// Check all required parameters are provided
	for _, param := range template.Parameters {
		if param.Required {
			if _, exists := params[param.Name]; !exists {
				return fmt.Errorf("missing required parameter: %s", param.Name)
			}
		}
	}

	return nil
}

// createAuditLog is a helper to create audit logs (non-blocking)
func (s *Service) createAuditLog(ctx context.Context, orgID, userID, action, resourceType string, resourceID *string, details map[string]interface{}) {
	go func() {
		log := &organization.AuditLog{
			OrganizationID: &orgID,
			UserID:         userID,
			Action:         action,
			ResourceType:   resourceType,
			ResourceID:     resourceID,
			Details:        details,
		}
		if err := s.orgRepo.CreateAuditLog(context.Background(), log); err != nil {
			s.logger.WithError(err).Warn("Failed to create audit log")
		}
	}()
}

// Helper functions for parameter substitution and validation

// extractParameterReferences extracts parameter names from {{param_name}} syntax
func extractParameterReferences(sqlTemplate string) []string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := re.FindAllStringSubmatch(sqlTemplate, -1)

	params := make([]string, 0, len(matches))
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			if !seen[paramName] {
				params = append(params, paramName)
				seen[paramName] = true
			}
		}
	}

	return params
}

// containsSQLInjectionPatterns checks for common SQL injection patterns
func containsSQLInjectionPatterns(sql string) bool {
	// Basic check for dangerous patterns in template itself
	dangerous := []string{
		"';",
		"--",
		"/*",
		"*/",
		"xp_",
		"sp_",
		"exec(",
		"execute(",
	}

	sqlLower := strings.ToLower(sql)
	for _, pattern := range dangerous {
		if strings.Contains(sqlLower, pattern) {
			return true
		}
	}

	return false
}

// isValidParameterType checks if parameter type is valid
func isValidParameterType(paramType string) bool {
	validTypes := []string{"string", "number", "date", "boolean"}
	for _, t := range validTypes {
		if paramType == t {
			return true
		}
	}
	return false
}
