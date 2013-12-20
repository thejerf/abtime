package abtime

import (
	"time"
)

// It's best to allocate IDs like this for your time usages.
const (
	timeoutID = iota
)

func ExampleAbstractTime() {
	// Suppose you have a goroutine feeding you something from a socket,
	// and you want to do something if that times out. You can test this
	// with:
	manualTime := NewManual()
	timedOut := make(chan struct{})

	go ReadSocket(manualTime, timedOut)

	manualTime.Trigger(timeoutID)

	// This will read the struct{}{} from above. Getting here asserts
	// that we did what we wanted when we timed out.
	<-timedOut
}

// In production code, at would be a RealTime, and thus use the "real"
// time.After function, ignoring the ID.
func ReadSocket(at AbstractTime, timedOut chan struct{}) {
	timeout := at.After(time.Second, timeoutID)

	// in this example, this will never be filled
	fromSocket := make(chan []byte)

	select {
	case <-fromSocket:
		// handle socketData
	case <-timeout:
		timedOut <- struct{}{}
	}
}
