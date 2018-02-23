package notifier

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/blippar/aragorn/log"
)

type printer struct{}

// NewLogNotifier returns a new log printer.
func NewLogNotifier() Notifier {
	return &printer{}
}

func (*printer) Notify(r *Report) {
	nbFailed := 0
	for _, tr := range r.TestReports {
		fields := []zapcore.Field{
			zap.String("suite", r.Suite.Name()),
			zap.String("name", tr.Test.Name()),
			zap.Time("started_at", tr.Start),
			zap.Duration("duration", tr.Duration),
		}
		if len(tr.Errs) > 0 {
			fields = append(fields, zap.Errors("errs", tr.Errs))
			log.Warn("test failed", fields...)
			nbFailed++
		} else {
			log.Info("test passed", fields...)
		}
	}
	log.Info("test suite done",
		zap.String("suite", r.Suite.Name()),
		zap.Bool("failfast", r.Suite.FailFast()),
		zap.Int("nb_tests", len(r.Suite.Tests())),
		zap.Int("nb_test_reports", len(r.TestReports)),
		zap.Int("nb_failed", nbFailed),
		zap.Time("started_at", r.Start),
		zap.Duration("duration", r.Duration),
	)
}
