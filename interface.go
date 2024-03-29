package abtime

import (
	"context"
	"time"
)

// Ticker defines an interface for the functions that return *time.Ticker
// in the original Time module.
type Ticker interface {
	Channel() <-chan time.Time
	Reset(time.Duration)
	Stop()
}

// Timer defines an interface for the functions that return *time.Timer
// in the original Time module.
type Timer interface {
	Stop() bool
	Reset(time.Duration) bool
	Channel() <-chan time.Time
}

// The AbstractTime interface abstracts the time module into an interface.
type AbstractTime interface {
	Now() time.Time
	After(time.Duration, int) <-chan time.Time
	Sleep(time.Duration, int)
	Tick(time.Duration, int) <-chan time.Time
	NewTicker(time.Duration, int) Ticker
	AfterFunc(time.Duration, func(), int) Timer
	NewTimer(time.Duration, int) Timer

	WithDeadline(context.Context, time.Time, int) (context.Context, context.CancelFunc)
	WithTimeout(context.Context, time.Duration, int) (context.Context, context.CancelFunc)
}
