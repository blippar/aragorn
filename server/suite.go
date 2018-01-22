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

var errNoSchedulingRule = errors.New("no scheduling rule set in test suite file: please set runCron or runEvery")

type Suite struct {
	name     string
	suite    testsuite.Suite
	notifier notifier.Notifier

	typ      string
	runCron  string
	runEvery time.Duration
	failfast bool
}

type SuiteConfig struct {
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

func (s *Server) NewSuiteFromReader(dir string, r io.Reader) (*Suite, error) {
	var cfg SuiteConfig
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode test suite file: %v", err)
	}
	var runEvery time.Duration
	if cfg.RunCron == "" && cfg.RunEvery == "" {
		return nil, errNoSchedulingRule
	} else if cfg.RunEvery != "" {
		d, err := time.ParseDuration(cfg.RunEvery)
		if err != nil {
			return nil, fmt.Errorf("could not parse duration in test suite file: %v", err)
		}
		runEvery = d
	}
	newSuite, err := testsuite.Get(cfg.Type)
	if err != nil {
		return nil, err
	}
	n := notifier.NewPrinter()
	if cfg.Slack.Webhook != "" && cfg.Slack.Username != "" && cfg.Slack.Channel != "" {
		n = notifier.Multi(n, slack.New(cfg.Slack.Webhook, cfg.Slack.Username, cfg.Slack.Channel))
	}
	suite, err := newSuite(dir, cfg.Suite)
	if err != nil {
		return nil, err
	}
	return &Suite{
		name:     cfg.Name,
		suite:    suite,
		notifier: n,
		typ:      cfg.Type,
		runCron:  cfg.RunCron,
		runEvery: runEvery,
		failfast: s.failfast,
	}, nil
}

func (s *Server) NewSuiteFromFile(path string) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open test suite file: %v", err)
	}
	defer f.Close()
	dir := filepath.Dir(path)
	return s.NewSuiteFromReader(dir, f)
}

func (s *Suite) Run() {
	report := notifier.NewReport(s.name)
	s.suite.Run(report, s.failfast)
	report.Done()
	s.notifier.Notify(report)
}
