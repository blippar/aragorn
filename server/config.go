package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go4.org/errorutil"
)

type Config struct {
	Plugins  map[string]json.RawMessage
	Services map[string]ServiceConfig
}

type ServiceConfig struct {
	Path     string
	URL      string
	RunEvery string // parsable by time.ParseDuration
	RunCron  string // cron string
	FailFast bool   // stop after first test failure
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
