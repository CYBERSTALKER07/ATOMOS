// Package orderroutes mounts the order URL surface onto chi. Handlers live in
// the order package; this file is thin by design.
package orderroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/claims"
	"github.com/pegasusx/pegasusx/apps/backend-go/compliance"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"
)

// Deps is the narrow dependency contract for this routes package.
type Deps struct {
	Service             *order.Service
	ClaimsService       *claims.Service
	TaxService          *tax.Service
	ComplianceHandler   *compliance.Handler
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

// RegisterRoutes mounts the order endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}

	// Public platform receipt QR target (no auth — redacted commercial view).
	r.Get("/v1/platform/receipts/{receiptID}", d.Service.HandleGetPlatformReceipt)

	mount := func(gr chi.Router) {
		gr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/order/create", d.Service.HandleCreate)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Patch("/v1/order/{orderID}/status", d.Service.HandleUpdateStatus)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleFactoryAdmin)).Post("/v1/orders/{orderID}/assign", d.Service.HandleAssignOrder)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/arrive", d.Service.HandleMarkArrived)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/deliver", d.Service.HandleSubmitDelivery)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/sync/batch", d.Service.HandleSyncBatch)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/confirm-offload", d.Service.HandleConfirmOffload)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/complete", d.Service.HandleCompleteOrder)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/order/collect-cash", d.Service.HandleCollectCash)
		gr.With(auth.RequireRole(auth.RoleDriver, auth.RoleAdmin, auth.RoleWarehouseAdmin)).Post("/v1/order/{orderID}/fiscal/retry", d.Service.HandleFiscalRetry)
		gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Post("/v1/order/{orderID}/force-complete", d.Service.HandleForceComplete)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/shop-closed/resolve", d.Service.HandleResolveShopClosed)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/fleet/route/request-early-complete", d.Service.HandleRequestEarlyComplete)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/confirm-payment-bypass", d.Service.HandleConfirmPaymentBypass)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/report-damage", d.Service.HandleReportDamage)
		gr.With(auth.RequireRole(auth.RoleDriver, auth.RoleRetailer, auth.RoleWarehouseAdmin, auth.RoleWarehouse, auth.RoleFactoryAdmin)).Post("/v1/delivery/report-condition", d.Service.HandleReportCondition)
		gr.With(auth.RequireRole(auth.RoleDriver, auth.RoleRetailer, auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse, auth.RoleFactoryAdmin)).Get("/v1/order/{orderID}/condition-reports", d.Service.ListConditionReports)

		gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse)).Get("/v1/order/{orderID}/timeline", d.Service.HandleGetOrderTimeline)
		gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse)).Get("/v1/order/{orderID}/status-context", d.Service.HandleGetOrderStatusContext)
		gr.With(auth.RequireRole(auth.RoleRetailer)).Get("/v1/order/{orderID}/qr-payload", d.Service.HandleGetQRPayload)
		gr.With(auth.RequireRole(auth.RoleDriver)).Post("/v1/delivery/scan-qr", d.Service.HandleDeliveryScanQR)
		gr.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/delivery/confirm-cash", d.Service.HandleRetailerConfirmCash)

		gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/reconciliation", d.Service.HandleListReconciliationOrders)
		gr.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/reconciliation/resolve", d.Service.HandleResolveReconciliation)

		// Post-delivery logistics claims (retailer concealed damage / OS&D).
		if d.ClaimsService != nil {
			gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin)).Post("/v1/orders/{orderID}/claims", d.ClaimsService.HandleFileOrderClaim)
			gr.With(auth.RequireRole(auth.RoleRetailer, auth.RoleAdmin, auth.RoleWarehouseAdmin)).Get("/v1/orders/{orderID}/claims", d.ClaimsService.HandleListOrderClaims)
			// Adjudication → supplier chargeback + optional GP partial refund.
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Get("/v1/supplier/claims", d.ClaimsService.HandleListSupplierClaims)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Post("/v1/claims/{claimID}/approve", d.ClaimsService.HandleApproveClaim)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Post("/v1/claims/{claimID}/reject", d.ClaimsService.HandleRejectClaim)
		}

		// Tax regime versioning (admin CRUD).
		if d.TaxService != nil {
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Post("/v1/admin/tax-regimes", d.TaxService.HandleCreateRegime)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Get("/v1/admin/tax-regimes", d.TaxService.HandleListRegimes)
			gr.With(auth.RequireRole(auth.RoleAdmin, auth.RoleWarehouseAdmin)).Get("/v1/admin/tax-regimes/{regimeID}", d.TaxService.HandleGetRegime)
		}
		
		// Compliance / Audit Dashboard
		if d.ComplianceHandler != nil {
			gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/compliance/dashboard", d.ComplianceHandler.GetDashboard)
			gr.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/compliance/export", d.ComplianceHandler.ExportCSV)
		}
	}

	auth.ProtectMutations(r, auth.MutationGuardConfig{
		FirebaseEnabled:  d.FirebaseAuthEnabled,
		FirebaseVerifier: d.FirebaseVerifier,
		AllowBypass:      d.AllowAuthBypass,
	}, mount)
}
