package telemetry

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestGPSBuffer_IngestOverwrites(t *testing.T) {
	buf := &GPSBuffer{
		entries:  make(map[string]*GPSEntry),
		interval: 1 * time.Hour, // won't auto-flush
		done:     make(chan struct{}),
	}

	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.0, Longitude: 69.0, Timestamp: 100})
	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.1, Longitude: 69.1, Timestamp: 200})

	buf.mu.Lock()
	entry := buf.entries["d1"]
	buf.mu.Unlock()

	if entry == nil {
		t.Fatal("expected entry for d1")
	}
	if entry.Latitude != 41.1 {
		t.Errorf("lat = %f, want 41.1 (latest overwrites)", entry.Latitude)
	}
	if entry.Timestamp != 200 {
		t.Errorf("timestamp = %d, want 200", entry.Timestamp)
	}
}

func TestGPSBuffer_FlushDrainsEntries(t *testing.T) {
	// Create a buffer with a nil hub (flush will skip broadcast but drain the map)
	buf := &GPSBuffer{
		entries:  make(map[string]*GPSEntry),
		previous: make(map[string]*GPSEntry),
		interval: 1 * time.Hour,
		hub:      &Hub{Clients: make(map[*websocket.Conn]*clientMeta), subscribed: make(map[string]bool)},
		done:     make(chan struct{}),
	}

	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.0, Longitude: 69.0, SupplierID: "s1"})
	buf.Ingest(GPSEntry{DriverID: "d2", Latitude: 41.1, Longitude: 69.1, SupplierID: "s1"})

	buf.flush()

	buf.mu.Lock()
	remaining := len(buf.entries)
	buf.mu.Unlock()

	if remaining != 0 {
		t.Errorf("expected 0 entries after flush, got %d", remaining)
	}
}

func TestGPSBuffer_EmptyFlushNoOp(t *testing.T) {
	buf := &GPSBuffer{
		entries:  make(map[string]*GPSEntry),
		previous: make(map[string]*GPSEntry),
		interval: 1 * time.Hour,
		hub:      &Hub{Clients: make(map[*websocket.Conn]*clientMeta), subscribed: make(map[string]bool)},
		done:     make(chan struct{}),
	}

	// Should not panic on empty flush
	buf.flush()
}

func TestGPSBuffer_SignificantChangeFilter(t *testing.T) {
	buf := &GPSBuffer{
		entries:  make(map[string]*GPSEntry),
		previous: make(map[string]*GPSEntry),
		interval: 1 * time.Hour,
		hub:      &Hub{Clients: make(map[*websocket.Conn]*clientMeta), subscribed: make(map[string]bool)},
		done:     make(chan struct{}),
	}

	// First flush — no previous, so all entries are "significant"
	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.0, Longitude: 69.0, SupplierID: "s1"})
	buf.flush()

	// Now d1 has a previous position. Ingest a tiny move (< threshold)
	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.00005, Longitude: 69.00005, SupplierID: "s1"})
	buf.flush()

	// The previous should NOT have been updated (move was insignificant)
	prev := buf.previous["d1"]
	if prev == nil {
		t.Fatal("expected previous entry for d1")
	}
	// Previous should still be the first position (41.0) since the tiny move was filtered
	if prev.Latitude != 41.0 {
		t.Errorf("previous lat = %f, want 41.0 (tiny move should be filtered)", prev.Latitude)
	}

	// Ingest a significant move (> threshold)
	buf.Ingest(GPSEntry{DriverID: "d1", Latitude: 41.001, Longitude: 69.001, SupplierID: "s1"})
	buf.flush()

	prev = buf.previous["d1"]
	if prev.Latitude != 41.001 {
		t.Errorf("previous lat = %f, want 41.001 (significant move should update)", prev.Latitude)
	}
}

func TestIsSignificantMove(t *testing.T) {
	cases := []struct {
		name string
		prev GPSEntry
		curr GPSEntry
		want bool
	}{
		{
			name: "no_move",
			prev: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			curr: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			want: false,
		},
		{
			name: "tiny_lat_move",
			prev: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			curr: GPSEntry{Latitude: 41.00005, Longitude: 69.0},
			want: false,
		},
		{
			name: "significant_lat_move",
			prev: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			curr: GPSEntry{Latitude: 41.0002, Longitude: 69.0},
			want: true,
		},
		{
			name: "significant_lng_move",
			prev: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			curr: GPSEntry{Latitude: 41.0, Longitude: 69.0002},
			want: true,
		},
		{
			name: "both_significant",
			prev: GPSEntry{Latitude: 41.0, Longitude: 69.0},
			curr: GPSEntry{Latitude: 41.001, Longitude: 69.001},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSignificantMove(&tc.prev, &tc.curr)
			if got != tc.want {
				t.Errorf("isSignificantMove(%v, %v) = %v, want %v", tc.prev, tc.curr, got, tc.want)
			}
		})
	}
}
