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
	// TopicWebhooks is RETIRED (W1 2026-08-12). Payment settlement and partner
	// webhook delivery consume TopicMain / TopicOrders. The Kafka topic may still
	// exist in infra for compatibility; producers must not emit to it.
	TopicWebhooks = topicFromEnv("KAFKA_TOPIC_WEBHOOKS", "pegasusx-webhooks")
	// TopicExceptions carries logistics claims / OS&D / reverse-logistics.
	TopicExceptions = topicFromEnv("KAFKA_TOPIC_EXCEPTIONS", "logistics.exceptions.v1")
	// TopicTelemetryLogistics carries temperature / seal / sensor exception telemetry.
	TopicTelemetryLogistics = topicFromEnv("KAFKA_TOPIC_TELEMETRY_LOGISTICS", "logistics.telemetry.v1")
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
	for _, t := range []string{TopicMain, TopicOrders, TopicDispatch, TopicRealtime} {
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

// TwinConsumerTopics returns Kafka topics for the digital-twin projector.
// Always includes TopicMain; when domain consume is on, also fans in orders,
// dispatch (route created), and realtime (driver location).
func TwinConsumerTopics() []string {
	if !ConsumeDomainTopics() {
		return []string{TopicMain}
	}
	seen := make(map[string]struct{}, 4)
	out := make([]string, 0, 4)
	for _, t := range []string{TopicMain, TopicOrders, TopicDispatch, TopicRealtime} {
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

// DomainTopicForEventType maps an event type to its domain topic, or "" when
// the event should remain on TopicMain only.
func DomainTopicForEventType(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case EventOrderCreated, EventOrderStatusChanged, EventOrderValidationFailed,
		EventOrderAssigned, EventOrderReassigned, EventOrderFinalized, EventOrderAmended,
		EventOrderAllocated,
		EventPreOrderNotified, EventPreOrderNudge, EventPreOrderConfirmation,
		EventPreOrderConfirmed, EventPreOrderEdited, EventPreOrderCancelled,
		EventPreOrderAutoAccepted, EventPreOrderDateProposed, EventPreOrderDateAccepted,
		EventPreOrderDateRejected, EventShopClosed, EventShopClosedResponse,
		EventShopClosedEscalated, EventShopClosedResolved, EventShopClosedTimeout,
		EventProximityUnlocked, EventPartialOffload, EventCreditLeave,
		EventNegotiationProposed,
		EventNegotiationResolved, EventPaymentRequired, EventPaymentCleared, EventPaymentFailed,
		EventSettlementRequired, EventDeliverySessionUpdated, EventDeliveryDisputed,
		EventMissingItemsReported, EventSplitPaymentCreated,
		EventFiscalReceiptRequested, EventFiscalReceiptSucceeded, EventFiscalReceiptFailed,
		EventRefundRequested, EventRefundSucceeded, EventRefundFailed, EventFiscalCorrectiveRequested,
		EventOrderForceCompleted, EventCashShortfall, EventCashOverage:
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
	case EventClaimFiled, EventClaimResolved, EventLogisticsExceptionReported,
		EventReverseLogisticsRequired:
		// MISSING_ITEMS_REPORTED stays on TopicOrders (above); dual-emit to exceptions
		// is done explicitly in order handlers when needed.
		return TopicExceptions
	case EventLogisticsTelemetry:
		return TopicTelemetryLogistics
	case EventDemandSignal:
		// STORE_POS flywheel → dedicated demand topic for supplier consumers.
		return TopicDemand
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
