package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
)

const runHelp = `Schedule the test suites in the configuration file`

type runCommand struct {
	config   string
	failfast bool
}

func (*runCommand) Name() string { return "run" }
func (*runCommand) Args() string {
	return ""
}
func (*runCommand) ShortHelp() string { return runHelp }
func (*runCommand) LongHelp() string  { return runHelp }
func (*runCommand) Hidden() bool      { return false }

func (cmd *runCommand) Register(fs *flag.FlagSet) {
	fs.StringVar(&cmd.config, "config", "config.json", "Path to your config file")
	fs.BoolVar(&cmd.failfast, "failfast", false, "Stop after first test failure")
}

func (cmd *runCommand) Run(args []string) error {
	srv, err := server.New(cmd.config, cmd.failfast)
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return err
	}
	handleSignals()
	srv.Stop()
	return nil
}

func handleSignals() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sigCh:
		log.Debug("received signal", zap.String("signal", s.String()))
	}
}
