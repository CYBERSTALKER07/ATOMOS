package order

import (
	"context"
	"math"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/uber/h3-go/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProximityConfig struct {
	H3Resolution      int           // usually 10 or 11 for ~100 m
	MaxDistanceMeters float64       // 100.0
	TelemetryMaxAge   time.Duration // e.g. 90 * time.Second
}

type Location struct {
	Lat float64
	Lng float64
	At  time.Time // when the fix was taken
}

// DriverTelemetry is the standard payload sent by the driver on critical actions.
type DriverTelemetry struct {
	Lat            float64   `json:"lat"`
	Lng            float64   `json:"lng"`
	AccuracyMeters float64   `json:"accuracyMeters"`
	RecordedAt     time.Time `json:"recordedAt"`
}

func (t DriverTelemetry) ToLocation() Location {
	return Location{
		Lat: t.Lat,
		Lng: t.Lng,
		At:  t.RecordedAt,
	}
}

func (t DriverTelemetry) Validate(maxAccuracy float64) error {
	if t.AccuracyMeters > maxAccuracy {
		return status.Errorf(codes.FailedPrecondition, "telemetry accuracy %.1fm exceeds max %.1fm", t.AccuracyMeters, maxAccuracy)
	}
	skew := time.Since(t.RecordedAt)
	if skew < -5*time.Minute || skew > 5*time.Minute {
		return status.Errorf(codes.FailedPrecondition, "telemetry recordedAt is skewed")
	}
	return nil
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000 // Earth radius in meters
	h1 := lat1 * math.Pi / 180
	h2 := lat2 * math.Pi / 180
	dh := (lat2 - lat1) * math.Pi / 180
	dl := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dh/2)*math.Sin(dh/2) +
		math.Cos(h1)*math.Cos(h2)*
			math.Sin(dl/2)*math.Sin(dl/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}

func CheckProximity(
	cfg ProximityConfig,
	driverLoc Location,
	retailerLat, retailerLng float64,
	retailerH3 string, // pre-computed H3 index of the retailer
) (unlocked bool, method string, err error) {

	// 1. Freshness check
	if time.Since(driverLoc.At) > cfg.TelemetryMaxAge {
		return false, "", status.Errorf(codes.FailedPrecondition, "telemetry too old")
	}

	// 2. H3 match (fast path)
	driverH3, err := h3.LatLngToCell(h3.NewLatLng(driverLoc.Lat, driverLoc.Lng), cfg.H3Resolution)
	if err == nil && driverH3.String() == retailerH3 {
		return true, "H3", nil
	}

	// 3. Distance fallback
	dist := haversineMeters(driverLoc.Lat, driverLoc.Lng, retailerLat, retailerLng)
	if dist <= cfg.MaxDistanceMeters {
		return true, "GEOFENCE_100M", nil
	}

	return false, "", nil
}

func ptr[T any](v T) *T {
	return &v
}

func (s *Service) ensureProximityUnlocked(ctx context.Context, txn *spanner.ReadWriteTransaction, order *Order, driverLoc Location, opts TransitionOpts) error {
	if order.ProximityUnlockedAt != nil {
		return nil // already unlocked
	}

	if opts.SkipProximity {
		if opts.SupervisorToken == "" {
			return status.Error(codes.PermissionDenied, "cannot skip proximity without supervisor token")
		}
		// In a real app we'd verify opts.SupervisorToken or s.auth.CanForceProximity(ctx, opts.Actor)
		
		order.ProximityUnlockedAt = ptr(time.Now().UTC())
		order.ProximityMethod = ProximityMethodForceBypass
	} else {
		unlocked, method, err := CheckProximity(s.proximityCfg, driverLoc, order.Lat, order.Lng, order.H3Cell)
		if err != nil {
			return err
		}
		if !unlocked {
			return status.Errorf(codes.FailedPrecondition, "driver not within proximity")
		}

		order.ProximityUnlockedAt = ptr(time.Now().UTC())
		order.ProximityMethod = method
	}

	buf := &spannerTxnBuffer{}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventProximityUnlocked, Timestamp: order.ProximityUnlockedAt.Format(time.RFC3339Nano)},
		OrderID:    order.OrderID,
		DriverID:   order.DriverID,
		RetailerID: order.RetailerID,
		SupplierID: order.SupplierID,
		Status:     order.ProximityMethod,
	}); err != nil {
		return err
	}

	mutations := []*spanner.Mutation{
		spanner.UpdateMap("Orders", map[string]any{
			"OrderId":             order.OrderID,
			"ProximityUnlockedAt": *order.ProximityUnlockedAt,
			"ProximityMethod":     order.ProximityMethod,
		}),
	}
	
	for _, e := range buf.events {
		mutations = append(mutations, outboxMutation(e))
	}

	if err := txn.BufferWrite(mutations); err != nil {
		return err
	}

	return nil
}
