package server

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
)

var errNoSchedulingRule = errors.New("no scheduling rule set for test suite: please set runCron or runEvery")

type Server struct {
	sch      *scheduler.Scheduler
	notifier notifier.Notifier
}

func New(n notifier.Notifier) *Server {
	return &Server{
		sch:      scheduler.New(),
		notifier: n,
	}
}

func (s *Server) AddSuite(suite *Suite) error {
	if err := s.scheduleSuite(suite); err != nil {
		return fmt.Errorf("could not schedule suite %s: %v", suite.path, err)
	}
	log.Info("test suite scheduled", zap.String("file", suite.path), zap.String("suite", suite.name), zap.String("type", suite.typ))
	return nil
}

func (s *Server) Start() error {
	return s.sch.Start()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.sch.Stop(ctx)
}

func (s *Server) scheduleSuite(suite *Suite) error {
	sr := &suiteRunner{
		s: suite,
		n: s.notifier,
	}
	if suite.runCron != "" {
		return s.sch.AddCron(suite.path, sr, suite.runCron)
	} else if suite.runEvery > 0 {
		return s.sch.Add(suite.path, sr, suite.runEvery)
	}
	return errNoSchedulingRule
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
