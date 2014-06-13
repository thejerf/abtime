/*

Package abtime provides abstracted time functionality that can be swapped
between testing and real without changing application code.

In any code that seriously uses time, such as billing or scheduling code,
best software engineering practices are that you should not directly
access the operating system time.

Other people's discussions: http://blog.plover.com/prog/Moonpig.html#testing-sucks
http://stackoverflow.com/questions/5622194/time-dependent-unit-tests/5622222#5622222
http://jim-mcbeath.blogspot.com/2009/02/unit-testing-with-dates-and-times.html

This module wraps the parts of the time module of Go that do access
the OS time directly, as it stands at Go 1.2 and 1.3 (which are both the
same.) Unfortunately, due to the fact I can not re-export types, you'll 
still need to import "time" for its types.

This module declares an interface for time functions AbstractTime,
provides an implementation that simply backs to the "real" time functions
"RealTime", and provides an implementation that allows you to fully control
the time "ManualTime", including setting "now", and requiring you to
manually trigger all time-based events, such as alerts and alarms.

Since there is no way to distinguish between different calls to the
standard time functions, each of the methods in the AbstractTime interface
adds an "id". The RealTime implementation simply ignores them. The
ManualTime implementations uses these to trigger specific time events.
Be sure to see the example for usage of the ManualTime implementation.

Avoid re-using IDs on the Tick functions; it becomes confusing which
.Trigger is affecting which Tick.

Be sure to see the Example below.

Quality: At the moment I would call this alpha code. Go lint clean, go vet
clean, 100% coverage in the tests. You and I both know that doesn't prove
this is bug-free, but at least it shows I care.

*/
package abtime
