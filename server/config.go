package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go4.org/errorutil"

	"github.com/blippar/aragorn/log"
	"github.com/blippar/aragorn/notifier"
	"github.com/blippar/aragorn/plugin"
)

type Config struct {
	Notifiers map[string]json.RawMessage
	Suites    []*SuiteConfig
}

func NewConfigFromReader(r io.Reader) (*Config, error) {
	cfg := &Config{}
	if err := decodeReaderJSON(r, cfg); err != nil {
		return nil, fmt.Errorf("could not decode config: %v", jsonDecodeError(r, err))
	}
	return cfg, nil
}

func NewConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %v", err)
	}
	defer f.Close()
	return NewConfigFromReader(f)
}

func (cfg *Config) Notifier() notifier.Notifier {
	var notifiers []notifier.Notifier
	for id, ncfg := range cfg.Notifiers {
		n, err := genNotifierPlugin(id, ncfg)
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

func (cfg *Config) GenSuites(failfast bool) ([]*Suite, error) {
	suites := make([]*Suite, len(cfg.Suites))
	for i, scfg := range cfg.Suites {
		suite, err := scfg.GenSuite(failfast)
		if err != nil {
			return nil, err
		}
		suites[i] = suite
	}
	return suites, nil
}

func genNotifierPlugin(id string, cfg []byte) (notifier.Notifier, error) {
	reg := plugin.Get(plugin.NotifierPlugin, id)
	if reg == nil {
		return nil, errors.New("plugin not found")
	}
	ic := plugin.NewContext(reg, "")
	if err := decodeJSON(cfg, ic.Config); err != nil {
		return nil, fmt.Errorf("could not decode notifier config: %v", err)
	}
	n, err := reg.Init(ic)
	if err != nil {
		return nil, err
	}
	return n.(notifier.Notifier), nil
}

type duration time.Duration

// UnmarshalJSON unmarshals b from string if double quotes around e.g. "1m10s" or an int in nanosecond.
func (d *duration) UnmarshalJSON(b []byte) error {
	if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
		sd := string(b[1 : len(b)-1])
		dur, err := time.ParseDuration(sd)
		*d = duration(dur)
		return err
	}
	dur, err := strconv.ParseInt(string(b), 10, 64)
	*d = duration(dur)
	return err
}

type namer interface {
	Name() string
}

func decodeReaderJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(v)
}

func decodeJSON(b []byte, v interface{}) error {
	return decodeReaderJSON(bytes.NewReader(b), v)
}

func jsonDecodeError(r io.Reader, err error) error {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return err
	}
	serr, ok := err.(*json.SyntaxError)
	if !ok {
		return err
	}
	if _, err := rs.Seek(0, os.SEEK_SET); err != nil {
		return fmt.Errorf("seek error: %v", err)
	}
	line, col, highlight := errorutil.HighlightBytePosition(rs, serr.Offset)
	extra := ""
	if n, ok := r.(namer); ok {
		extra = fmt.Sprintf("%s:%d:%d", n.Name(), line, col)
	}
	return fmt.Errorf("%s\nError at line %d, column %d (file offset %d):\n%s", extra, line, col, serr.Offset, highlight)
}
