package schemadiff

import (
	"testing"

	"github.com/jbeck018/howlerops/pkg/database"
)

func TestNewComparator(t *testing.T) {
	comparator := NewComparator()
	if comparator == nil {
		t.Fatal("NewComparator() returned nil")
	}
}

func TestColumnsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        database.ColumnInfo
		b        database.ColumnInfo
		expected bool
	}{
		{
			name: "identical columns",
			a: database.ColumnInfo{
				Name:     "id",
				DataType: "bigint",
				Nullable: false,
			},
			b: database.ColumnInfo{
				Name:     "id",
				DataType: "bigint",
				Nullable: false,
			},
			expected: true,
		},
		{
			name: "different types",
			a: database.ColumnInfo{
				Name:     "id",
				DataType: "bigint",
				Nullable: false,
			},
			b: database.ColumnInfo{
				Name:     "id",
				DataType: "integer",
				Nullable: false,
			},
			expected: false,
		},
		{
			name: "different nullable",
			a: database.ColumnInfo{
				Name:     "email",
				DataType: "varchar",
				Nullable: true,
			},
			b: database.ColumnInfo{
				Name:     "email",
				DataType: "varchar",
				Nullable: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := columnsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("columnsEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIndexesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        database.IndexInfo
		b        database.IndexInfo
		expected bool
	}{
		{
			name: "identical indexes",
			a: database.IndexInfo{
				Name:    "idx_user_email",
				Columns: []string{"email"},
				Unique:  true,
				Method:  "btree",
			},
			b: database.IndexInfo{
				Name:    "idx_user_email",
				Columns: []string{"email"},
				Unique:  true,
				Method:  "btree",
			},
			expected: true,
		},
		{
			name: "different uniqueness",
			a: database.IndexInfo{
				Name:    "idx_user_email",
				Columns: []string{"email"},
				Unique:  true,
				Method:  "btree",
			},
			b: database.IndexInfo{
				Name:    "idx_user_email",
				Columns: []string{"email"},
				Unique:  false,
				Method:  "btree",
			},
			expected: false,
		},
		{
			name: "different columns",
			a: database.IndexInfo{
				Name:    "idx_composite",
				Columns: []string{"first_name", "last_name"},
				Unique:  false,
				Method:  "btree",
			},
			b: database.IndexInfo{
				Name:    "idx_composite",
				Columns: []string{"last_name", "first_name"},
				Unique:  false,
				Method:  "btree",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("indexesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestForeignKeysEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        database.ForeignKeyInfo
		b        database.ForeignKeyInfo
		expected bool
	}{
		{
			name: "identical foreign keys",
			a: database.ForeignKeyInfo{
				Name:              "fk_user_role",
				Columns:           []string{"role_id"},
				ReferencedTable:   "roles",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
				OnUpdate:          "CASCADE",
			},
			b: database.ForeignKeyInfo{
				Name:              "fk_user_role",
				Columns:           []string{"role_id"},
				ReferencedTable:   "roles",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
				OnUpdate:          "CASCADE",
			},
			expected: true,
		},
		{
			name: "different on delete",
			a: database.ForeignKeyInfo{
				Name:              "fk_user_role",
				Columns:           []string{"role_id"},
				ReferencedTable:   "roles",
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
				OnUpdate:          "CASCADE",
			},
			b: database.ForeignKeyInfo{
				Name:              "fk_user_role",
				Columns:           []string{"role_id"},
				ReferencedTable:   "roles",
				ReferencedColumns: []string{"id"},
				OnDelete:          "SET NULL",
				OnUpdate:          "CASCADE",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := foreignKeysEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("foreignKeysEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBuildTableMap(t *testing.T) {
	snapshot := &SchemaSnapshot{
		Structures: map[string]*database.TableStructure{
			"public.users": {
				Table: database.TableInfo{
					Schema: "public",
					Name:   "users",
				},
			},
			"public.roles": {
				Table: database.TableInfo{
					Schema: "public",
					Name:   "roles",
				},
			},
		},
	}

	tableMap := buildTableMap(snapshot)

	if len(tableMap) != 2 {
		t.Errorf("Expected 2 tables in map, got %d", len(tableMap))
	}

	if _, exists := tableMap["public.users"]; !exists {
		t.Error("Expected public.users in table map")
	}

	if _, exists := tableMap["public.roles"]; !exists {
		t.Error("Expected public.roles in table map")
	}
}
