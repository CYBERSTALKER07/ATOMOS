package proximity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ─── Redis Key Schema ─────────────────────────────────────────────────────────
const (
	GeoKey              = "geo:proximity"      // Single sorted set for GEODIST between driver ↔ retailer — matches cache.KeyGeoProximity
	ArrivingKey         = "proximity:arriving" // SET of order IDs already processed for approach alerts — matches cache.KeyArrivingSet
	DefaultBreachRadius = 100.0                // meters — fallback when config service unavailable
	// Keep local constant in proximity to avoid a package cycle through kafka/telemetry.
	// Must stay aligned with kafka/events.go:EventDriverApproaching.
	eventDriverApproaching = "DRIVER_APPROACHING"
)

// ─── Internal Projections ──────────────────────────────────────────────────────

// transitOrder is a lightweight projection of an IN_TRANSIT order needed for proximity evaluation.
type transitOrder struct {
	OrderID       string
	RetailerID    string
	SupplierID    string
	SupplierName  string
	DeliveryToken string
	ShopLat       float64
	ShopLng       float64
	SequenceIndex int64 // Route stop index — used to prevent drive-by false positives
	RouteID       string
}

// DriverApproachingEvent is the Kafka payload fired when a driver breaches the 100m perimeter.
type DriverApproachingEvent struct {
	OrderID         string    `json:"order_id"`
	SupplierID      string    `json:"supplier_id"`
	SupplierName    string    `json:"supplier_name"`
	RetailerID      string    `json:"retailer_id"`
	DeliveryToken   string    `json:"delivery_token"`
	DriverLatitude  float64   `json:"driver_latitude"`
	DriverLongitude float64   `json:"driver_longitude"`
	Timestamp       time.Time `json:"timestamp"`
}

// ─── Engine ────────────────────────────────────────────────────────────────────

// Engine is the Redis GEO proximity detector.
// Nil-safe on Redis: if Redis is nil, all operations degrade silently.
type Engine struct {
	Redis     redis.UniversalClient
	Spanner   *spanner.Client
	Producer  *kafka.Writer
	ConfigSvc ConfigProvider // Phase I: config-driven breach radius (nil = use DefaultBreachRadius)
}

// ConfigProvider abstracts the country config service for proximity decisions.
type ConfigProvider interface {
	GetBreachRadius(ctx context.Context, supplierID, countryCode string) float64
}

// ProcessPing is the main entry point — called by the telemetry WebSocket hub
// on every driver GPS update. Runs in a goroutine so it never blocks the WS read loop.
func (e *Engine) ProcessPing(ctx context.Context, driverID string, lat, lng float64) {
	if e.Redis == nil {
		return // degraded mode — proximity detection offline
	}

	// 1. GEOADD driver location into Redis GEO sorted set
	if err := e.Redis.GeoAdd(ctx, GeoKey, &redis.GeoLocation{
		Name:      driverKey(driverID),
		Longitude: lng,
		Latitude:  lat,
	}).Err(); err != nil {
		log.Printf("[PROXIMITY] GEOADD failed for driver %s: %v", driverID, err)
		return
	}

	// 2. Fetch active IN_TRANSIT orders assigned to this driver (indexed Spanner read via IDX_Orders_DriverId)
	orders, err := e.activeTransitOrders(ctx, driverID)
	if err != nil {
		log.Printf("[PROXIMITY] Failed to fetch transit orders for driver %s: %v", driverID, err)
		return
	}

	if len(orders) == 0 {
		return // No active deliveries — stand down
	}

	// 3. Evaluate breach for each active order
	// Determine current stop index: the minimum SequenceIndex among all transit orders
	// on this route to prevent drive-by false positives on future stops.
	currentStopIdx := int64(-1)
	if len(orders) > 0 {
		currentStopIdx = orders[0].SequenceIndex
		for _, o := range orders[1:] {
			if o.SequenceIndex >= 0 && (currentStopIdx < 0 || o.SequenceIndex < currentStopIdx) {
				currentStopIdx = o.SequenceIndex
			}
		}
	}

	for i := range orders {
		e.evaluateBreach(ctx, driverID, lat, lng, &orders[i], currentStopIdx)
	}
}

// activeTransitOrders queries Spanner for all orders assigned to this driver
// that are currently IN_TRANSIT.
func (e *Engine) activeTransitOrders(ctx context.Context, driverID string) ([]transitOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.RetailerId, o.SupplierId, o.DeliveryToken, o.ShopLocation,
		             COALESCE(s.Name, '') AS SupplierName,
		             COALESCE(o.SequenceIndex, -1) AS SequenceIndex,
		             COALESCE(o.RouteId, '') AS RouteId
		      FROM Orders o
		      LEFT JOIN Suppliers s ON o.SupplierId = s.SupplierId
		      WHERE o.DriverId = @driverId AND o.State = 'IN_TRANSIT'`,
		Params: map[string]interface{}{
			"driverId": driverID,
		},
	}

	iter := e.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []transitOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("spanner query failed: %w", err)
		}

		var orderID, retailerID string
		var supplierID, deliveryToken, shopLocation, supplierName spanner.NullString
		var sequenceIndex spanner.NullInt64
		var routeID string

		if err := row.Columns(&orderID, &retailerID, &supplierID, &deliveryToken, &shopLocation, &supplierName, &sequenceIndex, &routeID); err != nil {
			return nil, fmt.Errorf("column parse failed: %w", err)
		}

		if !shopLocation.Valid {
			continue // No shop coordinates — cannot run proximity check
		}

		lat, lng, err := parseWKT(shopLocation.StringVal)
		if err != nil {
			log.Printf("[PROXIMITY] Bad WKT for order %s: %v", orderID, err)
			continue
		}

		results = append(results, transitOrder{
			OrderID:       orderID,
			RetailerID:    retailerID,
			SupplierID:    supplierID.StringVal,
			SupplierName:  supplierName.StringVal,
			DeliveryToken: deliveryToken.StringVal,
			ShopLat:       lat,
			ShopLng:       lng,
			SequenceIndex: sequenceIndex.Int64,
			RouteID:       routeID,
		})
	}

	return results, nil
}

// evaluateBreach runs the Redis GEO tripwire for a single order.
// currentStopIdx is the lowest SequenceIndex among the driver's active transit orders.
// Only fires for the current/next stop to prevent drive-by false positives.
func (e *Engine) evaluateBreach(ctx context.Context, driverID string, driverLat, driverLng float64, ord *transitOrder, currentStopIdx int64) {
	// ─── SEQUENCE GUARD: prevent drive-by false positives ───────────────────
	// Only trigger approaching for the current stop (lowest sequence index).
	// If the driver is passing a future stop (e.g., stop #5 while on stop #2),
	// we skip silently — the event will fire when it's actually that stop's turn.
	if ord.SequenceIndex >= 0 && currentStopIdx >= 0 && ord.SequenceIndex > currentStopIdx {
		return // Future stop — not the driver's current target
	}

	// 1. Seed retailer location into the same GEO key (idempotent — GEOADD overwrites same member)
	if err := e.Redis.GeoAdd(ctx, GeoKey, &redis.GeoLocation{
		Name:      retailerKey(ord.RetailerID),
		Longitude: ord.ShopLng,
		Latitude:  ord.ShopLat,
	}).Err(); err != nil {
		log.Printf("[PROXIMITY] Failed to seed retailer %s location: %v", ord.RetailerID, err)
		return
	}

	// 2. GEODIST — single key, two members, result in meters
	dist, err := e.Redis.GeoDist(ctx, GeoKey, driverKey(driverID), retailerKey(ord.RetailerID), "m").Result()
	if err != nil {
		log.Printf("[PROXIMITY] GEODIST failed for driver=%s retailer=%s: %v — falling back to Haversine", driverID, ord.RetailerID, err)
		dist = haversineMeters(driverLat, driverLng, ord.ShopLat, ord.ShopLng)
	}

	// Resolve breach radius from config (per-supplier, per-country) or use default
	breachRadius := DefaultBreachRadius
	if e.ConfigSvc != nil {
		breachRadius = e.ConfigSvc.GetBreachRadius(ctx, ord.SupplierID, "")
	}
	if dist > breachRadius {
		return // Outside the perimeter — no action
	}

	// ─── BREACH DETECTED ───────────────────────────────────────────────────

	// 3. Idempotency gate — check Redis SET before touching Spanner
	alreadyArriving, err := e.Redis.SIsMember(ctx, ArrivingKey, ord.OrderID).Result()
	if err != nil {
		log.Printf("[PROXIMITY] Redis SISMEMBER check failed for order %s: %v", ord.OrderID, err)
		return
	}
	if alreadyArriving {
		return // Already processed — do NOT spam Spanner
	}

	log.Printf("[PROXIMITY] BREACH — Driver %s within %.0fm of retailer %s (order: %s)",
		driverID, dist, ord.RetailerID, ord.OrderID)

	// 4. Mark in Redis so we only emit one DRIVER_APPROACHING event per order.
	e.Redis.SAdd(ctx, ArrivingKey, ord.OrderID)
	// Set TTL on the arriving set to prevent unbounded memory growth.
	// 24h is generous — orders should be completed well within that window.
	e.Redis.Expire(ctx, ArrivingKey, 24*time.Hour)

	// 5. Fire DRIVER_APPROACHING as a best-effort real-time signal.
	// Intentionally NOT outbox-backed: this path does not mutate durable state,
	// is already idempotency-gated by Redis SET membership above, and should not
	// add Spanner transaction overhead to high-frequency telemetry pings.
	go e.emitApproachingEvent(context.WithoutCancel(ctx), ord, driverLat, driverLng)

	log.Printf("[PROXIMITY] Order %s breached the perimeter. DRIVER_APPROACHING dispatched.", ord.OrderID)
}

// emitApproachingEvent publishes DRIVER_APPROACHING to Kafka as best-effort UX fan-out.
func (e *Engine) emitApproachingEvent(ctx context.Context, ord *transitOrder, driverLat, driverLng float64) {
	if e.Producer == nil {
		log.Printf("[PROXIMITY] Kafka producer unavailable; skipping %s for order %s", eventDriverApproaching, ord.OrderID)
		return
	}

	event := DriverApproachingEvent{
		OrderID:         ord.OrderID,
		SupplierID:      ord.SupplierID,
		SupplierName:    ord.SupplierName,
		RetailerID:      ord.RetailerID,
		DeliveryToken:   ord.DeliveryToken,
		DriverLatitude:  driverLat,
		DriverLongitude: driverLng,
		Timestamp:       time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[PROXIMITY] Failed to marshal DRIVER_APPROACHING for order %s: %v", ord.OrderID, err)
		return
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = e.Producer.WriteMessages(publishCtx, kafka.Message{
		Key: []byte(eventDriverApproaching),
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(eventDriverApproaching)},
		},
		Value: payload,
	})
	if err != nil {
		log.Printf("[PROXIMITY] Kafka %s failed for order %s: %v", eventDriverApproaching, ord.OrderID, err)
	} else {
		log.Printf("[PROXIMITY] Kafka %s broadcast for order %s", eventDriverApproaching, ord.OrderID)
	}
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// driverKey and retailerKey delegate to the canonical cache.DriverGeoMember/RetailerGeoMember.
// Kept as package-internal shortcuts to avoid importing cache (import cycle risk).
func driverKey(id string) string   { return "d:" + id }
func retailerKey(id string) string { return "r:" + id }

// parseWKT extracts (latitude, longitude) from a Spanner WKT string: POINT(lng lat)
func parseWKT(wkt string) (float64, float64, error) {
	if len(wkt) < 7 || wkt[:6] != "POINT(" || wkt[len(wkt)-1] != ')' {
		return 0, 0, fmt.Errorf("invalid WKT: %s", wkt)
	}
	parts := strings.Fields(wkt[6 : len(wkt)-1])
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid WKT coordinate count: %s", wkt)
	}
	lng, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %w", err)
	}
	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %w", err)
	}
	return lat, lng, nil
}

// haversineMeters returns the great-circle distance in meters between two lat/lng points.
// Used as a fallback when Redis GEODIST is unavailable.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6_371_000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}
