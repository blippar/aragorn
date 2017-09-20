package notifier

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
)

var _ = Notifier(&Printer{})

// Printer is a reporter that stacks errors for later use.
// Stacked errors are printed on each report and removed from the stack.
type Printer struct {
	name     string
	start    time.Time
	failures []error
	err      error
}

// NewPrinter returns a new Printer.
func NewPrinter() *Printer {
	return &Printer{}
}

// BeforeTest implements the Notifier interface.
func (r *Printer) BeforeTest(name string) {
	r.name = name
	r.start = time.Now()
}

// Report implements the Notifier interface.
func (r *Printer) Report(err error) {
	r.failures = append(r.failures, err)
}

// Reportf implements the notifier interface.
func (r *Printer) Reportf(format string, args ...interface{}) {
	r.Report(fmt.Errorf(format, args...))
}

// TestError implements the Notifier interface.
func (r *Printer) TestError(err error) {
	r.err = err
}

// AfterTest implements the Notifier interface.
func (r *Printer) AfterTest() {
	if len(r.failures) > 0 || r.err != nil {
		if r.err != nil {
			log.Info("could not run test", zap.String("name", r.name), zap.Duration("took", time.Since(r.start)), zap.Error(r.err))
		} else {
			log.Info("test failed", zap.String("name", r.name), zap.Duration("took", time.Since(r.start)), zap.Errors("failures", r.failures))
		}
	} else {
		log.Info("test passed", zap.String("name", r.name), zap.Duration("took", time.Since(r.start)))
	}
	r.failures = nil
	r.err = nil
}

// SuiteDone implements the Notifier interface.
func (r *Printer) SuiteDone() {
}
