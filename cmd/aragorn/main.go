package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/config"
	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"
)

func main() {
	cfg, err := config.FromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := log.Init(cfg.Humanize); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer log.L().Sync()

	srv := server.New(cfg)
	if err = srv.Start(); err != nil {
		log.Fatal("can not start service", zap.Error(err))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		srv.Stop()
	case <-srv.Wait():
	}
}
