package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/redis/go-redis/v9"
)

// saveDriverLocationLua atomically compares the incoming timestamp (in nanoseconds)
// against the stored timestamp key ({driver:<id>}:last_location_ts) on the same
// Redis Cluster slot and only updates {driver:<id>}:last_location when the incoming
// telemetry point is newer or equal, preventing TOCTOU out-of-order GPS overwrites.
var saveDriverLocationLua = redis.NewScript(`
local locKey = KEYS[1]
local tsKey = KEYS[2]
local payload = ARGV[1]
local incomingNanos = tonumber(ARGV[2])
local receivedNanos = tonumber(ARGV[3])
local ttlMs = tonumber(ARGV[4])

local currentMeta = redis.call('GET', tsKey)
if currentMeta then
    local sep = string.find(currentMeta, ":")
    if sep then
        local existingNanos = tonumber(string.sub(currentMeta, 1, sep - 1))
        local existingRecv = tonumber(string.sub(currentMeta, sep + 1))
        if existingNanos and incomingNanos > 0 and existingNanos > 0 then
            if incomingNanos < existingNanos then
                return 0
            end
            if incomingNanos == existingNanos and existingRecv and receivedNanos > 0 and receivedNanos < existingRecv then
                return 0
            end
        end
    end
end

redis.call('SET', locKey, payload, 'PX', ttlMs)
redis.call('SET', tsKey, tostring(incomingNanos) .. ":" .. tostring(receivedNanos), 'PX', ttlMs)
return 1
`)

const (
	lastDriverLocationKeyPrefix = "telemetry:driver:last_location:"
	defaultStaleAfterSeconds    = 30
)

// DefaultLastLocationTTL bounds live driver location cache residency.
const DefaultLastLocationTTL = 2 * time.Minute

// LastLocationWriter persists the latest authenticated driver location.
type LastLocationWriter interface {
	SaveDriverLocation(ctx context.Context, location DriverLocation) error
}

// LastLocationReader resolves the latest cached location for one driver.
type LastLocationReader interface {
	GetDriverLocation(ctx context.Context, driverID string) (DriverLocation, bool, error)
}

// LastLocationStore supports both live telemetry writes and scoped tracking reads.
type LastLocationStore interface {
	LastLocationWriter
	LastLocationReader
}

// DriverLocation is the cache and tracking projection for a driver's last point.
type DriverLocation struct {
	DriverID          string    `json:"driver_id"`
	SupplierID        string    `json:"supplier_id"`
	Lat               float64   `json:"lat"`
	Lng               float64   `json:"lng"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	Velocity          *float64  `json:"velocity,omitempty"`
	Heading           *float64  `json:"heading,omitempty"`
	Humidity          *float64  `json:"humidity,omitempty"`
	ShockG            *float64  `json:"shock_g,omitempty"`
	Tilt              *float64  `json:"tilt,omitempty"`
	Tampered          *bool     `json:"tampered,omitempty"`
	ReportedAt        time.Time `json:"reported_at"`
	ReceivedAt        time.Time `json:"received_at"`
	StaleAfterSeconds int       `json:"stale_after_seconds"`
}

// IsLive reports whether the location is fresh enough for retailer tracking.
func (l DriverLocation) IsLive(now time.Time) bool {
	if strings.TrimSpace(l.DriverID) == "" || strings.TrimSpace(l.SupplierID) == "" || l.ReceivedAt.IsZero() {
		return false
	}
	staleAfter := l.StaleAfterSeconds
	if staleAfter <= 0 {
		staleAfter = defaultStaleAfterSeconds
	}
	age := now.UTC().Sub(l.ReceivedAt.UTC())
	if age < 0 {
		return true
	}
	return age <= time.Duration(staleAfter)*time.Second
}

// CacheLastLocationStore stores last locations in Redis-compatible cache.
type CacheLastLocationStore struct {
	cache *cache.Cache
	ttl   time.Duration
}

// NewCacheLastLocationStore constructs a cache-backed last-location store.
func NewCacheLastLocationStore(cacheClient *cache.Cache, ttl time.Duration) *CacheLastLocationStore {
	if ttl <= 0 {
		ttl = DefaultLastLocationTTL
	}
	return &CacheLastLocationStore{cache: cacheClient, ttl: ttl}
}

// SaveDriverLocation persists the latest driver point with a bounded TTL.
func (s *CacheLastLocationStore) SaveDriverLocation(ctx context.Context, location DriverLocation) error {
	if s == nil || s.cache == nil {
		return nil
	}
	location = normalizeDriverLocation(location)
	if strings.TrimSpace(location.DriverID) == "" {
		return fmt.Errorf("save driver location: driver_id required")
	}
	raw, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("marshal driver location %s: %w", location.DriverID, err)
	}
	if client := extractRedisClient(s.cache); client != nil {
		locKey := lastDriverLocationKey(location.DriverID)
		tsKey := locKey + "_ts"
		orderTS := locationOrderingTimestamp(location)
		var orderNanos, recvNanos int64
		if !orderTS.IsZero() {
			orderNanos = orderTS.UnixNano()
		}
		if !location.ReceivedAt.IsZero() {
			recvNanos = location.ReceivedAt.UTC().UnixNano()
		}
		ttlMs := s.ttl.Milliseconds()
		if ttlMs <= 0 {
			ttlMs = DefaultLastLocationTTL.Milliseconds()
		}
		if luaErr := saveDriverLocationLua.Run(ctx, client, []string{locKey, tsKey}, raw, orderNanos, recvNanos, ttlMs).Err(); luaErr == nil {
			return nil
		}
	}
	existing, found, err := s.GetDriverLocation(ctx, location.DriverID)
	if err != nil {
		return fmt.Errorf("read existing driver location %s: %w", location.DriverID, err)
	}
	if found && shouldKeepExistingLocation(existing, location) {
		return nil
	}
	return s.cache.Set(ctx, lastDriverLocationKey(location.DriverID), raw, s.ttl)
}

func extractRedisClient(c *cache.Cache) *redis.Client {
	if c == nil || c.Backend() == nil {
		return nil
	}
	if cg, ok := c.Backend().(interface{ Client() *redis.Client }); ok {
		return cg.Client()
	}
	return nil
}

// GetDriverLocation reads one driver's latest cached location.
func (s *CacheLastLocationStore) GetDriverLocation(ctx context.Context, driverID string) (DriverLocation, bool, error) {
	if s == nil || s.cache == nil {
		return DriverLocation{}, false, nil
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return DriverLocation{}, false, nil
	}
	raw, found, err := s.cache.Get(ctx, lastDriverLocationKey(driverID))
	if err != nil || !found {
		return DriverLocation{}, found, err
	}
	var location DriverLocation
	if err := json.Unmarshal(raw, &location); err != nil {
		return DriverLocation{}, false, fmt.Errorf("decode driver location %s: %w", driverID, err)
	}
	return normalizeDriverLocation(location), true, nil
}

func normalizeDriverLocation(location DriverLocation) DriverLocation {
	location.DriverID = strings.TrimSpace(location.DriverID)
	location.SupplierID = strings.TrimSpace(location.SupplierID)
	if !location.ReportedAt.IsZero() {
		location.ReportedAt = location.ReportedAt.UTC()
	}
	if !location.ReceivedAt.IsZero() {
		location.ReceivedAt = location.ReceivedAt.UTC()
	}
	if location.ReportedAt.IsZero() && !location.ReceivedAt.IsZero() {
		location.ReportedAt = location.ReceivedAt
	}
	if location.ReceivedAt.IsZero() && !location.ReportedAt.IsZero() {
		location.ReceivedAt = location.ReportedAt
	}
	if location.Latitude == 0 {
		location.Latitude = location.Lat
	}
	if location.Longitude == 0 {
		location.Longitude = location.Lng
	}
	if location.Lat == 0 {
		location.Lat = location.Latitude
	}
	if location.Lng == 0 {
		location.Lng = location.Longitude
	}
	if location.StaleAfterSeconds <= 0 {
		location.StaleAfterSeconds = defaultStaleAfterSeconds
	}
	return location
}

func lastDriverLocationKey(driverID string) string {
	return fmt.Sprintf("{driver:%s}:last_location", strings.TrimSpace(driverID))
}

func shouldKeepExistingLocation(existing, incoming DriverLocation) bool {
	existingAt := locationOrderingTimestamp(existing)
	incomingAt := locationOrderingTimestamp(incoming)
	if existingAt.IsZero() || incomingAt.IsZero() {
		return false
	}
	if incomingAt.Before(existingAt) {
		return true
	}
	if incomingAt.Equal(existingAt) && !existing.ReceivedAt.IsZero() && !incoming.ReceivedAt.IsZero() {
		return incoming.ReceivedAt.Before(existing.ReceivedAt)
	}
	return false
}

func locationOrderingTimestamp(location DriverLocation) time.Time {
	if !location.ReportedAt.IsZero() {
		return location.ReportedAt.UTC()
	}
	if !location.ReceivedAt.IsZero() {
		return location.ReceivedAt.UTC()
	}
	return time.Time{}
}
