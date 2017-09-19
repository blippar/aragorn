package testsuite

import (
	"encoding/json"
	"fmt"

	"github.com/blippar/aragorn/scheduler"
)

type Config struct {
	// Name used to identify this tests suite.
	Name string

	RunEvery string // parsable by time.ParseDuration
	RunCron  string // cron string

	SlackWebhook string

	// Type of the tests suite, can be HTTP or GRPC.
	Type string

	// Description of the tests suite, depends on Type.
	Suite json.RawMessage
}

type RegisterFunc func(*Config) (scheduler.Job, error)

var m map[string]RegisterFunc

func init() {
	m = make(map[string]RegisterFunc)
}

func CreateJob(cfg *Config) (scheduler.Job, error) {
	fn, ok := m[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported tests suite type: %q", cfg.Type)
	}

	return fn(cfg)
}

func Register(typ string, fn RegisterFunc) {
	m[typ] = fn
}
