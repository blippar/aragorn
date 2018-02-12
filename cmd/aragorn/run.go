package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/blippar/aragorn/server"
)

const runHelp = `Monitor and schedule the test suites in the directories`

type runCommand struct {
	failfast bool
}

func (*runCommand) Name() string { return "run" }
func (*runCommand) Args() string {
	return "[file ...]"
}
func (*runCommand) ShortHelp() string { return runHelp }
func (*runCommand) LongHelp() string  { return runHelp }
func (*runCommand) Hidden() bool      { return false }

func (cmd *runCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.failfast, "failfast", false, "stop after first test failure")
}

func (cmd *runCommand) Run(args []string) error {
	srv := server.New(cmd.failfast)
	if err := srv.Watch(args); err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		srv.Stop()
	case <-srv.Wait():
	}
	return nil
}
