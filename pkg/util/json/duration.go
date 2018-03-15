package json

import (
	"strconv"
	"time"
)

// A Duration represents the elapsed time between two instants.
type Duration time.Duration

// UnmarshalJSON unmarshals b from string if double quotes around e.g. "1m10s" or an int in nanosecond.
func (d *Duration) UnmarshalJSON(b []byte) error {
	if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
		sd := string(b[1 : len(b)-1])
		dur, err := time.ParseDuration(sd)
		*d = Duration(dur)
		return err
	}
	dur, err := strconv.ParseInt(string(b), 10, 64)
	*d = Duration(dur)
	return err
}
