package notifier

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
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
}

func NewReport(s Suite) *Report {
	return &Report{
		Suite: s,
		Start: time.Now(),
	}
}

func (r *Report) AddTest(t testsuite.Test) testsuite.TestReport {
	log.Debug("running test", zap.String("name", t.Name()), zap.String("description", t.Description()))
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

func (tr *TestReport) Done() bool {
	tr.Duration = time.Since(tr.Start)
	return len(tr.Errs) > 0
}
