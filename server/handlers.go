package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/notifier/slack"
	"github.com/blippar/aragorn/testsuite"
)

type Config struct {
	// Name used to identify this test suite.
	Name string

	RunEvery string // parsable by time.ParseDuration
	RunCron  string // cron string

	Slack struct {
		Webhook  string
		Username string
		Channel  string
	}

	// Type of the test suite, can be HTTP or GRPC.
	Type string

	// Description of the test suite, depends on Type.
	Suite json.RawMessage
}

func newConfigFromReader(r io.Reader) (*Config, error) {
	cfg := &Config{}
	if err := json.NewDecoder(r).Decode(cfg); err != nil {
		return nil, fmt.Errorf("could not decode test suite file: %v", err)
	}
	return cfg, nil
}

func newConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open test suite file: %v", err)
	}
	defer f.Close()
	return newConfigFromReader(f)
}

// newTestSuiteFromDisk unmarshals the test suite located at path
// initializes it. If once is set to true, the test suite is executed once.
// Otherwise, the test suite is added to the scheduler.
func (s *Server) newTestSuiteFromDisk(path string, once bool) error {
	cfg, err := newConfigFromFile(path)
	if err != nil {
		return err
	}
	newSuite, err := testsuite.Get(cfg.Type)
	if err != nil {
		return err
	}
	n := notifier.NewPrinter()
	if cfg.Slack.Webhook != "" && cfg.Slack.Username != "" && cfg.Slack.Channel != "" {
		n = notifier.Multi(n, slack.New(cfg.Slack.Webhook, cfg.Slack.Username, cfg.Slack.Channel))
	}
	dir := filepath.Dir(path)
	suite, err := newSuite(dir, cfg.Suite)
	if err != nil {
		return err
	}
	sr := newSuiteRunner(cfg.Name, suite, n)
	if once {
		sr.Run()
	} else if cfg.RunCron != "" {
		s.sch.AddCron(path, sr, cfg.RunCron)
	} else if cfg.RunEvery != "" {
		d, err := time.ParseDuration(cfg.RunEvery)
		if err != nil {
			return fmt.Errorf("could not parse duration in test suite file: %v", err)
		}
		s.sch.Add(path, sr, d)
	} else {
		return errors.New("no scheduling rule set in test suite file: please set runCron or runEvery")
	}
	return nil
}

func (s *Server) removeTestSuite(path string) error {
	return s.sch.Remove(path)
}
