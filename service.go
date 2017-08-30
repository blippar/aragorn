package main

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

type service struct {
	cfg *config.Config
	fsw *fsnotify.Watcher
	sch *scheduler.Scheduler

	doneCh chan struct{}
	stopCh chan struct{}
	errCh  chan error
}

func newService(cfg *config.Config) *service {
	s := &service{
		cfg:   cfg,
		sch:   &scheduler.Scheduler{},
		errCh: make(chan error, 1),
	}

	return s
}

func (s *service) start() error {
	s.doneCh = make(chan struct{})
	s.stopCh = make(chan struct{})

	var err error
	s.fsw, err = newFSWatcher(s.cfg)
	if err != nil {
		return err
	}

	if err := s.loadDirs(); err != nil {
		return err
	}

	s.sch.Start()

	go s.run()
	return nil
}

func (s *service) stop() {
	select {
	case s.stopCh <- struct{}{}:
		<-s.doneCh
	case <-s.doneCh:
	}
}

func (s *service) wait() <-chan struct{} {
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

func (s *service) run() {
	go s.fsWatchEventLoop()
	go s.fsWatchErrorLoop()
	log.Info("started service")

	select {
	case err := <-s.errCh:
		log.Error("service stopping after fatal error", zap.Error(err))
	case <-s.stopCh:
		log.Info("service received stop signal")
	}

	s.sch.Stop()
	if err := s.fsw.Close(); err != nil {
		log.Error("inotify close error", zap.Error(err))
	}
	close(s.doneCh)
	log.Info("service stopped")
}

func (s *service) loadDirs() error {
	walkFn := func(p string, info os.FileInfo, e error) error {
		if strings.HasSuffix(p, ".suite.json") {
			log.Info("loading tests suite", zap.String("file", p))
			if err := s.newTestSuiteFromDisk(p); err != nil {
				return fmt.Errorf("could not create tests suite from disk: %v", err)
			}
		}
		return e
	}

	log.Info("looking for existing tests suite files", zap.Strings("directories", s.cfg.Dirs))

	for _, dir := range s.cfg.Dirs {
		err := filepath.Walk(dir, walkFn)
		if err != nil {
			return fmt.Errorf("could not walk dir %q: %v", dir, err)
		}
	}
	return nil
}

func (s *service) fsWatchEventLoop() {
	for e := range s.fsw.Events {
		if strings.HasSuffix(e.Name, ".suite.json") {
			switch {
			case isCreateEvent(e.Op):
				log.Info("new tests suite", zap.String("file", e.Name))
				if err := s.newTestSuiteFromDisk(e.Name); err != nil {
					log.Error("could not create tests suite from disk", zap.Error(err))
					break
				}
			case isRenameEvent(e.Op):
				log.Info("removing tests suite", zap.String("file", e.Name))
				s.removeTestSuite(e.Name)
			case isWriteEvent(e.Op):
				log.Info("tests suite changed", zap.String("file", e.Name))
				s.removeTestSuite(e.Name)
				if err := s.newTestSuiteFromDisk(e.Name); err != nil {
					log.Error("could not create tests suite from disk", zap.Error(err))
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

func (s *service) fsWatchErrorLoop() {
	for err := range s.fsw.Errors {
		// NOTE: those errors might be fatal, so maybe its would
		// be better to return the first error encountered instead
		// of just logging it.
		log.Error("inotify watcher error", zap.Error(err))
	}
}
