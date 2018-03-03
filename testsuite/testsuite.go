package testsuite

import "context"

type Suite interface {
	Tests() []Test
}

type Test interface {
	Name() string
	Description() string
	Run(context.Context, Logger)
}

type Logger interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}
