// Receiving-window parsing for retailers. Spanner has no TIME scalar so the
// canonical wire shape is "HH:MM" 24-hour stored as STRING(5). Every callsite
// (mobile clients, admin portal, dispatcher) agrees on this shape.
package proximity

import (
	"errors"
	"strconv"
	"strings"
)

// ErrInvalidReceivingWindow is returned when a retailer-supplied window string
// fails HH:MM 24-hour validation. Handlers should surface this as 400.
var ErrInvalidReceivingWindow = errors.New("invalid receiving window: expected HH:MM 24-hour format")

// ValidateReceivingWindow checks that s is a valid 24-hour "HH:MM" string and
// returns the canonical zero-padded form. An empty input is treated as "not
// provided" and returned as ("", nil) so callers can keep optional semantics.
func ValidateReceivingWindow(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", ErrInvalidReceivingWindow
	}
	hh, err := strconv.Atoi(parts[0])
	if err != nil || hh < 0 || hh > 23 {
		return "", ErrInvalidReceivingWindow
	}
	mm, err := strconv.Atoi(parts[1])
	if err != nil || mm < 0 || mm > 59 {
		return "", ErrInvalidReceivingWindow
	}
	if len(parts[0]) == 1 {
		parts[0] = "0" + parts[0]
	}
	if len(parts[1]) == 1 {
		parts[1] = "0" + parts[1]
	}
	return parts[0] + ":" + parts[1], nil
}
