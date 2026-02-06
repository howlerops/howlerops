package main

import (
	"github.com/jbeck018/howlerops/services"
)

// WailsKeyboardService handles keyboard-related operations for Wails v3
type WailsKeyboardService struct {
	deps            *SharedDeps
	keyboardService *services.KeyboardService
}

// NewWailsKeyboardService creates a new WailsKeyboardService instance
func NewWailsKeyboardService(deps *SharedDeps, ks *services.KeyboardService) *WailsKeyboardService {
	return &WailsKeyboardService{
		deps:            deps,
		keyboardService: ks,
	}
}

// HandleKeyboardEvent handles keyboard events
func (a *WailsKeyboardService) HandleKeyboardEvent(event services.KeyboardEvent) {
	a.keyboardService.HandleKeyboardEvent(event)
}

// GetAllKeyboardBindings returns all keyboard bindings
func (a *WailsKeyboardService) GetAllKeyboardBindings() map[string]services.KeyboardAction {
	return a.keyboardService.GetAllBindings()
}

// GetKeyboardBindingsByCategory returns keyboard bindings grouped by category
func (a *WailsKeyboardService) GetKeyboardBindingsByCategory() map[string][]services.KeyboardAction {
	return a.keyboardService.GetBindingsByCategory()
}

// AddKeyboardBinding adds a new keyboard binding
func (a *WailsKeyboardService) AddKeyboardBinding(key string, action services.KeyboardAction) {
	a.keyboardService.AddBinding(key, action)
}

// RemoveKeyboardBinding removes a keyboard binding
func (a *WailsKeyboardService) RemoveKeyboardBinding(key string) {
	a.keyboardService.RemoveBinding(key)
}

// ResetKeyboardBindings resets keyboard bindings to defaults
func (a *WailsKeyboardService) ResetKeyboardBindings() {
	a.keyboardService.ResetToDefaults()
}

// ExportKeyboardBindings exports all keyboard bindings
func (a *WailsKeyboardService) ExportKeyboardBindings() map[string]services.KeyboardAction {
	return a.keyboardService.ExportBindings()
}

// ImportKeyboardBindings imports keyboard bindings
func (a *WailsKeyboardService) ImportKeyboardBindings(bindings map[string]services.KeyboardAction) {
	a.keyboardService.ImportBindings(bindings)
}
