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
	name     string
	start    time.Time
	duration time.Duration
	tests    []*TestReport
}

type TestReport struct {
	name     string
	start    time.Time
	duration time.Duration
	errs     []error
}

func NewReport(name string) *Report {
	return &Report{
		name:  name,
		start: time.Now(),
	}
}

func (r *Report) AddTest(name string) testsuite.TestReport {
	tr := newTestReport(name)
	r.tests = append(r.tests, tr)
	return tr
}

func (r *Report) Done() {
	r.duration = time.Since(r.start)
}

func newTestReport(name string) *TestReport {
	return &TestReport{
		name:  name,
		start: time.Now(),
	}
}

func (tr *TestReport) Error(args ...interface{}) {
	tr.errs = append(tr.errs, errors.New(fmt.Sprint(args...)))
}

func (tr *TestReport) Errorf(format string, args ...interface{}) {
	tr.errs = append(tr.errs, fmt.Errorf(format, args...))
}

func (tr *TestReport) Done() {
	tr.duration = time.Since(tr.start)
}
