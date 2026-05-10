package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// AuditJournal queues telemetry audit events for best-effort durable journaling.
type AuditJournal interface {
	Emit(ctx context.Context, event AuditEvent) error
	Close() error
}

// AuditEvent is the canonical durable journal shape for a driver GPS ping.
type AuditEvent struct {
	TraceID    string    `json:"trace_id"`
	DriverID   string    `json:"driver_id"`
	SupplierID string    `json:"supplier_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Velocity   float64   `json:"velocity,omitempty"`
	Heading    float64   `json:"heading,omitempty"`
	Timestamp  int64     `json:"timestamp"`
	EventTime  time.Time `json:"event_time"`
	Payload    []byte    `json:"payload"`
}

// NormalizePayloadTimestamp converts seconds-form timestamps into Unix millis.
func NormalizePayloadTimestamp(timestamp int64) int64 {
	if timestamp <= 0 {
		return time.Now().UnixMilli()
	}
	if timestamp < 1_000_000_000_000 {
		return timestamp * 1000
	}
	return timestamp
}

// TelemetryEventTime converts a normalized telemetry timestamp into UTC time.
func TelemetryEventTime(timestamp int64) time.Time {
	return time.UnixMilli(NormalizePayloadTimestamp(timestamp)).UTC()
}

// DeriveTraceID returns a stable telemetry trace identifier when the client has
// not yet supplied one on the wire.
func DeriveTraceID(supplierID string, payload GPSPayload) string {
	if payload.TraceID != "" {
		return payload.TraceID
	}
	normalizedTimestamp := NormalizePayloadTimestamp(payload.Timestamp)
	sum := sha256.Sum256([]byte(fmt.Sprintf(
		"%s|%s|%.6f|%.6f|%d",
		supplierID,
		payload.DriverID,
		payload.Latitude,
		payload.Longitude,
		normalizedTimestamp,
	)))
	return hex.EncodeToString(sum[:])
}

// BuildAuditEvent canonicalizes a GPS payload into the durable journal shape.
func BuildAuditEvent(payload GPSPayload, supplierID string) (AuditEvent, GPSPayload, error) {
	normalized := payload
	normalized.Timestamp = NormalizePayloadTimestamp(normalized.Timestamp)
	normalized.TraceID = DeriveTraceID(supplierID, normalized)

	body, err := json.Marshal(normalized)
	if err != nil {
		return AuditEvent{}, GPSPayload{}, fmt.Errorf("marshal canonical gps payload: %w", err)
	}

	return AuditEvent{
		TraceID:    normalized.TraceID,
		DriverID:   normalized.DriverID,
		SupplierID: supplierID,
		Latitude:   normalized.Latitude,
		Longitude:  normalized.Longitude,
		Velocity:   normalized.Velocity,
		Heading:    normalized.Heading,
		Timestamp:  normalized.Timestamp,
		EventTime:  TelemetryEventTime(normalized.Timestamp),
		Payload:    body,
	}, normalized, nil
}
