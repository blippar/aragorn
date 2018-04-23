---
id: intro
title: Introduction
---

Aragorn is a regression testing tool that can run a set of tests periodically.
It acts as a server that can execute or schedule test suites.
(a JSON file with the `.suite.json` extension, see example below for more information)

When a test suite is run, each of its tests is executed. For each test, there
are 3 possible outcomes:

* the test's request is successful and its response is expected: nothing happens
* the test's request is successful but its response is unexpected: failures are reported
* the test's request can not be performed : the error is reported

> A request error can be an HTTP error while trying to make the request for
> example. Those errors will be retried 3 times, with a 30s pause in between
> each try. A failure is not retried.

Where the failures/errors are reported and how they are reported depends on the
notifier configured. For now the only one implemented is the SlackNotifier. It
is set at the initialization of a test suite. During a test suite execution, it
stacks every potential failure and/or error. At the end of the suite execution,
it will send a notification describing what happened if there was any failure or error.