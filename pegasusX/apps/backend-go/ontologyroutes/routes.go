// Package ontologyroutes mounts the ATOMOS M0.5 Semantic Ontology HTTP API endpoints.
package ontologyroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
)

// Deps bundles dependencies required to mount /v1/ontology/* routes.
type Deps struct {
	Service         *ontology.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts:
//   - GET  /v1/ontology/types
//   - GET  /v1/ontology/objects
//   - GET  /v1/ontology/objects/{type}/{id}/neighbors
//   - POST /v1/ontology/rebuild
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	readRoleGuard := auth.RequireRole(
		auth.RoleAdmin,
		auth.RolePlatformAdmin,
		auth.RoleWarehouseAdmin,
		auth.RoleWarehouse,
		auth.RoleFactoryAdmin,
		auth.RoleFactory,
		auth.RolePayload,
	)
	readPermGuard := auth.RequireAnySupplierPerm(auth.PermAuditRead, auth.PermConfigManage)

	rebuildRoleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	rebuildPermGuard := auth.RequireSupplierPerm(auth.PermConfigManage)

	mount := func(gr chi.Router) {
		gr.With(readRoleGuard, readPermGuard).Get("/v1/ontology/types", d.Service.HandleListTypes)
		gr.With(readRoleGuard, readPermGuard).Get("/v1/ontology/objects", d.Service.HandleListObjects)
		gr.With(readRoleGuard, readPermGuard).Get("/v1/ontology/objects/{type}/{id}/neighbors", d.Service.HandleGetNeighbors)
		gr.With(rebuildRoleGuard, rebuildPermGuard).Post("/v1/ontology/rebuild", d.Service.HandleRebuild)
	}

	if d.AllowAuthBypass {
		r.Group(mount)
		return
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: false,
		}, mount)
	})
}
