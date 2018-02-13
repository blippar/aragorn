package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
	"go.uber.org/zap"
)

const execHelp = `Execute the test suites in the directories`

type execCommand struct {
	failfast bool
}

func (*execCommand) Name() string { return "exec" }
func (*execCommand) Args() string {
	return "[file ...]"
}
func (*execCommand) ShortHelp() string { return execHelp }
func (*execCommand) LongHelp() string  { return execHelp }
func (*execCommand) Hidden() bool      { return false }

func (cmd *execCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.failfast, "failfast", false, "stop after first test failure")
}

func (cmd *execCommand) Run(args []string) error {
	suites, err := getSuitesFromDirs(args, cmd.failfast)
	if err != nil {
		return err
	}
	for _, s := range suites {
		log.Info("running test suite", zap.String("file", s.Path()), zap.String("suite", s.Name()), zap.String("type", s.Type()))
		s.Run()
	}
	return nil
}

func getSuitesFromDirs(dirs []string, failfast bool) ([]*server.Suite, error) {
	if len(dirs) == 0 {
		dirs = []string{"."}
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
	for _, dir := range dirs {
		if strings.HasSuffix(dir, server.TestSuiteJSONSuffix) {
			paths = append(paths, dir)
			continue
		}
		if err := filepath.Walk(dir, walkFn); err != nil {
			return nil, fmt.Errorf("could not walk %q directory: %v", dir, err)
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
