// Package returnsroutes mounts reverse-logistics gate endpoints for payloader and warehouse roles.
package returnsroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/returns"
)

// Deps is the narrow dependency contract for returns gate routes.
type Deps struct {
	Service             *returns.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
}

// RegisterRoutes mounts inbound return scanning endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	mount := func(rr chi.Router) {
		rr.Get("/v1/returns/inbound", d.Service.HandleInboundList)
		rr.Post("/v1/returns/inbound/sessions", d.Service.HandleStartReceiveSession)
		rr.Post("/v1/returns/inbound/scan", d.Service.HandleInboundScan)
		rr.Post("/v1/returns/inbound/confirm", d.Service.HandleInboundConfirm)
		rr.Get("/v1/returns/history", d.Service.HandleReturnsHistory)
		rr.Get("/v1/catalog/barcode/{ean}", d.Service.HandleBarcodeLookup)
	}
	allowed := []auth.Role{auth.RolePayload, auth.RoleWarehouse, auth.RoleWarehouseAdmin, auth.RoleAdmin}
	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(allowed...))
			mount(gr)
		})
		return
	}
	r.Group(func(gr chi.Router) {
		gr.Use(auth.RequireRole(allowed...))
		mount(gr)
	})
}

// RegisterDriverRoutes mounts driver return-goods summary.
func RegisterDriverRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	mount := func(rr chi.Router) {
		rr.Get("/v1/driver/return-goods", d.Service.HandleDriverReturnGoods)
	}
	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(auth.RoleDriver))
			mount(gr)
		})
		return
	}
	r.Group(func(gr chi.Router) {
		gr.Use(auth.RequireRole(auth.RoleDriver))
		mount(gr)
	})
}

// RegisterSupplierHistory mounts supplier-scoped returns history.
func RegisterSupplierHistory(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		d.Service.HandleReturnsHistory(w, req)
	}
	allowed := []auth.Role{auth.RoleAdmin}
	if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
		r.Group(func(gr chi.Router) {
			gr.Use(auth.FirebaseAuth(d.FirebaseVerifier))
			gr.Use(auth.RequireRole(allowed...))
			gr.Get("/v1/supplier/returns/history", handler)
		})
		return
	}
	r.Group(func(gr chi.Router) {
		gr.Use(auth.RequireRole(allowed...))
		gr.Get("/v1/supplier/returns/history", handler)
	})
}
