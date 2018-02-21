package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blippar/aragorn/server"
)

const execShortHelp = `Execute the test suites`
const execLongHelp = `Execute the test suites` + fileHelp

type execCommand struct {
	config   string
	failfast bool
}

func (*execCommand) Name() string { return "exec" }
func (*execCommand) Args() string {
	return "[file ...]"
}
func (*execCommand) ShortHelp() string { return execShortHelp }
func (*execCommand) LongHelp() string  { return execLongHelp }
func (*execCommand) Hidden() bool      { return false }

func (cmd *execCommand) Register(fs *flag.FlagSet) {
	fs.StringVar(&cmd.config, "config", "", "Path to your config file")
	fs.BoolVar(&cmd.failfast, "failfast", false, "Stop after first test failure")
}

func (cmd *execCommand) Run(args []string) error {
	if cmd.config != "" {
		srv, err := server.New(cmd.config, cmd.failfast)
		if err != nil {
			return err
		}
		return srv.Exec()
	}
	suites, err := getSuitesFromArgs(args, cmd.failfast)
	if err != nil {
		return err
	}
	srv := &server.Server{
		Suites:   suites,
		Notifier: logNotifier,
	}
	return srv.Exec()
}

func getSuitesFromArgs(args []string, failfast bool) ([]*server.Suite, error) {
	if len(args) == 0 {
		args = []string{"."}
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
	for _, arg := range args {
		fi, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		switch mode := fi.Mode(); {
		case mode.IsDir():
			if err := filepath.Walk(arg, walkFn); err != nil {
				return nil, fmt.Errorf("could not walk %q directory: %v", arg, err)
			}
		case mode.IsRegular():
			paths = append(paths, arg)
		}
	}
	suites := make([]*server.Suite, len(paths))
	for i, path := range paths {
		s, err := server.NewSuiteFromFile(path, failfast)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		suites[i] = s
	}
	return suites, nil
}
