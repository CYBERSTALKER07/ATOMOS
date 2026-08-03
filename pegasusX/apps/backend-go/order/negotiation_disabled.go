package order

import "net/http"

// quantityNegotiationDisabled gates quantity negotiation across the ecosystem.
//
// Product-deferred (disabled). Delivery-time qty propose/resolve is NOT a
// substitute for claims, shop-closed, partial offload, or store stock quarantine.
// When true:
//   - POST /v1/delivery/negotiate → 410 feature_disabled
//   - POST /v1/supplier/negotiate/resolve → 410 feature_disabled
//   - GET  /v1/supplier/negotiations/pending → empty list
//   - StartNegotiationSweeper → no-op
//
// Other logistics paths (shop-closed, claims, missing-items, credit-leave,
// STORE_STOCK receive, DEMAND_SIGNAL, auto-order) are independent and unaffected.
const quantityNegotiationDisabled = true

func writeNegotiationDisabled(w http.ResponseWriter) {
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "feature_disabled",
		"feature": "quantity_negotiation",
	})
}
