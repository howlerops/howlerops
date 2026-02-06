package database

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// TestNormalizeValue_Nil tests nil value handling
func TestNormalizeValue_Nil(t *testing.T) {
	result := NormalizeValue(nil)
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

// TestNormalizeValue_SqlNullString tests sql.NullString normalization
func TestNormalizeValue_SqlNullString(t *testing.T) {
	testCases := []struct {
		name     string
		input    sql.NullString
		expected interface{}
	}{
		{"valid string", sql.NullString{String: "hello", Valid: true}, "hello"},
		{"empty valid string", sql.NullString{String: "", Valid: true}, ""},
		{"invalid string", sql.NullString{String: "ignored", Valid: false}, nil},
		{"null string", sql.NullString{Valid: false}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_SqlNullInt64 tests sql.NullInt64 normalization
func TestNormalizeValue_SqlNullInt64(t *testing.T) {
	testCases := []struct {
		name     string
		input    sql.NullInt64
		expected interface{}
	}{
		{"valid positive", sql.NullInt64{Int64: 42, Valid: true}, int64(42)},
		{"valid negative", sql.NullInt64{Int64: -100, Valid: true}, int64(-100)},
		{"valid zero", sql.NullInt64{Int64: 0, Valid: true}, int64(0)},
		{"max int64", sql.NullInt64{Int64: 9223372036854775807, Valid: true}, int64(9223372036854775807)},
		{"min int64", sql.NullInt64{Int64: -9223372036854775808, Valid: true}, int64(-9223372036854775808)},
		{"invalid", sql.NullInt64{Int64: 100, Valid: false}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_SqlNullInt32 tests sql.NullInt32 normalization
func TestNormalizeValue_SqlNullInt32(t *testing.T) {
	testCases := []struct {
		name     string
		input    sql.NullInt32
		expected interface{}
	}{
		{"valid positive", sql.NullInt32{Int32: 42, Valid: true}, int32(42)},
		{"valid negative", sql.NullInt32{Int32: -100, Valid: true}, int32(-100)},
		{"valid zero", sql.NullInt32{Int32: 0, Valid: true}, int32(0)},
		{"max int32", sql.NullInt32{Int32: 2147483647, Valid: true}, int32(2147483647)},
		{"min int32", sql.NullInt32{Int32: -2147483648, Valid: true}, int32(-2147483648)},
		{"invalid", sql.NullInt32{Int32: 100, Valid: false}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_SqlNullFloat64 tests sql.NullFloat64 normalization
func TestNormalizeValue_SqlNullFloat64(t *testing.T) {
	testCases := []struct {
		name     string
		input    sql.NullFloat64
		expected interface{}
	}{
		{"valid positive", sql.NullFloat64{Float64: 3.14159, Valid: true}, 3.14159},
		{"valid negative", sql.NullFloat64{Float64: -2.71828, Valid: true}, -2.71828},
		{"valid zero", sql.NullFloat64{Float64: 0.0, Valid: true}, 0.0},
		{"valid scientific", sql.NullFloat64{Float64: 1.23e10, Valid: true}, 1.23e10},
		{"invalid", sql.NullFloat64{Float64: 100.5, Valid: false}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_SqlNullBool tests sql.NullBool normalization
func TestNormalizeValue_SqlNullBool(t *testing.T) {
	testCases := []struct {
		name     string
		input    sql.NullBool
		expected interface{}
	}{
		{"valid true", sql.NullBool{Bool: true, Valid: true}, true},
		{"valid false", sql.NullBool{Bool: false, Valid: true}, false},
		{"invalid true", sql.NullBool{Bool: true, Valid: false}, nil},
		{"invalid false", sql.NullBool{Bool: false, Valid: false}, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_SqlNullTime tests sql.NullTime normalization
func TestNormalizeValue_SqlNullTime(t *testing.T) {
	fixedTime := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)

	testCases := []struct {
		name     string
		input    sql.NullTime
		expected interface{}
	}{
		{"valid time", sql.NullTime{Time: fixedTime, Valid: true}, fixedTime.Format(time.RFC3339)},
		{"invalid time", sql.NullTime{Time: fixedTime, Valid: false}, nil},
		{"valid zero time", sql.NullTime{Time: time.Time{}, Valid: true}, time.Time{}.Format(time.RFC3339)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_ByteArray tests []byte normalization
func TestNormalizeValue_ByteArray(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected interface{}
	}{
		{
			"valid JSON object",
			[]byte(`{"key":"value","number":42}`),
			map[string]interface{}{"key": "value", "number": float64(42)},
		},
		{
			"valid JSON array",
			[]byte(`[1,2,3]`),
			[]interface{}{float64(1), float64(2), float64(3)},
		},
		{
			"plain text",
			[]byte("hello world"),
			"hello world",
		},
		{
			"empty bytes",
			[]byte{},
			"",
		},
		{
			"binary data",
			[]byte{0x00, 0x01, 0x02, 0xFF},
			string([]byte{0x00, 0x01, 0x02, 0xFF}),
		},
		{
			"invalid JSON",
			[]byte(`{"incomplete":`),
			`{"incomplete":`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tc.expected, tc.expected, result, result)
			}
		})
	}
}

// TestNormalizeValue_Time tests time.Time normalization
func TestNormalizeValue_Time(t *testing.T) {
	testCases := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			"specific time",
			time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC),
			"2024-03-15T10:30:45Z",
		},
		{
			"zero time",
			time.Time{},
			"0001-01-01T00:00:00Z",
		},
		{
			"time with timezone",
			time.Date(2024, 12, 25, 15, 30, 0, 0, time.FixedZone("EST", -5*3600)),
			time.Date(2024, 12, 25, 15, 30, 0, 0, time.FixedZone("EST", -5*3600)).Format(time.RFC3339),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_Pointers tests pointer type normalization
func TestNormalizeValue_Pointers(t *testing.T) {
	str := "hello"
	i := 42
	i64 := int64(100)
	f64 := 3.14159
	b := true
	tm := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)

	testCases := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"nil string pointer", (*string)(nil), nil},
		{"valid string pointer", &str, "hello"},
		{"nil int pointer", (*int)(nil), nil},
		{"valid int pointer", &i, 42},
		{"nil int64 pointer", (*int64)(nil), nil},
		{"valid int64 pointer", &i64, int64(100)},
		{"nil float64 pointer", (*float64)(nil), nil},
		{"valid float64 pointer", &f64, 3.14159},
		{"nil bool pointer", (*bool)(nil), nil},
		{"valid bool pointer", &b, true},
		{"nil time pointer", (*time.Time)(nil), nil},
		{"valid time pointer", &tm, tm.Format(time.RFC3339)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_Arrays tests []interface{} normalization
func TestNormalizeValue_Arrays(t *testing.T) {
	testCases := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			"simple array",
			[]interface{}{1, "hello", true},
			[]interface{}{1, "hello", true},
		},
		{
			"empty array",
			[]interface{}{},
			[]interface{}{},
		},
		{
			"nested with null values",
			[]interface{}{sql.NullString{String: "test", Valid: true}, sql.NullInt64{Valid: false}},
			[]interface{}{"test", nil},
		},
		{
			"nested arrays",
			[]interface{}{
				[]interface{}{1, 2},
				[]interface{}{3, 4},
			},
			[]interface{}{
				[]interface{}{1, 2},
				[]interface{}{3, 4},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_Maps tests map[string]interface{} normalization
func TestNormalizeValue_Maps(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			"simple map",
			map[string]interface{}{"key": "value", "number": 42},
			map[string]interface{}{"key": "value", "number": 42},
		},
		{
			"empty map",
			map[string]interface{}{},
			map[string]interface{}{},
		},
		{
			"map with null values",
			map[string]interface{}{
				"valid":   sql.NullString{String: "test", Valid: true},
				"invalid": sql.NullInt64{Valid: false},
			},
			map[string]interface{}{
				"valid":   "test",
				"invalid": nil,
			},
		},
		{
			"nested maps",
			map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
			map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_DefaultPassthrough tests default case
func TestNormalizeValue_DefaultPassthrough(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"int", 42, 42},
		{"int64", int64(100), int64(100)},
		{"float64", 3.14159, 3.14159},
		{"string", "hello", "hello"},
		{"bool", true, true},
		{"empty string", "", ""},
		{"zero int", 0, 0},
		{"false bool", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_ComplexNesting tests deeply nested structures
func TestNormalizeValue_ComplexNesting(t *testing.T) {
	input := map[string]interface{}{
		"array": []interface{}{
			sql.NullString{String: "test", Valid: true},
			map[string]interface{}{
				"nested": sql.NullInt64{Int64: 42, Valid: true},
			},
		},
		"null":   sql.NullBool{Valid: false},
		"direct": "value",
	}

	expected := map[string]interface{}{
		"array": []interface{}{
			"test",
			map[string]interface{}{
				"nested": int64(42),
			},
		},
		"null":   nil,
		"direct": "value",
	}

	result := NormalizeValue(input)
	if !reflect.DeepEqual(result, expected) {
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedJSON, resultJSON)
	}
}

// TestNormalizeRow tests row normalization
func TestNormalizeRow(t *testing.T) {
	testCases := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			"simple row",
			[]interface{}{
				sql.NullString{String: "name", Valid: true},
				sql.NullInt64{Int64: 42, Valid: true},
				sql.NullBool{Bool: true, Valid: true},
			},
			[]interface{}{"name", int64(42), true},
		},
		{
			"row with nulls",
			[]interface{}{
				sql.NullString{String: "test", Valid: true},
				sql.NullInt64{Valid: false},
				sql.NullString{Valid: false},
			},
			[]interface{}{"test", nil, nil},
		},
		{
			"empty row",
			[]interface{}{},
			[]interface{}{},
		},
		{
			"mixed types",
			[]interface{}{
				"direct string",
				42,
				true,
				sql.NullFloat64{Float64: 3.14, Valid: true},
				time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC),
			},
			[]interface{}{
				"direct string",
				42,
				true,
				3.14,
				"2024-03-15T10:30:45Z",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeRow(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestApplyPagination tests pagination SQL modification
func TestApplyPagination(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		limit    int
		offset   int
		expected string
	}{
		{
			"basic limit",
			"SELECT * FROM users",
			10,
			0,
			"SELECT * FROM users LIMIT 10",
		},
		{
			"limit with offset",
			"SELECT * FROM users",
			10,
			20,
			"SELECT * FROM users LIMIT 10 OFFSET 20",
		},
		{
			"zero limit (no pagination)",
			"SELECT * FROM users",
			0,
			0,
			"SELECT * FROM users",
		},
		{
			"negative limit (no pagination)",
			"SELECT * FROM users",
			-1,
			0,
			"SELECT * FROM users",
		},
		{
			"limit 1",
			"SELECT * FROM users",
			1,
			0,
			"SELECT * FROM users LIMIT 1",
		},
		{
			"large limit",
			"SELECT * FROM users",
			1000,
			0,
			"SELECT * FROM users LIMIT 1000",
		},
		{
			"zero offset with limit",
			"SELECT * FROM users WHERE active = true",
			25,
			0,
			"SELECT * FROM users WHERE active = true LIMIT 25",
		},
		{
			"offset without changing limit",
			"SELECT * FROM users ORDER BY created_at",
			50,
			100,
			"SELECT * FROM users ORDER BY created_at LIMIT 50 OFFSET 100",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ApplyPagination(tc.query, tc.limit, tc.offset)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

// TestExtractTotalCount tests total count query generation
func TestExtractTotalCount(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		expected string
	}{
		{
			"simple select",
			"SELECT * FROM users",
			"SELECT COUNT(*) FROM (SELECT * FROM users) AS count_query",
		},
		{
			"select with where",
			"SELECT * FROM users WHERE active = true",
			"SELECT COUNT(*) FROM (SELECT * FROM users WHERE active = true) AS count_query",
		},
		{
			"select with joins",
			"SELECT u.*, p.name FROM users u JOIN profiles p ON u.id = p.user_id",
			"SELECT COUNT(*) FROM (SELECT u.*, p.name FROM users u JOIN profiles p ON u.id = p.user_id) AS count_query",
		},
		{
			"select with order by",
			"SELECT * FROM users ORDER BY created_at DESC",
			"SELECT COUNT(*) FROM (SELECT * FROM users ORDER BY created_at DESC) AS count_query",
		},
		{
			"complex query",
			"SELECT DISTINCT u.id, u.name FROM users u WHERE u.active = true ORDER BY u.name",
			"SELECT COUNT(*) FROM (SELECT DISTINCT u.id, u.name FROM users u WHERE u.active = true ORDER BY u.name) AS count_query",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractTotalCount(tc.query)
			if result != tc.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tc.expected, result)
			}
		})
	}
}

// TestNormalizeValue_JSONBytes tests JSON byte handling edge cases
func TestNormalizeValue_JSONBytes(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		validate func(t *testing.T, result interface{})
	}{
		{
			"nested JSON",
			[]byte(`{"user":{"name":"John","age":30}}`),
			func(t *testing.T, result interface{}) {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				user, ok := m["user"].(map[string]interface{})
				if !ok {
					t.Error("Expected nested user object")
					return
				}
				if user["name"] != "John" {
					t.Errorf("Expected name John, got %v", user["name"])
				}
				if user["age"] != float64(30) {
					t.Errorf("Expected age 30, got %v", user["age"])
				}
			},
		},
		{
			"JSON with null",
			[]byte(`{"key":null}`),
			func(t *testing.T, result interface{}) {
				m, ok := result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map, got %T", result)
					return
				}
				if m["key"] != nil {
					t.Errorf("Expected nil, got %v", m["key"])
				}
			},
		},
		{
			"JSON boolean",
			[]byte(`true`),
			func(t *testing.T, result interface{}) {
				if result != true {
					t.Errorf("Expected true, got %v", result)
				}
			},
		},
		{
			"JSON number",
			[]byte(`42.5`),
			func(t *testing.T, result interface{}) {
				if result != 42.5 {
					t.Errorf("Expected 42.5, got %v", result)
				}
			},
		},
		{
			"JSON string",
			[]byte(`"hello"`),
			func(t *testing.T, result interface{}) {
				if result != "hello" {
					t.Errorf("Expected hello, got %v", result)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeValue(tc.input)
			tc.validate(t, result)
		})
	}
}

// TestNormalizeValue_EdgeCases tests various edge cases
func TestNormalizeValue_EdgeCases(t *testing.T) {
	// Test that zero values are preserved
	t.Run("preserve zero values", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    interface{}
			expected interface{}
		}{
			{"zero int", 0, 0},
			{"zero float", 0.0, 0.0},
			{"false bool", false, false},
			{"empty string", "", ""},
			{"valid zero int64", sql.NullInt64{Int64: 0, Valid: true}, int64(0)},
			{"valid zero float", sql.NullFloat64{Float64: 0.0, Valid: true}, 0.0},
			{"valid false bool", sql.NullBool{Bool: false, Valid: true}, false},
			{"valid empty string", sql.NullString{String: "", Valid: true}, ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := NormalizeValue(tc.input)
				if result != tc.expected {
					t.Errorf("Expected %v, got %v", tc.expected, result)
				}
			})
		}
	})

	// Test large values
	t.Run("large values", func(t *testing.T) {
		largeString := string(make([]byte, 10000))
		result := NormalizeValue(largeString)
		if result != largeString {
			t.Error("Large string not preserved")
		}

		largeArray := make([]interface{}, 1000)
		for i := range largeArray {
			largeArray[i] = i
		}
		result = NormalizeValue(largeArray)
		resultArray, ok := result.([]interface{})
		if !ok || len(resultArray) != 1000 {
			t.Error("Large array not preserved")
		}
	})
}
