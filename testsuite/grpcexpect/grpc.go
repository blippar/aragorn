package grpcexpect

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
	"github.com/blippar/aragorn/testsuite"
)

// Suite describes a GRPC test suite.
type Suite struct {
	notifier notifier.Notifier
}

// Init initializes a gRPC test suite.
func (s *Suite) Init(n notifier.Notifier) error {
	s.notifier = n
	return nil
}

// Run runs all the tests in the suite.
func (s *Suite) Run() {
	fmt.Fprintln(os.Stderr, "not implemented")
}

func init() {
	f := testsuite.RegisterFunc(func(cfg *testsuite.Config) (scheduler.Job, error) {
		var suite Suite
		if err := json.Unmarshal(cfg.Suite, &suite); err != nil {
			return nil, fmt.Errorf("could not unmarshal gRPC test suite: %v", err)
		}

		if err := suite.Init(
			notifier.NewPrinter(),
		); err != nil {
			return nil, fmt.Errorf("could not init gRPC test suite: %v", err)
		}

		return &suite, nil
	})

	testsuite.Register("GRPC", f)
}
