package outbox

import "os"

const defaultMaxRelayRetries = 10

// MaxRelayRetries is the number of publish attempts before an event moves to OutboxDLQ.
func MaxRelayRetries() int {
	if v := os.Getenv("OUTBOX_MAX_RETRIES"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			return n
		}
	}
	return defaultMaxRelayRetries
}

func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, errInvalidInt
	}
	return n, nil
}

var errInvalidInt = errorString("invalid int")

type errorString string

func (e errorString) Error() string { return string(e) }
