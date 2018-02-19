package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
)

const execShortHelp = `Execute the test suites`
const execLongHelp = `Execute the test suites` + fileHelp

type execCommand struct {
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
	fs.BoolVar(&cmd.failfast, "failfast", false, "Stop after first test failure")
}

func (cmd *execCommand) Run(args []string) error {
	suites, err := getSuitesFromArgs(args, cmd.failfast)
	if err != nil {
		return err
	}
	for _, s := range suites {
		log.Info("running test suite", zap.String("file", s.Path()), zap.String("suite", s.Name()), zap.String("type", s.Type()))
		s.Run()
	}
	return nil
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
		if !strings.HasSuffix(path, server.TestSuiteJSONSuffix) {
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
