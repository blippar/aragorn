package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/testsuite"
)

type Config struct {
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

// newTestSuiteFromDisk unmarshals the test suite located at path
// initializes it. If schedule is set to true, it also adds it to the scheduler.
// Otherwise, the test suite is executed only once without delay.
func (s *Server) newTestSuiteFromDisk(path string, schedule bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open test suite file: %v", err)
	}
	defer f.Close()

	cfg := &Config{}
	if err = json.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("could not decode test suite file: %v", err)
	}

	newSuite, err := testsuite.Get(cfg.Type)
	if err != nil {
		return err
	}
	n := notifier.NewPrinter()
	if cfg.Slack.Webhook != "" && cfg.Slack.Username != "" && cfg.Slack.Channel != "" {
		n = notifier.Multi(n, notifier.NewSlackNotifier(cfg.Slack.Webhook, cfg.Slack.Username, cfg.Slack.Channel))
	}
	suite, err := newSuite(cfg.Suite)
	if err != nil {
		return err
	}
	sr := newSuiteRunner(cfg.Name, suite, n)

	if schedule {
		if cfg.RunCron != "" {
			s.sch.AddCron(path, sr, cfg.RunCron)
		} else if cfg.RunEvery != "" {
			d, err := time.ParseDuration(cfg.RunEvery)
			if err != nil {
				return fmt.Errorf("could not parse duration in test suite file: %v", err)
			}
			s.sch.Add(path, sr, d)
		}
	} else {
		sr.Run()
	}

	return nil
}

func (s *Server) removeTestSuite(path string) error {
	return s.sch.Remove(path)
}
