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
	"github.com/blippar/aragorn/plugin"
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

	FailFast bool // stop after first test failure

	// Type of the test suite, can be HTTP or GRPC.
	Type string

	// Description of the test suite, depends on Type.
	Suite json.RawMessage
}

func (s *Server) NewSuiteFromReader(r io.Reader) (*Suite, error) {
	var cfg SuiteConfig
	if err := decodeReaderJSON(r, &cfg); err != nil {
		return nil, fmt.Errorf("could not decode suite: %v", jsonDecodeError(r, err))
	}
	var runEvery time.Duration
	if cfg.RunCron == "" && cfg.RunEvery == "" {
		return nil, errNoSchedulingRule
	} else if cfg.RunEvery != "" {
		d, err := time.ParseDuration(cfg.RunEvery)
		if err != nil {
			return nil, fmt.Errorf("could not parse runEvery duration in suite: %v", err)
		}
		runEvery = d
	}
	reg := plugin.Get(plugin.TestSuitePlugin, cfg.Type)
	if reg == nil {
		return nil, fmt.Errorf("unsupported test suite type: %q", cfg.Type)
	}
	root := ""
	if n, ok := r.(namer); ok {
		root = filepath.Dir(n.Name())
	}
	ic := plugin.NewContext(reg, root)
	if err := decodeJSON(cfg.Suite, ic.Config); err != nil {
		return nil, fmt.Errorf("could not decode test suite: %v", err)
	}
	suite, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	n := notifier.NewPrinter()
	if cfg.Slack.Webhook != "" && cfg.Slack.Username != "" && cfg.Slack.Channel != "" {
		n = notifier.Multi(n, slack.New(cfg.Slack.Webhook, cfg.Slack.Username, cfg.Slack.Channel))
	}
	return &Suite{
		name:     cfg.Name,
		suite:    suite.(testsuite.Suite),
		notifier: n,
		typ:      cfg.Type,
		runCron:  cfg.RunCron,
		runEvery: runEvery,
		failfast: s.failfast || cfg.FailFast,
	}, nil
}

func (s *Server) NewSuiteFromFile(path string) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	return s.NewSuiteFromReader(f)
}

func (s *Suite) Run() {
	report := notifier.NewReport(s.name, s.failfast)
	s.suite.Run(report)
	report.Done()
	s.notifier.Notify(report)
}
