package outbox

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
)

// SupplierBackfillEnabled reports whether the Gate 5 outbox SupplierId backfill loop should run.
// Default on for PEGASUSX_ENV=ssmr|production; override with OUTBOX_SUPPLIER_BACKFILL=0/1.
func SupplierBackfillEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("OUTBOX_SUPPLIER_BACKFILL")); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")))
	return env == "ssmr" || env == "production"
}

func supplierBackfillLimit() int {
	n := 200
	if raw := strings.TrimSpace(os.Getenv("OUTBOX_SUPPLIER_BACKFILL_LIMIT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			n = v
		}
	}
	return n
}

func supplierBackfillInterval() time.Duration {
	d := 5 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("OUTBOX_SUPPLIER_BACKFILL_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			d = parsed
		}
	}
	return d
}

// StartSupplierIDBackfill runs BackfillSupplierID once immediately, then on an interval
// until ctx is cancelled. Missing SupplierId column is treated as a soft skip.
func StartSupplierIDBackfill(ctx context.Context, client *spanner.Client, log *slog.Logger) {
	if client == nil || !SupplierBackfillEnabled() {
		return
	}
	if log == nil {
		log = slog.Default()
	}
	run := func() {
		n, err := BackfillSupplierID(ctx, client, supplierBackfillLimit())
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "SupplierId") || strings.Contains(msg, "not found") || strings.Contains(msg, "Unrecognized name") {
				log.Info("outbox supplier backfill skipped (column missing)", "err", err)
				return
			}
			log.Warn("outbox supplier backfill failed", "err", err)
			return
		}
		if n > 0 {
			log.Info("outbox supplier backfill stamped rows", "updated", n)
		}
	}
	run()
	ticker := time.NewTicker(supplierBackfillInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
