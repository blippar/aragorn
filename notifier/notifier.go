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

type Report struct {
	Name     string
	Start    time.Time
	Duration time.Duration
	Tests    []*TestReport
	failfast bool
}

type TestReport struct {
	Name     string
	Start    time.Time
	Duration time.Duration
	Errs     []error
	failfast bool
}

func NewReport(name string, failfast bool) *Report {
	return &Report{
		Name:     name,
		Start:    time.Now(),
		failfast: failfast,
	}
}

func (r *Report) AddTest(name string) testsuite.TestReport {
	tr := newTestReport(name, r.failfast)
	r.Tests = append(r.Tests, tr)
	return tr
}

func (r *Report) Done() {
	r.Duration = time.Since(r.Start)
}

func newTestReport(name string, failfast bool) *TestReport {
	return &TestReport{
		Name:     name,
		Start:    time.Now(),
		failfast: failfast,
	}
}

func (tr *TestReport) Error(args ...interface{}) {
	tr.Errs = append(tr.Errs, errors.New(fmt.Sprint(args...)))
}

func (tr *TestReport) Errorf(format string, args ...interface{}) {
	tr.Errs = append(tr.Errs, fmt.Errorf(format, args...))
}

func (tr *TestReport) Done() bool {
	tr.Duration = time.Since(tr.Start)
	return tr.failfast && len(tr.Errs) > 0
}
