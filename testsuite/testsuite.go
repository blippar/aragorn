package testsuite

import "context"

type RPCInfo struct {
	FailFast bool
}

type rpcInfoContextKey struct{}

func NewContextWithRPCInfo(ctx context.Context, failfast bool) context.Context {
	return context.WithValue(ctx, rpcInfoContextKey{}, &RPCInfo{FailFast: failfast})
}

func RPCInfoFromContext(ctx context.Context) (s *RPCInfo, ok bool) {
	s, ok = ctx.Value(rpcInfoContextKey{}).(*RPCInfo)
	return
}

type Suite interface {
	Run(context.Context, Report)
	Tests() []Test
}

type Test interface {
	Name() string
	Description() string
}

type Report interface {
	AddTest(t Test) TestReport
}

type TestReport interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Done() bool
}
