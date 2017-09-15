package notifier

import (
	"fmt"
	"time"
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

// BeforeTest implements the Reporter interface.
func (r *Printer) BeforeTest(name string) {
	r.name = name
	r.start = time.Now()
}

// Report implements the Reporter interface.
func (r *Printer) Report(err error) {
	r.failures = append(r.failures, err)
}

// Reportf implements the notifier interface.
func (r *Printer) Reportf(format string, args ...interface{}) {
	r.Report(fmt.Errorf(format, args...))
}

// TestError implements the Reporter interface.
func (r *Printer) TestError(err error) {
	r.err = err
}

// AfterTest implements the Reporter interface.
func (r *Printer) AfterTest() {
	if len(r.failures) > 0 || r.err != nil {
		if r.err != nil {
			fmt.Println(r.err.Error())
		} else {
			for _, err := range r.failures {
				fmt.Println(err.Error())
			}
		}
		fmt.Printf("[%s] FAILED (%v)\n", r.name, time.Since(r.start))
	} else {
		fmt.Printf("[%s] PASS (%v)\n", r.name, time.Since(r.start))
	}
	r.failures = nil
	r.err = nil
}

// SuiteDone implements the Reporter interface.
func (r *Printer) SuiteDone() {
}
