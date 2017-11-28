package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/server"

	_ "github.com/blippar/aragorn/testsuite/grpcexpect"
	_ "github.com/blippar/aragorn/testsuite/httpexpect"
)

var (
	humanize = flag.Bool("humanize", false, "humanize logs")
	runOnce  = flag.Bool("run-once", false, "run all the tests only once and exits")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options...] dir1 [dir2...]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	dirs := flag.Args()
	if len(dirs) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err := log.Init(*humanize); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer log.L().Sync()

	srv := server.New(dirs)
	if *runOnce {
		if err := srv.RunTests(); err != nil {
			log.Fatal("could not run test suites", zap.Error(err))
		}
		return
	}
	if err := srv.Start(); err != nil {
		log.Fatal("could not start service", zap.Error(err))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		srv.Stop()
	case <-srv.Wait():
	}
}
