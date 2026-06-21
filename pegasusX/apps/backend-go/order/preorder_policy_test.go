package order

import (
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func TestPreorderPromoteAtTMinusOne(t *testing.T) {
	loc := proximity.TashkentLocation
	delivery := time.Date(2026, 6, 20, 0, 0, 0, 0, loc)

	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"T-2 no promote", time.Date(2026, 6, 18, 12, 0, 0, 0, loc), false},
		{"T-1 promote", time.Date(2026, 6, 19, 0, 0, 0, 0, loc), true},
		{"delivery day promote", time.Date(2026, 6, 20, 8, 0, 0, 0, loc), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lead := PreorderLeadDays(tc.now, &delivery)
			promote := lead <= 1
			if promote != tc.want {
				t.Fatalf("lead=%d promote=%v want %v", lead, promote, tc.want)
			}
		})
	}
}
