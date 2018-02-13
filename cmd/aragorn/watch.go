package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

const watchHelp = `Watch the test suites in the directories and execute them on create or update`

var (
	errFSEventsChClosed = errors.New("fsnotify events channel closed")
	errFSErrorsChClosed = errors.New("fsnotify errors channel closed")
)

type watchCommand struct {
	failfast bool
}

func (*watchCommand) Name() string { return "watch" }
func (*watchCommand) Args() string {
	return "[file ...]"
}
func (*watchCommand) ShortHelp() string { return watchHelp }
func (*watchCommand) LongHelp() string  { return watchHelp }
func (*watchCommand) Hidden() bool      { return false }

func (cmd *watchCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.failfast, "failfast", false, "stop after first test failure")
}

func (cmd *watchCommand) Run(dirs []string) error {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create new fsnotify watcher: %v", err)
	}
	defer fsw.Close()
	log.Info("adding directories to fsnotify watcher")
	for _, dir := range dirs {
		if err := fsw.Add(dir); err != nil {
			return fmt.Errorf("could not add %q directory to fsnotify watcher: %v", dir, err)
		}
	}
	errCh := make(chan error, 2)
	go func() { errCh <- fsWatchEventLoop(fsw.Events) }()
	go func() { errCh <- fsWatchErrorLoop(fsw.Errors) }()
	select {
	case err := <-errCh:
		log.Error("server stopping after fatal error", zap.Error(err))
	}
	return nil
}

func fsWatchEventLoop(events <-chan fsnotify.Event) error {
	for e := range events {
		log.Debug("watch event", zap.String("file", e.Name), zap.String("op", e.Op.String()))
		if strings.HasSuffix(e.Name, server.TestSuiteJSONSuffix) {
			fsHandleTestSuiteFileEvent(e)
		}
	}
	return errFSEventsChClosed
}

func fsHandleTestSuiteFileEvent(e fsnotify.Event) {
	switch {
	case isCreateEvent(e.Op) || isWriteEvent(e.Op):
		runSuiteFromFile(e.Name)
	}
}

func runSuiteFromFile(path string) {
	s, err := server.NewSuiteFromFile(path, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v", path, err)
		return
	}
	log.Info("running test suite", zap.String("file", s.Path()), zap.String("suite", s.Name()), zap.String("type", s.Type()))
	s.Run()
}

func fsWatchErrorLoop(errc <-chan error) error {
	for err := range errc {
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
