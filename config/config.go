package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
)

// Config is the configuration for the API.
type Config struct {
	Dirs     []string
	Humanize bool
	RunOnce  bool
}

// LoadFromFile loads the configuration from a JSON file named by filename.
func LoadFromFile(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	c := &Config{}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}

// FromArgs loads the configuration from the CLI arguments.
func FromArgs() *Config {
	flag.Usage = func() {
		fmt.Printf("Usage: aragorn dir1 [dir2...]\n\n")
		flag.PrintDefaults()
	}

	humanize := flag.Bool("humanize", false, "humanize logs")
	runOnce := flag.Bool("run-once", false, "run all the tests only once and exits")
	flag.Parse()

	cfg := &Config{
		Dirs:     flag.Args(),
		Humanize: *humanize,
		RunOnce:  *runOnce,
	}
	return cfg
}
