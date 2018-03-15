package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/server"
)

const execShortHelp = `Execute the test suites`
const execLongHelp = `Execute the test suites` + fileHelp

var errSomethingWentWrong = errors.New("something went wrong")

type execCommand struct {
	config   string
	failfast bool
	wait     bool
	filter   string
	timeout  time.Duration
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
	fs.StringVar(&cmd.filter, "filter", "", "Execute only the tests that match the regular expression")
	fs.DurationVar(&cmd.timeout, "timeout", 30*time.Second, "Timeout specifies a time limit for each test")
	fs.BoolVar(&cmd.wait, "wait", false, "Wait")
}

func (cmd *execCommand) Run(args []string) error {
	var (
		ctx       = context.Background()
		suites    []*server.Suite
		n         notifier.Notifier
		err       error
		suiteOpts = []server.SuiteOption{
			server.FailFast(cmd.failfast),
			server.Filter(cmd.filter),
			server.Timeout(cmd.timeout),
		}
	)
	if cmd.config != "" {
		cfg, err := server.NewConfigFromFile(cmd.config)
		if err != nil {
			return err
		}
		suites, err = cfg.GenSuites(suiteOpts...)
		if err != nil {
			return err
		}
		n = cfg.GenNotifier()
	} else {
		suites, err = getSuitesFromArgs(args, suiteOpts...)
		if err != nil {
			return err
		}
	}
	err = cmd.exec(ctx, suites, n)
	if cmd.wait {
		select {}
	}
	return err
}

func (cmd *execCommand) exec(ctx context.Context, suites []*server.Suite, n notifier.Notifier) error {
	var err error
	for _, suite := range suites {
		report := suite.Run(ctx)
		if n != nil {
			n.Notify(report)
		}
		if err == nil && report.NbFailed > 0 {
			err = errSomethingWentWrong
			if cmd.failfast {
				break
			}
		}
	}
	return err
}

func getSuitesFromArgs(args []string, options ...server.SuiteOption) ([]*server.Suite, error) {
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
		s, err := server.NewSuiteFromFile(path, options...)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		suites[i] = s
	}
	return suites, nil
}
