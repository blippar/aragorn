package testsuite

type Suite interface {
	Run(report Report)
}

type Report interface {
	AddTest(name string) TestReport
}

type TestReport interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Done() bool
}
