package abtime

import (
	"time"
)

// NewRealTime returns a AbTime-conforming object that backs to the
// standard time module.
func NewRealTime() RealTime {
	return RealTime{}
}

// TimerWrap wraps a Timer-conforming wrapper around a *time.Timer.
type TimerWrap struct {
	T *time.Timer
}

// Channel returns the channel the *time.Timer will signal on.
func (tw TimerWrap) Channel() <-chan time.Time {
	return tw.T.C
}

// Stop wraps the *time.Timer.Stop().
func (tw TimerWrap) Stop() bool {
	return tw.T.Stop()
}

// Reset wraps the *time.Timer.Reset().
func (tw TimerWrap) Reset(d time.Duration) bool {
	return tw.T.Reset(d)
}

// The RealTime object implements the direct calls to the time module.
type RealTime struct{}

// Now wraps time.Now.
func (rt RealTime) Now() time.Time {
	return time.Now()
}

// After wraps time.After.
func (rt RealTime) After(d time.Duration, token int) <-chan time.Time {
	return time.After(d)
}

// Sleep wraps time.Sleep.
func (rt RealTime) Sleep(d time.Duration, token int) {
	time.Sleep(d)
}

// Tick wraps time.Tick.
func (rt RealTime) Tick(d time.Duration, token int) <-chan time.Time {
	return time.Tick(d) // nolint: megacheck
}

// NewTicker wraps time.NewTicker. It returns something conforming to the
// abtime.Ticker interface.
func (rt RealTime) NewTicker(d time.Duration, token int) Ticker {
	return tickerWrapper{time.NewTicker(d)}
}

// AfterFunc wraps time.AfterFunc. It returns something conforming to the
// abtime.Timer interface.
func (rt RealTime) AfterFunc(d time.Duration, f func(), token int) Timer {
	return TimerWrap{time.AfterFunc(d, f)}
}

// NewTimer wraps time.NewTimer. It returns something conforming to the
// abtime.Timer interface.
func (rt RealTime) NewTimer(d time.Duration, token int) Timer {
	return TimerWrap{time.NewTimer(d)}
}

type tickerWrapper struct {
	*time.Ticker
}

func (tw tickerWrapper) Channel() <-chan time.Time {
	return tw.C
}
