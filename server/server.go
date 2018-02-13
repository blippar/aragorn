package server

import (
	"errors"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
)

const TestSuiteJSONSuffix = ".suite.json"

var errNoSchedulingRule = errors.New("no scheduling rule set in test suite file: please set runCron or runEvery")

var logNotifier = notifier.NewLogNotifier()

type Server struct {
	sch    *scheduler.Scheduler
	suites []*Suite
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
		suites: suites,
	}, nil
}

func (s *Server) Start() error {
	s.sch = scheduler.New()
	for _, suite := range s.suites {
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

func (s *Server) Exec() {
	for _, suite := range s.suites {
		log.Info("running test suite", zap.String("file", suite.Path()), zap.String("suite", suite.Name()), zap.String("type", suite.Type()))
		suite.Run()
	}
}

func (s *Server) scheduleSuite(suite *Suite) error {
	if suite.runCron != "" {
		return s.sch.AddCron(suite.path, suite, suite.runCron)
	} else if suite.runEvery > 0 {
		return s.sch.Add(suite.path, suite, suite.runEvery)
	}
	return errNoSchedulingRule
}
