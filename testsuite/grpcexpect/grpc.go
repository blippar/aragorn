package grpcexpect

import (
	"errors"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/testsuite"
)

// Suite describes a GRPC test suite.
type Suite struct{}

// New returns a Suite.
func New() (*Suite, error) {
	return nil, errors.New("Not implemented")
}

// NewSuiteFromJSON returns a `testsuite.Suite` using the cfg to construct the config.
func NewSuiteFromJSON(path string, data []byte) (testsuite.Suite, error) {
	return New()
}

// Run runs all the tests in the suite.
func (s *Suite) Run(r testsuite.Report) {
	log.Error("not implemented")
}

func init() {
	testsuite.Register("GRPC", NewSuiteFromJSON)
}
