package services

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// KeyboardService handles keyboard shortcuts and events
type KeyboardService struct {
	logger   *logrus.Logger
	ctx      context.Context
	mu       sync.RWMutex
	bindings map[string]KeyboardAction
}

// KeyboardAction represents a keyboard shortcut action
type KeyboardAction struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Handler     string `json:"handler"`
}

// KeyboardEvent represents a keyboard event
type KeyboardEvent struct {
	Key       string `json:"key"`
	CtrlKey   bool   `json:"ctrlKey"`
	AltKey    bool   `json:"altKey"`
	ShiftKey  bool   `json:"shiftKey"`
	MetaKey   bool   `json:"metaKey"`
	Timestamp int64  `json:"timestamp"`
}

// NewKeyboardService creates a new keyboard service
func NewKeyboardService(logger *logrus.Logger) *KeyboardService {
	service := &KeyboardService{
		logger:   logger,
		bindings: make(map[string]KeyboardAction),
	}

	// Initialize default keyboard shortcuts
	service.initializeDefaultBindings()

	return service
}

// SetContext sets the Wails context
func (k *KeyboardService) SetContext(ctx context.Context) {
	k.ctx = ctx
}

// initializeDefaultBindings sets up the default keyboard shortcuts
func (k *KeyboardService) initializeDefaultBindings() {
	k.mu.Lock()
	defer k.mu.Unlock()

	// File operations
	k.bindings["ctrl+n"] = KeyboardAction{
		Key:         "Ctrl+N",
		Description: "New Query",
		Category:    "File",
		Handler:     "file:new-query",
	}
	k.bindings["cmd+n"] = KeyboardAction{
		Key:         "Cmd+N",
		Description: "New Query",
		Category:    "File",
		Handler:     "file:new-query",
	}
	k.bindings["ctrl+o"] = KeyboardAction{
		Key:         "Ctrl+O",
		Description: "Open File",
		Category:    "File",
		Handler:     "file:open",
	}
	k.bindings["cmd+o"] = KeyboardAction{
		Key:         "Cmd+O",
		Description: "Open File",
		Category:    "File",
		Handler:     "file:open",
	}
	k.bindings["ctrl+s"] = KeyboardAction{
		Key:         "Ctrl+S",
		Description: "Save File",
		Category:    "File",
		Handler:     "file:save",
	}
	k.bindings["cmd+s"] = KeyboardAction{
		Key:         "Cmd+S",
		Description: "Save File",
		Category:    "File",
		Handler:     "file:save",
	}
	k.bindings["ctrl+shift+s"] = KeyboardAction{
		Key:         "Ctrl+Shift+S",
		Description: "Save As",
		Category:    "File",
		Handler:     "file:save-as",
	}
	k.bindings["cmd+shift+s"] = KeyboardAction{
		Key:         "Cmd+Shift+S",
		Description: "Save As",
		Category:    "File",
		Handler:     "file:save-as",
	}
	k.bindings["ctrl+w"] = KeyboardAction{
		Key:         "Ctrl+W",
		Description: "Close Tab",
		Category:    "File",
		Handler:     "file:close-tab",
	}
	k.bindings["cmd+w"] = KeyboardAction{
		Key:         "Cmd+W",
		Description: "Close Tab",
		Category:    "File",
		Handler:     "file:close-tab",
	}

	// Edit operations
	k.bindings["ctrl+z"] = KeyboardAction{
		Key:         "Ctrl+Z",
		Description: "Undo",
		Category:    "Edit",
		Handler:     "edit:undo",
	}
	k.bindings["cmd+z"] = KeyboardAction{
		Key:         "Cmd+Z",
		Description: "Undo",
		Category:    "Edit",
		Handler:     "edit:undo",
	}
	k.bindings["ctrl+y"] = KeyboardAction{
		Key:         "Ctrl+Y",
		Description: "Redo",
		Category:    "Edit",
		Handler:     "edit:redo",
	}
	k.bindings["cmd+y"] = KeyboardAction{
		Key:         "Cmd+Y",
		Description: "Redo",
		Category:    "Edit",
		Handler:     "edit:redo",
	}
	k.bindings["ctrl+x"] = KeyboardAction{
		Key:         "Ctrl+X",
		Description: "Cut",
		Category:    "Edit",
		Handler:     "edit:cut",
	}
	k.bindings["cmd+x"] = KeyboardAction{
		Key:         "Cmd+X",
		Description: "Cut",
		Category:    "Edit",
		Handler:     "edit:cut",
	}
	k.bindings["ctrl+c"] = KeyboardAction{
		Key:         "Ctrl+C",
		Description: "Copy",
		Category:    "Edit",
		Handler:     "edit:copy",
	}
	k.bindings["cmd+c"] = KeyboardAction{
		Key:         "Cmd+C",
		Description: "Copy",
		Category:    "Edit",
		Handler:     "edit:copy",
	}
	k.bindings["ctrl+v"] = KeyboardAction{
		Key:         "Ctrl+V",
		Description: "Paste",
		Category:    "Edit",
		Handler:     "edit:paste",
	}
	k.bindings["cmd+v"] = KeyboardAction{
		Key:         "Cmd+V",
		Description: "Paste",
		Category:    "Edit",
		Handler:     "edit:paste",
	}
	k.bindings["ctrl+f"] = KeyboardAction{
		Key:         "Ctrl+F",
		Description: "Find",
		Category:    "Edit",
		Handler:     "edit:find",
	}
	k.bindings["cmd+f"] = KeyboardAction{
		Key:         "Cmd+F",
		Description: "Find",
		Category:    "Edit",
		Handler:     "edit:find",
	}
	k.bindings["ctrl+h"] = KeyboardAction{
		Key:         "Ctrl+H",
		Description: "Replace",
		Category:    "Edit",
		Handler:     "edit:replace",
	}
	k.bindings["cmd+h"] = KeyboardAction{
		Key:         "Cmd+H",
		Description: "Replace",
		Category:    "Edit",
		Handler:     "edit:replace",
	}

	// Query operations
	k.bindings["ctrl+enter"] = KeyboardAction{
		Key:         "Ctrl+Enter",
		Description: "Run Query",
		Category:    "Query",
		Handler:     "query:run",
	}
	k.bindings["cmd+enter"] = KeyboardAction{
		Key:         "Cmd+Enter",
		Description: "Run Query",
		Category:    "Query",
		Handler:     "query:run",
	}
	k.bindings["ctrl+shift+enter"] = KeyboardAction{
		Key:         "Ctrl+Shift+Enter",
		Description: "Run Selection",
		Category:    "Query",
		Handler:     "query:run-selection",
	}
	k.bindings["cmd+shift+enter"] = KeyboardAction{
		Key:         "Cmd+Shift+Enter",
		Description: "Run Selection",
		Category:    "Query",
		Handler:     "query:run-selection",
	}
	k.bindings["ctrl+e"] = KeyboardAction{
		Key:         "Ctrl+E",
		Description: "Explain Query",
		Category:    "Query",
		Handler:     "query:explain",
	}
	k.bindings["cmd+e"] = KeyboardAction{
		Key:         "Cmd+E",
		Description: "Explain Query",
		Category:    "Query",
		Handler:     "query:explain",
	}
	k.bindings["ctrl+shift+f"] = KeyboardAction{
		Key:         "Ctrl+Shift+F",
		Description: "Format Query",
		Category:    "Query",
		Handler:     "query:format",
	}
	k.bindings["cmd+shift+f"] = KeyboardAction{
		Key:         "Cmd+Shift+F",
		Description: "Format Query",
		Category:    "Query",
		Handler:     "query:format",
	}

	// Connection operations
	k.bindings["ctrl+shift+n"] = KeyboardAction{
		Key:         "Ctrl+Shift+N",
		Description: "New Connection",
		Category:    "Connection",
		Handler:     "connection:new",
	}
	k.bindings["cmd+shift+n"] = KeyboardAction{
		Key:         "Cmd+Shift+N",
		Description: "New Connection",
		Category:    "Connection",
		Handler:     "connection:new",
	}
	k.bindings["ctrl+t"] = KeyboardAction{
		Key:         "Ctrl+T",
		Description: "Test Connection",
		Category:    "Connection",
		Handler:     "connection:test",
	}
	k.bindings["cmd+t"] = KeyboardAction{
		Key:         "Cmd+T",
		Description: "Test Connection",
		Category:    "Connection",
		Handler:     "connection:test",
	}
	k.bindings["ctrl+r"] = KeyboardAction{
		Key:         "Ctrl+R",
		Description: "Refresh",
		Category:    "Connection",
		Handler:     "connection:refresh",
	}
	k.bindings["cmd+r"] = KeyboardAction{
		Key:         "Cmd+R",
		Description: "Refresh",
		Category:    "Connection",
		Handler:     "connection:refresh",
	}

	// View operations
	k.bindings["ctrl+b"] = KeyboardAction{
		Key:         "Ctrl+B",
		Description: "Toggle Sidebar",
		Category:    "View",
		Handler:     "view:toggle-sidebar",
	}
	k.bindings["cmd+b"] = KeyboardAction{
		Key:         "Cmd+B",
		Description: "Toggle Sidebar",
		Category:    "View",
		Handler:     "view:toggle-sidebar",
	}
	k.bindings["ctrl+shift+r"] = KeyboardAction{
		Key:         "Ctrl+Shift+R",
		Description: "Toggle Results Panel",
		Category:    "View",
		Handler:     "view:toggle-results",
	}
	k.bindings["cmd+shift+r"] = KeyboardAction{
		Key:         "Cmd+Shift+R",
		Description: "Toggle Results Panel",
		Category:    "View",
		Handler:     "view:toggle-results",
	}
	k.bindings["ctrl+="] = KeyboardAction{
		Key:         "Ctrl+=",
		Description: "Zoom In",
		Category:    "View",
		Handler:     "view:zoom-in",
	}
	k.bindings["cmd+="] = KeyboardAction{
		Key:         "Cmd+=",
		Description: "Zoom In",
		Category:    "View",
		Handler:     "view:zoom-in",
	}
	k.bindings["ctrl+-"] = KeyboardAction{
		Key:         "Ctrl+-",
		Description: "Zoom Out",
		Category:    "View",
		Handler:     "view:zoom-out",
	}
	k.bindings["cmd+-"] = KeyboardAction{
		Key:         "Cmd+-",
		Description: "Zoom Out",
		Category:    "View",
		Handler:     "view:zoom-out",
	}
	k.bindings["ctrl+0"] = KeyboardAction{
		Key:         "Ctrl+0",
		Description: "Reset Zoom",
		Category:    "View",
		Handler:     "view:reset-zoom",
	}
	k.bindings["cmd+0"] = KeyboardAction{
		Key:         "Cmd+0",
		Description: "Reset Zoom",
		Category:    "View",
		Handler:     "view:reset-zoom",
	}

	// Navigation
	k.bindings["ctrl+tab"] = KeyboardAction{
		Key:         "Ctrl+Tab",
		Description: "Next Tab",
		Category:    "Navigation",
		Handler:     "nav:next-tab",
	}
	k.bindings["cmd+tab"] = KeyboardAction{
		Key:         "Cmd+Tab",
		Description: "Next Tab",
		Category:    "Navigation",
		Handler:     "nav:next-tab",
	}
	k.bindings["ctrl+shift+tab"] = KeyboardAction{
		Key:         "Ctrl+Shift+Tab",
		Description: "Previous Tab",
		Category:    "Navigation",
		Handler:     "nav:prev-tab",
	}
	k.bindings["cmd+shift+tab"] = KeyboardAction{
		Key:         "Cmd+Shift+Tab",
		Description: "Previous Tab",
		Category:    "Navigation",
		Handler:     "nav:prev-tab",
	}

	// Help
	k.bindings["ctrl+?"] = KeyboardAction{
		Key:         "Ctrl+?",
		Description: "Keyboard Shortcuts",
		Category:    "Help",
		Handler:     "help:shortcuts",
	}
	k.bindings["cmd+?"] = KeyboardAction{
		Key:         "Cmd+?",
		Description: "Keyboard Shortcuts",
		Category:    "Help",
		Handler:     "help:shortcuts",
	}
	k.bindings["f1"] = KeyboardAction{
		Key:         "F1",
		Description: "Help",
		Category:    "Help",
		Handler:     "help:show",
	}

	// Special keys
	k.bindings["escape"] = KeyboardAction{
		Key:         "Escape",
		Description: "Cancel/Close",
		Category:    "General",
		Handler:     "general:escape",
	}
	k.bindings["f5"] = KeyboardAction{
		Key:         "F5",
		Description: "Refresh",
		Category:    "General",
		Handler:     "general:refresh",
	}
}

// HandleKeyboardEvent processes a keyboard event
func (k *KeyboardService) HandleKeyboardEvent(event KeyboardEvent) {
	key := k.normalizeKeyString(event)

	k.mu.RLock()
	action, exists := k.bindings[key]
	k.mu.RUnlock()

	if !exists {
		return
	}

	k.logger.WithFields(logrus.Fields{
		"key":     key,
		"handler": action.Handler,
	}).Debug("Keyboard shortcut triggered")

	// Emit keyboard event
	runtime.EventsEmit(k.ctx, "keyboard:shortcut", map[string]interface{}{
		"key":     key,
		"action":  action.Handler,
		"event":   event,
	})

	// Emit specific handler event
	runtime.EventsEmit(k.ctx, action.Handler, event)
}

// normalizeKeyString converts keyboard event to normalized key string
func (k *KeyboardService) normalizeKeyString(event KeyboardEvent) string {
	var parts []string

	// Add modifiers (order matters for consistency)
	if event.CtrlKey {
		parts = append(parts, "ctrl")
	}
	if event.MetaKey {
		parts = append(parts, "cmd")
	}
	if event.AltKey {
		parts = append(parts, "alt")
	}
	if event.ShiftKey {
		parts = append(parts, "shift")
	}

	// Add the main key
	parts = append(parts, event.Key)

	return joinKeys(parts)
}

// joinKeys joins key parts with "+"
func joinKeys(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "+" + parts[i]
	}
	return result
}

// GetAllBindings returns all keyboard bindings
func (k *KeyboardService) GetAllBindings() map[string]KeyboardAction {
	k.mu.RLock()
	defer k.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	bindings := make(map[string]KeyboardAction)
	for key, action := range k.bindings {
		bindings[key] = action
	}

	return bindings
}

// GetBindingsByCategory returns keyboard bindings grouped by category
func (k *KeyboardService) GetBindingsByCategory() map[string][]KeyboardAction {
	k.mu.RLock()
	defer k.mu.RUnlock()

	categories := make(map[string][]KeyboardAction)

	for _, action := range k.bindings {
		categories[action.Category] = append(categories[action.Category], action)
	}

	return categories
}

// AddBinding adds a new keyboard binding
func (k *KeyboardService) AddBinding(key string, action KeyboardAction) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.bindings[key] = action

	k.logger.WithFields(logrus.Fields{
		"key":     key,
		"handler": action.Handler,
	}).Info("Keyboard binding added")
}

// RemoveBinding removes a keyboard binding
func (k *KeyboardService) RemoveBinding(key string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	delete(k.bindings, key)

	k.logger.WithField("key", key).Info("Keyboard binding removed")
}

// IsBindingExists checks if a keyboard binding exists
func (k *KeyboardService) IsBindingExists(key string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()

	_, exists := k.bindings[key]
	return exists
}

// GetBinding returns a specific keyboard binding
func (k *KeyboardService) GetBinding(key string) (KeyboardAction, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	action, exists := k.bindings[key]
	return action, exists
}

// UpdateBinding updates an existing keyboard binding
func (k *KeyboardService) UpdateBinding(key string, action KeyboardAction) bool {
	k.mu.Lock()
	defer k.mu.Unlock()

	if _, exists := k.bindings[key]; !exists {
		return false
	}

	k.bindings[key] = action

	k.logger.WithFields(logrus.Fields{
		"key":     key,
		"handler": action.Handler,
	}).Info("Keyboard binding updated")

	return true
}

// ResetToDefaults resets all bindings to default values
func (k *KeyboardService) ResetToDefaults() {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.bindings = make(map[string]KeyboardAction)
	k.initializeDefaultBindings()

	k.logger.Info("Keyboard bindings reset to defaults")

	// Emit reset event
	runtime.EventsEmit(k.ctx, "keyboard:reset", nil)
}

// ExportBindings exports keyboard bindings as JSON
func (k *KeyboardService) ExportBindings() map[string]KeyboardAction {
	return k.GetAllBindings()
}

// ImportBindings imports keyboard bindings from a map
func (k *KeyboardService) ImportBindings(bindings map[string]KeyboardAction) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for key, action := range bindings {
		k.bindings[key] = action
	}

	k.logger.WithField("count", len(bindings)).Info("Keyboard bindings imported")

	// Emit import event
	runtime.EventsEmit(k.ctx, "keyboard:imported", map[string]interface{}{
		"count": len(bindings),
	})
}