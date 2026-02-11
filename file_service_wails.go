package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/jbeck018/howlerops/services"
)

// WailsFileService handles file operations, dialogs, credentials, and utility functions for Wails v3
type WailsFileService struct {
	deps              *SharedDeps
	fileService       *services.FileService
	credentialService *services.CredentialService
	passwordManager   *services.PasswordManager
}

// NewWailsFileService creates a new WailsFileService instance
func NewWailsFileService(deps *SharedDeps, fs *services.FileService, cs *services.CredentialService, pm *services.PasswordManager) *WailsFileService {
	return &WailsFileService{
		deps:              deps,
		fileService:       fs,
		credentialService: cs,
		passwordManager:   pm,
	}
}

// ===============================
// File Dialog Methods
// ===============================

// OpenFileDialog opens a file dialog and returns the selected file path
func (a *WailsFileService) OpenFileDialog() (string, error) {
	return a.fileService.OpenFile(nil)
}

// OpenEnvFileDialog opens a file dialog for .env files and returns the selected file path and content
func (a *WailsFileService) OpenEnvFileDialog() (map[string]string, error) {
	// ShowHiddenFiles(true) is required on macOS to see dotfiles like .env
	// NOTE: Do NOT use AddFilter() - .env files have no extension and macOS
	// will grey them out as unselectable when any filter is applied
	dialog := a.deps.App.Dialog.OpenFile().
		SetTitle("Open .env File").
		ShowHiddenFiles(true)

	filePath, err := dialog.PromptForSingleSelection()
	if err != nil {
		return nil, err
	}

	if filePath == "" {
		return nil, nil // User cancelled
	}

	// Read the file content
	content, err := a.fileService.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return map[string]string{
		"path":    filePath,
		"content": content,
	}, nil
}

// SaveFileDialog opens a save file dialog and returns the selected file path
func (a *WailsFileService) SaveFileDialog() (string, error) {
	return a.fileService.SaveFile("query.sql")
}

// ReadFile reads a file and returns its content
func (a *WailsFileService) ReadFile(filePath string) (string, error) {
	return a.fileService.ReadFile(filePath)
}

// WriteFile writes content to a file
func (a *WailsFileService) WriteFile(filePath, content string) error {
	return a.fileService.WriteFile(filePath, content)
}

// ===============================
// Credential Management Methods
// ===============================

// StorePassword stores a password securely using the hybrid password manager
// connectionID is used as the identifier for the stored credential
// masterKeyBase64 is an optional base64-encoded master key from the authenticated session
// If empty, uses keychain only (graceful degradation)
func (a *WailsFileService) StorePassword(connectionID, password, masterKeyBase64 string) error {
	a.deps.Logger.WithField("connection_id", connectionID).Debug("Storing password with hybrid password manager")

	if a.passwordManager == nil {
		// Fallback to legacy keychain-only if password manager not initialized
		return a.credentialService.StorePassword(connectionID, password)
	}

	userID := ""
	if a.deps.StorageManager != nil {
		userID = a.deps.StorageManager.GetUserID()
	}

	// Decode master key if provided
	var masterKey []byte
	var err error
	if masterKeyBase64 != "" {
		masterKey, err = base64.StdEncoding.DecodeString(masterKeyBase64)
		if err != nil {
			a.deps.Logger.WithError(err).Warn("Invalid master key provided, falling back to keychain only")
			masterKey = nil
		}
	}

	ctx := context.Background()
	if err := a.passwordManager.StorePassword(ctx, userID, connectionID, password, masterKey); err != nil {
		a.deps.Logger.WithError(err).Error("Failed to store password")
		return fmt.Errorf("failed to store password securely: %w", err)
	}

	return nil
}

// GetPassword retrieves a password using the hybrid password manager
// connectionID is the identifier for the stored credential
// masterKeyBase64 is an optional base64-encoded master key from the authenticated session
// If empty, tries keychain fallback
func (a *WailsFileService) GetPassword(connectionID, masterKeyBase64 string) (string, error) {
	a.deps.Logger.WithField("connection_id", connectionID).Debug("Retrieving password with hybrid password manager")

	if a.passwordManager == nil {
		// Fallback to legacy keychain-only if password manager not initialized
		return a.credentialService.GetPassword(connectionID)
	}

	userID := ""
	if a.deps.StorageManager != nil {
		userID = a.deps.StorageManager.GetUserID()
	}

	// Decode master key if provided
	var masterKey []byte
	var err error
	if masterKeyBase64 != "" {
		masterKey, err = base64.StdEncoding.DecodeString(masterKeyBase64)
		if err != nil {
			a.deps.Logger.WithError(err).Warn("Invalid master key provided, falling back to keychain")
			masterKey = nil
		}
	}

	ctx := context.Background()
	password, err := a.passwordManager.GetPassword(ctx, userID, connectionID, masterKey)
	if err != nil {
		a.deps.Logger.WithError(err).Error("Failed to retrieve password")
		return "", fmt.Errorf("failed to retrieve password: %w", err)
	}

	return password, nil
}

// DeletePassword deletes a password using the hybrid password manager
func (a *WailsFileService) DeletePassword(connectionID string) error {
	a.deps.Logger.WithField("connection_id", connectionID).Debug("Deleting password with hybrid password manager")

	if a.passwordManager == nil {
		// Fallback to legacy keychain-only if password manager not initialized
		return a.credentialService.DeletePassword(connectionID)
	}

	userID := ""
	if a.deps.StorageManager != nil {
		userID = a.deps.StorageManager.GetUserID()
	}

	ctx := context.Background()
	if err := a.passwordManager.DeletePassword(ctx, userID, connectionID); err != nil {
		a.deps.Logger.WithError(err).Error("Failed to delete password")
		return fmt.Errorf("failed to delete password: %w", err)
	}

	return nil
}

// HasPassword checks if a password exists in either keychain or encrypted DB
// Returns true if the password exists, false otherwise
func (a *WailsFileService) HasPassword(connectionID string) bool {
	if a.passwordManager == nil {
		// Fallback to legacy keychain-only if password manager not initialized
		return a.credentialService.HasPassword(connectionID)
	}

	// Try to get password - if successful, it exists
	// Pass empty master key since we're just checking existence
	_, err := a.GetPassword(connectionID, "")
	return err == nil
}

// ===============================
// Dialog Methods
// ===============================

// ShowInfoDialog shows an information dialog
func (a *WailsFileService) ShowInfoDialog(title, message string) {
	if a.deps.App == nil {
		a.deps.Logger.Warn("Cannot show info dialog - application not initialized")
		return
	}
	a.deps.App.Dialog.Info().SetTitle(title).SetMessage(message).Show()
}

// ShowErrorDialog shows an error dialog
func (a *WailsFileService) ShowErrorDialog(title, message string) {
	if a.deps.App == nil {
		a.deps.Logger.Warn("Cannot show error dialog - application not initialized")
		return
	}
	a.deps.App.Dialog.Error().SetTitle(title).SetMessage(message).Show()
}

// ShowQuestionDialog shows a question dialog and returns the result
func (a *WailsFileService) ShowQuestionDialog(title, message string) (bool, error) {
	if a.deps.App == nil {
		return false, fmt.Errorf("application not initialized")
	}
	// v3 Question dialog requires adding buttons and checking which was clicked
	// For now, use a simple implementation that shows the dialog and defaults to false
	dialog := a.deps.App.Dialog.Question().SetTitle(title).SetMessage(message)
	yesBtn := dialog.AddButton("Yes")
	dialog.AddButton("No")
	dialog.SetDefaultButton(yesBtn)
	dialog.Show()
	// Note: v3's Show() is async. For sync result, we'd need a different approach.
	// This is a simplification - the dialog shows but result isn't captured.
	return false, nil
}

// ShowNotification shows a notification dialog
func (a *WailsFileService) ShowNotification(title, message string, isError bool) {
	if isError {
		a.ShowErrorDialog(title, message)
	} else {
		a.ShowInfoDialog(title, message)
	}
}

// ===============================
// Utility Methods
// ===============================

// GetAppVersion returns the application version
func (a *WailsFileService) GetAppVersion() string {
	return "1.0.0"
}

// GetHomePath returns the user's home directory
func (a *WailsFileService) GetHomePath() (string, error) {
	return a.fileService.GetHomePath()
}

// ===============================
// File Service Delegation Methods
// ===============================

// GetFileInfo returns file information
func (a *WailsFileService) GetFileInfo(filePath string) (*services.FileInfo, error) {
	return a.fileService.GetFileInfo(filePath)
}

// FileExists checks if a file exists
func (a *WailsFileService) FileExists(filePath string) bool {
	return a.fileService.FileExists(filePath)
}

// GetRecentFiles returns recently accessed files
func (a *WailsFileService) GetRecentFiles() ([]services.RecentFile, error) {
	return a.fileService.GetRecentFiles()
}

// ClearRecentFiles clears the recent files list
func (a *WailsFileService) ClearRecentFiles() {
	a.fileService.ClearRecentFiles()
}

// RemoveFromRecentFiles removes a file from the recent files list
func (a *WailsFileService) RemoveFromRecentFiles(filePath string) {
	a.fileService.RemoveFromRecentFiles(filePath)
}

// GetWorkspaceFiles returns files in a workspace directory
func (a *WailsFileService) GetWorkspaceFiles(dirPath string, extensions []string) ([]services.FileInfo, error) {
	return a.fileService.GetWorkspaceFiles(dirPath, extensions)
}

// CreateDirectory creates a new directory
func (a *WailsFileService) CreateDirectory(dirPath string) error {
	return a.fileService.CreateDirectory(dirPath)
}

// DeleteFile deletes a file
func (a *WailsFileService) DeleteFile(filePath string) error {
	return a.fileService.DeleteFile(filePath)
}

// CopyFile copies a file from source to destination
func (a *WailsFileService) CopyFile(srcPath, destPath string) error {
	return a.fileService.CopyFile(srcPath, destPath)
}

// GetTempDir returns the system temporary directory
func (a *WailsFileService) GetTempDir() string {
	return a.fileService.GetTempDir()
}

// CreateTempFile creates a temporary file with the given content
func (a *WailsFileService) CreateTempFile(content, prefix, suffix string) (string, error) {
	return a.fileService.CreateTempFile(content, prefix, suffix)
}

// GetDownloadsPath returns the user's downloads directory
func (a *WailsFileService) GetDownloadsPath() (string, error) {
	return a.fileService.GetDownloadsPath()
}

// SaveToDownloads saves content to a file in the downloads directory
func (a *WailsFileService) SaveToDownloads(filename, content string) (string, error) {
	return a.fileService.SaveToDownloads(filename, content)
}

// ===============================
// Icon Management Methods
// ===============================

// GetAppIcon returns the main application icon
func (a *WailsFileService) GetAppIcon() ([]byte, error) {
	return iconFS.ReadFile("assets/howlerops-transparent.png")
}

// GetLightIcon returns the light theme icon
func (a *WailsFileService) GetLightIcon() ([]byte, error) {
	return iconFS.ReadFile("assets/howlerops-light.png")
}

// GetDarkIcon returns the dark theme icon
func (a *WailsFileService) GetDarkIcon() ([]byte, error) {
	return iconFS.ReadFile("assets/howlerops-dark.png")
}
