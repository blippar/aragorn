package server

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
)

var (
	errNoSchedulingRule   = errors.New("no scheduling rule set for test suite: please set runCron or runEvery")
	errSomethingWentWrong = errors.New("something went wrong")
)

type Server struct {
	sch      *scheduler.Scheduler
	Suites   []*Suite
	Notifier notifier.Notifier
	Failfast bool
}

func New(cfgPath string, failfast bool) (*Server, error) {
	cfg, err := NewConfigFromFile(cfgPath)
	if err != nil {
		return nil, err
	}
	suites, err := cfg.GenSuites(failfast)
	if err != nil {
		return nil, err
	}
	return &Server{
		Suites:   suites,
		Notifier: cfg.Notifier(),
		Failfast: failfast,
	}, nil
}

func (s *Server) Start() error {
	s.sch = scheduler.New()
	for _, suite := range s.Suites {
		if err := s.scheduleSuite(suite); err != nil {
			log.Error("test suite scheduling", zap.String("file", suite.Path()), zap.Error(err))
		} else {
			log.Info("test suite scheduled", zap.String("file", suite.Path()), zap.String("suite", suite.Name()), zap.String("type", suite.Type()))
		}
	}
	s.sch.Start()
	return nil
}

func (s *Server) Stop() {
	if s.sch != nil {
		s.sch.Stop()
	}
}

func (s *Server) Exec() error {
	ctx := context.Background()
	ok := true
	for _, suite := range s.Suites {
		report := suite.Run(ctx)
		if s.Notifier != nil {
			s.Notifier.Notify(report)
		}
		if ok {
			for _, tr := range report.TestReports {
				if len(tr.Errs) > 0 {
					ok = false
					if s.Failfast {
						return errSomethingWentWrong
					}
					break
				}
			}
		}
	}
	if !ok {
		return errSomethingWentWrong
	}
	return nil
}

func (s *Server) scheduleSuite(suite *Suite) error {
	sr := &suiteRunner{
		s: suite,
		n: s.Notifier,
	}
	if suite.runCron != "" {
		return s.sch.AddCron(suite.path, sr, suite.runCron)
	} else if suite.runEvery > 0 {
		return s.sch.Add(suite.path, sr, suite.runEvery)
	}
	return errNoSchedulingRule
}
