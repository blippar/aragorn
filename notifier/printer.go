package notifier

import (
	"fmt"
	"time"
)

var _ = Notifier(&Printer{})

// Printer is a reporter that stacks errors for later use.
// Stacked errors are printed on each report and removed from the stack.
type Printer struct {
	name  string
	start time.Time
	errs  []error
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
	r.errs = append(r.errs, err)
}

// TestError implements the Reporter interface.
func (r *Printer) TestError(err error) {
}

// AfterTest implements the Reporter interface.
func (r *Printer) AfterTest() {
	if len(r.errs) > 0 {
		for _, err := range r.errs {
			fmt.Println(err.Error())
		}
		fmt.Printf("[%s] FAILED (%v)\n", r.name, time.Since(r.start))
	} else {
		fmt.Printf("[%s] PASS (%v)\n", r.name, time.Since(r.start))
	}
	r.errs = nil
}

// SuiteDone implements the Reporter interface.
func (r *Printer) SuiteDone() {
}
