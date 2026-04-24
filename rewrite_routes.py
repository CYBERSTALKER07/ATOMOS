with open("the-lab-monorepo/apps/backend-go/webhookroutes/routes.go", "w") as f:
    f.write('''// Package webhookroutes owns the /v1/webhooks/* surface. Endpoints here are
// intentionally unauthenticated at the JWT layer — each gateway authenticates
// its webhook via signature or HTTP Basic (verified inside the handler). The
// PriorityGuard shedder is applied so a webhook storm cannot starve the user
// traffic tier.
package webhookroutes

import (
\t"net/http"

\t"github.com/go-chi/chi/v5"

\t"backend-go/payment"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register webhook routes.
type Deps struct {
\tWebhookSvc    *payment.WebhookService
\tLog           Middleware
\tPriorityGuard Middleware
}

// RegisterRoutes mounts the payment gateway webhook surface:
//
//\tPOST /v1/webhooks/global-pay — Global Pay HPP return
//\tPOST /v1/webhooks/stripe     — Stripe event (signature-verified)
func RegisterRoutes(r chi.Router, d Deps) {
\tguard := d.PriorityGuard
\tlog := d.Log
\tsvc := d.WebhookSvc

\tr.HandleFunc("/v1/webhooks/global-pay", guard(log(svc.HandleGlobalPayWebhook)))
\tr.HandleFunc("/v1/webhooks/stripe", guard(log(svc.HandleStripeWebhook)))
}
''')
