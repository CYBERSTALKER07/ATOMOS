// Package drivercmd prepares driver WebSocket command envelopes with stable
// command_id stamps and persists issued commands in DriverCommands so
// POST /v1/ws/ack can reject unissued or event-mismatched acknowledgments.
package drivercmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// PrepareJSON ensures any driver JSON object carrying a non-empty event type
// includes a non-empty command_id, returning updated JSON plus extracted metadata.
func PrepareJSON(payload []byte) (enriched []byte, commandID, eventType, traceID string, ok bool) {
	if len(payload) == 0 {
		return payload, "", "", "", false
	}
	var env map[string]any
	if err := json.Unmarshal(payload, &env); err != nil || env == nil {
		return payload, "", "", "", false
	}
	eventType = stringField(env, "type")
	if eventType == "" {
		eventType = stringField(env, "event_type")
	}
	if eventType == "" {
		return payload, "", "", "", false
	}
	commandID = stringField(env, "command_id")
	if commandID == "" {
		commandID = uuid.NewString()
		env["command_id"] = commandID
		if updated, err := json.Marshal(env); err == nil {
			payload = updated
		}
	}
	if len(commandID) > 128 {
		commandID = commandID[:128]
	}
	if len(eventType) > 64 {
		eventType = eventType[:64]
	}
	traceID = stringField(env, "trace_id")
	if len(traceID) > 128 {
		traceID = traceID[:128]
	}
	return payload, commandID, eventType, traceID, true
}

// PrepareAndRecordJSON stamps command_id on driver WS envelopes and persists the
// issued command into DriverCommands before broadcast when client != nil.
func PrepareAndRecordJSON(ctx context.Context, client *spanner.Client, driverID string, payload []byte) []byte {
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return payload
	}
	enriched, commandID, eventType, traceID, ok := PrepareJSON(payload)
	if !ok || client == nil {
		return enriched
	}
	if err := RecordIssued(ctx, client, driverID, commandID, eventType, traceID); err != nil {
		slog.WarnContext(ctx, "record issued driver command failed",
			"driver_id", driverID,
			"command_id", commandID,
			"event_type", eventType,
			"err", err,
		)
	}
	return enriched
}

// RecordIssued persists an issued command row in DriverCommands.
func RecordIssued(ctx context.Context, client *spanner.Client, driverID, commandID, eventType, traceID string) error {
	driverID = strings.TrimSpace(driverID)
	commandID = strings.TrimSpace(commandID)
	eventType = strings.TrimSpace(eventType)
	traceID = strings.TrimSpace(traceID)
	if client == nil || driverID == "" || commandID == "" || eventType == "" {
		return nil
	}
	if len(commandID) > 128 {
		commandID = commandID[:128]
	}
	if len(eventType) > 64 {
		eventType = eventType[:64]
	}
	if ctx == nil {
		ctx = context.Background()
	}
	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cols := map[string]any{
		"DriverId":  driverID,
		"CommandId": commandID,
		"EventType": eventType,
		"IssuedAt":  spanner.CommitTimestamp,
	}
	if traceID != "" {
		if len(traceID) > 128 {
			traceID = traceID[:128]
		}
		cols["TraceId"] = traceID
	}
	_, err := client.Apply(writeCtx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("DriverCommands", cols),
	})
	return err
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
