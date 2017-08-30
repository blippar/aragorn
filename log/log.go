package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var l *zap.Logger

func Init(debug bool) error {
	var err error
	if debug {
		l, err = zap.NewDevelopment()
	} else {
		l, err = zap.NewProduction()
	}
	if err != nil {
		return err
	}
	return nil
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
