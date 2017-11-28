package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type SuiteConfig struct {
	// Name used to identify this test suite.
	Name string

	RunEvery string // parsable by time.ParseDuration
	RunCron  string // cron string

	Slack struct {
		Webhook  string
		Username string
		Channel  string
	}

	// Type of the test suite, can be HTTP or GRPC.
	Type string

	// Description of the test suite, depends on Type.
	Suite json.RawMessage
}

func NewSuiteConfigFromReader(r io.Reader) (*SuiteConfig, error) {
	cfg := &SuiteConfig{}
	if err := json.NewDecoder(r).Decode(cfg); err != nil {
		return nil, fmt.Errorf("could not decode test suite file: %v", err)
	}
	return cfg, nil
}

func NewSuiteConfigFromFile(path string) (*SuiteConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open test suite file: %v", err)
	}
	defer f.Close()
	return NewSuiteConfigFromReader(f)
}
