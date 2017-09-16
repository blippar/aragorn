package grpcexpect

import (
	"fmt"
	"os"

	"github.com/blippar/aragorn/notifier"
)

// Suite describes a GRPC tests suite.
type Suite struct {
	notifier notifier.Notifier
}

// Init initializes a gRPC tests suite.
func (s *Suite) Init(n notifier.Notifier) error {
	s.notifier = n
	return nil
}

// Run runs all the tests in the suite.
func (s *Suite) Run() {
	fmt.Fprintln(os.Stderr, "not implemented")
}
