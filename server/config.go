package server

import (
	gojson "encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/pkg/util/json"
	"github.com/blippar/aragorn/plugin"
)

type Config struct {
	Notifiers map[string]gojson.RawMessage
	Suites    []*SuiteConfig
	path      string
	dir       string
}

func NewConfigFromReader(r io.Reader, options ...SuiteOption) (*Config, error) {
	cfg := &Config{}
	if err := json.Decode(r, cfg); err != nil {
		return nil, fmt.Errorf("could not decode config: %v", err)
	}
	if n, ok := r.(namer); ok {
		cfg.path = n.Name()
		cfg.dir = filepath.Dir(cfg.path)
	}
	return cfg, nil
}

func NewConfigFromFile(path string, options ...SuiteOption) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer f.Close()
	return NewConfigFromReader(f)
}

func (cfg *Config) GenSuites(options ...SuiteOption) ([]*Suite, error) {
	suites := make([]*Suite, len(cfg.Suites))
	for i, scfg := range cfg.Suites {
		path := cfg.getFilePath(scfg.Path)
		suite, err := cfg.genSuite(path, scfg, options...)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		suites[i] = suite
	}
	return suites, nil
}

func (cfg *Config) genSuite(path string, scfg *SuiteConfig, options ...SuiteOption) (*Suite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open suite file: %v", err)
	}
	defer f.Close()
	var opts []SuiteOption
	if scfg.Suite != nil {
		opts = []SuiteOption{baseSuite(scfg.Suite)}
	}
	s, err := NewSuiteFromReader(f, opts...)
	if err != nil {
		return nil, err
	}
	scfg.applyOptions(options...)
	if err := s.applyConfig(scfg); err != nil {
		return nil, err
	}
	return s, nil
}

func (cfg *Config) GenNotifier() notifier.Notifier {
	var notifiers []notifier.Notifier
	for id, ncfg := range cfg.Notifiers {
		n, err := cfg.genNotifierPlugin(id, ncfg)
		if err != nil {
			log.Error("setup notifier plugin", zap.String("id", id), zap.Error(err))
			continue
		}
		notifiers = append(notifiers, n.(notifier.Notifier))
	}
	switch len(notifiers) {
	case 0:
		return nil
	case 1:
		return notifiers[0]
	}
	return notifier.Multi(notifiers...)
}

func (cfg *Config) genNotifierPlugin(id string, b []byte) (notifier.Notifier, error) {
	reg := plugin.Get(plugin.NotifierPlugin, id)
	if reg == nil {
		return nil, errors.New("plugin not found")
	}
	ic := plugin.NewContext(reg, cfg.path)
	if err := json.Unmarshal(b, ic.Config); err != nil {
		return nil, fmt.Errorf("could not decode notifier config: %v", err)
	}
	n, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	return n.(notifier.Notifier), nil
}

func (cfg *Config) getFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cfg.dir, path)
}
