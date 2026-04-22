// Package driverroutes owns the /v1/driver/* surface. Handler implementations
// are sourced from backend-go/fleet, backend-go/supplier, backend-go/order,
// backend-go/crypto, and backend-go/supplier.ManifestService — this package
// composes them under a single DRIVER-role mount.
package driverroutes

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/crypto"
	"backend-go/fleet"
	"backend-go/order"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register /v1/driver routes.
// OrderService is narrowed to the pending-collections handler factory.
type Deps struct {
	Spanner     *spanner.Client
	Order       *order.OrderService
	ManifestSvc *supplier.ManifestService
	Cache       *cache.Cache
	CacheFlight *singleflight.Group
	Log         Middleware
}

// RegisterRoutes mounts the driver-facing surface:
//
//	GET  /v1/driver/earnings            — per-driver earnings report
//	GET  /v1/driver/history             — delivery history
//	GET  /v1/driver/availability        — on-shift toggle
//	GET  /v1/driver/pending-collections — outstanding cash collections
//	GET  /v1/driver/profile             — driver profile + vehicle assignment
//	GET  /v1/driver/manifest-gate       — Ghost Stop Prevention gate check
//	GET  /v1/driver/manifest            — Hash Manifest Protocol (token hashes)
func RegisterRoutes(r chi.Router, d Deps) {
	s := d.Spanner
	log := d.Log
	driver := []string{"DRIVER"}

	r.HandleFunc("/v1/driver/earnings",
		auth.RequireRole(driver, log(fleet.HandleDriverEarnings(s))))
	r.HandleFunc("/v1/driver/history",
		auth.RequireRole(driver, log(fleet.HandleDriverHistory(s))))
	r.HandleFunc("/v1/driver/availability",
		auth.RequireRole(driver, log(fleet.HandleDriverAvailability(s))))
	r.HandleFunc("/v1/driver/pending-collections",
		auth.RequireRole(driver, log(order.HandlePendingCollections(d.Order))))
	r.HandleFunc("/v1/driver/profile",
		auth.RequireRole(driver, log(supplier.HandleDriverProfile(s, d.Cache, d.CacheFlight))))

	// Ghost Stop Prevention gate — wraps the manifest service's method.
	r.HandleFunc("/v1/driver/manifest-gate",
		auth.RequireRole(driver, log(func(w http.ResponseWriter, req *http.Request) {
			d.ManifestSvc.HandleDriverManifestGate()(w, req)
		})))

	// Hash Manifest Protocol — downloads today's SHA-256 token hashes for
	// offline drop verification. DRIVER role only — raw tokens never transmitted.
	r.HandleFunc("/v1/driver/manifest",
		auth.RequireRole(driver, log(crypto.GetDriverManifestHandler(s))))
}
