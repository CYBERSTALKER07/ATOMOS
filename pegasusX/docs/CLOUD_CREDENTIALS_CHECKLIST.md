# Cloud Credentials Checklist

## Required GSM secrets (backend / payments)

ExternalSecret `backend-go-secrets` is **atomic** (all `spec.data` keys required). Tenant prefix today: `pegasusx-ssmr-*` (ssmr-shaped; see [`PROD_WIRING_AND_THIRD_PARTIES.md`](../artifacts/PROD_WIRING_AND_THIRD_PARTIES.md)).

| K8s `secretKey` | GSM id | Real vs stub |
|-----------------|--------|----------------|
| `jwt-secret` | `…-jwt-secret` | **Real** before any auth |
| `internal-api-key` | `…-internal-api-key` | **Real** for optimizer / internal calls |
| `google-maps-api-key` | `…-google-maps-api-key` | **Real** for geocode / Routes |
| `global-pay-webhook-secret` | `…-global-pay-webhook-secret` | **Real** when GP webhooks live |
| `global-pay-service-id` | `…-global-pay-service-id` | **Real** for GP merchant |
| `global-pay-username` | `…-global-pay-username` | **Real** for GP merchant |
| `global-pay-password` | `…-global-pay-password` | **Real** (merchant-owned) |
| `redis-password` | `…-redis-auth` | **Real** Memorystore AUTH |
| `adyen-webhook-secret` | `…-adyen-webhook-secret` | Stub OK until Adyen rail live |
| `stripe-webhook-secret` | `…-stripe-webhook-secret` | Stub OK until Stripe rail live |
| `payme-webhook-secret` | `…-payme-webhook-secret` | Stub OK until Payme rail live |
| `click-webhook-secret` | `…-click-webhook-secret` | Stub OK until Click rail live |

- Terraform shells all 12 names (`infra/terraform/main.tf`). Versions: pass TF vars or run `scripts/phase0_sync_gsm_secrets.sh` (`ENSURE_ES_STUBS=1` writes `unused-rail-placeholder` when env empty).
- Emergency without ESO: `scripts/bootstrap_k8s_secrets.sh` (`SECRET_PREFIX=pegasusx-ssmr`).
- Env aliases: `GLOBAL_PAY_USERNAME` / `GLOBAL_PAY_PASSWORD` / `GLOBAL_PAY_WEBHOOK_SECRET`, `JWT_SECRET`, `GOOGLE_MAPS_API_KEY`

## Maps Platform — key model (do not share one key everywhere)

| Key | Consumers | Restrictions |
|-----|-----------|----------------|
| Server (`…-google-maps-api-key`) | backend-go | IP / GKE NAT; Geocoding + Places + Routes (`routes.googleapis.com`). Used for `/v1/platform/geocode/*` and route geometry (`ROUTING_PROVIDER=auto\|google`). |
| Android SDK (`…-maps-android-api-key`) | Driver/retailer (and any Google Maps Compose apps) | Package name + SHA-1/SHA-256 |
| iOS SDK (`…-maps-ios-api-key`) | Native Google Maps iOS if used; MapKit needs no Google key | Bundle ID |

Terraform: [`infra/terraform/maps_platform.tf`](../infra/terraform/maps_platform.tf) enables Maps APIs and optional GSM shells for Android/iOS keys.

## Billing

- Project-level budget: `infra/terraform/budget.tf` (`monthly_budget_usd` + `budget_alert_emails`)
- After enabling Routes, watch Maps Platform SKU spend; prefer persisted polylines (seal) over live Routes on every GET

## Optional / regional

- `ROUTING_OSRM_URL` — self-hosted OSRM fallback when PVC extract is loaded
- MapLibre portals use free Carto-style tiles (no Mapbox token required)

## Dispatch optimizer (not a GSM third-party)

- `OPTIMIZER_BASE_URL` — in-cluster DNS to `optimizer-core:8082` when the sidecar is deployed
- **Not required for API boot**; missing/unreachable → H3 BinPack (`fallback_phase1`)
- Cloud SSMR/prod: sidecar not live until AR image + replicas ≥ 1 — [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md)

## Partner API (Gate 3)

- No new GSM secrets for Wave 1 — keys and webhook signing secrets live in Spanner (`PartnerApiKeys`, `WebhookSubscriptions.SigningSecret`)
- Optional `PARTNER_JWT_SECRET` for OAuth2 access tokens (else derived from `JWT_SECRET` so tokens never verify as human sessions)
- AS2 PEM material via SecretRef → GSM / `PARTNER_AS2_SECRET_<REF>` (never store PEMs in Spanner)
- Issue keys via authenticated `POST /v1/admin/partner-keys`; never commit plaintext `pxk_` / `whsec_` secrets
- OAuth: `POST /partner/v1/oauth/token` (`client_credentials`) — clients are existing partner keys
- Apply DDL migration `apps/backend-go/schema/migrations/20260805_partner_integration_layer.ddl` (+ partner AS2 `20260806_partner_as2.ddl`)
- Contract: [`PARTNER_API.md`](./PARTNER_API.md), [`PARTNER_AS2.md`](./PARTNER_AS2.md), [`contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml)
