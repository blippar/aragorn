package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var l *zap.Logger

func Init(debug bool) (err error) {
	var (
		cfg  zap.Config
		opts []zap.Option
	)

	if debug {
		cfg = zap.NewDevelopmentConfig()
		opts = []zap.Option{zap.AddCallerSkip(1)}
	} else {
		cfg = zap.NewProductionConfig()
	}

	cfg.DisableStacktrace = true

	l, err = cfg.Build(opts...)
	return
}

func L() *zap.Logger {
	return l
}

func Debug(msg string, fields ...zapcore.Field) {
	l.Debug(msg, fields...)
}

func Info(msg string, fields ...zapcore.Field) {
	l.Info(msg, fields...)
}

func Warn(msg string, fields ...zapcore.Field) {
	l.Warn(msg, fields...)
}

func Error(msg string, fields ...zapcore.Field) {
	l.Error(msg, fields...)
}

func DPanic(msg string, fields ...zapcore.Field) {
	l.DPanic(msg, fields...)
}

func Panic(msg string, fields ...zapcore.Field) {
	l.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zapcore.Field) {
	l.Fatal(msg, fields...)
}
