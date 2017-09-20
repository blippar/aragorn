package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/config"
	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/scheduler"
)

type Server struct {
	cfg *config.Config
	fsw *fsnotify.Watcher
	sch *scheduler.Scheduler

	doneCh chan struct{}
	stopCh chan struct{}
	errCh  chan error
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:   cfg,
		sch:   scheduler.New(),
		errCh: make(chan error, 1),
	}

	return s
}

func (s *Server) Start() error {
	s.doneCh = make(chan struct{})
	s.stopCh = make(chan struct{})

	if err := s.loadDirs(); err != nil {
		return err
	}

	if s.cfg.RunOnce {
		close(s.doneCh)
		return nil
	}

	var err error
	s.fsw, err = newFSWatcher(s.cfg)
	if err != nil {
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

func newFSWatcher(cfg *config.Config) (*fsnotify.Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create new fsnotify watcher: %v", err)
	}

	for _, dir := range cfg.Dirs {
		log.Info("adding directory to fsnotify watcher", zap.String("directory", dir))
		if err = fsw.Add(dir); err != nil {
			return nil, fmt.Errorf("could not add directory %q to fsnotify watcher: %v", dir, err)
		}
	}
	return fsw, nil
}

func (s *Server) run() {
	go s.fsWatchEventLoop()
	go s.fsWatchErrorLoop()
	log.Info("started server")

	select {
	case err := <-s.errCh:
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

func (s *Server) walkFn(p string, info os.FileInfo, e error) error {
	if strings.HasSuffix(p, ".suite.json") {
		log.Info("loading test suite", zap.String("file", p))
		if err := s.newTestSuiteFromDisk(p, !s.cfg.RunOnce); err != nil {
			log.Error("could not load test suite", zap.String("file", p), zap.Error(err))
		}
	}
	return e
}

func (s *Server) loadDirs() error {
	log.Info("looking for existing test suite files", zap.Strings("directories", s.cfg.Dirs))

	for _, dir := range s.cfg.Dirs {
		err := filepath.Walk(dir, s.walkFn)
		if err != nil {
			return fmt.Errorf("could not walk dir %q: %v", dir, err)
		}
	}
	return nil
}

func (s *Server) fsWatchEventLoop() {
	for e := range s.fsw.Events {
		if strings.HasSuffix(e.Name, ".suite.json") {
			switch {
			case isCreateEvent(e.Op):
				log.Info("new test suite", zap.String("file", e.Name))
				if err := s.newTestSuiteFromDisk(e.Name, true); err != nil {
					log.Error("could not create test suite from disk", zap.Error(err))
					break
				}
			case isRenameEvent(e.Op):
				log.Info("removing test suite", zap.String("file", e.Name))
				s.removeTestSuite(e.Name)
			case isWriteEvent(e.Op):
				log.Info("test suite changed", zap.String("file", e.Name))
				s.removeTestSuite(e.Name)
				if err := s.newTestSuiteFromDisk(e.Name, true); err != nil {
					log.Error("could not create test suite from disk", zap.Error(err))
					break
				}
			}
		}
	}
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

func (s *Server) fsWatchErrorLoop() {
	for err := range s.fsw.Errors {
		// NOTE: those errors might be fatal, so maybe its would
		// be better to return the first error encountered instead
		// of just logging it.
		log.Error("inotify watcher error", zap.Error(err))
	}
}
