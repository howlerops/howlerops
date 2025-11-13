package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sql-studio/backend-go/pkg/version"
)

const (
	// GitHubAPIURL is the GitHub API endpoint for releases
	GitHubAPIURL = "https://api.github.com/repos/sql-studio/sql-studio/releases/latest"

	// UpdateCheckInterval is how often to check for updates (24 hours)
	UpdateCheckInterval = 24 * time.Hour

	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second
)

// Release represents a GitHub release
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	CreatedAt  string  `json:"created_at"`
	Assets     []Asset `json:"assets"`
}

// Asset represents a release asset (downloadable file)
type Asset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	DownloadURL    string
	ChecksumURL    string
	Available      bool
}

// Updater handles version checking and updates
type Updater struct {
	githubAPIURL string
	configDir    string
	httpClient   *http.Client
}

// NewUpdater creates a new updater instance
func NewUpdater(configDir string) *Updater {
	return &Updater{
		githubAPIURL: GitHubAPIURL,
		configDir:    configDir,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// CheckForUpdate checks if a newer version is available
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: version.Version,
		LatestVersion:  strings.TrimPrefix(release.TagName, "v"),
		ReleaseNotes:   release.Body,
		Available:      false,
	}

	// Compare versions
	if u.isNewerVersion(info.LatestVersion, info.CurrentVersion) {
		info.Available = true

		// Find appropriate binary for current platform
		binaryName := u.getBinaryName()
		checksumName := fmt.Sprintf("%s.sha256", binaryName)

		for _, asset := range release.Assets {
			if asset.Name == binaryName {
				info.DownloadURL = asset.DownloadURL
			}
			if asset.Name == checksumName {
				info.ChecksumURL = asset.DownloadURL
			}
		}

		if info.DownloadURL == "" {
			return nil, fmt.Errorf("no binary found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
		}
	}

	return info, nil
}

// ShouldCheckForUpdate checks if it's time to check for updates
func (u *Updater) ShouldCheckForUpdate() bool {
	lastCheckFile := filepath.Join(u.configDir, ".last_update_check")

	info, err := os.Stat(lastCheckFile)
	if err != nil {
		// File doesn't exist, should check
		return true
	}

	// Check if 24 hours have passed
	return time.Since(info.ModTime()) >= UpdateCheckInterval
}

// RecordUpdateCheck records that an update check was performed
func (u *Updater) RecordUpdateCheck() error {
	if err := os.MkdirAll(u.configDir, 0600); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	lastCheckFile := filepath.Join(u.configDir, ".last_update_check")
	// #nosec G304 - path is constructed from u.configDir, not user input
	f, err := os.Create(lastCheckFile)
	if err != nil {
		return fmt.Errorf("failed to create last check file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	_, err = f.WriteString(time.Now().Format(time.RFC3339))
	return err
}

// DownloadUpdate downloads and installs the update
func (u *Updater) DownloadUpdate(ctx context.Context, info *UpdateInfo) error {
	if !info.Available {
		return fmt.Errorf("no update available")
	}

	// Download binary
	binaryData, err := u.downloadFile(ctx, info.DownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}

	// Download and verify checksum if available
	if info.ChecksumURL != "" {
		expectedChecksum, err := u.downloadChecksum(ctx, info.ChecksumURL)
		if err != nil {
			return fmt.Errorf("failed to download checksum: %w", err)
		}

		actualChecksum := sha256.Sum256(binaryData)
		actualChecksumStr := hex.EncodeToString(actualChecksum[:])

		if actualChecksumStr != expectedChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksumStr)
		}
	}

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Resolve symlinks
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Create backup
	backupPath := currentExe + ".backup"
	if err := u.copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write new binary
	tmpPath := currentExe + ".tmp"
	if err := os.WriteFile(tmpPath, binaryData, 0600); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// Replace current binary
	if err := os.Rename(tmpPath, currentExe); err != nil {
		// Restore backup on failure
		_ = os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove backup on success
	_ = os.Remove(backupPath)

	return nil
}

// fetchLatestRelease fetches the latest release from GitHub
func (u *Updater) fetchLatestRelease(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u.githubAPIURL, nil)
	if err != nil {
		return nil, err
	}

	// GitHub API requires User-Agent
	req.Header.Set("User-Agent", fmt.Sprintf("sql-studio/%s", version.Version))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// downloadFile downloads a file from the given URL
func (u *Updater) downloadFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// downloadChecksum downloads and parses the checksum file
func (u *Updater) downloadChecksum(ctx context.Context, url string) (string, error) {
	data, err := u.downloadFile(ctx, url)
	if err != nil {
		return "", err
	}

	// Checksum file format: "<hash>  <filename>"
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid checksum file format")
	}

	return parts[0], nil
}

// isNewerVersion compares version strings (simple semantic version comparison)
func (u *Updater) isNewerVersion(latest, current string) bool {
	// Remove 'v' prefix if present
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")

	// Skip comparison for dev builds
	if current == "dev" {
		return false
	}

	// Simple string comparison (works for semantic versions)
	return latest > current
}

// getBinaryName returns the appropriate binary name for the current platform
func (u *Updater) getBinaryName() string {
	base := "sql-studio-backend"

	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return fmt.Sprintf("%s-darwin-amd64", base)
		case "arm64":
			return fmt.Sprintf("%s-darwin-arm64", base)
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return fmt.Sprintf("%s-linux-amd64", base)
		case "arm64":
			return fmt.Sprintf("%s-linux-arm64", base)
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return fmt.Sprintf("%s-windows-amd64.exe", base)
		}
	}

	return fmt.Sprintf("%s-%s-%s", base, runtime.GOOS, runtime.GOARCH)
}

// copyFile copies a file from src to dst
func (u *Updater) copyFile(src, dst string) error {
	// #nosec G304 - src is the current executable path, not user input
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.Printf("Failed to close source file: %v", err)
		}
	}()

	// #nosec G304 - dst is validated destination path for backup, not user input
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.Printf("Failed to close destination file: %v", err)
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
