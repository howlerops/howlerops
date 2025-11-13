package pii

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles persistence of PII field metadata
type Store interface {
	CreatePIIField(ctx context.Context, field *PIIField) error
	GetPIIField(ctx context.Context, tableName, fieldName string) (*PIIField, error)
	ListPIIFields(ctx context.Context) ([]*PIIField, error)
	ListTablePIIFields(ctx context.Context, tableName string) ([]*PIIField, error)
	UpdatePIIField(ctx context.Context, field *PIIField) error
	DeletePIIField(ctx context.Context, id string) error
	VerifyPIIField(ctx context.Context, id string) error
	GetPIIFieldsByType(ctx context.Context, piiType string) ([]*PIIField, error)
}

type store struct {
	db *sql.DB
}

// NewStore creates a new PII store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreatePIIField(ctx context.Context, field *PIIField) error {
	if field.ID == "" {
		field.ID = uuid.New().String()
	}
	now := time.Now()
	if field.CreatedAt.IsZero() {
		field.CreatedAt = now
	}
	if field.UpdatedAt.IsZero() {
		field.UpdatedAt = now
	}

	query := `
		INSERT INTO pii_fields (
			id, table_name, field_name, pii_type, detection_method,
			confidence_score, verified, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(table_name, field_name) DO UPDATE SET
			pii_type = excluded.pii_type,
			detection_method = excluded.detection_method,
			confidence_score = excluded.confidence_score,
			updated_at = excluded.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		field.ID,
		field.TableName,
		field.FieldName,
		field.PIIType,
		field.DetectionMethod,
		field.ConfidenceScore,
		field.Verified,
		field.CreatedAt.Unix(),
		field.UpdatedAt.Unix(),
	)

	return err
}

func (s *store) GetPIIField(ctx context.Context, tableName, fieldName string) (*PIIField, error) {
	query := `
		SELECT id, table_name, field_name, pii_type, detection_method,
			confidence_score, verified, created_at, updated_at
		FROM pii_fields
		WHERE table_name = ? AND field_name = ?
	`

	var field PIIField
	var createdAtUnix, updatedAtUnix int64
	var confidenceScore sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, tableName, fieldName).Scan(
		&field.ID,
		&field.TableName,
		&field.FieldName,
		&field.PIIType,
		&field.DetectionMethod,
		&confidenceScore,
		&field.Verified,
		&createdAtUnix,
		&updatedAtUnix,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("PII field not found")
		}
		return nil, err
	}

	field.CreatedAt = time.Unix(createdAtUnix, 0)
	field.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if confidenceScore.Valid {
		field.ConfidenceScore = confidenceScore.Float64
	}

	return &field, nil
}

func (s *store) ListPIIFields(ctx context.Context) ([]*PIIField, error) {
	query := `
		SELECT id, table_name, field_name, pii_type, detection_method,
			confidence_score, verified, created_at, updated_at
		FROM pii_fields
		ORDER BY table_name, field_name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	return s.scanPIIFields(rows)
}

func (s *store) ListTablePIIFields(ctx context.Context, tableName string) ([]*PIIField, error) {
	query := `
		SELECT id, table_name, field_name, pii_type, detection_method,
			confidence_score, verified, created_at, updated_at
		FROM pii_fields
		WHERE table_name = ?
		ORDER BY field_name
	`

	rows, err := s.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	return s.scanPIIFields(rows)
}

func (s *store) GetPIIFieldsByType(ctx context.Context, piiType string) ([]*PIIField, error) {
	query := `
		SELECT id, table_name, field_name, pii_type, detection_method,
			confidence_score, verified, created_at, updated_at
		FROM pii_fields
		WHERE pii_type = ?
		ORDER BY table_name, field_name
	`

	rows, err := s.db.QueryContext(ctx, query, piiType)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	return s.scanPIIFields(rows)
}

func (s *store) scanPIIFields(rows *sql.Rows) ([]*PIIField, error) {
	var fields []*PIIField

	for rows.Next() {
		var field PIIField
		var createdAtUnix, updatedAtUnix int64
		var confidenceScore sql.NullFloat64

		err := rows.Scan(
			&field.ID,
			&field.TableName,
			&field.FieldName,
			&field.PIIType,
			&field.DetectionMethod,
			&confidenceScore,
			&field.Verified,
			&createdAtUnix,
			&updatedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		field.CreatedAt = time.Unix(createdAtUnix, 0)
		field.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if confidenceScore.Valid {
			field.ConfidenceScore = confidenceScore.Float64
		}

		fields = append(fields, &field)
	}

	return fields, rows.Err()
}

func (s *store) UpdatePIIField(ctx context.Context, field *PIIField) error {
	field.UpdatedAt = time.Now()

	query := `
		UPDATE pii_fields
		SET pii_type = ?, detection_method = ?, confidence_score = ?,
		    verified = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		field.PIIType,
		field.DetectionMethod,
		field.ConfidenceScore,
		field.Verified,
		field.UpdatedAt.Unix(),
		field.ID,
	)

	return err
}

func (s *store) DeletePIIField(ctx context.Context, id string) error {
	query := `DELETE FROM pii_fields WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *store) VerifyPIIField(ctx context.Context, id string) error {
	query := `UPDATE pii_fields SET verified = true, updated_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, time.Now().Unix(), id)
	return err
}
