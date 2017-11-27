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
}

type TestReport struct {
	Name     string
	Start    time.Time
	Duration time.Duration
	Errs     []error
}

func NewReport(name string) *Report {
	return &Report{
		Name:  name,
		Start: time.Now(),
	}
}

func (r *Report) AddTest(name string) testsuite.TestReport {
	tr := newTestReport(name)
	r.Tests = append(r.Tests, tr)
	return tr
}

func (r *Report) Done() {
	r.Duration = time.Since(r.Start)
}

func newTestReport(name string) *TestReport {
	return &TestReport{
		Name:  name,
		Start: time.Now(),
	}
}

func (tr *TestReport) Error(args ...interface{}) {
	tr.Errs = append(tr.Errs, errors.New(fmt.Sprint(args...)))
}

func (tr *TestReport) Errorf(format string, args ...interface{}) {
	tr.Errs = append(tr.Errs, fmt.Errorf(format, args...))
}

func (tr *TestReport) Done() {
	tr.Duration = time.Since(tr.Start)
}
