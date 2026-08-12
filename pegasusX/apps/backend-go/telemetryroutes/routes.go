package telemetryroutes

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

const (
	locationSchemaVersion = 1
	locationStaleAfterSec = 30
)

// DeliveryTokenResolver resolves the active QR handoff token for an order.
type DeliveryTokenResolver interface {
	ResolveDeliveryToken(ctx context.Context, orderID string) (string, error)
}

// ReturnApproachChecker evaluates warehouse depot proximity for inbound returns.
type ReturnApproachChecker interface {
	CheckWarehouseApproach(ctx context.Context, driverID, supplierID string, lat, lng float64) error
}

type Deps struct {
	TelemetryHub        *ws.Hub
	RetailerHub         *ws.Hub
	LastLocations       telemetry.LastLocationWriter
	DeliveryTokens      DeliveryTokenResolver
	ReturnApproach      ReturnApproachChecker
	SupplierID          string
	Log                 *slog.Logger
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	// LocationBusEmitter publishes a throttled DRIVER_LOCATION_UPDATED to the
	// transactional outbox (TopicRealtime) so the notification dispatcher and
	// digital-twin consumers are live. Nil disables bus emit (WS/Redis still run
	// at full fidelity). Wired in main.go from the Spanner outbox store.
	LocationBusEmitter LocationBusEmitter
	// LocationBusInterval throttles bus emits per driver (default 5s). Full
	// fidelity stays on WS/Redis; only the bus copy is throttled.
	LocationBusInterval time.Duration
	// RouteLookup optionally resolves a driver's active route for twin consumers.
	// Nil-safe: events are emitted without route_id when unavailable.
	RouteLookup DriverRouteLookup
}

// LocationBusEmitter emits one outbox event for a driver location update. The
// payload is the canonical location envelope JSON (same bytes fanned out to WS).
type LocationBusEmitter interface {
	EmitDriverLocation(ctx context.Context, supplierID, driverID, routeID string, payload []byte) error
}

// DriverRouteLookup resolves the driver's currently-active route id, if any.
type DriverRouteLookup interface {
	ActiveRouteForDriver(ctx context.Context, driverID string) (string, error)
}

type LocationUpdate struct {
	DriverID           string   `json:"driver_id,omitempty"`
	Lat                *float64 `json:"lat,omitempty"`
	Lng                *float64 `json:"lng,omitempty"`
	Latitude           *float64 `json:"latitude,omitempty"`
	Longitude          *float64 `json:"longitude,omitempty"`
	Velocity           *float64 `json:"velocity,omitempty"`
	Heading            *float64 `json:"heading,omitempty"`
	Timestamp          string   `json:"timestamp,omitempty"`
	NextStopRetailerID string   `json:"next_stop_retailer_id,omitempty"`
	NextStopOrderID    string   `json:"next_stop_order_id,omitempty"`
	NextStopLat        *float64 `json:"next_stop_lat,omitempty"`
	NextStopLng        *float64 `json:"next_stop_lng,omitempty"`
}

type locationIdentity struct {
	DriverID   string
	SupplierID string
}

type driverLocationPayload struct {
	DriverID          string   `json:"driver_id"`
	SupplierID        string   `json:"supplier_id"`
	Lat               float64  `json:"lat"`
	Lng               float64  `json:"lng"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	Velocity          *float64 `json:"velocity,omitempty"`
	Heading           *float64 `json:"heading,omitempty"`
	ReportedAt        string   `json:"reported_at"`
	ReceivedAt        string   `json:"received_at"`
	StaleAfterSeconds int      `json:"stale_after_seconds"`
}

type locationEnvelope struct {
	Type          string                `json:"type"`
	TraceID       string                `json:"trace_id"`
	Timestamp     string                `json:"timestamp"`
	Version       int                   `json:"v"`
	SchemaVersion int                   `json:"schema_version"`
	Data          driverLocationPayload `json:"data"`
}

func RegisterRoutes(r chi.Router, d Deps) {
	if d.TelemetryHub == nil {
		return
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}

	handler := http.HandlerFunc(d.handleLocation)

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.With(auth.FirebaseAuth(d.FirebaseVerifier), auth.RequireRole(auth.RoleDriver)).Post("/v1/telemetry/location", handler)
		return
	}
	r.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/telemetry/location", handler)
}

func (d Deps) handleLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTelemetryJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	identity, err := d.resolveDriverIdentity(r)
	if err != nil {
		d.writeIdentityError(w, err)
		return
	}
	loc, err := decodeLocation(w, r)
	if err != nil {
		writeTelemetryJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	payload, err := buildLocationEnvelope(r.Context(), identity, loc, time.Now().UTC())
	if err != nil {
		writeTelemetryJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		d.Log.Error("telemetry location marshal failed", "driver_id", identity.DriverID, "err", err)
		writeTelemetryJSON(w, http.StatusInternalServerError, map[string]string{"error": "telemetry_marshal_failed"})
		return
	}
	if d.LastLocations != nil {
		if err := d.LastLocations.SaveDriverLocation(r.Context(), driverLocationFromEnvelope(payload)); err != nil {
			d.Log.Warn("telemetry last location save failed", "driver_id", identity.DriverID, "err", err)
		}
	}
	d.emitLocationToBus(r.Context(), identity, raw)
	for _, room := range telemetryRooms(identity) {
		d.TelemetryHub.Broadcast(r.Context(), room, raw)
	}

	if loc.NextStopRetailerID != "" && loc.NextStopLat != nil && loc.NextStopLng != nil && payload.Data.Lat != 0 && payload.Data.Lng != 0 {
		dist := proximity.HaversineDistance(payload.Data.Lat, payload.Data.Lng, *loc.NextStopLat, *loc.NextStopLng)
		if proximity.WithinDeliveryApproach(dist) {
			arrivalPayload, _ := json.Marshal(map[string]any{
				"type":        "DELIVERY_ARRIVING",
				"driver_id":   identity.DriverID,
				"order_id":    loc.NextStopOrderID,
				"retailer_id": loc.NextStopRetailerID,
				"distance_km": dist,
			})
			d.TelemetryHub.Broadcast(r.Context(), "retailer:"+loc.NextStopRetailerID, arrivalPayload)

			if d.RetailerHub != nil && strings.TrimSpace(loc.NextStopOrderID) != "" {
				deliveryToken := strings.TrimSpace(loc.NextStopOrderID)
				if d.DeliveryTokens != nil {
					if resolved, err := d.DeliveryTokens.ResolveDeliveryToken(r.Context(), loc.NextStopOrderID); err == nil && strings.TrimSpace(resolved) != "" {
						deliveryToken = strings.TrimSpace(resolved)
					}
				}
				approachPayload, _ := json.Marshal(map[string]any{
					"type":             "DRIVER_APPROACHING",
					"order_id":         loc.NextStopOrderID,
					"retailer_id":      loc.NextStopRetailerID,
					"driver_id":        identity.DriverID,
					"delivery_token":   deliveryToken,
					"driver_latitude":  payload.Data.Lat,
					"driver_longitude": payload.Data.Lng,
					"distance_km":      dist,
				})
				d.RetailerHub.Broadcast(r.Context(), "retailer:"+loc.NextStopRetailerID, approachPayload)
			}
		}
	}

	if d.ReturnApproach != nil && payload.Data.Lat != 0 && payload.Data.Lng != 0 {
		if err := d.ReturnApproach.CheckWarehouseApproach(
			r.Context(),
			identity.DriverID,
			identity.SupplierID,
			payload.Data.Lat,
			payload.Data.Lng,
		); err != nil {
			d.Log.Warn("warehouse return approach check failed", "driver_id", identity.DriverID, "err", err)
		}
	}

	writeTelemetryJSON(w, http.StatusAccepted, map[string]any{
		"status":      "accepted",
		"driver_id":   identity.DriverID,
		"supplier_id": identity.SupplierID,
	})
}

// locationBusThrottle tracks the last bus emit per driver so the outbox/Kafka
// copy is throttled while WS/Redis stay full-fidelity.
var locationBusThrottle = struct {
	sync.Mutex
	last map[string]time.Time
}{last: map[string]time.Time{}}

// emitLocationToBus publishes a throttled DRIVER_LOCATION_UPDATED to the outbox
// so dispatcher + twin consumers receive it. Best-effort: a bus failure never
// fails the driver's location request (WS/Redis already delivered full copy).
func (d Deps) emitLocationToBus(ctx context.Context, identity locationIdentity, raw []byte) {
	if d.LocationBusEmitter == nil {
		return
	}
	interval := d.LocationBusInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	locationBusThrottle.Lock()
	last, seen := locationBusThrottle.last[identity.DriverID]
	if seen && time.Since(last) < interval {
		locationBusThrottle.Unlock()
		return
	}
	locationBusThrottle.last[identity.DriverID] = time.Now()
	locationBusThrottle.Unlock()

	routeID := ""
	if d.RouteLookup != nil {
		if rid, err := d.RouteLookup.ActiveRouteForDriver(ctx, identity.DriverID); err == nil {
			routeID = rid
		}
	}
	if err := d.LocationBusEmitter.EmitDriverLocation(ctx, identity.SupplierID, identity.DriverID, routeID, raw); err != nil {
		d.Log.Warn("driver location bus emit failed", "driver_id", identity.DriverID, "err", err)
	}
}

func driverLocationFromEnvelope(envelope locationEnvelope) telemetry.DriverLocation {
	data := envelope.Data
	reportedAt, _ := time.Parse(time.RFC3339Nano, data.ReportedAt)
	receivedAt, _ := time.Parse(time.RFC3339Nano, data.ReceivedAt)
	return telemetry.DriverLocation{
		DriverID:          data.DriverID,
		SupplierID:        data.SupplierID,
		Lat:               data.Lat,
		Lng:               data.Lng,
		Latitude:          data.Latitude,
		Longitude:         data.Longitude,
		Velocity:          data.Velocity,
		Heading:           data.Heading,
		ReportedAt:        reportedAt,
		ReceivedAt:        receivedAt,
		StaleAfterSeconds: data.StaleAfterSeconds,
	}
}

func (d Deps) resolveDriverIdentity(r *http.Request) (locationIdentity, error) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return locationIdentity{}, errTelemetryUnauthorized
	}
	if claims.Role != auth.RoleDriver {
		return locationIdentity{}, errTelemetryForbidden
	}
	driverID := strings.TrimSpace(claims.Subject)
	if driverID == "" {
		return locationIdentity{}, errTelemetryUnauthorized
	}
	supplierID := strings.TrimSpace(claims.SupplierID)
	if supplierID == "" {
		supplierID = strings.TrimSpace(d.SupplierID)
	}
	if supplierID == "" {
		return locationIdentity{}, errTelemetrySupplierMissing
	}
	return locationIdentity{DriverID: driverID, SupplierID: supplierID}, nil
}

func decodeLocation(w http.ResponseWriter, r *http.Request) (LocationUpdate, error) {
	defer r.Body.Close()
	var loc LocationUpdate
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := decoder.Decode(&loc); err != nil {
		return LocationUpdate{}, errors.New("invalid_json")
	}
	return loc, nil
}

func buildLocationEnvelope(ctx context.Context, identity locationIdentity, loc LocationUpdate, receivedAt time.Time) (locationEnvelope, error) {
	lat, lng, err := loc.coordinates()
	if err != nil {
		return locationEnvelope{}, err
	}
	reportedAt, err := parseLocationTimestamp(loc.Timestamp, receivedAt)
	if err != nil {
		return locationEnvelope{}, err
	}
	return locationEnvelope{
		Type:          events.EventDriverLocationUpdated,
		TraceID:       outbox.TraceIDFromContext(ctx),
		Timestamp:     receivedAt.Format(time.RFC3339Nano),
		Version:       1,
		SchemaVersion: locationSchemaVersion,
		Data: driverLocationPayload{
			DriverID:          identity.DriverID,
			SupplierID:        identity.SupplierID,
			Lat:               lat,
			Lng:               lng,
			Latitude:          lat,
			Longitude:         lng,
			Velocity:          loc.Velocity,
			Heading:           loc.Heading,
			ReportedAt:        reportedAt.Format(time.RFC3339Nano),
			ReceivedAt:        receivedAt.Format(time.RFC3339Nano),
			StaleAfterSeconds: locationStaleAfterSec,
		},
	}, nil
}

func (l LocationUpdate) coordinates() (float64, float64, error) {
	lat := firstFloat(l.Lat, l.Latitude)
	lng := firstFloat(l.Lng, l.Longitude)
	if lat == nil || lng == nil {
		return 0, 0, errors.New("latitude_longitude_required")
	}
	if *lat < -90 || *lat > 90 || *lng < -180 || *lng > 180 {
		return 0, 0, errors.New("latitude_longitude_out_of_range")
	}
	return *lat, *lng, nil
}

func firstFloat(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func parseLocationTimestamp(raw string, fallback time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC(), nil
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || value <= 0 {
		return time.Time{}, errors.New("invalid_timestamp")
	}
	if value < 1_000_000_000_000 {
		return time.Unix(value, 0).UTC(), nil
	}
	return time.UnixMilli(value).UTC(), nil
}

func telemetryRooms(identity locationIdentity) []string {
	return []string{
		"telemetry:driver:" + identity.DriverID,
		"telemetry:supplier:" + identity.SupplierID,
	}
}

var (
	errTelemetryUnauthorized    = errors.New("unauthorized")
	errTelemetryForbidden       = errors.New("forbidden")
	errTelemetrySupplierMissing = errors.New("supplier_scope_required")
)

func (d Deps) writeIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errTelemetryForbidden):
		writeTelemetryJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, errTelemetrySupplierMissing):
		writeTelemetryJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
	default:
		writeTelemetryJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
}

func writeTelemetryJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
