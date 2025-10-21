package testutil

import (
\t"testing"
)

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
\tt.Helper()
\tif err != nil {
\t\tt.Fatalf("unexpected error: %v", err)
\t}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
\tt.Helper()
\tif err == nil {
\t\tt.Fatal("expected error, got nil")
\t}
}

// AssertErrorContains fails if err is nil or doesn't contain substr
func AssertErrorContains(t *testing.T, err error, substr string) {
\tt.Helper()
\tif err == nil {
\t\tt.Fatal("expected error, got nil")
\t}
\tif !contains(err.Error(), substr) {
\t\tt.Fatalf("error %q does not contain %q", err.Error(), substr)
\t}
}

// AssertEqual fails if got != want (simple equality check)
func AssertEqual[T comparable](t *testing.T, got, want T) {
\tt.Helper()
\tif got != want {
\t\tt.Fatalf("got %v, want %v", got, want)
\t}
}

// AssertNotNil fails if value is nil
func AssertNotNil(t *testing.T, value interface{}) {
\tt.Helper()
\tif value == nil {
\t\tt.Fatal("expected non-nil value, got nil")
\t}
}

// AssertNil fails if value is not nil
func AssertNil(t *testing.T, value interface{}) {
\tt.Helper()
\tif value != nil {
\t\tt.Fatalf("expected nil, got %v", value)
\t}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
\treturn len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
\t\t(len(s) > 0 && len(substr) > 0 && s[:len(substr)] == substr) ||
\t\t(len(s) > len(substr) && contains(s[1:], substr)))
}
