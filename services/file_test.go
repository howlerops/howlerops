package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newSilentFileLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestFileServiceWriteReadAndEvents(t *testing.T) {
	emitter := newRecordingEmitter()
	logger := newSilentFileLogger()
	service := NewFileServiceWithEmitter(logger, emitter)
	ctx := context.Background()
	service.SetContext(ctx)

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "sample.sql")

	if err := service.WriteFile(target, "SELECT 42;"); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, ok := emitter.WaitFor("file:saved", 100*time.Millisecond); !ok {
		t.Fatalf("expected file:saved event")
	}

	content, err := service.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if content != "SELECT 42;" {
		t.Fatalf("unexpected file content: %s", content)
	}

	recent, err := service.GetRecentFiles()
	if err != nil {
		t.Fatalf("GetRecentFiles returned error: %v", err)
	}
	if len(recent) == 0 || recent[0].Path != target {
		t.Fatalf("expected recent files to include %s, got %#v", target, recent)
	}
}

func TestFileServiceDeleteAndRecentClearedEvents(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewFileServiceWithEmitter(newSilentFileLogger(), emitter)
	service.SetContext(context.Background())

	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "to_delete.sql")
	if err := os.WriteFile(target, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	service.addToRecentFiles(target)

	if err := service.DeleteFile(target); err != nil {
		t.Fatalf("DeleteFile returned error: %v", err)
	}
	if _, ok := emitter.WaitFor("file:deleted", 100*time.Millisecond); !ok {
		t.Fatalf("expected file:deleted event")
	}

	service.ClearRecentFiles()
	if _, ok := emitter.WaitFor("file:recent-cleared", 100*time.Millisecond); !ok {
		t.Fatalf("expected file:recent-cleared event")
	}
}

func TestFileServiceSaveToDownloadsHonorsHome(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewFileServiceWithEmitter(newSilentFileLogger(), emitter)
	service.SetContext(context.Background())

	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	path, err := service.SaveToDownloads("report.sql", "select now();")
	if err != nil {
		t.Fatalf("SaveToDownloads returned error: %v", err)
	}
	if _, ok := emitter.WaitFor("file:saved", 100*time.Millisecond); !ok {
		t.Fatalf("expected file:saved event for downloads")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected downloads file to exist: %v", err)
	}
	if !strings.HasPrefix(path, filepath.Join(tempHome, "Downloads")) {
		t.Fatalf("expected file in Downloads directory, got %s", path)
	}
}
