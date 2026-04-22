// Package webhookroutes owns the /v1/webhooks/* surface. Endpoints here are
// intentionally unauthenticated at the JWT layer — each gateway authenticates
// its webhook via signature or HTTP Basic (verified inside the handler). The
// PriorityGuard shedder is applied so a webhook storm cannot starve the user
// traffic tier.
package webhookroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend-go/payment"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register webhook routes.
type Deps struct {
	WebhookSvc    *payment.WebhookService
	Log           Middleware
	PriorityGuard Middleware
}

// RegisterRoutes mounts the payment gateway webhook surface:
//
//	POST /v1/webhooks/click      — Click gateway notification
//	POST /v1/webhooks/payme      — Payme JSON-RPC
//	POST /v1/webhooks/global-pay — Global Pay HPP return
//	POST /v1/webhooks/stripe     — Stripe event (signature-verified)
func RegisterRoutes(r chi.Router, d Deps) {
	guard := d.PriorityGuard
	log := d.Log
	svc := d.WebhookSvc

	r.HandleFunc("/v1/webhooks/click", guard(log(svc.HandleClickWebhook)))
	r.HandleFunc("/v1/webhooks/payme", guard(log(svc.HandlePaymeWebhook)))
	r.HandleFunc("/v1/webhooks/global-pay", guard(log(svc.HandleGlobalPayWebhook)))
	r.HandleFunc("/v1/webhooks/stripe", guard(log(svc.HandleStripeWebhook)))
}
