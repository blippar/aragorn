package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/blippar/aragorn/testsuite"
)

// newTestSuiteFromDisk unmarshals the test suite located at path
// initializes it. If schedule is set to true, it also adds it to the scheduler.
// Otherwise, the test suite is executed only once without delay.
func (s *Server) newTestSuiteFromDisk(path string, schedule bool) error {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read test suite file: %v", err)
	}

	var cfg testsuite.Config
	if err = json.Unmarshal(d, &cfg); err != nil {
		return fmt.Errorf("could not unmarshal test suite file: %v", err)
	}

	job, err := testsuite.CreateJob(&cfg)
	if err != nil {
		return err
	}

	if schedule {
		if cfg.RunCron != "" {
			s.sch.AddCron(path, job, cfg.RunCron)
		} else if cfg.RunEvery != "" {
			d, err := time.ParseDuration(cfg.RunEvery)
			if err != nil {
				return fmt.Errorf("could not parse duration in test suite file: %v", err)
			}
			s.sch.Add(path, job, d)
		}
	} else {
		job.Run()
	}

	return nil
}

func (s *Server) removeTestSuite(path string) {
	s.sch.Remove(path)
}
