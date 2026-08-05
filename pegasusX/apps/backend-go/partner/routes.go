package partner

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RegisterPartnerRoutes mounts /partner/v1/* behind partner API key auth.
func RegisterPartnerRoutes(r chi.Router, keys KeyRepository, h *Handlers) {
	r.Route("/partner/v1", func(pr chi.Router) {
		pr.Use(AuthMiddleware(keys))
		pr.With(RequirePartner(ScopeOrdersWrite)).Post("/orders", h.HandleCreateOrder)
		pr.With(RequirePartner(ScopeOrdersRead)).Get("/orders/{orderID}", h.HandleGetOrder)
		pr.With(RequirePartner(ScopeCatalogRead)).Get("/catalog", h.HandleCatalog)
		pr.With(RequirePartner(ScopeInventoryRead)).Get("/inventory/availability", h.HandleAvailability)
		pr.With(RequirePartner(ScopeWebhooksManage)).Get("/webhooks", h.HandleListWebhooks)
		pr.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks", h.HandleCreateWebhook)
		pr.With(RequirePartner(ScopeWebhooksManage)).Delete("/webhooks/{subscriptionID}", h.HandleDeactivateWebhook)
		pr.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks/{subscriptionID}/ping", h.HandlePingWebhook)
		pr.With(RequirePartner(ScopeWebhooksManage)).Get("/webhooks/dead-letter", h.HandleListDeadLetter)
		pr.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks/dead-letter/{attemptID}/replay", h.HandleReplayDeadLetter)
		pr.With(RequirePartner(ScopeExportsRead)).Post("/exports", h.HandleCreateExport)
		pr.With(RequirePartner(ScopeExportsRead)).Get("/exports", h.HandleListExports)
		pr.With(RequirePartner(ScopeExportsRead)).Get("/exports/{jobID}", h.HandleGetExport)
		pr.With(RequirePartner(ScopeExportsRead)).Get("/edi/documents", h.HandleListEdiDocuments)
		pr.With(RequirePartner(ScopeExportsRead)).Get("/edi/documents/{documentID}", h.HandleGetEdiDocument)
		pr.With(RequirePartner(ScopeExportsRead)).Post("/edi/documents/{documentID}/replay", h.HandleReplayEdiDocument)
	})
}

// RegisterAdminKeyRoutes mounts JWT-gated key issuance for humans.
func RegisterAdminKeyRoutes(r chi.Router, h *Handlers) {
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Post("/v1/admin/partner-keys", h.HandleIssueKey)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Get("/v1/admin/partner-keys", h.HandleListKeys)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Post("/v1/admin/partner-keys/{keyID}/revoke", h.HandleRevokeKey)
	// Supplier portal convenience alias (ADMIN role is supplier portal session).
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-keys", h.HandleIssueKey)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-keys", h.HandleListKeys)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-keys/{keyID}/revoke", h.HandleRevokeKey)

	// Wave 2A: webhook ops + SFTP + exports via supplier JWT
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-webhooks", h.HandleListWebhooks)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks", h.HandleCreateWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Delete("/v1/supplier/partner-webhooks/{subscriptionID}", h.HandleDeactivateWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks/{subscriptionID}/ping", h.HandlePingWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-webhooks/dead-letter", h.HandleListDeadLetter)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks/dead-letter/{attemptID}/replay", h.HandleReplayDeadLetter)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-sftp", h.HandleGetSftp)
	r.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/supplier/partner-sftp", h.HandlePutSftp)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-exports", h.HandleCreateExport)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-exports", h.HandleListExports)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-exports/{jobID}", h.HandleGetExport)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-edi/documents", h.HandleListEdiDocuments)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-edi/documents/{documentID}", h.HandleGetEdiDocument)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-edi/documents/{documentID}/replay", h.HandleReplayEdiDocument)

	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Get("/v1/admin/partner-sftp", h.HandleGetSftp)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer)).Put("/v1/admin/partner-sftp", h.HandlePutSftp)
}

// PartnerRateLimitKey returns a limiter key for reliability middleware (KeyId).
func PartnerRateLimitKey(r *http.Request) string {
	if p, ok := PrincipalFromContext(r.Context()); ok && p.KeyID != "" {
		return "partner:" + p.KeyID
	}
	return ""
}
