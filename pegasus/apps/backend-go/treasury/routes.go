package treasury

import (
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"

	"backend-go/auth"
)

// Middleware is the handler-wrap contract for the logging observability layer.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators required to register the treasury surface.
type Deps struct {
	Spanner *spanner.Client
	Log     Middleware
}

// RegisterRoutes mounts the /v1/treasury/* surface plus the supplier
// settlement-report endpoint (which is read-mostly treasury data, served
// under /v1/supplier/settlement-report for UI routing symmetry with the
// Supplier Cockpit).
func RegisterRoutes(r chi.Router, d Deps) {
	supplierAdmin := []string{"SUPPLIER", "ADMIN"}

	r.HandleFunc("/v1/treasury/ledger",
		auth.RequireRole(supplierAdmin, d.Log(TreasuryHandler(d.Spanner))))
	r.HandleFunc("/v1/treasury/cash-holdings",
		auth.RequireRole(supplierAdmin, d.Log(CashHoldingsHandler(d.Spanner))))
	r.HandleFunc("/v1/treasury/batch-settle",
		auth.RequireRole(supplierAdmin, d.Log(HandleBatchSettle(d.Spanner))))
	r.HandleFunc("/v1/treasury/invoice/status",
		auth.RequireRole(supplierAdmin, d.Log(HandleInvoiceStatusOverride(d.Spanner))))
	r.HandleFunc("/v1/supplier/settlement-report",
		auth.RequireRole(supplierAdmin, d.Log(HandleSettlementReport(d.Spanner))))
}
