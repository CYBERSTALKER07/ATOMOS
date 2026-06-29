package notifications

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// HandoffCardMetadata is persisted on inbox rows for actionable handoff cards.
type HandoffCardMetadata struct {
	Kind        string            `json:"kind"`
	Title       string            `json:"title"`
	Subtitle    string            `json:"subtitle,omitempty"`
	PrimaryCTA  string            `json:"primary_cta,omitempty"`
	PrimaryLink string            `json:"primary_link,omitempty"`
	EntityType  string            `json:"entity_type,omitempty"`
	EntityID    string            `json:"entity_id,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
}

// BuildHandoffMetadata maps high-signal events to structured inbox cards.
func BuildHandoffMetadata(eventType string, payload []byte) *HandoffCardMetadata {
	eventType = strings.TrimSpace(eventType)
	switch eventType {
	case "DISPATCH_COMMITTED":
		return handoffFromManifestEvent(eventType, payload, "Dispatch committed", "Review new manifests and loading queue", "/dispatch", "manifest")
	case events.EventManifestDraftCreated:
		return handoffFromManifestEvent(eventType, payload, "Manifest draft created", "Assign payloader and begin loading", "/manifests", "manifest")
	case events.EventManifestSealed:
		return handoffFromManifestEvent(eventType, payload, "Manifest sealed", "Ready for driver departure", "/manifests", "manifest")
	case events.EventManifestDispatched:
		return handoffFromManifestEvent(eventType, payload, "Manifest dispatched", "Track live route progress", "/fleet", "manifest")
	case events.EventPreOrderDateProposed:
		return handoffFromOrderEvent(eventType, payload, "Delivery date proposed", "Retailer review required", "/orders", "order")
	default:
		return nil
	}
}

func handoffFromManifestEvent(eventType string, payload []byte, title, subtitle, linkPrefix, entityType string) *HandoffCardMetadata {
	var e events.ManifestEvent
	if json.Unmarshal(payload, &e) != nil || strings.TrimSpace(e.ManifestID) == "" {
		return nil
	}
	fields := map[string]string{
		"manifest_id": e.ManifestID,
	}
	if e.DriverID != "" {
		fields["driver_id"] = e.DriverID
	}
	if e.WarehouseID != "" {
		fields["warehouse_id"] = e.WarehouseID
	}
	return &HandoffCardMetadata{
		Kind:        eventType,
		Title:       title,
		Subtitle:    subtitle,
		PrimaryCTA:  "Open",
		PrimaryLink: fmt.Sprintf("%s/%s", linkPrefix, e.ManifestID),
		EntityType:  entityType,
		EntityID:    e.ManifestID,
		Fields:      fields,
	}
}

func handoffFromOrderEvent(eventType string, payload []byte, title, subtitle, linkPrefix, entityType string) *HandoffCardMetadata {
	var e events.OrderEvent
	if json.Unmarshal(payload, &e) != nil || strings.TrimSpace(e.OrderID) == "" {
		return nil
	}
	fields := map[string]string{"order_id": e.OrderID}
	if e.RequestedDeliveryDate != "" {
		fields["proposed_delivery_date"] = e.RequestedDeliveryDate
	}
	return &HandoffCardMetadata{
		Kind:        eventType,
		Title:       title,
		Subtitle:    subtitle,
		PrimaryCTA:  "Review",
		PrimaryLink: fmt.Sprintf("%s/%s?review=1", linkPrefix, e.OrderID),
		EntityType:  entityType,
		EntityID:    e.OrderID,
		Fields:      fields,
	}
}

// EncodeHandoffMetadata serializes handoff metadata for Spanner storage.
func EncodeHandoffMetadata(meta *HandoffCardMetadata) []byte {
	if meta == nil {
		return nil
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return nil
	}
	return raw
}

// DecodeHandoffMetadata parses stored handoff metadata bytes.
func DecodeHandoffMetadata(raw []byte) *HandoffCardMetadata {
	if len(raw) == 0 {
		return nil
	}
	var meta HandoffCardMetadata
	if json.Unmarshal(raw, &meta) != nil {
		return nil
	}
	if strings.TrimSpace(meta.Kind) == "" {
		return nil
	}
	return &meta
}

// HandoffCardFreshness returns whether metadata should be shown prominently.
func HandoffCardFreshness(createdAt time.Time, now time.Time) bool {
	return now.Sub(createdAt) <= 7*24*time.Hour
}
