package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FileService handles file operations for HowlerOps
type FileService struct {
	logger         *logrus.Logger
	ctx            context.Context
	recentFiles    []string
	maxRecentFiles int
}

// FileInfo represents file metadata
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"modTime"`
	IsDirectory  bool      `json:"isDirectory"`
	Extension    string    `json:"extension"`
	Permissions  string    `json:"permissions"`
}

// RecentFile represents a recently opened file
type RecentFile struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	LastOpened  time.Time `json:"lastOpened"`
	Size        int64     `json:"size"`
}

// NewFileService creates a new file service
func NewFileService(logger *logrus.Logger) *FileService {
	return &FileService{
		logger:         logger,
		recentFiles:    make([]string, 0),
		maxRecentFiles: 10,
	}
}

// SetContext sets the Wails context
func (f *FileService) SetContext(ctx context.Context) {
	f.ctx = ctx
}

// OpenFile opens a file dialog and returns the selected file path
func (f *FileService) OpenFile(filters []runtime.FileFilter) (string, error) {
	if filters == nil {
		filters = []runtime.FileFilter{
			{
				DisplayName: "SQL Files (*.sql)",
				Pattern:     "*.sql",
			},
			{
				DisplayName: "Text Files (*.txt)",
				Pattern:     "*.txt",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		}
	}

	options := runtime.OpenDialogOptions{
		Title:   "Open SQL File",
		Filters: filters,
	}

	filePath, err := runtime.OpenFileDialog(f.ctx, options)
	if err != nil {
		return "", err
	}

	if filePath != "" {
		f.addToRecentFiles(filePath)
		f.logger.WithField("file_path", filePath).Info("File opened")

		// Emit file opened event
		runtime.EventsEmit(f.ctx, "file:opened", map[string]interface{}{
			"path": filePath,
		})
	}

	return filePath, nil
}

// SaveFile opens a save file dialog and returns the selected file path
func (f *FileService) SaveFile(defaultFilename string) (string, error) {
	if defaultFilename == "" {
		defaultFilename = "query.sql"
	}

	options := runtime.SaveDialogOptions{
		Title: "Save SQL File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "SQL Files (*.sql)",
				Pattern:     "*.sql",
			},
			{
				DisplayName: "Text Files (*.txt)",
				Pattern:     "*.txt",
			},
		},
		DefaultFilename: defaultFilename,
	}

	filePath, err := runtime.SaveFileDialog(f.ctx, options)
	if err != nil {
		return "", err
	}

	if filePath != "" {
		f.logger.WithField("file_path", filePath).Info("File save location selected")

		// Emit file save dialog event
		runtime.EventsEmit(f.ctx, "file:save-dialog", map[string]interface{}{
			"path": filePath,
		})
	}

	return filePath, nil
}

// ReadFile reads a file and returns its content
func (f *FileService) ReadFile(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"file_path": filePath,
			"error":     err,
		}).Error("Failed to read file")
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	f.addToRecentFiles(filePath)
	f.logger.WithFields(logrus.Fields{
		"file_path": filePath,
		"size":      len(content),
	}).Info("File read successfully")

	return string(content), nil
}

// WriteFile writes content to a file
func (f *FileService) WriteFile(filePath, content string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"file_path": filePath,
			"error":     err,
		}).Error("Failed to write file")
		return fmt.Errorf("failed to write file: %w", err)
	}

	f.addToRecentFiles(filePath)
	f.logger.WithFields(logrus.Fields{
		"file_path": filePath,
		"size":      len(content),
	}).Info("File written successfully")

	// Emit file saved event
	runtime.EventsEmit(f.ctx, "file:saved", map[string]interface{}{
		"path": filePath,
		"size": len(content),
	})

	return nil
}

// GetFileInfo returns file metadata
func (f *FileService) GetFileInfo(filePath string) (*FileInfo, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Name:        stat.Name(),
		Path:        filePath,
		Size:        stat.Size(),
		ModTime:     stat.ModTime(),
		IsDirectory: stat.IsDir(),
		Extension:   filepath.Ext(filePath),
		Permissions: stat.Mode().String(),
	}, nil
}

// FileExists checks if a file exists
func (f *FileService) FileExists(filePath string) bool {
	if filePath == "" {
		return false
	}

	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// GetRecentFiles returns recently opened files
func (f *FileService) GetRecentFiles() ([]RecentFile, error) {
	recentFiles := make([]RecentFile, 0, len(f.recentFiles))

	for _, filePath := range f.recentFiles {
		if !f.FileExists(filePath) {
			continue
		}

		stat, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		recentFiles = append(recentFiles, RecentFile{
			Path:       filePath,
			Name:       filepath.Base(filePath),
			LastOpened: stat.ModTime(),
			Size:       stat.Size(),
		})
	}

	return recentFiles, nil
}

// ClearRecentFiles clears the recent files list
func (f *FileService) ClearRecentFiles() {
	f.recentFiles = make([]string, 0)
	f.logger.Info("Recent files list cleared")

	// Emit recent files cleared event
	runtime.EventsEmit(f.ctx, "file:recent-cleared")
}

// RemoveFromRecentFiles removes a file from recent files list
func (f *FileService) RemoveFromRecentFiles(filePath string) {
	for i, path := range f.recentFiles {
		if path == filePath {
			f.recentFiles = append(f.recentFiles[:i], f.recentFiles[i+1:]...)
			break
		}
	}

	f.logger.WithField("file_path", filePath).Info("File removed from recent files")
}

// addToRecentFiles adds a file to the recent files list
func (f *FileService) addToRecentFiles(filePath string) {
	// Remove if already exists
	f.RemoveFromRecentFiles(filePath)

	// Add to beginning
	f.recentFiles = append([]string{filePath}, f.recentFiles...)

	// Trim to max size
	if len(f.recentFiles) > f.maxRecentFiles {
		f.recentFiles = f.recentFiles[:f.maxRecentFiles]
	}
}

// GetWorkspaceFiles returns files in a directory
func (f *FileService) GetWorkspaceFiles(dirPath string, extensions []string) ([]FileInfo, error) {
	if dirPath == "" {
		var err error
		dirPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	fileInfos := make([]FileInfo, 0)
	for _, file := range files {
		filePath := filepath.Join(dirPath, file.Name())
		ext := strings.ToLower(filepath.Ext(file.Name()))

		// Filter by extensions if provided
		if len(extensions) > 0 {
			found := false
			for _, allowedExt := range extensions {
				if ext == strings.ToLower(allowedExt) {
					found = true
					break
				}
			}
			if !found && !file.IsDir() {
				continue
			}
		}

		fileInfos = append(fileInfos, FileInfo{
			Name:        file.Name(),
			Path:        filePath,
			Size:        file.Size(),
			ModTime:     file.ModTime(),
			IsDirectory: file.IsDir(),
			Extension:   ext,
			Permissions: file.Mode().String(),
		})
	}

	return fileInfos, nil
}

// CreateDirectory creates a directory
func (f *FileService) CreateDirectory(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"dir_path": dirPath,
			"error":    err,
		}).Error("Failed to create directory")
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f.logger.WithField("dir_path", dirPath).Info("Directory created successfully")
	return nil
}

// DeleteFile deletes a file
func (f *FileService) DeleteFile(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	err := os.Remove(filePath)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"file_path": filePath,
			"error":     err,
		}).Error("Failed to delete file")
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from recent files
	f.RemoveFromRecentFiles(filePath)

	f.logger.WithField("file_path", filePath).Info("File deleted successfully")

	// Emit file deleted event
	runtime.EventsEmit(f.ctx, "file:deleted", map[string]interface{}{
		"path": filePath,
	})

	return nil
}

// CopyFile copies a file to a new location
func (f *FileService) CopyFile(srcPath, destPath string) error {
	if srcPath == "" || destPath == "" {
		return fmt.Errorf("source and destination paths cannot be empty")
	}

	content, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Ensure destination directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	err = ioutil.WriteFile(destPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"src_path":  srcPath,
		"dest_path": destPath,
	}).Info("File copied successfully")

	return nil
}

// GetHomePath returns the user's home directory
func (f *FileService) GetHomePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return home, nil
}

// GetTempDir returns the system temporary directory
func (f *FileService) GetTempDir() string {
	return os.TempDir()
}

// CreateTempFile creates a temporary file with the given content
func (f *FileService) CreateTempFile(content, prefix, suffix string) (string, error) {
	tempFile, err := ioutil.TempFile("", prefix+"*"+suffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tempFile.Close()

	if content != "" {
		_, err = tempFile.WriteString(content)
		if err != nil {
			return "", fmt.Errorf("failed to write to temporary file: %w", err)
		}
	}

	filePath := tempFile.Name()
	f.logger.WithField("temp_file", filePath).Info("Temporary file created")

	return filePath, nil
}