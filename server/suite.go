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
	typ      string
	failfast bool

	runCron  string
	runEvery time.Duration

	ts testsuite.Suite
}

func (s *Suite) Path() string            { return s.path }
func (s *Suite) Name() string            { return s.name }
func (s *Suite) Type() string            { return s.typ }
func (s *Suite) FailFast() bool          { return s.failfast }
func (s *Suite) Tests() []testsuite.Test { return s.ts.Tests() }

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
		failfast: failfast,
		typ:      cfg.Type,
		ts:       suite.(testsuite.Suite),
	}
	s.fromConfig(cfg)
	if baseCfg != nil {
		s.fromConfig(baseCfg)
	}
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

func (baseCfg *SuiteConfig) GenSuite(failfast bool) (*Suite, error) {
	f, err := os.Open(baseCfg.Path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	return NewSuiteFromReader(f, failfast, baseCfg)
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

func (s *Suite) Run(ctx context.Context) *notifier.Report {
	ctx = testsuite.NewContextWithRPCInfo(ctx, s.failfast)
	report := notifier.NewReport(s)
	s.ts.Run(ctx, report)
	report.Done()
	return report
}

func (s *Suite) RunNotify(ctx context.Context, n notifier.Notifier) *notifier.Report {
	report := s.Run(ctx)
	n.Notify(report)
	return report
}

type suiteRunner struct {
	s *Suite
	n notifier.Notifier
}

func (sr *suiteRunner) Run() {
	ctx := context.Background()
	sr.s.RunNotify(ctx, sr.n)
}
