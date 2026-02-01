// Stub package to satisfy import requirements during Wails binding generation on arm64
// This replaces github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 which has no
// buildable files on arm64 systems.

package duckdb_go_bindings_platform

// This file exists to satisfy the Go import analyzer.
// The actual darwin-amd64 CGO bindings are never used on arm64.
