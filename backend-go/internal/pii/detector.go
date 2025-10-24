package pii

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// Detector detects PII in data
type Detector struct {
	store    Store
	patterns map[string]*PIIPattern
	logger   *logrus.Logger
}

// NewDetector creates a new PII detector
func NewDetector(store Store, logger *logrus.Logger) *Detector {
	d := &Detector{
		store:    store,
		patterns: make(map[string]*PIIPattern),
		logger:   logger,
	}

	// Initialize default patterns
	d.initializePatterns()

	return d
}

// initializePatterns sets up common PII detection patterns
func (d *Detector) initializePatterns() {
	d.patterns["email"] = &PIIPattern{
		Type:        "email",
		Pattern:     `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
		Description: "Email address",
		Confidence:  0.95,
	}

	d.patterns["phone"] = &PIIPattern{
		Type:        "phone",
		Pattern:     `^\+?1?\d{9,15}$`,
		Description: "Phone number",
		Confidence:  0.85,
	}

	d.patterns["ssn"] = &PIIPattern{
		Type:        "ssn",
		Pattern:     `^\d{3}-\d{2}-\d{4}$`,
		Description: "Social Security Number",
		Confidence:  0.99,
	}

	d.patterns["credit_card"] = &PIIPattern{
		Type:        "credit_card",
		Pattern:     `^\d{4}[\s\-]?\d{4}[\s\-]?\d{4}[\s\-]?\d{4}$`,
		Description: "Credit card number",
		Confidence:  0.90,
	}

	d.patterns["zipcode"] = &PIIPattern{
		Type:        "address",
		Pattern:     `^\d{5}(-\d{4})?$`,
		Description: "US ZIP code",
		Confidence:  0.70,
	}
}

// ScanQueryResults scans query results for PII
func (d *Detector) ScanQueryResults(ctx context.Context, results []map[string]interface{}) (*PIIScanResult, error) {
	scanResult := &PIIScanResult{
		Matches: make([]PIIMatch, 0),
	}

	if len(results) == 0 {
		return scanResult, nil
	}

	// Get field names from first row
	var fieldNames []string
	for field := range results[0] {
		fieldNames = append(fieldNames, field)
		scanResult.TotalFields++
	}

	// Track unique PII fields found
	piiFieldsFound := make(map[string]bool)

	// Scan each row
	for _, row := range results {
		for field, value := range row {
			if value == nil {
				continue
			}

			// Detect PII in this field
			piiType, confidence := d.detectPIIInValue(field, value)
			if piiType != "" {
				piiFieldsFound[field] = true

				match := PIIMatch{
					Field:           field,
					Type:            piiType,
					Value:           value,
					ConfidenceScore: confidence,
					Masked:          false,
				}

				// Generate masked value
				match.MaskedValue = d.maskValue(piiType, value)

				scanResult.Matches = append(scanResult.Matches, match)
			}
		}
	}

	scanResult.PIIFieldsFound = len(piiFieldsFound)

	return scanResult, nil
}

// detectPIIInValue detects PII type in a value
func (d *Detector) detectPIIInValue(fieldName string, value interface{}) (string, float64) {
	// Check field name patterns first
	piiType, confidence := d.classifyByFieldName(fieldName)
	if piiType != "" && confidence > 0.7 {
		return piiType, confidence
	}

	// Check value patterns
	strValue := fmt.Sprintf("%v", value)
	piiType, confidence = d.classifyByValue(strValue)
	if piiType != "" {
		return piiType, confidence
	}

	return "", 0.0
}

// classifyByFieldName classifies PII based on field name
func (d *Detector) classifyByFieldName(fieldName string) (string, float64) {
	lowerField := strings.ToLower(fieldName)

	// Email patterns
	if strings.Contains(lowerField, "email") || strings.Contains(lowerField, "e_mail") {
		return "email", 0.90
	}

	// Phone patterns
	phonePatterns := []string{"phone", "mobile", "cell", "tel", "telephone"}
	for _, pattern := range phonePatterns {
		if strings.Contains(lowerField, pattern) {
			return "phone", 0.85
		}
	}

	// SSN patterns
	ssnPatterns := []string{"ssn", "social_security", "social_sec"}
	for _, pattern := range ssnPatterns {
		if strings.Contains(lowerField, pattern) {
			return "ssn", 0.95
		}
	}

	// Credit card patterns
	cardPatterns := []string{"credit_card", "card_number", "cc_number", "creditcard"}
	for _, pattern := range cardPatterns {
		if strings.Contains(lowerField, pattern) {
			return "credit_card", 0.90
		}
	}

	// Address patterns
	addressPatterns := []string{"address", "street", "city", "zip", "postal", "location"}
	for _, pattern := range addressPatterns {
		if strings.Contains(lowerField, pattern) {
			return "address", 0.75
		}
	}

	// Name patterns
	namePatterns := []string{"name", "first_name", "last_name", "full_name", "firstname", "lastname"}
	for _, pattern := range namePatterns {
		if strings.Contains(lowerField, pattern) {
			return "name", 0.80
		}
	}

	return "", 0.0
}

// classifyByValue classifies PII based on value patterns
func (d *Detector) classifyByValue(value string) (string, float64) {
	value = strings.TrimSpace(value)

	// Check each pattern
	for piiType, pattern := range d.patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			d.logger.WithError(err).WithField("pattern", piiType).Error("Invalid regex pattern")
			continue
		}

		// For phone numbers, clean the value first
		if piiType == "phone" {
			cleaned := strings.Map(func(r rune) rune {
				if (r >= '0' && r <= '9') || r == '+' {
					return r
				}
				return -1
			}, value)
			if regex.MatchString(cleaned) {
				return piiType, pattern.Confidence
			}
		} else {
			if regex.MatchString(value) {
				// Additional validation for credit cards
				if piiType == "credit_card" {
					if d.luhnCheck(value) {
						return piiType, pattern.Confidence
					}
				} else {
					return piiType, pattern.Confidence
				}
			}
		}
	}

	return "", 0.0
}

// maskValue masks a PII value for display
func (d *Detector) maskValue(piiType string, value interface{}) string {
	strValue := fmt.Sprintf("%v", value)

	switch piiType {
	case "email":
		parts := strings.Split(strValue, "@")
		if len(parts) == 2 {
			username := parts[0]
			if len(username) > 2 {
				return username[:2] + "***@" + parts[1]
			}
			return "***@" + parts[1]
		}
		return "***"

	case "phone":
		// Keep last 4 digits
		cleaned := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, strValue)
		if len(cleaned) >= 4 {
			return "***-***-" + cleaned[len(cleaned)-4:]
		}
		return "***-***-****"

	case "ssn":
		if len(strValue) >= 4 {
			return "***-**-" + strValue[len(strValue)-4:]
		}
		return "***-**-****"

	case "credit_card":
		cleaned := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, strValue)
		if len(cleaned) >= 4 {
			return "****-****-****-" + cleaned[len(cleaned)-4:]
		}
		return "****-****-****-****"

	case "address":
		// Mask everything except last few characters
		if len(strValue) > 10 {
			return "*** " + strValue[len(strValue)-10:]
		}
		return "***"

	case "name":
		parts := strings.Fields(strValue)
		if len(parts) > 1 {
			return parts[0][:1] + "*** " + parts[len(parts)-1][:1] + "***"
		}
		if len(strValue) > 1 {
			return strValue[:1] + "***"
		}
		return "***"

	default:
		return "***"
	}
}

// luhnCheck validates credit card numbers using Luhn algorithm
func (d *Detector) luhnCheck(cardNumber string) bool {
	// Remove non-digits
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cardNumber)

	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}

	sum := 0
	alternate := false

	// Process digits from right to left
	for i := len(cleaned) - 1; i >= 0; i-- {
		digit := int(cleaned[i] - '0')

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// RegisterPIIField manually registers a field as containing PII
func (d *Detector) RegisterPIIField(ctx context.Context, tableName, fieldName, piiType string) error {
	field := &PIIField{
		TableName:       tableName,
		FieldName:       fieldName,
		PIIType:         piiType,
		DetectionMethod: "manual",
		ConfidenceScore: 1.0,
		Verified:        true,
	}

	return d.store.CreatePIIField(ctx, field)
}

// GetRegisteredPIIFields retrieves all registered PII fields
func (d *Detector) GetRegisteredPIIFields(ctx context.Context) ([]*PIIField, error) {
	return d.store.ListPIIFields(ctx)
}

// GetTablePIIFields retrieves PII fields for a specific table
func (d *Detector) GetTablePIIFields(ctx context.Context, tableName string) ([]*PIIField, error) {
	return d.store.ListTablePIIFields(ctx, tableName)
}
