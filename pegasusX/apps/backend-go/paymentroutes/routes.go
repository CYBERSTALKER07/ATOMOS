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
	Service             *payment.Service
	JWTSecret           string
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

// RegisterRoutes mounts checkout and payment mutation endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountCheckout := func(rr chi.Router) {
		rr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/checkout/b2b", d.Service.HandleB2BCheckout)
		rr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/checkout/unified", d.Service.HandleUnifiedCheckout)
	}

	mountAdminPayment := func(rr chi.Router) {
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/chargeback", d.Service.HandleChargeback)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/chargeback/reversal", d.Service.HandleChargebackReversal)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/ledger", d.Service.HandleLedger)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/settlement/authority", d.Service.HandleSettlementAuthority)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/payment/reconciliation/mismatches", d.Service.HandleReconciliationMismatches)
		rr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/payment/global_pay/initiate", d.Service.HandleDeprecatedGlobalPayInitiate)
	}

	guard := auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}

	if d.AllowAuthBypass {
		mountCheckout(r)
		r.Group(func(gr chi.Router) {
			gr.Use(auth.CookieAuth(d.JWTSecret))
			gr.Use(auth.RequireRole(auth.RoleAdmin))
			mountAdminPayment(gr)
		})
		return
	}

	auth.ProtectMutations(r, guard, func(gr chi.Router) {
		mountCheckout(gr)
	})
	auth.ProtectMutations(r, guard, func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		mountAdminPayment(gr)
	})
}
