package events

import (
	"encoding/json"
	"os"
	"strings"
)

var (
	// TopicOrders carries order lifecycle events (dual-write target).
	TopicOrders = topicFromEnv("KAFKA_TOPIC_ORDERS", "pegasusx-orders")
	// TopicDispatch carries dispatch, manifest, and fleet routing events.
	TopicDispatch = topicFromEnv("KAFKA_TOPIC_DISPATCH", "pegasusx-dispatch")
	// TopicRealtime carries driver location and live telemetry fan-out.
	TopicRealtime = topicFromEnv("KAFKA_TOPIC_REALTIME", "pegasusx-realtime")
	// TopicWebhooks carries payment gateway settlement events.
	TopicWebhooks = topicFromEnv("KAFKA_TOPIC_WEBHOOKS", "pegasusx-webhooks")
)

// DualWriteDomainTopics mirrors events onto domain topics while consumers migrate
// off pegasusx-main. Enabled when KAFKA_TOPIC_DUAL_WRITE=true.
func DualWriteDomainTopics() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("KAFKA_TOPIC_DUAL_WRITE")), "true")
}

// ConsumeDomainTopics switches domain-specific consumers off pegasusx-main.
// Requires KAFKA_TOPIC_DUAL_WRITE=true in production cutover. When enabled,
// the notification dispatcher fans in TopicMain plus domain topics.
func ConsumeDomainTopics() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("KAFKA_TOPIC_CONSUME_DOMAIN")), "true")
}

// OrderConsumerTopic returns the Kafka topic for the order mutator consumer.
func OrderConsumerTopic() string {
	if ConsumeDomainTopics() {
		return TopicOrders
	}
	return TopicMain
}

// DispatcherConsumerTopics returns Kafka topics for the notification dispatcher.
func DispatcherConsumerTopics() []string {
	if !ConsumeDomainTopics() {
		return []string{TopicMain}
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	for _, t := range []string{TopicMain, TopicOrders, TopicDispatch, TopicRealtime, TopicLogisticsExceptions, TopicLogisticsTelemetry} {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// DispatchConsumerTopic returns the Kafka topic for warehouse/dispatch consumers.
func DispatchConsumerTopic() string {
	if ConsumeDomainTopics() {
		return TopicDispatch
	}
	return TopicMain
}

// DomainTopicForEventType maps an event type to its domain topic, or "" when
// the event should remain on TopicMain only.
func DomainTopicForEventType(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case EventOrderCreated, EventOrderStatusChanged, EventOrderValidationFailed,
		EventOrderAssigned, EventOrderReassigned, EventOrderFinalized, EventOrderAmended,
		EventPreOrderNotified, EventPreOrderNudge, EventPreOrderConfirmation,
		EventPreOrderConfirmed, EventPreOrderEdited, EventPreOrderCancelled,
		EventPreOrderAutoAccepted, EventPreOrderDateProposed, EventPreOrderDateAccepted,
		EventPreOrderDateRejected, EventShopClosed, EventShopClosedResponse,
		EventShopClosedEscalated, EventShopClosedResolved, EventNegotiationProposed,
		EventNegotiationResolved, EventPaymentRequired, EventPaymentCleared, EventPaymentFailed,
		EventSettlementRequired, EventDeliverySessionUpdated, EventDeliveryDisputed,
		EventSplitPaymentCreated,
		EventFiscalReceiptRequested, EventFiscalReceiptSucceeded, EventFiscalReceiptFailed,
		EventOrderForceCompleted:
		return TopicOrders
	case EventWarehouseDispatchLockChanged, EventManifestDraftCreated,
		EventManifestLoadingStarted, EventManifestOrderInjected, EventManifestOrderException,
		EventManifestDLQEscalation, EventManifestRebalanced, EventManifestCancelled,
		EventManifestSealed, EventManifestDispatched, EventManifestCompleted,
		EventRouteCreated, EventRouteReordered, EventDriverAvailabilityChanged,
		EventVehicleAvailabilityChanged, EventFreezeLockAcquired, EventFreezeLockReleased:
		return TopicDispatch
	case EventDriverLocationUpdated, EventDriverReturnApproaching,
		EventSupplyTransferApproaching, EventCommandDispatched, EventCommandReceived,
		EventCommandSettled:
		return TopicRealtime
	case EventMissingItemsReported, EventReverseLogisticsRequired:
		return TopicLogisticsExceptions
	default:
		return ""
	}
}

// RelayPublishTopics returns topics the outbox relay should publish to for one
// stored outbox row. When dual-write is disabled, only storedTopic is returned.
func RelayPublishTopics(storedTopic string, payload []byte) []string {
	storedTopic = strings.TrimSpace(storedTopic)
	if storedTopic == "" {
		storedTopic = TopicMain
	}
	if !DualWriteDomainTopics() {
		return []string{storedTopic}
	}
	eventType := extractEventTypeFromPayload(payload)
	domain := DomainTopicForEventType(eventType)
	if domain == "" || domain == storedTopic {
		return []string{storedTopic}
	}
	return []string{storedTopic, domain}
}

func extractEventTypeFromPayload(payload []byte) string {
	if len(payload) == 0 || payload[0] != '{' {
		return ""
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ""
	}
	return envelope.Type
}
