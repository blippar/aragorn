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

type runCommand struct{}

func (*runCommand) Name() string { return "run" }
func (*runCommand) Args() string {
	return "[file ...]"
}
func (*runCommand) ShortHelp() string { return runHelp }
func (*runCommand) LongHelp() string  { return runHelp }
func (*runCommand) Hidden() bool      { return false }

func (*runCommand) Register(fs *flag.FlagSet) {}

func (*runCommand) Run(args []string) error {
	srv := server.New(args)
	if err := srv.Start(); err != nil {
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
