// Package payloaderroutes owns the /v1/payloader/* surface that serves the
// Warehouse-staff Payloader app. Handlers live in backend-go/supplier and
// backend-go/fleet.
package payloaderroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
	"backend-go/fleet"
	"backend-go/idempotency"
	"backend-go/proximity"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/payloader routes.
type Deps struct {
	Spanner    *spanner.Client
	ReadRouter proximity.ReadRouter
	Log        Middleware
}

// RegisterRoutes mounts the payloader-facing surface:
//
//	GET  /v1/payloader/trucks             — vehicles for the payloader's supplier
//	GET  /v1/payloader/orders             — orders scoped to the payloader's vehicles
//	POST /v1/payloader/recommend-reassign — GPS-based truck recommendations
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	payloader := []string{"PAYLOADER"}
	payloaderSupplyAdmin := []string{"PAYLOADER", "ADMIN", "SUPPLIER"}

	r.HandleFunc("/v1/payloader/trucks",
		auth.RequireRole(payloader, log(supplier.HandlePayloaderTrucks(s))))
	r.HandleFunc("/v1/payloader/orders",
		auth.RequireRole(payloader, log(supplier.HandlePayloaderOrders(s))))
	r.HandleFunc("/v1/payloader/recommend-reassign",
		auth.RequireRole(payloaderSupplyAdmin, log(idempotency.Guard(fleet.HandleRecommendReassign(s, d.ReadRouter)))))
}
