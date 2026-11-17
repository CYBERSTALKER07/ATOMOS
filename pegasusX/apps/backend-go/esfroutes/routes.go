// Package esfroutes mounts ATOMOS M1.4 ESF document, Soliq submission, VAT tie-out, and IKPU audit endpoints onto chi.
package esfroutes

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
)

// Deps holds runtime dependencies for mounting M1.4 fiscal ESF routes.
type Deps struct {
	Service         *esf.Service
	JWTSecret       string
	AllowAuthBypass bool
}

// RegisterRoutes mounts all M1.4 ESF and fiscal compliance HTTP endpoints.
func RegisterRoutes(r chi.Router, d Deps) {
	if r == nil || d.Service == nil {
		return
	}

	roleGuard := auth.RequireRole(auth.RoleAdmin, auth.RolePlatformAdmin)
	readPerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermReconManage,
		auth.PermAuditRead,
		auth.PermCreditNoteApprove,
	)
	writePerm := auth.RequireAnySupplierPerm(
		auth.PermJournalPost,
		auth.PermCreditNoteApprove,
		auth.PermReconManage,
	)

	mount := func(gr chi.Router) {
		gr.With(roleGuard, readPerm).Get("/v1/fiscal/esf", d.Service.HandleList)
		gr.With(roleGuard, writePerm).Post("/v1/fiscal/esf", d.Service.HandleCreateOutbound)
		gr.With(roleGuard, writePerm).Post("/v1/fiscal/esf/inbound", d.Service.HandleCreateInbound)
		gr.With(roleGuard, readPerm).Get("/v1/fiscal/esf/{id}", d.Service.HandleGet)
		gr.With(roleGuard, writePerm).Post("/v1/fiscal/esf/{id}/resend", d.Service.HandleResend)
		gr.With(roleGuard, writePerm).Post("/v1/fiscal/esf/{id}/status", d.Service.HandleTransitionStatus)
		gr.With(roleGuard, readPerm).Get("/v1/fiscal/vat-summary", d.Service.HandleGetVATSummary)
		gr.With(roleGuard, readPerm).Get("/v1/fiscal/missing-ikpu", d.Service.HandleListMissingIKPU)
	}

	r.Group(func(gr chi.Router) {
		if strings.TrimSpace(d.JWTSecret) != "" {
			gr.Use(auth.CookieAuth(d.JWTSecret))
		}
		gr.Use(auth.AttachTenantFromClaims)
		auth.ProtectMutations(gr, auth.MutationGuardConfig{
			AllowBypass: d.AllowAuthBypass,
		}, mount)
	})
}
