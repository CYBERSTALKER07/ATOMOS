package order

import (
	"context"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultStaleReservationInterval = 1 * time.Minute
	DefaultStaleReservationTTL      = 15 * time.Minute
)

// StartStaleReservationSweeper starts an autonomous background daemon that periodically
// sweeps unpaid PENDING orders that have exceeded their reservation TTL,
// canceling them, restoring inventory reservations, and releasing credit reserves.
func StartStaleReservationSweeper(svc *Service) {
	if svc == nil || svc.spannerClient == nil {
		return
	}
	interval := staleReservationInterval()
	ttl := staleReservationTTL()
	if svc.log != nil {
		svc.log.Info("stale reservation sweeper started", "interval", interval.String(), "ttl", ttl.String())
	}

	go func() {
		// Initial jitter to avoid thundering herd on cluster reboot
		jitter := time.Duration(rand.Int63n(int64(10 * time.Second)))
		time.Sleep(jitter)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			if n, err := svc.RunStaleReservationSweeperOnce(ctx); err != nil {
				if svc.log != nil {
					svc.log.Warn("stale reservation sweeper tick failed", "err", err)
				}
			} else if n > 0 && svc.log != nil {
				svc.log.Info("stale reservation sweeper completed pass", "released_count", n)
			}
			cancel()
		}
	}()
}

// RunStaleReservationSweeperOnce executes a single stale reservation sweeper pass.
func (s *Service) RunStaleReservationSweeperOnce(ctx context.Context) (int, error) {
	ttl := staleReservationTTL()
	return s.ReleaseStaleReservations(ctx, ttl)
}

func staleReservationTTL() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("STALE_RESERVATION_TTL_MINUTES")); raw != "" {
		if m, err := strconv.Atoi(raw); err == nil && m > 0 {
			return time.Duration(m) * time.Minute
		}
	}
	if raw := strings.TrimSpace(os.Getenv("INVENTORY_HOLD_TTL_MINUTES")); raw != "" {
		if m, err := strconv.Atoi(raw); err == nil && m > 0 {
			return time.Duration(m) * time.Minute
		}
	}
	return DefaultStaleReservationTTL
}

func staleReservationInterval() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("STALE_RESERVATION_INTERVAL_SECONDS")); raw != "" {
		if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return DefaultStaleReservationInterval
}
