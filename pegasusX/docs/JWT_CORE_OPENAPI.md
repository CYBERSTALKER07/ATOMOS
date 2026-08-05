# JWT Core OpenAPI

Codegen-ready OpenAPI **3.0.3** contract for the high-traffic human JWT spine.

**Contract:** [`contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml)

**Gate:** `make jwt-openapi-gate` (existence + required path markers)

## Scope

~45 operations across auth, profiles, orders/delivery, warehouse ops, supplier inventory/manifests/pricing/topology, retailer stock/settings/suppliers, payloader manifests/seal, returns inbound, notifications, and catalog products.

- Auth: `BearerJWT` only (role login/refresh issue tokens)
- Does **not** duplicate `/partner/v1/*` — see [`PARTNER_API.md`](./PARTNER_API.md) / [`partner.openapi.yaml`](../contracts/partner.openapi.yaml)
- Paths reconciled against `*routes/routes.go` and `@pegasusx/api-client`

## Codegen status

| Surface | Status |
|---------|--------|
| Core JWT OpenAPI contract | **Wired** |
| Hand-written `@pegasusx/api-client` | Unchanged (still canonical for clients) |
| Full ~411-path catalog + SDK replace of ApiClient | Residual |

## Usage

```bash
make jwt-openapi-gate
```

Future: generate TypeScript / native clients from this file (e.g. openapi-typescript) without changing event contracts (`gen-contracts` / Quicktype remain separate).
