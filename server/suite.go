package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/blippar/aragorn/log"
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

	retryCount int
	retryWait  time.Duration
	timeout    time.Duration

	ts testsuite.Suite
}

func (s *Suite) Path() string            { return s.path }
func (s *Suite) Name() string            { return s.name }
func (s *Suite) Type() string            { return s.typ }
func (s *Suite) FailFast() bool          { return s.failfast }
func (s *Suite) Tests() []testsuite.Test { return s.ts.Tests() }

type SuiteConfig struct {
	Path       string          `json:"path,omitempty"`     // only used in base config.
	Name       string          `json:"name,omitempty"`     // identifier for this test suite
	RunEvery   duration        `json:"runEvery,omitempty"` // scheduling every duration.
	RunCron    string          `json:"runCron,omitempty"`  // cron string.
	RetryCount int             `json:"retryCount,omitempty"`
	RetryWait  duration        `json:"retryWait,omitempty"`
	Timeout    duration        `json:"timeout,omitempty"`
	FailFast   bool            `json:"failFast,omitempty"` // stop after first test failure.
	Type       string          `json:"type,omitempty"`     // type of the test suite, can be HTTP, GRPC...
	Suite      json.RawMessage `json:"suite,omitempty"`    // description of the test suite, depends on Type.
}

func NewSuiteFromReader(r io.Reader, failfast bool, baseSuite json.RawMessage) (*Suite, error) {
	cfg := &SuiteConfig{
		RetryCount: 1,
		RetryWait:  duration(1 * time.Second),
		Timeout:    duration(30 * time.Second),
	}
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
		return nil, fmt.Errorf("could not decode test suite: %v", jsonDecodeError(nil, err))
	}
	if baseSuite != nil {
		if err := decodeJSON(baseSuite, ic.Config); err != nil {
			return nil, fmt.Errorf("could not decode base test suite: %v", jsonDecodeError(nil, err))
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
	s, err := NewSuiteFromReader(f, failfast, baseCfg.Suite)
	if err != nil {
		return nil, err
	}
	s.fromConfig(baseCfg)
	return s, nil
}

func (s *Suite) fromConfig(cfg *SuiteConfig) {
	if cfg.Name != "" {
		s.name = cfg.Name
	}
	if cfg.RunEvery > 0 {
		s.runEvery = time.Duration(cfg.RunEvery)
	}
	if cfg.RunCron != "" {
		s.runCron = cfg.RunCron
	}
	s.failfast = s.failfast || cfg.FailFast
	if cfg.RetryCount > 0 {
		s.retryCount = cfg.RetryCount
	}
	if cfg.RetryWait > 0 {
		s.retryWait = time.Duration(cfg.RetryWait)
	}
	if cfg.Timeout > 0 {
		s.timeout = time.Duration(cfg.Timeout)
	}
}

func (s *Suite) Run(ctx context.Context) *notifier.Report {
	log.Info("running suite", zap.String("file", s.path), zap.String("suite", s.name), zap.String("type", s.typ))

	span, ctx := ot.StartSpanFromContext(ctx, s.name)
	defer span.Finish()

	report := notifier.NewReport(s)

	for _, t := range s.ts.Tests() {
		ok := s.runTestWithRetry(ctx, t, report)
		if (s.failfast && !ok) || ctx.Err() != nil {
			span.SetTag("failfast", true)
			break
		}
	}

	report.Done()

	log.Info("test suite done",
		zap.String("suite", s.name),
		zap.Bool("failfast", s.failfast),
		zap.Int("nb_tests", len(s.Tests())),
		zap.Int("nb_test_reports", len(report.TestReports)),
		zap.Int("nb_failed", report.NbFailed),
		zap.Time("started_at", report.Start),
		zap.Duration("duration", report.Duration),
	)

	return report
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try.
func (s *Suite) runTestWithRetry(ctx context.Context, t testsuite.Test, r *notifier.Report) bool {
	for attempt := 1; ; attempt++ {
		if ok := s.runTest(ctx, t, r); ok {
			return ok
		}
		if attempt >= s.retryCount {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(s.retryWait):
		}
	}
}

func (s *Suite) runTest(ctx context.Context, t testsuite.Test, r *notifier.Report) bool {
	log.Info("running test", zap.String("name", t.Name()), zap.String("description", t.Description()))

	span, ctx := ot.StartSpanFromContext(ctx, t.Name())
	defer span.Finish()

	tr := r.NewTestReport(t)

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	t.Run(ctx, tr)

	tr.Done()
	fields := []zapcore.Field{
		zap.String("name", t.Name()),
		zap.Time("started_at", tr.Start),
		zap.Duration("duration", tr.Duration),
	}
	ok := len(tr.Errs) == 0
	if !ok {
		r.NbFailed++
		otext.Error.Set(span, true)

		errs := make([]string, len(tr.Errs))
		for i, err := range tr.Errs {
			str := err.Error()
			errs[i] = str
			span.LogFields(otlog.String("event", str))
		}
		fields = append(fields, zap.Strings("errors", errs))
		log.Warn("test failed", fields...)
	} else {
		log.Info("test passed", fields...)
	}
	return ok
}

type suiteRunner struct {
	s *Suite
	n notifier.Notifier
}

func (sr *suiteRunner) Run() {
	ctx := context.Background()
	r := sr.s.Run(ctx)
	if sr.n != nil {
		sr.n.Notify(r)
	}
}
