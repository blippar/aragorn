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
	for _, tr := range r.tests {
		fields := []zapcore.Field{
			zap.String("test_sute", r.name),
			zap.String("name", tr.name),
			zap.Time("started_at", tr.start),
			zap.Duration("duration", tr.duration),
		}
		msg := "test passed"
		if len(tr.errs) > 0 {
			msg = "test failed"
			fields = append(fields, zap.Errors("errs", tr.errs))
		}
		log.Info(msg, fields...)
	}
	log.Info("test suite done",
		zap.String("name", r.name),
		zap.Time("started_at", r.start),
		zap.Duration("duration", r.duration),
		zap.Int("nb_tests", len(r.tests)),
	)
}
