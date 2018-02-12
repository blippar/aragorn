package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

type Suite struct {
	path     string
	name     string
	suite    testsuite.Suite
	notifier notifier.Notifier
	typ      string
	runCron  string
	runEvery time.Duration
	failfast bool
}

type SuiteConfig struct {
	Path     string
	Name     string // identifier for this test suite.
	RunEvery duration
	RunCron  string // cron string
	FailFast bool   // stop after first test failure
}

type fullSuiteConfig struct {
	SuiteConfig
	// Type of the test suite, can be HTTP or GRPC.
	Type string
	// Description of the test suite, depends on Type.
	Suite json.RawMessage
}

func NewSuiteFromReader(r io.Reader) (*Suite, error) {
	var cfg fullSuiteConfig
	if err := decodeReaderJSON(r, &cfg); err != nil {
		return nil, fmt.Errorf("could not decode suite: %v", jsonDecodeError(r, err))
	}
	reg := plugin.Get(plugin.TestSuitePlugin, cfg.Type)
	if reg == nil {
		return nil, fmt.Errorf("unsupported test suite type: %q", cfg.Type)
	}
	path, root := "", ""
	if n, ok := r.(namer); ok {
		path = n.Name()
		root = filepath.Dir(path)
	}
	ic := plugin.NewContext(reg, root)
	if err := decodeJSON(cfg.Suite, ic.Config); err != nil {
		return nil, fmt.Errorf("could not decode test suite: %v", err)
	}
	suite, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	return &Suite{
		path:     path,
		name:     cfg.Name,
		suite:    suite.(testsuite.Suite),
		notifier: logNotifier,
		typ:      cfg.Type,
		runCron:  cfg.RunCron,
		runEvery: time.Duration(cfg.RunEvery),
		failfast: cfg.FailFast,
	}, nil
}

func NewSuiteFromFile(path string) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	return NewSuiteFromReader(f)
}

func NewSuiteFromSuiteConfig(scfg *SuiteConfig) (*Suite, error) {
	s, err := NewSuiteFromFile(scfg.Path)
	if err != nil {
		return nil, err
	}
	if scfg.Name != "" {
		s.name = scfg.Name
	}
	if scfg.RunEvery != 0 {
		s.runEvery = time.Duration(scfg.RunEvery)
	}
	if scfg.RunCron != "" {
		s.runCron = scfg.RunCron
	}
	if scfg.FailFast != false {
		s.failfast = scfg.FailFast
	}
	return s, nil
}

func (s *Suite) Run() {
	report := notifier.NewReport(s.name, s.failfast)
	s.suite.Run(report)
	report.Done()
	if s.notifier != nil {
		s.notifier.Notify(report)
	}
}
