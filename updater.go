package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// Current version - this will be updated by build process
	CurrentVersion = "0.0.2"

	// GitHub API endpoint for latest release
	GitHubReleasesAPI = "https://api.github.com/repos/jbeck018/howlerops/releases/latest"

	// Update check interval (24 hours)
	UpdateCheckInterval = 24 * time.Hour
)

// UpdateInfo represents information about an available update
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	DownloadURL    string `json:"downloadUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
	PublishedAt    string `json:"publishedAt"`
}

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// UpdateChecker handles checking for application updates
type UpdateChecker struct {
	ctx           interface{}
	lastCheckTime time.Time
	latestRelease *GitHubRelease
	httpClient    *http.Client
}

// NewUpdateChecker creates a new update checker instance
func NewUpdateChecker() *UpdateChecker {
	return &UpdateChecker{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckForUpdates checks if a new version is available
func (u *UpdateChecker) CheckForUpdates() (*UpdateInfo, error) {
	// Fetch latest release from GitHub
	release, err := u.fetchLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	u.latestRelease = release
	u.lastCheckTime = time.Now()

	// Parse versions
	currentVer := normalizeVersion(CurrentVersion)
	latestVer := normalizeVersion(release.TagName)

	// Check if update is available
	updateAvailable := compareVersions(latestVer, currentVer) > 0

	// Get platform-specific download URL
	downloadURL := u.getDownloadURL(release)

	return &UpdateInfo{
		Available:      updateAvailable,
		CurrentVersion: CurrentVersion,
		LatestVersion:  release.TagName,
		DownloadURL:    downloadURL,
		ReleaseNotes:   release.Body,
		PublishedAt:    release.PublishedAt,
	}, nil
}

// GetCurrentVersion returns the current application version
func (u *UpdateChecker) GetCurrentVersion() string {
	return CurrentVersion
}

// OpenDownloadPage opens the download page in the default browser
func (u *UpdateChecker) OpenDownloadPage() error {
	if u.latestRelease == nil {
		return fmt.Errorf("no release information available")
	}

	if u.ctx != nil {
		wailsRuntime.BrowserOpenURL(u.ctx.(context.Context), u.latestRelease.HTMLURL)
	}
	return nil
}

// fetchLatestRelease fetches the latest release from GitHub API
func (u *UpdateChecker) fetchLatestRelease() (*GitHubRelease, error) {
	req, err := http.NewRequest("GET", GitHubReleasesAPI, nil)
	if err != nil {
		return nil, err
	}

	// Add user agent to avoid rate limiting
	req.Header.Set("User-Agent", fmt.Sprintf("HowlerOps/%s", CurrentVersion))

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// getDownloadURL returns the appropriate download URL for the current platform
func (u *UpdateChecker) getDownloadURL(release *GitHubRelease) string {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Look for platform-specific asset
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)

		switch platform {
		case "darwin":
			// macOS: Look for universal binary or architecture-specific
			if strings.Contains(name, "darwin-universal") {
				return asset.BrowserDownloadURL
			}
			if strings.Contains(name, "darwin") && (strings.Contains(name, arch) || strings.Contains(name, "universal")) {
				return asset.BrowserDownloadURL
			}
		case "windows":
			if strings.Contains(name, "windows") && strings.Contains(name, arch) {
				return asset.BrowserDownloadURL
			}
		case "linux":
			if strings.Contains(name, "linux") && strings.Contains(name, arch) {
				return asset.BrowserDownloadURL
			}
		}
	}

	// Fallback to release page
	return release.HTMLURL
}

// normalizeVersion removes 'v' prefix and trims whitespace
func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

// compareVersions compares two semantic versions
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}

	return 0
}
