package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
)

const runHelp = `Schedule the test suites in the configuration file`

type runCommand struct {
	config          string
	shutdownTimeout time.Duration
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
	fs.DurationVar(&cmd.shutdownTimeout, "shutdown-timeout", 10*time.Second, "grace period for which to wait before shutting down")
}

func (cmd *runCommand) Run(args []string) error {
	cfg, err := server.NewConfigFromFile(cmd.config)
	if err != nil {
		return err
	}
	suites, err := cfg.GenSuites()
	if err != nil {
		return err
	}
	srv := server.New(cfg.GenNotifier())
	for _, suite := range suites {
		if err := srv.AddSuite(suite); err != nil {
			return err
		}
	}
	if err := srv.Start(); err != nil {
		return err
	}
	handleSignals()
	ctx := context.Background()
	if cmd.shutdownTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.shutdownTimeout)
		defer cancel()
	}
	srv.Stop(ctx)
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
