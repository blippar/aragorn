package server

import (
	"context"
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
	Suite    testsuite.Suite
	notifier notifier.Notifier
	typ      string
	runCron  string
	runEvery time.Duration
	failfast bool
	ctx      context.Context
}

func (s *Suite) Path() string   { return s.path }
func (s *Suite) Name() string   { return s.name }
func (s *Suite) Type() string   { return s.typ }
func (s *Suite) FailFast() bool { return s.failfast }

type SuiteConfig struct {
	Path     string          `json:"path,omitempty"`     // only used in base config.
	Name     string          `json:"name,omitempty"`     // identifier for this test suite
	RunEvery duration        `json:"runEvery,omitempty"` // scheduling every duration.
	RunCron  string          `json:"runCron,omitempty"`  // cron string.
	FailFast bool            `json:"failFast,omitempty"` // stop after first test failure.
	Type     string          `json:"type,omitempty"`     // type of the test suite, can be HTTP, GRPC...
	Suite    json.RawMessage `json:"suite,omitempty"`    // description of the test suite, depends on Type.
}

func NewSuiteFromReader(r io.Reader, failfast bool, baseCfg *SuiteConfig) (*Suite, error) {
	cfg := &SuiteConfig{}
	if err := decodeReaderJSON(r, cfg); err != nil {
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
	if baseCfg != nil && baseCfg.Suite != nil {
		if err := decodeJSON(baseCfg.Suite, ic.Config); err != nil {
			return nil, fmt.Errorf("could not decode base test suite: %v", err)
		}
	}
	suite, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	s := &Suite{
		path:     path,
		Suite:    suite.(testsuite.Suite),
		notifier: logNotifier,
		failfast: failfast,
		typ:      cfg.Type,
	}
	s.fromConfig(cfg)
	if baseCfg != nil {
		s.fromConfig(baseCfg)
	}
	ctx := context.Background()
	s.ctx = testsuite.NewContextWithRPCInfo(ctx, s.failfast)
	return s, nil
}

func NewSuiteFromFile(path string, failfast bool) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	return NewSuiteFromReader(f, failfast, nil)
}

func NewSuiteFromSuiteConfig(baseCfg *SuiteConfig, failfast bool, n notifier.Notifier) (*Suite, error) {
	f, err := os.Open(baseCfg.Path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	s, err := NewSuiteFromReader(f, failfast, baseCfg)
	if err != nil {
		return nil, err
	}
	s.notifier = n
	return s, nil
}

func (s *Suite) fromConfig(cfg *SuiteConfig) {
	if cfg.Name != "" {
		s.name = cfg.Name
	}
	if cfg.RunEvery != 0 {
		s.runEvery = time.Duration(cfg.RunEvery)
	}
	if cfg.RunCron != "" {
		s.runCron = cfg.RunCron
	}
	s.failfast = s.failfast || cfg.FailFast
}

func (s *Suite) Run() {
	report := notifier.NewReport(s)
	s.Suite.Run(s.ctx, report)
	report.Done()
	s.notifier.Notify(report)
}
