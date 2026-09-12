// Package paymentroutes mounts checkout and payment mutation endpoints.
// Gateway webhooks are mounted in webhookroutes.
package paymentroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// Deps is the narrow dependency contract for payment routes.
type Deps struct {
	Service         *payment.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts checkout and payment mutation endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountCheckout := func(rr chi.Router) {
		rr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/checkout/b2b", d.Service.HandleB2BCheckout)
		rr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/checkout/preview", d.Service.HandleCheckoutPreview)
		rr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/checkout/unified", d.Service.HandleUnifiedCheckout)
	}

	mountAdminPayment := func(rr chi.Router) {
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/chargeback", d.Service.HandleChargeback)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/chargeback/reversal", d.Service.HandleChargebackReversal)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/ledger", d.Service.HandleLedger)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/claim-chargebacks", d.Service.HandleClaimChargebacks)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/settlement/authority", d.Service.HandleSettlementAuthority)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/reconciliation/mismatches", d.Service.HandleReconciliationMismatches)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/global_pay/initiate", d.Service.HandleDeprecatedGlobalPayInitiate)
	}

	mountPayers := func(rr chi.Router) {
		rr.Post("/v1/payers", d.Service.HandleCreatePayer)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payers", d.Service.HandleListPayers)
		rr.Get("/v1/payers/{payerId}", d.Service.HandleGetPayer)
		rr.Put("/v1/payers/{payerId}", d.Service.HandleUpdatePayer)
	}

	guard := auth.MutationGuardConfig{
		AllowBypass: d.AllowAuthBypass,
	}

	if d.AllowAuthBypass {
		r.Group(func(rr chi.Router) {
			mountCheckout(rr)
			mountAdminPayment(rr)
			mountPayers(rr)
		})
	} else {
		auth.ProtectMutations(r, guard, func(gr chi.Router) {
			mountCheckout(gr)
			mountPayers(gr)
		})
		r.Group(func(gr chi.Router) {
			gr.Use(auth.CookieAuth(d.JWTSecret))
			gr.Use(auth.RequireRole(auth.RoleAdmin))
			mountAdminPayment(gr)
		})
	}
}
