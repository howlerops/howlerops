package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventsEmitter abstracts the Wails event emitter so services can be unit tested.
type EventsEmitter interface {
	Emit(event string, data interface{}) error
}

// V3EventsEmitter implements EventsEmitter using the v3 application
type V3EventsEmitter struct {
	app *application.App
}

// NewV3EventsEmitter creates a new v3 events emitter
func NewV3EventsEmitter(app *application.App) *V3EventsEmitter {
	return &V3EventsEmitter{app: app}
}

func (e *V3EventsEmitter) Emit(event string, data interface{}) error {
	if e.app != nil {
		e.app.Event.Emit(event, data)
	}
	return nil
}

// NoopEventsEmitter is a no-op emitter for testing
type NoopEventsEmitter struct{}

func (NoopEventsEmitter) Emit(event string, data interface{}) error {
	return nil
}

func defaultEventsEmitter() EventsEmitter {
	return NoopEventsEmitter{}
}
