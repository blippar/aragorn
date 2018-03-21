package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"go4.org/errorutil"
)

type namer interface {
	Name() string
}

// NewEncoder delegates to json.NewEncoder
// It is only here so this package can be a drop-in for common encoding/json uses
func NewEncoder(w io.Writer) *json.Encoder {
	return json.NewEncoder(w)
}

// Marshal delegates to json.Marshal
// It is only here so this package can be a drop-in for common encoding/json uses
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Decode decodes the reader data.
func Decode(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	return enhanceError(r, decoder.Decode(v))
}

// Unmarshal unmarshals the given data.
func Unmarshal(b []byte, v interface{}) error {
	return Decode(bytes.NewReader(b), v)
}

func enhanceError(r io.Reader, err error) error {
	if err == nil {
		return nil
	}
	var offset int64
	switch serr := err.(type) {
	case *json.SyntaxError:
		offset = serr.Offset
	case *json.UnmarshalTypeError:
		offset = serr.Offset
	default:
		return err
	}
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return fmt.Errorf("%v (offset %d)", err, offset)
	}
	if _, serr := rs.Seek(0, io.SeekStart); serr != nil {
		return fmt.Errorf("%v: seek error: %v", err, serr)
	}
	line, col, highlight := errorutil.HighlightBytePosition(rs, offset)
	extra := ""
	if n, ok := r.(namer); ok {
		extra = fmt.Sprintf("\n%s:%d:%d", n.Name(), line, col)
	}
	return fmt.Errorf("%v%s\nError at line %d, column %d (file offset %d):\n%s", err, extra, line, col, offset, highlight)
}
