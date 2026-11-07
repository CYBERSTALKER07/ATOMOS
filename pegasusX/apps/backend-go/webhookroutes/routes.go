// verification is performed by the payment service handlers.
package webhookroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// Deps is the narrow dependency contract for webhook routes.
type Deps struct {
	Service *payment.Service
}

// RegisterRoutes mounts payment webhook endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Group(func(r chi.Router) {
		// Apply log and any webhook-specific middlewares
		r.Use(middleware.Logger)
		r.Post("/v1/webhooks/global-pay", d.Service.HandleGlobalPayWebhook)
		r.Post("/v1/webhooks/adyen", d.Service.HandleAdyenWebhook)
		r.Post("/v1/webhooks/stripe", d.Service.HandleStripeWebhook)
		// UNWIRED: Payme Merchant API + Click SHOP handlers are implemented
		// (payment.HandlePaymeWebhook / HandleClickWebhook) but must not be
		// reachable until an explicit wire decision. Launch path is Cash +
		// Global Pay + MySoliq + bank-file.
		// r.Post("/v1/webhooks/payme", d.Service.HandlePaymeWebhook)
		// r.Post("/v1/webhooks/click", d.Service.HandleClickWebhook)
	})
}
