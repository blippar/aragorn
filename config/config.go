package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Config is the configuration for the API.
type Config struct {
	Dirs     []string
	Humanize bool
	RunOnce  bool
}

// FromArgs loads the configuration from the CLI arguments.
func FromArgs() (*Config, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options...] dir1 [dir2...]\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	humanize := flag.Bool("humanize", false, "humanize logs")
	runOnce := flag.Bool("run-once", false, "run all the tests only once and exits")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return nil, errors.New("no directory to monitor")
	}

	cfg := &Config{
		Dirs:     args,
		Humanize: *humanize,
		RunOnce:  *runOnce,
	}
	return cfg, nil
}
