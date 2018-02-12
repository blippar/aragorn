package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/scheduler"
)

const testSuiteJSONSuffix = ".suite.json"

var (
	errFSEventsChClosed = errors.New("fsnotify events channel closed")
	errFSErrorsChClosed = errors.New("fsnotify errors channel closed")
	errNoSchedulingRule = errors.New("no scheduling rule set in test suite file: please set runCron or runEvery")
)

var logNotifier = notifier.NewLogNotifier()

type Server struct {
	failfast bool

	fsw      *fsnotify.Watcher
	sch      *scheduler.Scheduler
	notifier notifier.Notifier

	doneCh chan struct{}
	stopCh chan struct{}
}

func New(failfast bool) *Server {
	return &Server{
		failfast: failfast,
		sch:      scheduler.New(),
		notifier: logNotifier,
	}
}

func (s *Server) Stop() {
	select {
	case s.stopCh <- struct{}{}:
		<-s.doneCh
	case <-s.doneCh:
	}
}

func (s *Server) Wait() <-chan struct{} {
	return s.doneCh
}

func List(dirs []string) error {
	suites, err := getSuitesFromDirs(dirs)
	if err != nil {
		return err
	}
	for _, s := range suites {
		log.Info("test suite", zap.String("file", s.path), zap.String("suite", s.name), zap.String("type", s.typ))
	}
	return nil
}

func Exec(dirs []string, failfast bool) error {
	suites, err := getSuitesFromDirs(dirs)
	if err != nil {
		return err
	}
	for _, s := range suites {
		if !s.failfast && failfast {
			s.failfast = true
		}
		s.Run()
	}
	return nil
}

func (s *Server) Watch(dirs []string) error {
	if err := s.initFSWatcher(dirs); err != nil {
		return err
	}
	s.doneCh = make(chan struct{})
	s.stopCh = make(chan struct{})
	go s.runWatch()
	return nil
}

func (s *Server) Start() {
	s.doneCh = make(chan struct{})
	s.stopCh = make(chan struct{})
	s.sch.Start()
	go s.run()
}

func getSuitesFromDirs(dirs []string) ([]*Suite, error) {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	var paths []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, testSuiteJSONSuffix) {
			return nil
		}
		paths = append(paths, path)
		return nil
	}
	for _, dir := range dirs {
		if strings.HasSuffix(dir, testSuiteJSONSuffix) {
			paths = append(paths, dir)
			continue
		}
		if err := filepath.Walk(dir, walkFn); err != nil {
			return nil, fmt.Errorf("could not walk %q directory: %v", dir, err)
		}
	}
	suites := make([]*Suite, len(paths))
	for i, path := range paths {
		s, err := NewSuiteFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		suites[i] = s
	}
	return suites, nil
}

func (s *Server) addSuite(suite *Suite) {
	suite.notifier = s.notifier
	if !suite.failfast && s.failfast {
		suite.failfast = true
	}
	log.Info("test suite", zap.String("file", suite.path), zap.String("suite", suite.name), zap.String("type", suite.typ))
	if err := s.scheduleSuite(suite); err != nil {
		log.Error("test suite scheduling", zap.String("file", suite.path), zap.Error(err))
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

func (s *Server) initFSWatcher(dirs []string) error {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create new fsnotify watcher: %v", err)
	}
	log.Info("adding directories to fsnotify watcher")
	for _, dir := range dirs {
		if err := fsw.Add(dir); err != nil {
			fsw.Close()
			return fmt.Errorf("could not add %q directory to fsnotify watcher: %v", dir, err)
		}
	}
	s.fsw = fsw
	return nil
}

func (s *Server) run() {
	log.Info("server started")
	<-s.stopCh
	log.Info("server stopping")
	s.sch.Stop()
	close(s.doneCh)
	log.Info("server stopped")
}

func (s Server) runWatch() {
	log.Info("server started")
	errCh := make(chan error, 2)
	go func() { errCh <- s.fsWatchEventLoop() }()
	go func() { errCh <- s.fsWatchErrorLoop() }()
	select {
	case err := <-errCh:
		log.Error("server stopping after fatal error", zap.Error(err))
	case <-s.stopCh:
		log.Info("server stopping")
	}
	if err := s.fsw.Close(); err != nil {
		log.Error("inotify close error", zap.Error(err))
	}
	close(s.doneCh)
	log.Info("server stopped")
}

func (s *Server) fsWatchEventLoop() error {
	for e := range s.fsw.Events {
		log.Debug("watch event", zap.String("file", e.Name), zap.String("op", e.Op.String()))
		if strings.HasSuffix(e.Name, testSuiteJSONSuffix) {
			s.fsHandleTestSuiteFileEvent(e)
		}
	}
	return errFSEventsChClosed
}

func (s *Server) fsHandleTestSuiteFileEvent(e fsnotify.Event) {
	switch {
	case isCreateEvent(e.Op) || isWriteEvent(e.Op):
		s.runSuiteFromFile(e.Name)
	}
}

func (s *Server) runSuiteFromFile(path string) {
	suite, err := NewSuiteFromFile(path)
	if err != nil {
		log.Error("test suite create", zap.String("file", path), zap.Error(err))
		return
	}
	suite.notifier = s.notifier
	if !suite.failfast && s.failfast {
		suite.failfast = true
	}
	log.Info("test suite", zap.String("file", suite.path), zap.String("suite", suite.name), zap.String("type", suite.typ))
	suite.Run()
}

func (s *Server) fsWatchErrorLoop() error {
	for err := range s.fsw.Errors {
		// Those errors might be fatal, so maybe its would
		// be better to return the first error encountered instead
		// of just logging it.
		log.Error("inotify watcher error", zap.Error(err))
	}
	return errFSErrorsChClosed
}

func isCreateEvent(o fsnotify.Op) bool {
	return o&fsnotify.Create == fsnotify.Create
}

func isWriteEvent(o fsnotify.Op) bool {
	return o&fsnotify.Write == fsnotify.Write
}
