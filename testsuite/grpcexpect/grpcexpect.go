package grpcexpect

import (
	"context"
	"errors"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

var _ testsuite.Suite = (*Suite)(nil)

// Suite describes a GRPC test suite.
type Suite struct{}

// New returns a Suite.
func New() (*Suite, error) {
	return nil, errors.New("Not implemented")
}

// Run runs all the tests in the suite.
func (s *Suite) Run(ctx context.Context, r testsuite.Report) {
	log.Error("not implemented")
}

func (s *Suite) Tests() []testsuite.Test {
	return nil
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.TestSuitePlugin,
		ID:   "GRPC",
		InitFn: func(ctx *plugin.InitContext) (interface{}, error) {
			return New()
		},
	})
}
