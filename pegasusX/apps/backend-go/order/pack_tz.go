package order

import (
	"context"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// resolveCalendarLocation prefers a stored IANA timezone, else the shipped pack (GS-M4).
func resolveCalendarLocation(ctx context.Context, supplierID, storedTZ string) (*time.Location, error) {
	if tz := strings.TrimSpace(storedTZ); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc, nil
		}
	}
	return auth.TimezoneFromContext(ctx, supplierID)
}
