package audit

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

// DetailedAuditLogger handles field-level audit logging
type DetailedAuditLogger struct {
	store  Store
	logger *logrus.Logger
}

// NewDetailedAuditLogger creates a new detailed audit logger
func NewDetailedAuditLogger(store Store, logger *logrus.Logger) *DetailedAuditLogger {
	return &DetailedAuditLogger{
		store:  store,
		logger: logger,
	}
}

// LogUpdate logs field-level changes for an audit event
func (l *DetailedAuditLogger) LogUpdate(ctx context.Context, auditLogID string, changes []AuditChange) error {
	if len(changes) == 0 {
		return nil
	}

	logs := make([]*AuditLogDetailed, 0, len(changes))
	for _, change := range changes {
		logs = append(logs, &AuditLogDetailed{
			AuditLogID: auditLogID,
			TableName:  change.TableName,
			RecordID:   change.RecordID,
			FieldName:  change.FieldName,
			OldValue:   formatValue(change.OldValue),
			NewValue:   formatValue(change.NewValue),
			FieldType:  change.FieldType,
		})
	}

	err := l.store.CreateDetailedLogs(ctx, logs)
	if err != nil {
		l.logger.WithError(err).Error("Failed to create detailed audit logs")
		return err
	}

	l.logger.WithFields(logrus.Fields{
		"audit_log_id": auditLogID,
		"changes":      len(changes),
	}).Info("Logged detailed changes")

	return nil
}

// DetectChanges compares two records and returns field-level changes
func (l *DetailedAuditLogger) DetectChanges(tableName, recordID string, oldRecord, newRecord interface{}) []AuditChange {
	changes := make([]AuditChange, 0)

	// Handle map comparisons
	if oldMap, ok := oldRecord.(map[string]interface{}); ok {
		if newMap, ok := newRecord.(map[string]interface{}); ok {
			changes = l.compareMap(tableName, recordID, oldMap, newMap)
			return changes
		}
	}

	// Handle struct comparisons using reflection
	oldVal := reflect.ValueOf(oldRecord)
	newVal := reflect.ValueOf(newRecord)

	if oldVal.Kind() == reflect.Ptr {
		oldVal = oldVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	if oldVal.Kind() != reflect.Struct || newVal.Kind() != reflect.Struct {
		l.logger.Warn("Cannot compare non-struct values")
		return changes
	}

	oldType := oldVal.Type()
	for i := 0; i < oldVal.NumField(); i++ {
		field := oldType.Field(i)
		fieldName := field.Name

		// Get JSON tag if available
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		oldFieldVal := oldVal.Field(i).Interface()
		newFieldVal := newVal.Field(i).Interface()

		if !reflect.DeepEqual(oldFieldVal, newFieldVal) {
			fieldType := l.classifyField(fieldName, oldFieldVal, newFieldVal)
			changes = append(changes, AuditChange{
				TableName: tableName,
				RecordID:  recordID,
				FieldName: fieldName,
				OldValue:  oldFieldVal,
				NewValue:  newFieldVal,
				FieldType: fieldType,
			})
		}
	}

	return changes
}

// compareMap compares two maps and returns changes
func (l *DetailedAuditLogger) compareMap(tableName, recordID string, oldMap, newMap map[string]interface{}) []AuditChange {
	changes := make([]AuditChange, 0)

	// Check for changed and deleted fields
	for key, oldVal := range oldMap {
		newVal, exists := newMap[key]
		if !exists {
			// Field was deleted
			fieldType := l.classifyField(key, oldVal, nil)
			changes = append(changes, AuditChange{
				TableName: tableName,
				RecordID:  recordID,
				FieldName: key,
				OldValue:  oldVal,
				NewValue:  nil,
				FieldType: fieldType,
			})
		} else if !reflect.DeepEqual(oldVal, newVal) {
			// Field was changed
			fieldType := l.classifyField(key, oldVal, newVal)
			changes = append(changes, AuditChange{
				TableName: tableName,
				RecordID:  recordID,
				FieldName: key,
				OldValue:  oldVal,
				NewValue:  newVal,
				FieldType: fieldType,
			})
		}
	}

	// Check for new fields
	for key, newVal := range newMap {
		if _, exists := oldMap[key]; !exists {
			fieldType := l.classifyField(key, nil, newVal)
			changes = append(changes, AuditChange{
				TableName: tableName,
				RecordID:  recordID,
				FieldName: key,
				OldValue:  nil,
				NewValue:  newVal,
				FieldType: fieldType,
			})
		}
	}

	return changes
}

// classifyField determines if a field contains PII or sensitive data
func (l *DetailedAuditLogger) classifyField(fieldName string, oldVal, newVal interface{}) string {
	lowerField := strings.ToLower(fieldName)

	// PII field patterns
	piiPatterns := []string{
		"email", "phone", "mobile", "ssn", "social_security",
		"credit_card", "card_number", "passport", "driver_license",
		"tax_id", "national_id", "address", "street", "zip", "postal",
	}

	for _, pattern := range piiPatterns {
		if strings.Contains(lowerField, pattern) {
			return "pii"
		}
	}

	// Sensitive field patterns
	sensitivePatterns := []string{
		"password", "secret", "token", "key", "api_key",
		"access_token", "refresh_token", "private", "credential",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerField, pattern) {
			return "sensitive"
		}
	}

	// Check actual values for PII patterns
	if l.containsPII(oldVal) || l.containsPII(newVal) {
		return "pii"
	}

	return "normal"
}

// containsPII checks if a value contains PII data
func (l *DetailedAuditLogger) containsPII(value interface{}) bool {
	if value == nil {
		return false
	}

	str := fmt.Sprintf("%v", value)

	// Simple email pattern
	if strings.Contains(str, "@") && strings.Contains(str, ".") {
		return true
	}

	// Simple phone pattern (digits with possible separators)
	digitsOnly := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, str)

	if len(digitsOnly) >= 10 && len(digitsOnly) <= 15 {
		return true
	}

	return false
}

// formatValue converts a value to a string for storage
func formatValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// GetChangeHistory retrieves the complete change history for a record
func (l *DetailedAuditLogger) GetChangeHistory(ctx context.Context, tableName, recordID string) (*ChangeHistory, error) {
	return l.store.GetChangeHistory(ctx, tableName, recordID)
}

// GetFieldHistory retrieves the change history for a specific field
func (l *DetailedAuditLogger) GetFieldHistory(ctx context.Context, tableName, recordID, fieldName string) ([]FieldChange, error) {
	return l.store.GetFieldHistory(ctx, tableName, recordID, fieldName)
}
