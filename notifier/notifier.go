package notifier

import (
	"errors"
	"fmt"
	"time"

	"github.com/blippar/aragorn/testsuite"
)

// Notifier notifies report.
type Notifier interface {
	Notify(r *Report)
}

type Suite interface {
	Name() string
	Type() string
	FailFast() bool
	Tests() []testsuite.Test
}

type Report struct {
	Suite       Suite
	Start       time.Time
	Duration    time.Duration
	TestReports []*TestReport
	NbFailed    int
}

func NewReport(s Suite) *Report {
	return &Report{
		Suite: s,
		Start: time.Now(),
	}
}

func (r *Report) NewTestReport(t testsuite.Test) *TestReport {
	tr := &TestReport{
		Test:  t,
		Start: time.Now(),
	}
	r.TestReports = append(r.TestReports, tr)
	return tr
}

func (r *Report) Done() {
	r.Duration = time.Since(r.Start)
}

type TestReport struct {
	Test     testsuite.Test
	Start    time.Time
	Duration time.Duration
	Errs     []error
}

func (tr *TestReport) Error(args ...interface{}) {
	tr.Errs = append(tr.Errs, errors.New(fmt.Sprint(args...)))
}

func (tr *TestReport) Errorf(format string, args ...interface{}) {
	tr.Errs = append(tr.Errs, fmt.Errorf(format, args...))
}

func (tr *TestReport) Reset() {
	tr.Errs = nil
}

func (tr *TestReport) Done() {
	tr.Duration = time.Since(tr.Start)
}
