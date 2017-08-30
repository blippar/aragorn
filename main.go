package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/blippar/aragorn/config"
	"github.com/blippar/aragorn/log"
)

func main() {
	cfg := config.FromArgs()

	if err := log.Init(cfg.Humanize); err != nil {
		exit(err)
	}
	defer log.L().Sync()

	svc := newService(cfg)
	if err := svc.start(); err != nil {
		exit(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		svc.stop()
	case <-svc.wait():
	}
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
