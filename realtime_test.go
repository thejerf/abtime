package abtime

import (
	"testing"
	"time"
)

// given the simplicity of the implementations here, there isn't much that
// can subtly go wrong, so this is mostly a coverage test, although, it's
// legit to at least ensure all functions are covered and don't crash.

func TestConcrete(t *testing.T) {
	rt := NewRealTime()
	rt.Now()

	ch := rt.After(time.Nanosecond, 0)
	<-ch

	rt.Sleep(time.Nanosecond, 0)

	ch = rt.Tick(time.Nanosecond, 0)
	<-ch
	<-ch

	ticker := rt.NewTicker(time.Nanosecond, 0)
	ticker.Stop()

	sendAfter := make(chan struct{})
	rt.AfterFunc(time.Nanosecond, func() {
		sendAfter <- struct{}{}
	}, 0)
	<-sendAfter

	timer := rt.NewTimer(time.Nanosecond, 0)
	if timer.Channel() == nil {
		t.Fatal("Channel isn't working properly")
	}
	timer.Reset(time.Millisecond)
	timer.Stop()
}
