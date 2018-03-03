package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
)

const watchShortHelp = `Watch the test suites and execute them on create or save`
const watchLongHelp = `Watch the test suites and execute them on create or save` + fileHelp

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
func (*watchCommand) ShortHelp() string { return watchShortHelp }
func (*watchCommand) LongHelp() string  { return watchLongHelp }
func (*watchCommand) Hidden() bool      { return false }

func (cmd *watchCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.failfast, "failfast", false, "Stop after first test failure")
}

func (cmd *watchCommand) Run(args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create new fsnotify watcher: %v", err)
	}
	defer fsw.Close()
	log.Info("adding files to fsnotify watcher")
	argFiles := make(map[string]bool)
	for _, arg := range args {
		name := filepath.Clean(arg)
		if err := fsw.Add(name); err != nil {
			return fmt.Errorf("could not add %q directory to fsnotify watcher: %v", arg, err)
		}
		fi, err := os.Stat(name)
		if err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			argFiles[name] = true
		}
	}
	errCh := make(chan error, 2)
	go func() { errCh <- cmd.fsWatchEventLoop(fsw.Events, argFiles) }()
	go func() { errCh <- cmd.fsWatchErrorLoop(fsw.Errors) }()
	select {
	case err := <-errCh:
		log.Error("server stopping after fatal error", zap.Error(err))
	}
	return nil
}

func (cmd *watchCommand) fsWatchEventLoop(events <-chan fsnotify.Event, argFiles map[string]bool) error {
	for e := range events {
		log.Debug("watch event", zap.String("file", e.Name), zap.String("op", e.Op.String()))
		if cmd.isValidEvent(e.Op) && (strings.HasSuffix(e.Name, testSuiteJSONSuffix) || argFiles[e.Name]) {
			cmd.runSuiteFromFile(e.Name)
		}
	}
	return errFSEventsChClosed
}

func (cmd *watchCommand) runSuiteFromFile(path string) {
	ctx := context.Background()
	s, err := server.NewSuiteFromFile(path, cmd.failfast)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v", path, err)
		return
	}
	s.Run(ctx)
}

func (*watchCommand) fsWatchErrorLoop(errc <-chan error) error {
	for err := range errc {
		// Those errors might be fatal, so maybe it would
		// be better to return the first error encountered instead
		// of just logging it.
		log.Error("inotify watcher error", zap.Error(err))
	}
	return errFSErrorsChClosed
}

func (*watchCommand) isValidEvent(o fsnotify.Op) bool {
	return o&fsnotify.Create == fsnotify.Create || o&fsnotify.Write == fsnotify.Write
}
