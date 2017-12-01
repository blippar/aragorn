package notifier

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/blippar/aragorn/log"
)

// Printer is a reporter that stacks errors for later use.
// Stacked errors are printed on each report and removed from the stack.
type printer struct{}

// NewPrinter returns a new Printer.
func NewPrinter() Notifier {
	return &printer{}
}

func (*printer) Notify(r *Report) {
	for _, tr := range r.Tests {
		fields := []zapcore.Field{
			zap.String("suite", r.Name),
			zap.String("name", tr.Name),
			zap.Time("started_at", tr.Start),
			zap.Duration("duration", tr.Duration),
		}
		msg := "test passed"
		if len(tr.Errs) > 0 {
			msg = "test failed"
			fields = append(fields, zap.Errors("errs", tr.Errs))
		}
		log.Info(msg, fields...)
	}
	log.Info("test suite done",
		zap.String("suite", r.Name),
		zap.Time("started_at", r.Start),
		zap.Duration("duration", r.Duration),
		zap.Int("nb_tests", len(r.Tests)),
	)
}
