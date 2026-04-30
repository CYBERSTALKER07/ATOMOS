package sync

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
)

// RegisterRoutes mounts the Desert Protocol endpoints onto r. The `wrap`
// parameter is the logging/observability middleware supplied by the caller
// (typically loggingMiddleware from main) — passed explicitly to keep this
// package free of any dependency on the composition root.
//
// Routes:
//
//	POST /v1/sync/batch   — driver offline delivery queue upload (DRIVER only)
//	GET  /v1/sync/catchup — reconnection delta feed (DRIVER, ADMIN, SUPPLIER, RETAILER)
//	GET  /v1/sync/delta   — mobile-optimised unix-ms delta feed with pagination
func RegisterRoutes(r chi.Router, s *spanner.Client, wrap func(http.HandlerFunc) http.HandlerFunc) {
	r.HandleFunc("/v1/sync/batch",
		auth.RequireRole([]string{"DRIVER"}, wrap(HandleBatchSync(s))),
	)

	r.HandleFunc("/v1/sync/catchup",
		auth.RequireRole([]string{"DRIVER", "ADMIN", "SUPPLIER", "RETAILER"}, wrap(HandleCatchup(s))),
	)

	r.HandleFunc("/v1/sync/delta",
		auth.RequireRole([]string{"DRIVER", "ADMIN", "SUPPLIER", "RETAILER"}, wrap(HandleDelta(s))),
	)
}
