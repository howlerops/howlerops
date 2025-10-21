package services

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// EventsEmitter abstracts the Wails event emitter so services can be unit tested.
type EventsEmitter interface {
	Emit(ctx context.Context, event string, data interface{}) error
}

type runtimeEventsEmitter struct{}

func (runtimeEventsEmitter) Emit(ctx context.Context, event string, data interface{}) error {
	runtime.EventsEmit(ctx, event, data)
	return nil
}

func defaultEventsEmitter() EventsEmitter {
	return runtimeEventsEmitter{}
}
