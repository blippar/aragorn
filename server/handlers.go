package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/blippar/aragorn/grpcexpect"
	"github.com/blippar/aragorn/httpexpect"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
)

const (
	httpType = "HTTP"
	grpcType = "GRPC"
)

type jsonTestSuite struct {
	testSuiteConfig

	// Description of the tests suite, depends on Type.
	Suite json.RawMessage
}

type testSuiteConfig struct {
	// Name used to identify this tests suite.
	Name string

	RunEvery string // parsable by time.ParseDuration
	RunCron  string // cron string

	SlackWebhook string

	// Type of the tests suite, can be HTTP or GRPC.
	Type string
}

// newTestSuiteFromDisk unmarshals the tests suite located at path
// initializes it. If schedule is set to true, it also adds it to the scheduler.
// Otherwise, the tests suite is executed only once without delay.
func (s *Server) newTestSuiteFromDisk(path string, schedule bool) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read tests suite file: %v", err)
	}

	var ts jsonTestSuite
	if err = json.Unmarshal(d, &ts); err != nil {
		return fmt.Errorf("could not unmarshal tests suite file: %v", err)
	}

	var job scheduler.Job
	switch ts.Type {
	case httpType:
		job, err = s.newHTTPTestSuite(&ts)
	case grpcType:
		job, err = s.newGRPCTestSuite(&ts)
	default:
		return fmt.Errorf("unsupported tests suite (%s) type: %q", path, ts.Type)
	}
	if err != nil {
		return err
	}

	if schedule {
		if ts.RunCron != "" {
			s.sch.AddCron(path, job, ts.RunCron)
		} else if ts.RunEvery != "" {
			d, err := time.ParseDuration(ts.RunEvery)
			if err != nil {
				return fmt.Errorf("could not parse duration in tests suite file: %v", err)
			}
			s.sch.Add(path, job, d)
		}
	} else {
		job.Run()
	}

	return nil
}

func (s *Server) newHTTPTestSuite(ts *jsonTestSuite) (scheduler.Job, error) {
	var suite httpexpect.Suite
	if err := json.Unmarshal(ts.Suite, &suite); err != nil {
		return nil, fmt.Errorf("could not unmarshal HTTP tests suite: %v", err)
	}

	cfg := ts.testSuiteConfig
	if err := suite.Init(
		cfg.Name,
		notifier.NewSlackNotifier(cfg.SlackWebhook, cfg.Name),
		httpexpect.WithHTTPClient(&http.Client{Timeout: 20 * time.Second}),
		httpexpect.WithRetryPolicy(3, 15*time.Second),
	); err != nil {
		return nil, fmt.Errorf("could not init HTTP tests suite: %v", err)
	}
	return &suite, nil
}

func (s *Server) newGRPCTestSuite(ts *jsonTestSuite) (scheduler.Job, error) {
	var suite grpcexpect.Suite
	if err := json.Unmarshal(ts.Suite, &suite); err != nil {
		return nil, fmt.Errorf("could not unmarshal gRPC tests suite: %v", err)
	}
	if err := suite.Init(
		notifier.NewPrinter(),
	); err != nil {
		return nil, fmt.Errorf("could not init gRPC tests suite: %v", err)
	}
	return &suite, nil
}

func (s *Server) removeTestSuite(path string) {
	s.sch.Remove(path)
}
