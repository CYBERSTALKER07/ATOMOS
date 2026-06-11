// Package promotionroutes mounts supplier and retailer promotion endpoints.
package promotionroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
)

// Deps is the narrow contract for promotion route mounting.
type Deps struct {
	Service   *promotion.Service
	JWTSecret string
}

// RegisterRoutes mounts promotion CRUD (supplier) and quote (retailer).
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Service == nil {
		return
	}
	r.Group(func(gr chi.Router) {
		gr.Use(auth.CookieAuth(d.JWTSecret))
		gr.Use(auth.RequireRole(auth.RoleAdmin))
		gr.Get("/v1/supplier/promotions", d.Service.HandleListSupplierPromotions)
		gr.Post("/v1/supplier/promotions", d.Service.HandleCreateSupplierPromotion)
		gr.Patch("/v1/supplier/promotions/{promotionID}", d.Service.HandleUpdateSupplierPromotion)
		gr.Delete("/v1/supplier/promotions/{promotionID}", d.Service.HandleDeactivateSupplierPromotion)
	})

}
