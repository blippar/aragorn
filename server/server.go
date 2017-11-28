package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/notifier/slack"
	"github.com/blippar/aragorn/scheduler"
	"github.com/blippar/aragorn/testsuite"
)

const testSuiteJSONSuffix = ".suite.json"

var (
	errFSEventsChClosed = errors.New("fsnotify events channel closed")
	errFSErrorsChClosed = errors.New("fsnotify errors channel closed")
	errNoSchedulingRule = errors.New("no scheduling rule set in test suite file: please set runCron or runEvery")
)

type Server struct {
	dirs []string
	fsw  *fsnotify.Watcher
	sch  *scheduler.Scheduler

	doneCh chan struct{}
	stopCh chan struct{}
}

func New(dirs []string) *Server {
	return &Server{
		dirs: dirs,
		sch:  scheduler.New(),
	}
}

func (s *Server) Start() error {
	s.doneCh = make(chan struct{})
	s.stopCh = make(chan struct{})

	if err := s.loadDirs(false); err != nil {
		return err
	}

	if err := s.initFSWatcher(); err != nil {
		return err
	}

	s.sch.Start()

	go s.run()
	return nil
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

func (s *Server) RunTests() error {
	return s.loadDirs(true)
}

func (s *Server) loadDirs(once bool) error {
	walkFn := func(path string, info os.FileInfo, err error) error {
		// NOTE: All errors that arise visiting files and directories are filtered by walkFn.
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, testSuiteJSONSuffix) {
			return nil
		}
		if err := s.addTestSuite(path, once); err != nil {
			log.Error("could not add test suite", zap.String("file", path), zap.Error(err))
			return nil
		}
		if !once {
			log.Info("test suite added", zap.String("file", path))
		}
		return nil
	}
	log.Info("looking for existing test suite files in directories", zap.Strings("directories", s.dirs))
	for _, dir := range s.dirs {
		if err := filepath.Walk(dir, walkFn); err != nil {
			return fmt.Errorf("could not walk %q directory: %v", dir, err)
		}
	}
	return nil
}

func (s *Server) addTestSuite(path string, once bool) error {
	cfg, err := NewSuiteConfigFromFile(path)
	if err != nil {
		return err
	}
	newSuite, err := testsuite.Get(cfg.Type)
	if err != nil {
		return err
	}
	n := notifier.NewPrinter()
	if cfg.Slack.Webhook != "" && cfg.Slack.Username != "" && cfg.Slack.Channel != "" {
		n = notifier.Multi(n, slack.New(cfg.Slack.Webhook, cfg.Slack.Username, cfg.Slack.Channel))
	}
	dir := filepath.Dir(path)
	suite, err := newSuite(dir, cfg.Suite)
	if err != nil {
		return err
	}
	sr := NewSuiteRunner(cfg.Name, suite, n)
	if once {
		sr.Run()
	} else if cfg.RunCron != "" {
		s.sch.AddCron(path, sr, cfg.RunCron)
	} else if cfg.RunEvery != "" {
		d, err := time.ParseDuration(cfg.RunEvery)
		if err != nil {
			return fmt.Errorf("could not parse duration in test suite file: %v", err)
		}
		s.sch.Add(path, sr, d)
	} else {
		return errNoSchedulingRule
	}
	return nil
}

func (s *Server) removeTestSuite(path string) error {
	return s.sch.Remove(path)
}

func (s *Server) initFSWatcher() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create new fsnotify watcher: %v", err)
	}
	log.Info("adding directories to fsnotify watcher")
	for _, dir := range s.dirs {
		if err := fsw.Add(dir); err != nil {
			fsw.Close()
			return fmt.Errorf("could not add %q directory to fsnotify watcher: %v", dir, err)
		}
	}
	s.fsw = fsw
	return nil
}

func (s *Server) run() {
	errCh := make(chan error, 2)
	go func() { errCh <- s.fsWatchEventLoop() }()
	go func() { errCh <- s.fsWatchErrorLoop() }()
	log.Info("server started")

	select {
	case err := <-errCh:
		log.Error("server stopping after fatal error", zap.Error(err))
	case <-s.stopCh:
		log.Info("server received stop signal")
	}

	s.sch.Stop()
	if err := s.fsw.Close(); err != nil {
		log.Error("inotify close error", zap.Error(err))
	}
	close(s.doneCh)
	log.Info("server stopped")
}

func (s *Server) fsWatchEventLoop() error {
	for e := range s.fsw.Events {
		if strings.HasSuffix(e.Name, testSuiteJSONSuffix) {
			s.fsHandleTestSuiteFileEvent(e)
		}
	}
	return errFSEventsChClosed
}

func (s *Server) fsHandleTestSuiteFileEvent(e fsnotify.Event) {
	switch {
	case isCreateEvent(e.Op):
		if err := s.addTestSuite(e.Name, true); err != nil {
			log.Error("could not add test suite", zap.String("file", e.Name), zap.Error(err))
			return
		}
		log.Info("test suite added", zap.String("file", e.Name))
	case isRenameEvent(e.Op) || isRemoveEvent(e.Op):
		if err := s.removeTestSuite(e.Name); err != nil {
			log.Error("could not remove test suite", zap.String("file", e.Name), zap.Error(err))
			return
		}
		log.Info("test suite removed", zap.String("file", e.Name))
	case isWriteEvent(e.Op):
		if err := s.removeTestSuite(e.Name); err != nil {
			log.Error("could not remove test suite", zap.String("file", e.Name), zap.Error(err))
			return
		}
		if err := s.addTestSuite(e.Name, true); err != nil {
			log.Error("could not create test suite from disk", zap.String("file", e.Name), zap.Error(err))
			return
		}
		log.Info("test suite updated", zap.String("file", e.Name))
	}
}

func (s *Server) fsWatchErrorLoop() error {
	for err := range s.fsw.Errors {
		// NOTE: those errors might be fatal, so maybe its would
		// be better to return the first error encountered instead
		// of just logging it.
		log.Error("inotify watcher error", zap.Error(err))
	}
	return errFSErrorsChClosed
}

func isCreateEvent(o fsnotify.Op) bool {
	return o&fsnotify.Create == fsnotify.Create
}

func isRenameEvent(o fsnotify.Op) bool {
	return o&fsnotify.Rename == fsnotify.Rename
}

func isWriteEvent(o fsnotify.Op) bool {
	return o&fsnotify.Write == fsnotify.Write
}

func isRemoveEvent(o fsnotify.Op) bool {
	return o&fsnotify.Remove == fsnotify.Remove
}
