# abtime

[![Build Status](https://travis-ci.org/thejerf/abtime.png?branch=master)](https://travis-ci.org/thejerf/abtime)

    go get github.com/thejerf/abtime

A library for abstracting away from the literal Go time library, for testing and time control.

In any code that seriously uses time, such as billing or scheduling code,
best software engineering practices are that you should not directly
access the operating system time. This module provides you with code to
implement that principle in Go.

See some discussions:

* [blog.plover.com's Moonpig discussion](http://blog.plover.com/prog/Moonpig.html#testing-sucks)
* [Jon Skeet's discussion on Stack Overflow](http://stackoverflow.com/questions/5622194/time-dependent-unit-tests/5622222#5622222)
* [Jim McBeath's post](http://jim-mcbeath.blogspot.com/2009/02/unit-testing-with-dates-and-times.html)

This module is fully covered with
[godoc](http://godoc.org/github.com/thejerf/abtime), including examples,
usage, and everything else you might expect from a README.md on GitHub.
(DRY.)

This is at least beta quality now, and if you're thinking about using it,
bear in mind that what it really offers is a structure, not a huge pile of
code. Should the code turn out incorrect, it would be easy for you to fix.
