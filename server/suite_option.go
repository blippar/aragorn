package server

import (
	"fmt"
	"regexp"
	"time"

	"github.com/blippar/aragorn/pkg/util/json"
)

// SuiteOption is a function that sets some option on the suite.
type SuiteOption func(c *SuiteConfig)

func Timeout(timeout time.Duration) SuiteOption {
	return func(cfg *SuiteConfig) {
		cfg.Timeout = json.Duration(timeout)
	}
}

func FailFast(ff bool) SuiteOption {
	return func(cfg *SuiteConfig) {
		cfg.FailFast = ff
	}
}

func Filter(filter string) SuiteOption {
	return func(cfg *SuiteConfig) {
		cfg.filter = filter
	}
}

func baseSuite(bs []byte) SuiteOption {
	return func(cfg *SuiteConfig) {
		cfg.baseSuite = bs
	}
}

func (cfg *SuiteConfig) applyOptions(options ...SuiteOption) {
	for _, option := range options {
		option(cfg)
	}
}

func (s *Suite) applyConfig(cfg *SuiteConfig) error {
	if cfg.Name != "" {
		s.name = cfg.Name
	}
	if cfg.RunEvery > 0 {
		s.runEvery = time.Duration(cfg.RunEvery)
	}
	if cfg.RunCron != "" {
		s.runCron = cfg.RunCron
	}
	if cfg.RetryCount > 0 {
		s.retryCount = cfg.RetryCount
	}
	if cfg.RetryWait > 0 {
		s.retryWait = time.Duration(cfg.RetryWait)
	}
	if cfg.Timeout > 0 {
		s.timeout = time.Duration(cfg.Timeout)
	}
	if cfg.FailFast {
		s.failfast = true
	}
	if cfg.filter != "" {
		re, err := regexp.Compile(cfg.filter)
		if err != nil {
			return fmt.Errorf("filter: %v", err)
		}
		tests := s.tests[:0]
		for _, t := range s.tests {
			if re.MatchString(t.Name()) {
				tests = append(tests, t)
			}
		}
		s.tests = tests
	}
	return nil
}
