package main

// shared_deps.go
// Shared dependencies structure for all decomposed services
// This struct will be passed to each service to provide access to common resources

import (
	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jbeck018/howlerops/pkg/federation/duckdb"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/services"
)

// SharedDeps holds common dependencies shared across all decomposed services
type SharedDeps struct {
	Logger          *logrus.Logger
	StorageManager  *storage.Manager
	DatabaseService *services.DatabaseService
	DuckDBEngine    *duckdb.Engine
	App             *application.App
	MainWindow      application.Window
}

// emitEvent is a helper method to emit events through the Wails application
func (s *SharedDeps) emitEvent(eventName string, data interface{}) {
	if s.App != nil {
		s.App.Event.Emit(eventName, data)
	}
}
