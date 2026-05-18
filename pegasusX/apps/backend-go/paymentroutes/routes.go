// endpoints. Gateway webhooks are mounted in webhookroutes.
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
}

// RegisterRoutes mounts checkout and payment mutation endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	mountCheckout := func(rr chi.Router) {
		rr.Post("/v1/checkout/b2b", d.Service.HandleB2BCheckout)
		rr.Post("/v1/checkout/unified", d.Service.HandleUnifiedCheckout)
	}

	mountAdminPayment := func(rr chi.Router) {
		rr.Post("/v1/payment/chargeback", d.Service.HandleChargeback)
		rr.Post("/v1/payment/chargeback/reversal", d.Service.HandleChargebackReversal)
		rr.Post("/v1/payment/global_pay/initiate", d.Service.HandleDeprecatedGlobalPayInitiate)
	}

	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleRetailer))
			mountCheckout(gr)
		})
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleAdmin))
			mountAdminPayment(gr)
		})
		return
	}

	// Local scaffold fallback: checkout is open, admin payment mutations require
	// supplier cookie auth.
	mountCheckout(r)
	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin))
		mountAdminPayment(gr)
	})
}
