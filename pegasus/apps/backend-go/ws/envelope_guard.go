package ws

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// SchemaVersionV1 is the legacy websocket contract used by deployed native clients.
	SchemaVersionV1 = 1
	// SchemaVersionV2 is the current websocket contract with additive settlement and command fields.
	SchemaVersionV2 = 2
)

type guardedConnTarget struct {
	conn          *websocket.Conn
	schemaVersion int
}

type payloadGuardResult struct {
	payload             []byte
	eventType           string
	eventSchemaVersion  int
	clientSchemaVersion int
	downgraded          bool
	upgradeRequired     bool
}

var schemaEventMinVersion = map[string]int{
	EventCommandDispatched:      SchemaVersionV2,
	EventCommandReceived:        SchemaVersionV2,
	EventCommandSettled:         SchemaVersionV2,
	EventDeliverySessionUpdated: SchemaVersionV2,
	EventSettlementRequired:     SchemaVersionV2,
}

var schemaV2IncompatibleEvents = map[string]struct{}{
	EventCommandDispatched: {},
	EventCommandReceived:   {},
	EventCommandSettled:    {},
}

var schemaV2AdditiveFields = map[string]struct{}{
	"schema_version":   {},
	"currency":         {},
	"original_amount":  {},
	"adjusted_amount":  {},
	"fee_basis_points": {},
	"fee_amount":       {},
	"session_id":       {},
	"invoice_id":       {},
	"command_id":       {},
	"command_state":    {},
	"initiated_at":     {},
	"dispatched_at":    {},
}

// Breaking rename map placeholder for v2->v1 compatibility migrations.
var schemaV2ToV1Aliases = map[string]string{
	"uuid": "order_id",
}

// EnvelopeSchemaVersion returns the minimum schema version required to safely
// consume an event payload. Unknown events default to v1.
func EnvelopeSchemaVersion(eventType string, payload map[string]interface{}) int {
	eventType = strings.TrimSpace(eventType)
	if payload != nil {
		if explicit := parseSchemaVersion(payload["schema_version"]); explicit > 0 {
			return explicit
		}
		if explicit := parseSchemaVersion(payload["version"]); explicit > 0 {
			return explicit
		}
	}
	if minVersion, ok := schemaEventMinVersion[eventType]; ok {
		return minVersion
	}
	if payload != nil {
		for field := range schemaV2AdditiveFields {
			if _, ok := payload[field]; ok {
				return SchemaVersionV2
			}
		}
	}
	return SchemaVersionV1
}

func resolveClientSchemaVersion(r *http.Request) int {
	candidates := []string{
		strings.TrimSpace(r.URL.Query().Get("sv")),
		strings.TrimSpace(r.Header.Get("X-Schema-Version")),
		strings.TrimSpace(r.Header.Get("X-Client-Schema-Version")),
	}
	for _, raw := range candidates {
		if raw == "" {
			continue
		}
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return parsed
		}
	}

	// Browser clients are deployed continuously and should default to the latest
	// schema unless they explicitly pin a version.
	if strings.TrimSpace(r.Header.Get("Origin")) != "" {
		return SchemaVersionV2
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent"))), "mozilla/") {
		return SchemaVersionV2
	}

	// Native clients without explicit schema headers stay on the legacy fallback.
	return SchemaVersionV1
}

func parseSchemaVersion(raw interface{}) int {
	switch value := raw.(type) {
	case int:
		if value > 0 {
			return value
		}
	case int32:
		if value > 0 {
			return int(value)
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	case json.Number:
		if parsed, err := strconv.Atoi(value.String()); err == nil && parsed > 0 {
			return parsed
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func guardPayloadForClient(raw []byte, schemaVersion int) (payloadGuardResult, error) {
	clientVersion := schemaVersion
	if clientVersion <= 0 {
		clientVersion = SchemaVersionV1
	}
	result := payloadGuardResult{
		payload:             raw,
		clientSchemaVersion: clientVersion,
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil || payload == nil {
		return result, nil
	}

	eventType, _ := payload["type"].(string)
	result.eventType = strings.TrimSpace(eventType)
	result.eventSchemaVersion = EnvelopeSchemaVersion(result.eventType, payload)
	if result.eventSchemaVersion <= result.clientSchemaVersion {
		return result, nil
	}

	downgraded, err := downgradePayload(payload, result.eventType, result.eventSchemaVersion, result.clientSchemaVersion)
	if err != nil {
		outdated := buildOutdatedPayload(payload, result.eventType, result.eventSchemaVersion, result.clientSchemaVersion)
		outdatedWire, marshalErr := json.Marshal(outdated)
		if marshalErr != nil {
			return result, fmt.Errorf("marshal SYSTEM_APP_OUTDATED payload: %w", marshalErr)
		}
		result.payload = outdatedWire
		result.upgradeRequired = true
		return result, nil
	}

	wire, err := json.Marshal(downgraded)
	if err != nil {
		return result, fmt.Errorf("marshal downgraded payload: %w", err)
	}
	result.payload = wire
	result.downgraded = true
	return result, nil
}

func downgradePayload(payload map[string]interface{}, eventType string, fromVersion, targetVersion int) (map[string]interface{}, error) {
	if targetVersion >= fromVersion {
		return cloneEnvelopePayload(payload), nil
	}

	switch {
	case fromVersion == SchemaVersionV2 && targetVersion == SchemaVersionV1:
		if _, blocked := schemaV2IncompatibleEvents[eventType]; blocked {
			return nil, fmt.Errorf("event %s is incompatible with schema v%d", eventType, targetVersion)
		}

		downgraded := cloneEnvelopePayload(payload)
		for key := range schemaV2AdditiveFields {
			delete(downgraded, key)
		}
		for fromKey, toKey := range schemaV2ToV1Aliases {
			if _, exists := downgraded[toKey]; exists {
				continue
			}
			if value, exists := downgraded[fromKey]; exists {
				downgraded[toKey] = value
			}
		}
		return downgraded, nil
	default:
		return nil, fmt.Errorf("no downgrade path from schema v%d to v%d", fromVersion, targetVersion)
	}
}

func cloneEnvelopePayload(payload map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		clone[key] = value
	}
	return clone
}

func buildOutdatedPayload(source map[string]interface{}, blockedEventType string, requiredVersion, clientVersion int) map[string]interface{} {
	traceID, _ := source["trace_id"].(string)
	if strings.TrimSpace(traceID) == "" {
		traceID = fmt.Sprintf("trace-%d", time.Now().UTC().UnixNano())
	}

	return map[string]interface{}{
		"type":                    EventSystemAppOutdated,
		"message":                 "A critical app update is required to receive this event.",
		"required_schema_version": requiredVersion,
		"client_schema_version":   clientVersion,
		"blocked_event_type":      blockedEventType,
		"trace_id":                traceID,
		"timestamp":               time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func writeGuardedPayload(hub, recipientField, recipientID string, targets []guardedConnTarget, writeMu *sync.Mutex, raw []byte) bool {
	if len(targets) == 0 {
		return false
	}

	delivered := false
	writeMu.Lock()
	defer writeMu.Unlock()

	for _, target := range targets {
		wire := raw
		guarded, err := guardPayloadForClient(raw, target.schemaVersion)
		if err != nil {
			slog.Warn("ws envelope guard failed; falling back to raw payload",
				"hub", hub,
				recipientField, recipientID,
				"schema_version", target.schemaVersion,
				"error", err,
			)
		} else {
			wire = guarded.payload
			if guarded.downgraded {
				slog.Info("ws envelope downgraded for legacy client",
					"hub", hub,
					recipientField, recipientID,
					"event_type", guarded.eventType,
					"event_schema_version", guarded.eventSchemaVersion,
					"client_schema_version", guarded.clientSchemaVersion,
				)
			}
			if guarded.upgradeRequired {
				slog.Warn("ws envelope blocked for outdated client",
					"hub", hub,
					recipientField, recipientID,
					"event_type", guarded.eventType,
					"event_schema_version", guarded.eventSchemaVersion,
					"client_schema_version", guarded.clientSchemaVersion,
				)
			}
		}

		if err := target.conn.WriteMessage(websocket.TextMessage, wire); err != nil {
			slog.Warn("ws write failed; evicting dead connection",
				"hub", hub,
				recipientField, recipientID,
				"error", err,
			)
			target.conn.Close()
			continue
		}
		delivered = true
	}

	return delivered
}
