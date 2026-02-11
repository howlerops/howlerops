package main

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// UpdateService is a Wails-compatible service that wraps UpdateChecker
type UpdateService struct {
	deps    *SharedDeps
	checker *UpdateChecker
}

// NewUpdateService creates a new UpdateService instance
func NewUpdateService(deps *SharedDeps) *UpdateService {
	checker := NewUpdateChecker()

	// Set the application reference if available
	if deps != nil && deps.App != nil {
		checker.SetApp(deps.App)
	}

	return &UpdateService{
		deps:    deps,
		checker: checker,
	}
}

// CheckForUpdates checks if a new version is available
func (s *UpdateService) CheckForUpdates() (*UpdateInfo, error) {
	if s.checker == nil {
		return nil, fmt.Errorf("update checker not initialized")
	}
	return s.checker.CheckForUpdates()
}

// GetCurrentVersion returns the current application version
func (s *UpdateService) GetCurrentVersion() string {
	if s.checker == nil {
		return ""
	}
	return s.checker.GetCurrentVersion()
}

// OpenDownloadPage opens the download page in the default browser
func (s *UpdateService) OpenDownloadPage() error {
	if s.checker == nil {
		return fmt.Errorf("update checker not initialized")
	}
	return s.checker.OpenDownloadPage()
}

// SetApp updates the application reference for the update checker
func (s *UpdateService) SetApp(app *application.App) {
	if s.checker != nil {
		s.checker.SetApp(app)
	}
	if s.deps != nil {
		s.deps.App = app
	}
}
