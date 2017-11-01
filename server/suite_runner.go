package server

import (
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/testsuite"
)

type suiteRunner struct {
	name     string
	suite    testsuite.Suite
	notifier notifier.Notifier
}

func newSuiteRunner(name string, s testsuite.Suite, n notifier.Notifier) *suiteRunner {
	return &suiteRunner{
		name:     name,
		suite:    s,
		notifier: n,
	}
}

func (sr *suiteRunner) Run() {
	report := notifier.NewReport(sr.name)
	sr.suite.Run(report)
	report.Done()
	sr.notifier.Notify(report)
}
