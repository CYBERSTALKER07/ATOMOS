package telemetry

import (
	wsEvents "backend-go/ws"
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"strconv"
	"sync"
	"time"
)

// ── GPS Buffer & Flush ──────────────────────────────────────────────────────
// Instead of broadcasting every GPS ping to every admin connection, the buffer
// stores the LATEST position per driver and flushes a single FleetSnapshot to
// supplier-scoped admin connections at a fixed interval.
//
// Impact: reduces WebSocket packets by ~95% (from N pings/sec to 1 snapshot/interval)
// while maintaining real-time feel. Priority events (DRIVER_APPROACHING) bypass
// the buffer entirely.
//
// Configurable via GPS_FLUSH_INTERVAL env var (default: 5s).

// GPSEntry is the latest known position for a single driver.
type GPSEntry struct {
	DriverID   string  `json:"driver_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Timestamp  int64   `json:"timestamp"`
	SupplierID string  `json:"supplier_id,omitempty"`
}

// FleetSnapshot is a batched GPS update sent to admin connections.
type FleetSnapshot struct {
	Type    string     `json:"type"`
	Drivers []GPSEntry `json:"drivers"`
}

// GPSBuffer aggregates GPS pings and flushes them periodically.
type GPSBuffer struct {
	mu       sync.Mutex
	entries  map[string]*GPSEntry // driverID → latest position
	previous map[string]*GPSEntry // driverID → last flushed position (for significant-change filter)
	interval time.Duration
	hub      *Hub
	done     chan struct{}
}

// NewGPSBuffer creates and starts a GPS buffer with automatic flushing.
// The buffer writes to the given Hub's supplier-scoped broadcast.
func NewGPSBuffer(hub *Hub) *GPSBuffer {
	interval := 5 * time.Second
	if envVal := os.Getenv("GPS_FLUSH_INTERVAL"); envVal != "" {
		if secs, err := strconv.Atoi(envVal); err == nil && secs > 0 {
			interval = time.Duration(secs) * time.Second
		}
	}

	buf := &GPSBuffer{
		entries:  make(map[string]*GPSEntry),
		previous: make(map[string]*GPSEntry),
		interval: interval,
		hub:      hub,
		done:     make(chan struct{}),
	}
	go buf.flushLoop()
	slog.Info("gps buffer started",
		"component", "gps_buffer",
		"flush_interval", interval.String(),
	)
	return buf
}

// Ingest stores the latest GPS position for a driver (overwrites previous).
// This is called from the WebSocket read loop instead of broadcasting directly.
func (b *GPSBuffer) Ingest(entry GPSEntry) {
	b.mu.Lock()
	b.entries[entry.DriverID] = &entry
	b.mu.Unlock()
}

// flushLoop runs the periodic flush cycle.
func (b *GPSBuffer) flushLoop() {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			b.flush()
		}
	}
}

// GPSSignificantChangeThreshold is the minimum coordinate delta (in degrees)
// to qualify as a "significant move." ~0.0001° ≈ 11 meters at the equator.
// Positions that haven't moved beyond this threshold are suppressed from the flush,
// reducing WebSocket bandwidth by filtering stationary/parked drivers.
const GPSSignificantChangeThreshold = 0.0001

// flush takes a snapshot of all buffered positions, applies the significant-change
// filter against previous positions, groups by supplier, and broadcasts either a
// legacy FleetSnapshot or a compact DeltaEvent per supplier.
func (b *GPSBuffer) flush() {
	b.mu.Lock()
	if len(b.entries) == 0 {
		b.mu.Unlock()
		return
	}
	// Swap out the map (zero-alloc drain)
	snapshot := b.entries
	b.entries = make(map[string]*GPSEntry, len(snapshot))
	b.mu.Unlock()

	// Filter: only include drivers who moved significantly
	bySupplier := make(map[string][]GPSEntry)
	bySupplierDelta := make(map[string][]map[string]interface{})

	for _, entry := range snapshot {
		prev := b.previous[entry.DriverID]
		if prev != nil && !isSignificantMove(prev, entry) {
			continue // Stationary — suppress
		}

		bySupplier[entry.SupplierID] = append(bySupplier[entry.SupplierID], *entry)
		bySupplierDelta[entry.SupplierID] = append(bySupplierDelta[entry.SupplierID], map[string]interface{}{
			"d": entry.DriverID,
			"l": []float64{entry.Latitude, entry.Longitude},
		})

		// Update previous position for next cycle
		b.previous[entry.DriverID] = entry
	}

	totalDrivers := 0
	totalSuppliers := 0
	for supplierID, drivers := range bySupplier {
		if supplierID == "" {
			continue
		}

		// Emit both legacy FleetSnapshot (backward compat) and compact delta
		legacyMsg, err := json.Marshal(FleetSnapshot{
			Type:    "FLEET_SNAPSHOT",
			Drivers: drivers,
		})
		if err != nil {
			continue
		}
		b.hub.BroadcastToSupplier(supplierID, legacyMsg)

		// Also emit compact FLT_GPS delta (for clients that support delta-sync)
		deltaEvent := wsEvents.NewDelta(wsEvents.DeltaFleetGPS, supplierID, map[string]interface{}{
			"drivers": bySupplierDelta[supplierID],
		})
		deltaMsg, err := json.Marshal(deltaEvent)
		if err == nil {
			b.hub.BroadcastToSupplier(supplierID, deltaMsg)
		}

		totalDrivers += len(drivers)
		totalSuppliers++
	}

	if totalDrivers > 0 {
		slog.Info("gps buffer flush completed",
			"component", "gps_buffer",
			"driver_positions", totalDrivers,
			"supplier_channels", totalSuppliers,
			"filtered", "significant_change",
		)
	}
}

// isSignificantMove returns true if the driver moved beyond the threshold.
func isSignificantMove(prev, curr *GPSEntry) bool {
	return math.Abs(curr.Latitude-prev.Latitude) > GPSSignificantChangeThreshold ||
		math.Abs(curr.Longitude-prev.Longitude) > GPSSignificantChangeThreshold
}

// Stop gracefully stops the flush loop.
func (b *GPSBuffer) Stop() {
	close(b.done)
	slog.Info("gps buffer stopped", "component", "gps_buffer")
}
