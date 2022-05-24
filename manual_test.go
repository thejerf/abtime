package abtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

const (
	afterID = iota
	sleepID
	tickID
	tickID2
	afterFuncID
	timerID
	contextID
	childContextID
)

func TestAfter(t *testing.T) {
	at := NewManual()

	// assuredly trigger before the After is even called
	at.Trigger(afterID)

	sent := make(chan time.Time)
	go func() {
		ch := at.After(time.Second, afterID)
		t := <-ch
		sent <- t
	}()

	result := <-sent

	if result != at.now.Add(time.Second) {
		t.Fatal("Got wrong time sent for After")
	}

	go func() {
		ch := at.After(time.Second, afterID)
		t := <-ch
		sent <- t
	}()

	// Bootstrapping problem; we can't depend on abtime working to test
	// abtime...
	time.Sleep(time.Millisecond)
	at.Trigger(afterID)

	result = <-sent
	if result != at.now.Add(time.Second) {
		t.Fatal("Got the wrong time sent for the After after the call")
	}

	at.Advance(time.Second)
}

func TestSleep(t *testing.T) {
	at := NewManual()

	// trigger the sleep before it even exists
	at.Trigger(sleepID)

	finished := make(chan struct{})
	go func() {
		at.Sleep(time.Second, sleepID)
		finished <- struct{}{}
	}()

	<-finished

	go func() {
		at.Sleep(time.Second, sleepID)
		finished <- struct{}{}
	}()

	time.Sleep(time.Millisecond)
	at.Trigger(sleepID)
	<-finished

	at.Trigger(2, 4)
	for i := 0; i < 5; i++ {
		go func(innerI int) {
			at.Sleep(time.Second, innerI)
			finished <- struct{}{}
		}(i)
	}
	time.Sleep(time.Millisecond)
	at.Trigger(0, 1, 3)

	for i := 0; i < 5; i++ {
		<-finished
	}

	// if we get here, we must not have deadlocked
}

func TestTick(t *testing.T) {
	// significance of this date left as an exercise for the reader
	testTime := time.Date(2012, 3, 28, 12, 0, 0, 0, time.UTC)

	at := NewManualAtTime(testTime)
	if at.Now() != testTime {
		t.Fatal("Now is not working correctly.")
	}
	at.Trigger(tickID)

	received := make(chan time.Time)
	go func() {
		ch := at.Tick(time.Second, tickID)
		tick1 := <-ch
		tick2 := <-ch

		received <- tick1
		received <- tick2
	}()

	time.Sleep(time.Millisecond)
	at.Trigger(tickID)
	time1 := <-received
	time2 := <-received

	if time1 != testTime.Add(time.Second) || time2 != testTime.Add(2*time.Second) {
		t.Fatal("tick did not deliver the correct time")
	}

	ticker := at.NewTicker(time.Second, tickID2)
	ch := ticker.Channel()
	at.Trigger(tickID2)
	<-ch
	ticker.Reset(time.Second)
	ticker.Stop()
	at.Trigger(tickID2)
	// if this test failed, it would hang the test waiting to write on ch.
}

func TestAfterFunc(t *testing.T) {
	at := NewManual()

	funcRun := make(chan struct{})

	timer := at.AfterFunc(time.Second, func() {
		funcRun <- struct{}{}
	}, afterFuncID)

	if timer.Channel() != nil {
		t.Fatal("Channel on AfterFunc not working properly.")
	}

	// not that this really means much here
	if timer.Reset(time.Second * 2) {
		t.Fatal("Reset should not be returning true here")
	}
	at.Trigger(afterFuncID)

	<-funcRun

	timer = at.AfterFunc(time.Second, func() {
		panic("I should never be run!")
	}, afterFuncID)

	if !timer.Stop() {
		t.Fatal("Stop should not return true like this")
	}
	at.Trigger(afterFuncID)
	if timer.Stop() || !timer.Reset(time.Second*3) {
		t.Fatal("Stop/Reset should be returning true here")
	}
}

func TestTimer(t *testing.T) {
	at := NewManual()

	timer := at.NewTimer(time.Second, timerID)
	go func() {
		at.Trigger(timerID)
	}()

	curTime := <-timer.Channel()
	if at.now.Add(time.Second) != curTime {
		t.Fatal("Timer not sending proper time")
	}

	timer = at.NewTimer(time.Second, timerID)
	timer.Reset(2 * time.Second)
	if !timer.Stop() {
		t.Fatal("Stopping the timer should have returned true")
	}
	at.Trigger(timerID)

	// no good way to test the stop worked, the Stop description in
	// the time package explicitly says it does not close the channel.
}

func TestInterfaceConformance(t *testing.T) {
	// this just verifies that both implementations actually implement
	// ManualTime. Nothing else in the package actually does....
	var at AbstractTime // nolint: megacheck
	at = NewManual()
	at = NewRealTime()
	at.Now()
}

func TestNowQueueing(t *testing.T) {
	// this verifies that the "nows" queue in the expected manner
	at := NewManual()
	firstNow := at.Now()
	desired := []time.Time{firstNow.Add(10 * time.Second), firstNow.Add(20 * time.Second)}
	at.QueueNows(desired...)
	if at.Now() != desired[0] {
		t.Fatal("Failed to queue properly")
	}
	if at.Now() != desired[1] {
		t.Fatal("Failed to advance queue properly")
	}
	if at.Now() != desired[1] {
		t.Fatal("Failed to 'stick' the time properly.")
	}
}

func TestTimerReset(t *testing.T) {
	c := NewManual()
	d := time.Hour
	timer := c.NewTimer(d, timerID)

	wasActive := timer.Reset(d)
	if !wasActive {
		t.Fatal("Unexpected value from the Reset method")
	}

	go func() {
		c.Advance(d)
		c.Trigger(timerID)
	}()

	<-timer.Channel()
	wasActive = timer.Reset(d)
	if wasActive {
		t.Fatal("Unexpected reset result value")
	}
}

func TestMultipleTimerCreation(t *testing.T) {
	c := NewManual()

	_ = c.NewTimer(time.Second, timerID)

	// Klunky. More sign this is wrong. running "go" to register the
	// trigger doesn't make sense here.
	for {
		c.Lock()
		_, registered := c.triggers[timerID]
		c.Unlock()

		if registered {
			break
		}
		time.Sleep(time.Microsecond)
	}

	c.Unregister(timerID)
	timer2 := c.NewTimer(time.Second, timerID)
	go c.Trigger(timerID)
	<-timer2.Channel()

}

func TestContextCancel(t *testing.T) {
	mt := NewManual()

	ctx, cancelF := mt.WithTimeout(context.Background(), time.Minute, contextID)

	select {
	case <-ctx.Done():
		t.Fatal("context done channel already closed")
	default:
	}

	if ctx.Err() != nil {
		t.Fatal("context error is not nil")
	}

	cancelF()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context done channel open when it should be closed")
	}

	if ctx.Err() == nil || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("context error is not context.Canceled")
	}
}

func TestContextTrigger(t *testing.T) {
	mt := NewManual()

	ctx, cancelF := mt.WithTimeout(context.Background(), time.Minute, contextID)

	select {
	case <-ctx.Done():
		t.Fatal("context done channel already closed")
	default:
	}

	if ctx.Err() != nil {
		t.Fatal("context error is not nil")
	}

	mt.Trigger(contextID)

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context done channel open when it should be closed")
	}

	if ctx.Err() == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("context error is not context.DeadlineExceeded")
	}

	cancelF()

	if ctx.Err() == nil || !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatal("context error is not context.DeadlineExceeded")
	}
}

func TestContextNestedCancel(t *testing.T) {
	mt := NewManual()

	parent, parentCancel := context.WithCancel(context.Background())
	child, childCancel := mt.WithTimeout(parent, time.Minute, contextID)

	select {
	case <-child.Done():
		t.Fatal("context done channel already closed")
	default:
	}

	if child.Err() != nil {
		t.Fatal("context error is not nil")
	}

	parentCancel()

	select {
	case <-child.Done():
	case <-time.After(50 * time.Microsecond):
		t.Fatal("context done channel open when it should be closed")
	}

	if child.Err() == nil || !errors.Is(child.Err(), context.Canceled) {
		t.Fatal("context error is not context.Canceled")
	}

	childCancel()
}

func TestContextNestedTimeout(t *testing.T) {
	mt := NewManual()

	parent, parentCancel := mt.WithTimeout(context.Background(), time.Minute, contextID)
	child, childCancel := mt.WithTimeout(parent, time.Minute, childContextID)

	select {
	case <-child.Done():
		t.Fatal("context done channel already closed")
	default:
	}

	if child.Err() != nil {
		t.Fatal("context error is not nil")
	}

	mt.Trigger(contextID)

	select {
	case <-child.Done():
	case <-time.After(50 * time.Microsecond):
		t.Fatal("context done channel open when it should be closed")
	}

	if child.Err() == nil || !errors.Is(child.Err(), context.DeadlineExceeded) {
		t.Fatal("context error is not context.DeadlineExceeded")
	}

	childCancel()

	if child.Err() == nil || !errors.Is(child.Err(), context.DeadlineExceeded) {
		t.Fatal("context error is not context.DeadlineExceeded")
	}

	parentCancel()

	if child.Err() == nil || !errors.Is(child.Err(), context.DeadlineExceeded) {
		t.Fatal("context error is not context.DeadlineExceeded")
	}
}
