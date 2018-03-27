package server

import (
	"context"
	gojson "encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	ot "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/plugin"
	"github.com/blippar/aragorn/testsuite"
)

type Suite struct {
	path       string
	name       string
	typ        string
	runCron    string
	runEvery   time.Duration
	retryCount int
	retryWait  time.Duration
	timeout    time.Duration
	failfast   bool
	tests      []testsuite.Test
}

func (s *Suite) Path() string            { return s.path }
func (s *Suite) Name() string            { return s.name }
func (s *Suite) Type() string            { return s.typ }
func (s *Suite) FailFast() bool          { return s.failfast }
func (s *Suite) Tests() []testsuite.Test { return s.tests }

type SuiteConfig struct {
	Path string `json:"path,omitempty"` // only used in base config.

	Name       string        `json:"name,omitempty"`     // identifier for this test suite
	RunEvery   json.Duration `json:"runEvery,omitempty"` // scheduling every duration.
	RunCron    string        `json:"runCron,omitempty"`  // cron string.
	RetryCount int           `json:"retryCount,omitempty"`
	RetryWait  json.Duration `json:"retryWait,omitempty"`
	Timeout    json.Duration `json:"timeout,omitempty"`
	FailFast   bool          `json:"failFast,omitempty"` // stop after first test failure.

	Type  string            `json:"type,omitempty"`  // type of the test suite, can be HTTP, GRPC...
	Suite gojson.RawMessage `json:"suite,omitempty"` // description of the test suite, depends on Type.

	baseSuite []byte
	filter    string
}

type namer interface {
	Name() string
}

func NewSuite(path, typ string, tests []testsuite.Test, cfg *SuiteConfig) (*Suite, error) {
	s := &Suite{
		path:  path,
		typ:   typ,
		tests: tests,
	}
	if err := s.applyConfig(cfg); err != nil {
		return nil, err
	}
	return s, nil
}

func NewSuiteFromReader(r io.Reader, options ...SuiteOption) (*Suite, error) {
	cfg := &SuiteConfig{
		RetryCount: 1,
		RetryWait:  json.Duration(1 * time.Second),
		Timeout:    json.Duration(30 * time.Second),
	}
	if err := json.Decode(r, cfg); err != nil {
		return nil, fmt.Errorf("could not decode suite: %v", err)
	}
	cfg.applyOptions(options...)
	reg := plugin.Get(plugin.TestSuitePlugin, cfg.Type)
	if reg == nil {
		return nil, fmt.Errorf("unsupported test suite type: %q", cfg.Type)
	}
	path := ""
	if n, ok := r.(namer); ok {
		path = n.Name()
	}
	ic := plugin.NewContext(reg, path)
	if err := json.Unmarshal(cfg.Suite, ic.Config); err != nil {
		return nil, fmt.Errorf("could not decode test suite: %v", err)
	}
	if cfg.baseSuite != nil {
		if err := json.Unmarshal(cfg.baseSuite, ic.Config); err != nil {
			return nil, fmt.Errorf("could not decode base test suite: %v", err)
		}
	}
	suite, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	ts := suite.(testsuite.Suite)
	return NewSuite(path, cfg.Type, ts.Tests(), cfg)
}

func NewSuiteFromFile(path string, options ...SuiteOption) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	return NewSuiteFromReader(f, options...)
}

func (s *Suite) Run(ctx context.Context) *notifier.Report {
	log.Info("running suite", zap.String("file", s.path), zap.String("suite", s.name), zap.String("type", s.typ))
	span, ctx := ot.StartSpanFromContext(ctx, s.name)
	defer span.Finish()
	report := s.runTests(ctx, span)
	log.Info("test suite done",
		zap.String("suite", s.name),
		zap.Bool("failfast", s.failfast),
		zap.Int("nb_tests", len(s.tests)),
		zap.Int("nb_test_reports", len(report.TestReports)),
		zap.Int("nb_failed", report.NbFailed),
		zap.Time("started_at", report.Start),
		zap.Duration("duration", report.Duration),
	)
	return report
}

func (s *Suite) runTests(ctx context.Context, span ot.Span) *notifier.Report {
	report := notifier.NewReport(s)
	defer report.Done()
	ctx = testsuite.NewMDContext(ctx, testsuite.NewMD())
	for _, t := range s.tests {
		ok := s.runTestWithRetry(ctx, t, report)
		if !ok {
			report.NbFailed++
			if s.failfast {
				span.SetTag("failfast", true)
				break
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	return report
}

// runTestWithRetry will try to run the test t up to n times, waiting for n * wait time
// in between each try.
func (s *Suite) runTestWithRetry(ctx context.Context, t testsuite.Test, r *notifier.Report) bool {
	tr := r.NewTestReport(t)
	defer tr.Done()
	for attempt := 1; ; attempt++ {
		if ok := s.runTest(ctx, t, tr); ok {
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
		log.Info("test retry", zap.String("name", t.Name()), zap.Int("attempt", attempt+1), zap.Int("max_attempts", s.retryCount))
		tr.Reset()
	}
}

func (s *Suite) runTest(ctx context.Context, t testsuite.Test, tr *notifier.TestReport) bool {
	log.Info("running test", zap.String("name", t.Name()), zap.String("description", t.Description()))

	span, ctx := ot.StartSpanFromContext(ctx, t.Name())
	defer span.Finish()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	t.Run(ctx, tr)

	fields := []zapcore.Field{
		zap.String("name", t.Name()),
		zap.Time("started_at", tr.Start),
		zap.Duration("duration", tr.Duration),
	}
	ok := len(tr.Errs) == 0
	if !ok {
		errs := make([]string, len(tr.Errs))
		otfields := make([]otlog.Field, len(tr.Errs))
		for i, err := range tr.Errs {
			str := err.Error()
			errs[i] = str
			otfields[i] = otlog.String("error", str)
		}
		fields = append(fields, zap.Strings("errors", errs))
		log.Warn("test failed", fields...)
		otext.Error.Set(span, true)
		span.LogFields(otfields...)
	} else {
		log.Info("test passed", fields...)
	}
	return ok
}
