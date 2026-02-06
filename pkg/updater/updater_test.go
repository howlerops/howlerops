package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewUpdater(t *testing.T) {
	configDir := "/tmp/test-updater"
	u := NewUpdater(configDir)

	if u.configDir != configDir {
		t.Errorf("Expected configDir %s, got %s", configDir, u.configDir)
	}

	if u.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	if u.githubAPIURL != GitHubAPIURL {
		t.Errorf("Expected GitHub API URL %s, got %s", GitHubAPIURL, u.githubAPIURL)
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		expect  bool
	}{
		{
			name:    "newer version",
			latest:  "2.0.0",
			current: "1.0.0",
			expect:  true,
		},
		{
			name:    "same version",
			latest:  "2.0.0",
			current: "2.0.0",
			expect:  false,
		},
		{
			name:    "older version",
			latest:  "1.0.0",
			current: "2.0.0",
			expect:  false,
		},
		{
			name:    "with v prefix",
			latest:  "v2.0.0",
			current: "v1.0.0",
			expect:  true,
		},
		{
			name:    "dev version",
			latest:  "2.0.0",
			current: "dev",
			expect:  false,
		},
	}

	u := NewUpdater("/tmp/test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := u.isNewerVersion(tt.latest, tt.current)
			if got != tt.expect {
				t.Errorf("Expected %v, got %v for latest=%s current=%s", tt.expect, got, tt.latest, tt.current)
			}
		})
	}
}

func TestGetBinaryName(t *testing.T) {
	u := NewUpdater("/tmp/test")

	binaryName := u.getBinaryName()

	// Should contain the base name
	if binaryName == "" {
		t.Error("Binary name should not be empty")
	}

	// Should match current platform
	expectedBase := "sql-studio-backend"
	if !contains(binaryName, expectedBase) {
		t.Errorf("Binary name should contain %s, got %s", expectedBase, binaryName)
	}

	// Check OS specific
	if runtime.GOOS == "windows" && !contains(binaryName, ".exe") {
		t.Error("Windows binary should have .exe extension")
	}

	if runtime.GOOS == "darwin" && !contains(binaryName, "darwin") {
		t.Error("macOS binary should contain 'darwin'")
	}

	if runtime.GOOS == "linux" && !contains(binaryName, "linux") {
		t.Error("Linux binary should contain 'linux'")
	}
}

func TestCheckForUpdate(t *testing.T) {
	// Create mock server
	mockRelease := Release{
		TagName: "v3.0.0",
		Name:    "Version 3.0.0",
		Body:    "New features and bug fixes",
		Assets: []Asset{
			{
				Name:        "sql-studio-backend-darwin-amd64",
				DownloadURL: "https://example.com/darwin-amd64",
			},
			{
				Name:        "sql-studio-backend-darwin-amd64.sha256",
				DownloadURL: "https://example.com/darwin-amd64.sha256",
			},
		},
	}

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockRelease) // Best-effort encode in test
	}))
	defer server.Close()

	u := NewUpdater("/tmp/test")
	u.githubAPIURL = server.URL

	ctx := context.Background()
	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		t.Fatalf("CheckForUpdate failed: %v", err)
	}

	if info.LatestVersion != "3.0.0" {
		t.Errorf("Expected latest version 3.0.0, got %s", info.LatestVersion)
	}

	if info.ReleaseNotes != mockRelease.Body {
		t.Errorf("Expected release notes %q, got %q", mockRelease.Body, info.ReleaseNotes)
	}
}

func TestShouldCheckForUpdate(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // Best-effort cleanup in test

	u := NewUpdater(tmpDir)

	// First check - should return true (no file exists)
	if !u.ShouldCheckForUpdate() {
		t.Error("Should check for update when no last check file exists")
	}

	// Create last check file
	if err := u.RecordUpdateCheck(); err != nil {
		t.Fatalf("Failed to record update check: %v", err)
	}

	// Should not check immediately after recording
	if u.ShouldCheckForUpdate() {
		t.Error("Should not check for update immediately after recording")
	}

	// Modify file timestamp to simulate old check
	lastCheckFile := filepath.Join(tmpDir, ".last_update_check")
	oldTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(lastCheckFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to modify file timestamp: %v", err)
	}

	// Should check after 24 hours
	if !u.ShouldCheckForUpdate() {
		t.Error("Should check for update after 24 hours")
	}
}

func TestRecordUpdateCheck(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // Best-effort cleanup in test

	u := NewUpdater(tmpDir)

	if err := u.RecordUpdateCheck(); err != nil {
		t.Fatalf("RecordUpdateCheck failed: %v", err)
	}

	lastCheckFile := filepath.Join(tmpDir, ".last_update_check")
	if _, err := os.Stat(lastCheckFile); os.IsNotExist(err) {
		t.Error("Last check file should exist after recording")
	}

	// Verify content
	// #nosec G304 - test file path constructed from temp directory, not user input
	content, err := os.ReadFile(lastCheckFile)
	if err != nil {
		t.Fatalf("Failed to read last check file: %v", err)
	}

	// Should be a valid RFC3339 timestamp
	_, err = time.Parse(time.RFC3339, string(content))
	if err != nil {
		t.Errorf("Last check file should contain valid RFC3339 timestamp, got: %s", string(content))
	}
}

func TestFetchLatestRelease_Error(t *testing.T) {
	// Create mock server that returns error
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error")) // Best-effort write in test
	}))
	defer server.Close()

	u := NewUpdater("/tmp/test")
	u.githubAPIURL = server.URL

	ctx := context.Background()
	_, err := u.fetchLatestRelease(ctx)
	if err == nil {
		t.Error("Expected error when GitHub API returns 500")
	}
}

func TestFetchLatestRelease_InvalidJSON(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json")) // Best-effort write in test
	}))
	defer server.Close()

	u := NewUpdater("/tmp/test")
	u.githubAPIURL = server.URL

	ctx := context.Background()
	_, err := u.fetchLatestRelease(ctx)
	if err == nil {
		t.Error("Expected error when GitHub API returns invalid JSON")
	}
}

func TestDownloadChecksum(t *testing.T) {
	// Create mock server
	expectedHash := "abc123def456789"
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(expectedHash + "  filename.bin\n")) // Best-effort write in test
	}))
	defer server.Close()

	u := NewUpdater("/tmp/test")

	ctx := context.Background()
	checksum, err := u.downloadChecksum(ctx, server.URL)
	if err != nil {
		t.Fatalf("downloadChecksum failed: %v", err)
	}

	if checksum != expectedHash {
		t.Errorf("Expected checksum %s, got %s", expectedHash, checksum)
	}
}

func TestCopyFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "updater-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // Best-effort cleanup in test

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	u := NewUpdater("/tmp/test")
	if err := u.copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	// #nosec G304 - test file path from temp directory, not user input
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(copiedContent))
	}

	// Verify permissions
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Expected mode %v, got %v", srcInfo.Mode(), dstInfo.Mode())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(fmt.Sprint(r), "failed to listen on a port") {
				t.Skipf("Skipping test: cannot bind test server: %v", r)
			}
			panic(r)
		}
	}()

	return httptest.NewServer(handler)
}
