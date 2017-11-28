package server

import (
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/testsuite"
)

type SuiteRunner struct {
	name     string
	suite    testsuite.Suite
	notifier notifier.Notifier
}

func NewSuiteRunner(name string, s testsuite.Suite, n notifier.Notifier) *SuiteRunner {
	return &SuiteRunner{
		name:     name,
		suite:    s,
		notifier: n,
	}
}

func (sr *SuiteRunner) Run() {
	report := notifier.NewReport(sr.name)
	sr.suite.Run(report)
	report.Done()
	sr.notifier.Notify(report)
}
