package abtime

// In the docs, I say that "we can't distinguish between calls to After" or sleep.
// A clever programmer may decide that we could, if we require them all
// to use slightly different times; we could then key on times. However,
// that invites excessive binding of values between the concrete code and
// test suite. Plus that's just a weird binding that invites problems.

import (
	"sync"
	"time"
)

// The ManualTime object implements a time object you directly control.
//
// This allows you to manipulate "now", and control when events occur.
type ManualTime struct {
	now      time.Time
	nows     []time.Time
	triggers map[int]*triggerInfo

	sync.Mutex
}

type triggerInfo struct {
	count    uint
	triggers []trigger
}

type trigger interface {
	// Note this is always called while the lock for *ManualTime is
	// held.
	trigger(mt *ManualTime) bool // if true, delete the token; if false, keep it.
}

func (mt *ManualTime) register(id int, trig trigger) {
	mt.Lock()
	defer mt.Unlock()

	currentTriggerInfo, present := mt.triggers[id]
	if !present {
		mt.triggers[id] = &triggerInfo{0, []trigger{trig}}
		return
	}

	if currentTriggerInfo.count == 0 {
		currentTriggerInfo.triggers = append(currentTriggerInfo.triggers, trig)
		return
	}

	for currentTriggerInfo.count > 0 {
		currentTriggerInfo.count--

		discard := trig.trigger(mt)

		if discard {
			break
		}

		if currentTriggerInfo.count == 0 {
			currentTriggerInfo.triggers = append(currentTriggerInfo.triggers, trig)
			break
		}
	}
}

// NewManual returns a new ManualTime object, with the Now populated
// from the time.Now().
func NewManual() *ManualTime {
	return &ManualTime{now: time.Now(), nows: []time.Time{}, triggers: make(map[int]*triggerInfo)}
}

// NewManualAtTime returns a new ManualTime object, with the Now set to the
// time.Time you pass in.
func NewManualAtTime(now time.Time) *ManualTime {
	return &ManualTime{now: now, nows: []time.Time{}, triggers: make(map[int]*triggerInfo)}
}

// Trigger takes the given ids for time events, and causes them to "occur":
// triggering messages on channels, ending sleeps, etc.
//
// Note this is the ONLY way to "trigger" such events. While this package
// allows you to manipulate "Now" in a couple of different ways, advancing
// "now" past a Trigger's set time will NOT trigger it. First, this keeps
// it simple to understand when things are triggered, and second, reality
// isn't so deterministic anyhow....
func (mt *ManualTime) Trigger(ids ...int) {
	mt.Lock()
	defer mt.Unlock()

	for _, id := range ids {
		triggers, hasTriggers := mt.triggers[id]
		if !hasTriggers {
			mt.triggers[id] = &triggerInfo{1, []trigger{}}
			continue
		}
		if len(triggers.triggers) == 0 {
			triggers.count++
			continue
		}

		t := triggers.triggers[0]
		discard := t.trigger(mt)

		if discard {
			triggers.triggers = triggers.triggers[1:]
		}
	}
}

// Unregister will unregister a particular ID from the system. Normally the
// first one sticks, which means if you've got code that creates multiple
// timers in a loop or in multiple function calls, only the first one will
// work.
//
// NOTE: This method indicates a design flaw in abtime. It is not yet clear
// to me how to fix it in any reasonable way.
func (mt *ManualTime) Unregister(ids ...int) {
	for _, id := range ids {
		delete(mt.triggers, id)
	}
}

// UnregisterAll will unregister all current IDs from the manual time,
// returning you to a fresh view of the created channels and timers and
// such.
func (mt *ManualTime) UnregisterAll() {
	mt.triggers = map[int]*triggerInfo{}
}

// Now returns the ManualTime's current idea of "Now".
//
// If you have used QueueNow, this will advance to the next queued Now.
func (mt *ManualTime) Now() time.Time {
	mt.Lock()
	defer mt.Unlock()

	if len(mt.nows) > 0 {
		mt.now = mt.nows[0]
		mt.nows = mt.nows[1:]
		return mt.now
	}
	return mt.now
}

// Advance advances the manual time's idea of "now" by the given
// duration.
//
// If there is a queue of "Nows" from QueueNows, note this won't
// affect any of them.
func (mt *ManualTime) Advance(d time.Duration) {
	mt.Lock()
	defer mt.Unlock()

	mt.now = mt.now.Add(d)
}

// QueueNows allows you to set a number of times to be retrieved by
// successive calls to "Now". Once the queue is consumed by calls to Now(),
// the last time in the queue "sticks" as the new Now.
//
// This is useful if you have code that is timing how long something took
// by successive calls to .Now, with no other place for the test code to
// intercede.
//
// If multiple threads are accessing the Manual, it is of course
// non-deterministic who gets what time. However this could still be
// useful.
func (mt *ManualTime) QueueNows(times ...time.Time) {
	mt.Lock()
	defer mt.Unlock()

	mt.nows = append(mt.nows, times...)
}

type afterTrigger struct {
	mt *ManualTime
	d  time.Duration
	ch chan time.Time
}

func (afterT afterTrigger) trigger(mt *ManualTime) bool {
	afterT.ch <- afterT.mt.now.Add(afterT.d)
	return true
}

// After wraps time.After, and waits for the target id.
func (mt *ManualTime) After(d time.Duration, id int) <-chan time.Time {
	timeChan := make(chan time.Time)
	trigger := afterTrigger{mt, d, timeChan}
	go mt.register(id, trigger)
	return timeChan
}

type sleepTrigger struct {
	c chan struct{}
}

func (st sleepTrigger) trigger(mt *ManualTime) bool {
	st.c <- struct{}{}
	return true
}

// Sleep halts execution until you release it via Trigger.
func (mt *ManualTime) Sleep(d time.Duration, id int) {
	ch := make(chan struct{})

	go mt.register(id, sleepTrigger{ch})

	<-ch
}

type tickTrigger struct {
	C       chan time.Time
	now     time.Time
	d       time.Duration
	stopped bool
	sync.Mutex
}

func (tt *tickTrigger) trigger(mt *ManualTime) bool {
	tt.Lock()
	defer tt.Unlock()

	if tt.stopped {
		return true
	}

	tt.now = tt.now.Add(tt.d)
	tt.C <- tt.now
	return false
}

func (tt *tickTrigger) Stop() {
	tt.Lock()
	defer tt.Unlock()

	tt.stopped = true
}

func (tt *tickTrigger) Channel() <-chan time.Time {
	return tt.C
}

// NewTicker wraps time.NewTicker. It takes a snapshot of "now" at the
// point of the TickToken call, and will increment the time it returns
// by the Duration of the tick.
//
// Note that this can cause times to arrive out of order relative to
// each other if you have many of these going at once, if you manually
// trigger the ticks in such a way that they will be out of order.
func (mt *ManualTime) NewTicker(d time.Duration, id int) Ticker {
	ch := make(chan time.Time)
	tt := &tickTrigger{C: ch, now: mt.now, d: d}
	go mt.register(id, tt)
	return tt
}

// Tick allows you to create a ticker. See notes on NewTicker.
func (mt *ManualTime) Tick(d time.Duration, id int) <-chan time.Time {
	return mt.NewTicker(d, id).(*tickTrigger).C
}

type afterFuncTrigger struct {
	f       func()
	stopped bool
	sync.Mutex
}

func (af *afterFuncTrigger) Reset(d time.Duration) bool {
	af.Lock()
	defer af.Unlock()

	return af.stopped
}

func (af *afterFuncTrigger) Stop() bool {
	af.Lock()
	defer af.Unlock()

	ret := !af.stopped
	af.stopped = true
	return ret
}

func (af *afterFuncTrigger) Channel() <-chan time.Time {
	return nil
}

func (af *afterFuncTrigger) trigger(mt *ManualTime) bool {
	af.Lock()
	defer af.Unlock()

	if !af.stopped {
		go af.f()
	}
	af.stopped = true

	return true
}

// AfterFunc fires the function in its own goroutine when the id is
// .Trigger()ed. The resulting Timer object will return nil for its Channel().
func (mt *ManualTime) AfterFunc(d time.Duration, f func(), id int) Timer {
	af := &afterFuncTrigger{f: f, stopped: false}
	go mt.register(id, af)
	return af
}

type timerTrigger struct {
	c          chan time.Time
	initialNow time.Time
	duration   time.Duration
	stopped    bool
	sync.Mutex
}

func (tt *timerTrigger) Reset(d time.Duration) bool {
	tt.Lock()
	defer tt.Unlock()

	tt.duration = d
	return !tt.stopped
}

func (tt *timerTrigger) Stop() bool {
	tt.Lock()
	defer tt.Unlock()

	ret := !tt.stopped
	tt.stopped = true
	return ret
}

func (tt *timerTrigger) Channel() <-chan time.Time {
	return tt.c
}

func (tt *timerTrigger) trigger(mt *ManualTime) bool {
	if tt.stopped {
		return true
	}
	tt.stopped = true
	tt.c <- tt.initialNow.Add(tt.duration)
	return true
}

// NewTimer allows you to create a Ticker, which can be triggered
// via the given id, and also supports the Stop operation *time.Tickers have.
func (mt *ManualTime) NewTimer(d time.Duration, id int) Timer {
	tt := &timerTrigger{c: make(chan time.Time), initialNow: mt.now, duration: d}
	go mt.register(id, tt)
	return tt
}
