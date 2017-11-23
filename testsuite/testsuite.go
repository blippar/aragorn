package testsuite

import (
	"fmt"
)

type Suite interface {
	Run(Report)
}

type Report interface {
	AddTest(name string) TestReport
}

type TestReport interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Done()
}

type RegisterFunc func(data []byte) (Suite, error)

var m = make(map[string]RegisterFunc)

func Get(typ string) (RegisterFunc, error) {
	fn, ok := m[typ]
	if !ok {
		return nil, fmt.Errorf("unsupported test suite type: %q", typ)
	}
	return fn, nil
}

func Register(typ string, fn RegisterFunc) {
	if _, ok := m[typ]; ok {
		panic(fmt.Sprintf("type %q already registered", typ))
	}
	m[typ] = fn
}