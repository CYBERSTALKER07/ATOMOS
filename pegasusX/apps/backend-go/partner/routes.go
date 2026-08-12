package partner

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// RegisterPartnerRoutes mounts /partner/v1/* behind partner API key / OAuth auth.
func RegisterPartnerRoutes(r chi.Router, keys KeyRepository, h *Handlers) {
	RegisterPartnerRoutesOpts(r, AuthOptions{Keys: keys}, h)
}

// RegisterPartnerRoutesOpts mounts partner routes with OAuth JWT dual-accept.
func RegisterPartnerRoutesOpts(r chi.Router, authOpts AuthOptions, h *Handlers) {
	r.Route("/partner/v1", func(pr chi.Router) {
		// Token endpoint is unauthenticated (client_credentials).
		pr.Post("/oauth/token", h.HandleOAuthToken)
		// AS2 receive is cert/AS2-From identity — not Bearer (RFC 4130).
		pr.Post("/as2", h.HandleAS2Receive)

		pr.Group(func(ar chi.Router) {
			ar.Use(AuthMiddlewareOpts(authOpts))
			ar.With(RequirePartner(ScopeOrdersWrite)).Post("/orders", h.HandleCreateOrder)
			ar.With(RequirePartner(ScopeOrdersRead)).Get("/orders/{orderID}", h.HandleGetOrder)
			ar.With(RequirePartner(ScopeCatalogRead)).Get("/catalog", h.HandleCatalog)
			ar.With(RequirePartner(ScopeCatalogWrite)).Put("/catalog/products", h.HandleUpsertProducts)
			ar.With(RequirePartner(ScopeCatalogWrite)).Put("/catalog/prices", h.HandleUpsertPrices)
			ar.With(RequirePartner(ScopeInventoryRead)).Get("/inventory/availability", h.HandleAvailability)
			ar.With(RequirePartner(ScopeInventoryWrite)).Put("/inventory/stock", h.HandleUpsertStock)
			ar.With(RequirePartner(ScopeDemandWrite)).Post("/demand/pos-feed", h.HandlePOSDemandFeed)
			ar.With(RequirePartner(ScopeWebhooksManage)).Get("/webhooks", h.HandleListWebhooks)
			ar.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks", h.HandleCreateWebhook)
			ar.With(RequirePartner(ScopeWebhooksManage)).Delete("/webhooks/{subscriptionID}", h.HandleDeactivateWebhook)
			ar.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks/{subscriptionID}/ping", h.HandlePingWebhook)
			ar.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks/{subscriptionID}/rotate-secret", h.HandleRotateWebhookSecret)
			ar.With(RequirePartner(ScopeWebhooksManage)).Get("/webhooks/dead-letter", h.HandleListDeadLetter)
			ar.With(RequirePartner(ScopeWebhooksManage)).Post("/webhooks/dead-letter/{attemptID}/replay", h.HandleReplayDeadLetter)
			ar.With(RequirePartner(ScopeExportsRead)).Post("/exports", h.HandleCreateExport)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/exports", h.HandleListExports)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/exports/{jobID}", h.HandleGetExport)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/edi/documents", h.HandleListEdiDocuments)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/edi/documents/{documentID}", h.HandleGetEdiDocument)
			ar.With(RequirePartner(ScopeExportsRead)).Post("/edi/documents/{documentID}/replay", h.HandleReplayEdiDocument)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/coa", h.HandleGetCoa)
			ar.With(RequirePartner(ScopeExportsRead)).Put("/coa", h.HandlePutCoa)
			ar.With(RequirePartner(ScopeExportsRead)).Get("/as2/config", h.HandleGetAs2)
			ar.With(RequirePartner(ScopeExportsRead)).Put("/as2/config", h.HandlePutAs2)
		})
	})
}

// RegisterAdminKeyRoutes mounts JWT-gated key issuance for humans.
func RegisterAdminKeyRoutes(r chi.Router, h *Handlers) {
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Post("/v1/admin/partner-keys", h.HandleIssueKey)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Get("/v1/admin/partner-keys", h.HandleListKeys)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Post("/v1/admin/partner-keys/{keyID}/revoke", h.HandleRevokeKey)
	// Supplier portal convenience alias (ADMIN role is supplier portal session).
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-keys", h.HandleIssueKey)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-keys", h.HandleListKeys)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-keys/{keyID}/revoke", h.HandleRevokeKey)
	// Retailer self-serve partner keys (incl. environment=SANDBOX).
	r.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/retailer/partner-keys", h.HandleIssueKey)
	r.With(auth.RequireRole(auth.RoleRetailer)).Get("/v1/retailer/partner-keys", h.HandleListKeys)
	r.With(auth.RequireRole(auth.RoleRetailer)).Post("/v1/retailer/partner-keys/{keyID}/revoke", h.HandleRevokeKey)

	// Wave 2A: webhook ops + SFTP + exports via supplier JWT
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-webhooks", h.HandleListWebhooks)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks", h.HandleCreateWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Delete("/v1/supplier/partner-webhooks/{subscriptionID}", h.HandleDeactivateWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks/{subscriptionID}/ping", h.HandlePingWebhook)
	r.With(auth.RequireRole(auth.RoleAdmin)).Post("/v1/supplier/partner-webhooks/{subscriptionID}/rotate-secret", h.HandleRotateWebhookSecret)
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
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-coa", h.HandleGetCoa)
	r.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/supplier/partner-coa", h.HandlePutCoa)
	r.With(auth.RequireRole(auth.RoleAdmin)).Get("/v1/supplier/partner-as2", h.HandleGetAs2)
	r.With(auth.RequireRole(auth.RoleAdmin)).Put("/v1/supplier/partner-as2", h.HandlePutAs2)

	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Get("/v1/admin/partner-sftp", h.HandleGetSftp)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Put("/v1/admin/partner-sftp", h.HandlePutSftp)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Get("/v1/admin/partner-coa", h.HandleGetCoa)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Put("/v1/admin/partner-coa", h.HandlePutCoa)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Get("/v1/admin/partner-as2", h.HandleGetAs2)
	r.With(auth.RequireRole(auth.RoleAdmin, auth.RoleRetailer, auth.RolePlatformAdmin)).Put("/v1/admin/partner-as2", h.HandlePutAs2)
}

// PartnerRateLimitKey returns a limiter key for reliability middleware (KeyId).
func PartnerRateLimitKey(r *http.Request) string {
	if p, ok := PrincipalFromContext(r.Context()); ok && p.KeyID != "" {
		return "partner:" + p.KeyID
	}
	return ""
}
