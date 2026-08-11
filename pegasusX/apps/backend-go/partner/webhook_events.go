package partner

import "github.com/pegasusx/pegasusx/apps/backend-go/events"

// PartnerWebhookableEvents is the platform-safe subset of domain events that
// may be delivered to partner webhooks. Per-subscription EventTypes further
// filters within this set. Do not expose the full events catalog (159 types) —
// many are internal/ops-only.
var PartnerWebhookableEvents = map[string]bool{
	events.EventOrderCreated:             true,
	events.EventOrderStatusChanged:       true,
	events.EventClaimFiled:               true,
	events.EventClaimResolved:            true,
	events.EventPaymentCleared:           true,
	events.EventPaymentFailed:            true,
	events.EventPaymentRequired:          true,
	events.EventManifestSealed:           true,
	events.EventManifestDispatched:       true,
	events.EventManifestCompleted:        true,
	events.EventReturnReceivedAtWarehouse: true,
	events.EventDemandSignal:             true,
	events.EventProductHandlingUpdated:   true,
}

// IsPartnerWebhookable reports whether eventType is in the platform-safe set.
func IsPartnerWebhookable(eventType string) bool {
	return PartnerWebhookableEvents[eventType]
}
