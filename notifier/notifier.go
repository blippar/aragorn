package notifier

import "github.com/blippar/aragorn/reporter"

// Notifier is a set of methods called at specific steps duting a tests suite run.
type Notifier interface {
	reporter.Reporter

	// BeforeTest is called before each test run with the test name as a parameter.
	BeforeTest(name string)

	// TestError is called when a test could not be run.
	TestError(err error)

	// Report is called after each test run.
	AfterTest()

	// Done is called when the complete tests suite is done.
	SuiteDone()
}
