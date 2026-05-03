// Package retailerroutes owns the extracted retailer core route composition for
// analytics, profile and supplier reads, family-member management, server-side
// cart persistence, and retailer-initiated order actions such as AI review,
// preorder lifecycle updates, cancellation requests, and shop-closed response.
package retailerroutes

import (
	"context"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"

	"backend-go/analytics"
	"backend-go/auth"
	"backend-go/cache"
	"backend-go/order"
	"backend-go/proximity"
	"backend-go/supplier"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles the collaborators needed to mount the retailer core routes.
type Deps struct {
	Spanner        *spanner.Client
	ReadRouter     proximity.ReadRouter
	Cache          *cache.Cache
	CacheFlight    *singleflight.Group
	Order          *order.OrderService
	ShopClosedDeps *order.ShopClosedDeps
	Log            Middleware
}

// RegisterRoutes mounts the extracted retailer core surface:
//
//	GET /v1/retailer/analytics/expenses        — retailer expense analytics
//	GET /v1/retailer/analytics/detailed        — retailer detailed analytics
//	POST /v1/orders/request-cancel             — retailer cancellation request
//	POST /v1/retailer/shop-closed-response     — retailer response to shop-closed prompt
//	GET/POST /v1/retailer/family-members       — retailer family-member list/create
//	DELETE /v1/retailer/family-members/{id}    — retailer family-member delete
//	POST /v1/retailer/orders/confirm-ai        — retailer confirms AI order
//	POST /v1/retailer/orders/reject-ai         — retailer rejects AI order
//	POST /v1/orders/edit-preorder              — retailer edits scheduled preorder
//	POST /v1/orders/confirm-preorder           — retailer confirms scheduled preorder
//	GET/POST /v1/retailer/cart/sync            — retailer server-side cart sync
//	GET /v1/retailer/suppliers                 — retailer supplier favorites list
//	POST /v1/retailer/suppliers/{id}/{action}  — retailer supplier favorite add/remove
//	GET/PUT /v1/retailer/profile               — retailer profile
//	GET /v1/retailers/{retailerID}/orders      — retailer/mobile order list
//	GET /v1/retailer/tracking                  — retailer live tracking surface
func RegisterRoutes(r chi.Router, d Deps) {
	retailerRole := []string{"RETAILER"}
	log := d.Log

	r.HandleFunc("/v1/retailer/analytics/expenses",
		auth.RequireRole(retailerRole, log(analytics.HandleGetRetailerExpenses(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/retailer/analytics/detailed",
		auth.RequireRole(retailerRole, log(analytics.HandleRetailerDetailedAnalytics(d.Spanner, d.ReadRouter))))
	r.HandleFunc("/v1/orders/request-cancel",
		auth.RequireRole(retailerRole, log(order.HandleRequestCancel(d.Order))))
	r.HandleFunc("/v1/retailer/shop-closed-response",
		auth.RequireRole(retailerRole, log(d.Order.HandleShopClosedResponse(d.ShopClosedDeps))))
	r.HandleFunc("/v1/retailer/family-members",
		auth.RequireRole(retailerRole, log(familyMembersHandler(d))))
	r.HandleFunc("/v1/retailer/family-members/{memberID}",
		auth.RequireRole(retailerRole, log(auth.HandleDeleteFamilyMember(d.Spanner, retailerProfileInvalidator(d.Cache)))))
	r.HandleFunc("/v1/retailer/orders/confirm-ai",
		auth.RequireRole(retailerRole, log(order.HandleConfirmAiOrder(d.Order))))
	r.HandleFunc("/v1/retailer/orders/reject-ai",
		auth.RequireRole(retailerRole, log(order.HandleRejectAiOrder(d.Order))))
	r.HandleFunc("/v1/orders/edit-preorder",
		auth.RequireRole(retailerRole, log(order.HandleEditPreorder(d.Order))))
	r.HandleFunc("/v1/orders/confirm-preorder",
		auth.RequireRole(retailerRole, log(order.HandleConfirmPreorder(d.Order))))
	r.HandleFunc("/v1/retailer/cart/sync",
		auth.RequireRole(retailerRole, log(order.HandleCartSync(d.Spanner))))
	r.HandleFunc("/v1/retailer/suppliers",
		auth.RequireRole(retailerRole, log(supplier.HandleRetailerSuppliers(d.Spanner))))
	r.HandleFunc("/v1/retailer/suppliers/{supplierID}/{action}",
		auth.RequireRole(retailerRole, log(supplier.HandleRetailerSuppliers(d.Spanner))))
	r.HandleFunc("/v1/retailer/profile",
		auth.RequireRole(retailerRole, log(supplier.HandleRetailerProfile(d.Spanner, d.Cache, d.CacheFlight))))
	r.HandleFunc("/v1/retailers/{retailerID}/orders",
		auth.RequireRole([]string{"ADMIN", "RETAILER"}, log(handleRetailerOrders(d))))
	r.HandleFunc("/v1/retailer/tracking",
		auth.RequireRole(retailerRole, log(handleRetailerTracking(d))))
}

func familyMembersHandler(d Deps) http.HandlerFunc {
	listMembers := auth.HandleListFamilyMembers(d.Spanner)
	createMember := auth.HandleCreateFamilyMember(d.Spanner, retailerProfileInvalidator(d.Cache))

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listMembers(w, r)
		case http.MethodPost:
			createMember(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func retailerProfileInvalidator(c *cache.Cache) func(context.Context, ...string) {
	return func(ctx context.Context, keys ...string) {
		if c != nil {
			c.Invalidate(ctx, keys...)
		}
	}
}
