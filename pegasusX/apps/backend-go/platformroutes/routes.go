// Package platformroutes mounts version policy and device-token routes.
package platformroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/geolocation"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/tenantreg"
)

// Deps supplies platform handlers.
type Deps struct {
	Handler        *platform.Handler
	GeocodeHandler *geolocation.Handler
	TenantRegister *tenantreg.Service
	JWTSecret      string
	JWTIssuer      string
}

// RegisterRoutes mounts /v1/platform/* and shared device-token registration.
func RegisterRoutes(r chi.Router, d Deps) {
	if d.Handler == nil {
		return
	}
	r.Get("/v1/platform/client-policy", d.Handler.HandleClientPolicy)
	r.Get("/v1/platform/client-config", d.Handler.HandleClientConfig)
	r.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/platform/client-policy", d.Handler.HandleUpsertPolicy)
	// Evidence photo upload (claims / driver OS&D) — signed GCS PUT URL.
	r.With(auth.RequireRole(
		auth.RoleRetailer, auth.RoleDriver, auth.RolePayload,
		auth.RoleAdmin, auth.RoleWarehouseAdmin, auth.RoleWarehouse,
	)).Get("/v1/media/upload-ticket", d.Handler.HandleMediaUploadTicket)
	r.Post("/v1/user/device-token", d.Handler.HandleDeviceToken)
	if d.GeocodeHandler != nil {
		r.With(auth.RequireAnyAuthenticated()).Group(func(gr chi.Router) {
			geolocation.RegisterRoutes(gr, d.GeocodeHandler)
		})
	}
	if d.JWTSecret != "" {
		r.Post("/v1/auth/refresh", auth.HandleTokenRefresh(d.JWTSecret, d.JWTIssuer))
		r.Post("/v1/auth/logout", auth.HandleLogout(d.JWTSecret))
		// GS-A: session + market pack (any authenticated role).
		r.With(auth.RequireAnyAuthenticated()).Get("/v1/auth/session", auth.HandleSession)
	}
	if d.TenantRegister != nil {
		r.Post("/v1/platform/tenants/register", d.TenantRegister.HandleRegister)
	}
	r.Get("/v1/platform/cells", auth.HandleListCells)
	r.Get("/v1/platform/market-packs", auth.HandleListMarketPacks)
	r.Get("/v1/platform/market-packs/{code}", func(w http.ResponseWriter, req *http.Request) {
		auth.HandleGetMarketPack(chi.URLParam(req, "code"))(w, req)
	})
	r.Get("/v1/platform/partner-dialects", partner.HandleListPartnerDialects)
}
