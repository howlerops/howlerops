package services

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newSilentKeyboardLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestKeyboardServiceHandleKeyboardEventEmitsActions(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewKeyboardServiceWithEmitter(newSilentKeyboardLogger(), emitter)
	service.SetContext(context.Background())

	event := KeyboardEvent{
		Key:       "enter",
		CtrlKey:   true,
		Timestamp: 1,
	}
	service.HandleKeyboardEvent(event)

	if _, ok := emitter.WaitFor("keyboard:shortcut", 100*time.Millisecond); !ok {
		t.Fatalf("expected keyboard:shortcut event")
	}
	if _, ok := emitter.WaitFor("query:run", 100*time.Millisecond); !ok {
		t.Fatalf("expected specific handler event query:run")
	}
}

func TestKeyboardServiceAddRemoveBinding(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewKeyboardServiceWithEmitter(newSilentKeyboardLogger(), emitter)
	service.SetContext(context.Background())

	action := KeyboardAction{
		Key:         "Ctrl+K",
		Description: "Open Command Palette",
		Category:    "General",
		Handler:     "command:palette",
	}

	service.AddBinding("ctrl+k", action)
	if retrieved, ok := service.GetBinding("ctrl+k"); !ok || retrieved.Handler != "command:palette" {
		t.Fatalf("expected custom binding to be registered")
	}

	service.RemoveBinding("ctrl+k")
	if _, exists := service.GetBinding("ctrl+k"); exists {
		t.Fatalf("expected binding to be removed")
	}
}

func TestKeyboardServiceResetAndImportBindings(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewKeyboardServiceWithEmitter(newSilentKeyboardLogger(), emitter)
	service.SetContext(context.Background())

	service.ResetToDefaults()
	if _, ok := emitter.WaitFor("keyboard:reset", 100*time.Millisecond); !ok {
		t.Fatalf("expected keyboard:reset event")
	}

	service.ImportBindings(map[string]KeyboardAction{
		"ctrl+b": {
			Key:         "Ctrl+B",
			Description: "Bold",
			Category:    "Formatting",
			Handler:     "format:bold",
		},
	})
	if _, ok := emitter.WaitFor("keyboard:imported", 100*time.Millisecond); !ok {
		t.Fatalf("expected keyboard:imported event")
	}

	if action, ok := service.GetBinding("ctrl+b"); !ok || action.Handler != "format:bold" {
		t.Fatalf("expected imported binding to be available")
	}
}
