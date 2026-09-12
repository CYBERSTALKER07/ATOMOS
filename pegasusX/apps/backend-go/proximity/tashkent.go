package proximity

import (
	"context"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// TashkentLocation is the IANA load for Asia/Tashkent (UZ pack timezone).
// Product calendar law is PackLocation / TimezoneFromContext (GS-M4).
var TashkentLocation = mustLoadLocation("Asia/Tashkent")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("UZT", 5*60*60)
	}
	return loc
}

// PackLocation is the shipped-pack operational timezone (env pack when no JWT).
func PackLocation() *time.Location {
	loc, err := auth.TimezoneFromContext(context.Background(), "")
	if err != nil {
		return nil
	}
	return loc
}

// PackNow returns now in the shipped pack timezone. UTC if the pack is planned.
func PackNow() time.Time {
	if loc := PackLocation(); loc != nil {
		return time.Now().In(loc)
	}
	return time.Now().UTC()
}

// PackTodayStart returns midnight today in the shipped pack timezone.
func PackTodayStart(now time.Time) time.Time {
	loc := PackLocation()
	if loc == nil {
		return time.Time{}
	}
	return TodayStart(now, loc)
}

// TashkentNow is PackNow (GS-M4 — no silent Tashkent when the pack is planned).
func TashkentNow() time.Time {
	return PackNow()
}

// TashkentTodayStart is PackTodayStart.
func TashkentTodayStart(now time.Time) time.Time {
	return PackTodayStart(now)
}
