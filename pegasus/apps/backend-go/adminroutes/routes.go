// Package adminroutes owns the /v1/admin/* surface — the twenty-three
// platform-operator endpoints that span ledger reconciliation, the audit
// log, country-level configuration, order-stage admin resolvers
// (payment-bypass, approve-cancel, shop-closed resolve, early-complete,
// negotiate resolve, resolve-credit), the SystemConfig + platform-fee
// switches, retailer KYC triage, the Empathy Engine adoption dashboard,
// the broadcast push, DLQ inspection + replay, manual Global Pay
// reconciliation, the data-nuke, and the manual replenishment trigger.
//
// Handler bodies live in backend-go/admin, backend-go/order,
// backend-go/countrycfg, backend-go/analytics, backend-go/notifications,
// backend-go/replenishment, and backend-go/kafka. This package is a thin
// composer that mounts them behind the ADMIN / SUPPLIER role guards and
// the caller-supplied observability middleware.
//
// V.O.I.D. Wave B adoption notes:
//   - Role guards: ADMIN-only paths are /v1/admin/nuke, /v1/admin/config,
//     /v1/admin/retailer/{pending,approve,reject}, /v1/admin/dlq{,/replay},
//     and /v1/admin/payment/reconcile. Every other admin route also admits
//     SUPPLIER so portal operators can act inside their own tenant.
//   - Path-prefix dispatcher (/v1/admin/country-configs/*) registers on
//     chi wildcard routing so the {code} sub-path in countrycfg's handler
//     keeps compatibility semantics without http.DefaultServeMux.
//   - Outbox adoption for admin-triggered state transitions (nuke,
//     retailer KYC, DLQ replay) remains inside the delegated packages —
//     progressive migration, tracked separately.
package adminroutes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"cloud.google.com/go/spanner"

	"backend-go/admin"
	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/countrycfg"
	"backend-go/idempotency"
	internalKafka "backend-go/kafka"
	"backend-go/notifications"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/replenishment"
	"backend-go/settings"
)

// Middleware is the handler-wrap contract supplied by the caller (typically
// main.loggingMiddleware).
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to mount /v1/admin/*.
//
// The ShopClosed/EarlyComplete/Negotiation *Deps pointers match the order
// package's existing handler signatures. KafkaBrokerAddress is consumed by
// the DLQ endpoints which call the internal/kafka package directly.
type Deps struct {
	Spanner            *spanner.Client
	ReadRouter         proximity.ReadRouter
	Order              *order.OrderService
	CountryConfig      *countrycfg.Service
	PlatformCfg        *settings.PlatformConfig
	SessionSvc         *payment.SessionService
	GPReconciler       *payment.GlobalPayReconciler
	ReplenishEngine    *replenishment.ReplenishmentEngine
	BroadcastSvc       *notifications.BroadcastService
	KafkaBrokerAddress string

	ShopClosedDeps    *order.ShopClosedDeps
	EarlyCompleteDeps *order.EarlyCompleteDeps
	NegotiationDeps   *order.NegotiationDeps

	Log Middleware
}

// RegisterRoutes mounts the twenty-three admin endpoints:
//
//	GET   /v1/admin/reconciliation             — ledger anomalies feed
//	GET   /v1/admin/audit-log                  — compliance timeline
//	GET/PUT /v1/admin/country-configs          — country-level controls
//	GET/PUT/DELETE /v1/admin/country-configs/{code}
//	POST  /v1/admin/orders/payment-bypass      — issue bypass token
//	POST  /v1/admin/orders/approve-cancel      — approve cancel request
//	GET   /v1/admin/shop-closed/active         — active shop-closed escalations
//	POST  /v1/admin/shop-closed/resolve        — resolve shop-closed
//	POST  /v1/admin/route/approve-early-complete
//	POST  /v1/admin/negotiate/resolve          — approve/reject negotiation
//	POST  /v1/admin/orders/resolve-credit      — approve/deny credit delivery
//	DELETE /v1/admin/nuke                      — purge all data (dev only)
//	GET/PUT /v1/admin/config                   — SystemConfig upsert
//	GET/PATCH /v1/admin/config/platform-fee    — platform fee percent
//	GET   /v1/admin/empathy/adoption           — Empathy Engine metrics
//	GET   /v1/admin/retailer/pending           — retailer KYC queue
//	POST  /v1/admin/retailer/approve           — verify retailer
//	POST  /v1/admin/retailer/reject            — reject retailer
//	POST  /v1/admin/broadcast                  — system-wide notification push
//	GET   /v1/admin/dlq                        — DLQ inspection
//	POST  /v1/admin/dlq/replay                 — DLQ replay by offset
//	POST  /v1/admin/payment/reconcile          — manual Global Pay reconcile
//	POST  /v1/admin/replenishment/trigger      — manual replenishment cycle
func RegisterRoutes(r chi.Router, d Deps) {
	log := d.Log
	adminOnly := []string{"ADMIN"}
	adminOrSupplier := []string{"SUPPLIER", "ADMIN"}
	supplierOrAdmin := []string{"ADMIN", "SUPPLIER"}

	// 1. Ledger anomalies feed — method-gated GET in front of the admin handler.
	r.HandleFunc("/v1/admin/reconciliation",
		auth.RequireRole(adminOrSupplier, log(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodGet {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			admin.HandleGetReconciliation(d.Spanner)(w, req)
		})))

	// 2. Compliance + operational audit timeline.
	r.HandleFunc("/v1/admin/audit-log",
		auth.RequireRole(adminOrSupplier, log(admin.HandleGetAuditLog(d.Spanner))))

	// 3-4. Country-level operational controls (exact + wildcard dispatcher).
	r.HandleFunc("/v1/admin/country-configs",
		auth.RequireRole(adminOrSupplier, log(countrycfg.HandleCountryConfigs(d.CountryConfig))))
	r.HandleFunc("/v1/admin/country-configs/*",
		auth.RequireRole(adminOrSupplier, log(countrycfg.HandleCountryConfigByCode(d.CountryConfig))))

	// 5. Edge 5 — issue payment-bypass token for stuck AWAITING_PAYMENT orders.
	r.HandleFunc("/v1/admin/orders/payment-bypass",
		auth.RequireRole(adminOrSupplier, log(order.HandleIssuePaymentBypass(d.Order))))

	// 6. Edge 7 — approve retailer cancel request.
	r.HandleFunc("/v1/admin/orders/approve-cancel",
		auth.RequireRole(adminOrSupplier, log(idempotency.Guard(order.HandleApproveCancel(d.Order)))))

	// 7. P0 — list active shop-closed escalations.
	r.HandleFunc("/v1/admin/shop-closed/active",
		auth.RequireRole(adminOrSupplier, log(d.Order.HandleListActiveShopClosedAttempts(d.ShopClosedDeps))))

	// 8. P0 — resolve shop-closed escalation (WAIT | BYPASS | RETURN_TO_DEPOT).
	r.HandleFunc("/v1/admin/shop-closed/resolve",
		auth.RequireRole(adminOrSupplier, log(idempotency.Guard(d.Order.HandleResolveShopClosed(d.ShopClosedDeps)))))

	// 9. Edge 27 — supplier approves early route completion.
	r.HandleFunc("/v1/admin/route/approve-early-complete",
		auth.RequireRole(adminOrSupplier, log(idempotency.Guard(order.HandleApproveEarlyComplete(d.Order, d.EarlyCompleteDeps)))))

	// 10. Edge 28 — supplier approves/rejects driver-proposed negotiation.
	r.HandleFunc("/v1/admin/negotiate/resolve",
		auth.RequireRole(adminOrSupplier, log(idempotency.Guard(order.HandleResolveNegotiation(d.Order, d.NegotiationDeps)))))

	// 11. Edge 32 — supplier approves/denies credit delivery.
	r.HandleFunc("/v1/admin/orders/resolve-credit",
		auth.RequireRole(adminOrSupplier, log(idempotency.Guard(order.HandleResolveCreditDelivery(d.Order, d.EarlyCompleteDeps)))))

	// 11. Dev-only data purge (ADMIN only, irreversible).
	r.HandleFunc("/v1/admin/nuke",
		auth.RequireRole(adminOnly, log(admin.HandleNukeAllData(d.Spanner))))

	// 12. SystemConfig upsert (geofence, fees, etc.). Exact match; the
	//     /v1/admin/config/platform-fee route below registers as a sibling
	//     exact match so both can coexist on DefaultServeMux.
	r.HandleFunc("/v1/admin/config",
		auth.RequireRole(adminOnly, log(admin.HandleSystemConfig(d.Spanner))))

	// 13. Platform fee percent (Phase 4.1).
	r.HandleFunc("/v1/admin/config/platform-fee",
		auth.RequireRole(supplierOrAdmin, log(platformFeeHandler(d.PlatformCfg))))

	// 14. Empathy Engine adoption dashboard.
	r.HandleFunc("/v1/admin/empathy/adoption",
		auth.RequireRole(adminOrSupplier, log(analytics.HandleEmpathyAdoption(d.Spanner, d.ReadRouter))))

	// 15-17. Retailer KYC triage (ADMIN only).
	r.HandleFunc("/v1/admin/retailer/pending",
		auth.RequireRole(adminOnly, log(retailerPendingHandler(d.Order))))
	r.HandleFunc("/v1/admin/retailer/approve",
		auth.RequireRole(adminOnly, log(retailerStatusHandler(d.Order, "VERIFIED"))))
	r.HandleFunc("/v1/admin/retailer/reject",
		auth.RequireRole(adminOnly, log(retailerStatusHandler(d.Order, "REJECTED"))))

	// 18. System-wide notification push.
	r.HandleFunc("/v1/admin/broadcast",
		auth.RequireRole(adminOrSupplier, log(d.BroadcastSvc.HandleBroadcast)))

	// 19-20. Dead Letter Queue inspection + replay.
	r.HandleFunc("/v1/admin/dlq",
		auth.RequireRole(adminOnly, log(dlqListHandler(d.KafkaBrokerAddress))))
	r.HandleFunc("/v1/admin/dlq/replay",
		auth.RequireRole(adminOnly, log(dlqReplayHandler(d.KafkaBrokerAddress))))

	// 21. Operator-triggered manual Global Pay reconciliation.
	r.HandleFunc("/v1/admin/payment/reconcile",
		auth.RequireRole(adminOnly, log(paymentReconcileHandler(d.SessionSvc, d.GPReconciler))))

	// 22. Manual replenishment cycle trigger.
	r.HandleFunc("/v1/admin/replenishment/trigger",
		auth.RequireRole(adminOrSupplier, log(d.ReplenishEngine.HandleManualTrigger)))
}

// platformFeeHandler implements GET/PATCH /v1/admin/config/platform-fee.
func platformFeeHandler(cfg *settings.PlatformConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fee := cfg.PlatformFeePercent()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int64{"platform_fee_percent": fee})

		case http.MethodPatch:
			var body struct {
				FeePercent int64 `json:"fee_percent"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if body.FeePercent < 0 || body.FeePercent > 50 {
				http.Error(w, `{"error":"fee_percent must be 0-50"}`, http.StatusBadRequest)
				return
			}
			if err := cfg.Set(r.Context(), "platform_fee_percent", fmt.Sprintf("%d", body.FeePercent)); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int64{"platform_fee_percent": body.FeePercent})

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// retailerPendingHandler serves GET /v1/admin/retailer/pending.
func retailerPendingHandler(svc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		retailers, err := svc.ListPendingRetailers(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(retailers)
	}
}

// retailerStatusHandler serves POST /v1/admin/retailer/{approve,reject} by
// updating the retailer row to the supplied newStatus (VERIFIED | REJECTED).
func retailerStatusHandler(svc *order.OrderService, newStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			RetailerId string `json:"retailer_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := svc.UpdateRetailerStatus(r.Context(), req.RetailerId, newStatus); err != nil {
			log.Printf("Failed to update retailer status: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status":"%s"}`, newStatus)))
	}
}

// dlqListHandler serves GET /v1/admin/dlq — up to `limit` trapped events
// (offset-paginated) so the Admin Portal can assess financial damage.
func dlqListHandler(broker string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		limit := 100
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				http.Error(w, "Invalid limit", http.StatusBadRequest)
				return
			}
			limit = parsed
		}
		offset := 0
		if raw := r.URL.Query().Get("offset"); raw != "" {
			parsed, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				http.Error(w, "Invalid offset", http.StatusBadRequest)
				return
			}
			offset = parsed
		}

		messages, err := internalKafka.ListDLQMessages(broker, limit, offset)
		if err != nil {
			log.Printf("[DLQ API] Inspection failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"DLQ inspection failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		if messages == nil {
			messages = []internalKafka.DLQMessage{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

// dlqReplayHandler serves POST /v1/admin/dlq/replay — re-emits a single
// trapped event onto the main topic. Consumer idempotency handles
// exactly-once delivery on the receiving side.
func dlqReplayHandler(broker string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Offset int64 `json:"offset"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := internalKafka.ReplayDLQMessage(broker, req.Offset); err != nil {
			log.Printf("[DLQ REPLAY] Replay of offset %d failed: %v", req.Offset, err)
			http.Error(w, fmt.Sprintf(`{"error":"replay failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status":"REPLAYED","offset":%d}`, req.Offset)))
	}
}

// paymentReconcileHandler serves POST /v1/admin/payment/reconcile — the
// operator rescue button for stuck Global Pay sessions.
func paymentReconcileHandler(sessionSvc *payment.SessionService, gp *payment.GlobalPayReconciler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SessionID == "" {
			http.Error(w, `{"error":"session_id is required"}`, http.StatusBadRequest)
			return
		}

		session, err := sessionSvc.GetSession(r.Context(), req.SessionID)
		if err != nil {
			http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
			return
		}

		result, err := gp.ReconcileSession(r.Context(), session, "")
		if err != nil {
			log.Printf("[MANUAL_RECONCILE] Session %s reconciliation failed: %v", req.SessionID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
