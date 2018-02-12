package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"go4.org/errorutil"
)

type Config struct {
	Notifiers map[string]json.RawMessage
	Suites    []SuiteConfig
	// StartupDelay duration
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
