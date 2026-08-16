-- GS-I: per-supplier OIDC (buyer org SSO). Not a second tenant key.
-- No client_secret column — secrets stay in GSM/env, not Spanner.

CREATE TABLE SupplierOIDC (
  SupplierId              STRING(36)  NOT NULL,
  Issuer                  STRING(512) NOT NULL,
  ClientId                STRING(256) NOT NULL,
  Audience                STRING(256),
  AuthorizationEndpoint   STRING(512),
  RedirectURI             STRING(512),
  Enabled                 BOOL        NOT NULL DEFAULT (FALSE),
  UpdatedAt               TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (SupplierId);
