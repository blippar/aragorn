package grpcexpect

import (
	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/testsuite"
)

// Suite describes a GRPC test suite.
type Suite struct{}

// New returns a Suite.
func New() (testsuite.Suite, error) {
	return &Suite{}, nil
}

func newFromJSONData(data []byte) (testsuite.Suite, error) {
	return New()
}

// Run runs all the tests in the suite.
func (s *Suite) Run(r testsuite.Report) {
	log.Error("not implemented")
}

func init() {
	testsuite.Register("GRPC", newFromJSONData)
}