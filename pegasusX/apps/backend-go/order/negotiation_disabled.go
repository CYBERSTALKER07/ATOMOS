package order

import "net/http"

// quantityNegotiationDisabled gates quantity negotiation across the ecosystem.
// Product decision: feature deferred; clients must not surface negotiation UX.
const quantityNegotiationDisabled = true

func writeNegotiationDisabled(w http.ResponseWriter) {
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "feature_disabled",
		"feature": "quantity_negotiation",
	})
}
