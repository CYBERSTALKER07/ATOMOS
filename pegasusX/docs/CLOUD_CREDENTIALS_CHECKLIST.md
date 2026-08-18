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

### LB-3 fiscal / SMS shells (names only — do not put values in git)

ESO `spec.data` is atomic. **Do not** add these keys to the live ExternalSecret until GSM has ≥1 enabled version (TF apply of the shell + `phase0_sync` placeholder). PKCS#12 **bytes** are a volume (`FISCAL_MY_SOLIQ_PKCS12_FILE`), not a GSM string.

| K8s `secretKey` (when wired) | GSM id | Env | Real vs stub |
|------------------------------|--------|-----|----------------|
| `fiscal-my-soliq-base-url` | `…-fiscal-my-soliq-base-url` | `FISCAL_MY_SOLIQ_BASE_URL` | Real for MY_SOLIQ; stub until EDS |
| `fiscal-my-soliq-api-key` | `…-fiscal-my-soliq-api-key` | `FISCAL_MY_SOLIQ_API_KEY` | Real for MY_SOLIQ |
| `fiscal-my-soliq-tin` | `…-fiscal-my-soliq-tin` | `FISCAL_MY_SOLIQ_TIN` | Real for MY_SOLIQ |
| `fiscal-my-soliq-signer` | `…-fiscal-my-soliq-signer` | `FISCAL_MY_SOLIQ_SIGNER` | `pkcs12` in sandbox/prod |
| `fiscal-my-soliq-pkcs12-password` | `…-fiscal-my-soliq-pkcs12-password` | `FISCAL_MY_SOLIQ_PKCS12_PASSWORD` | Real with E-IMZO |
| `playmobile-login` | `…-playmobile-login` | `PLAYMOBILE_LOGIN` | Real for UZ SMS |
| `playmobile-password` | `…-playmobile-password` | `PLAYMOBILE_PASSWORD` | Real for UZ SMS |

TF shells: `infra/terraform/fiscal_sms_secrets.tf`. Phase0: `scripts/phase0_sync_gsm_secrets.sh`. Do not flip `FISCAL_PROVIDER` from a stub.

- Terraform shells all 12 payment names (`infra/terraform/main.tf`) plus the 7 fiscal/SMS shells. Versions: pass TF vars or run `scripts/phase0_sync_gsm_secrets.sh` (`ENSURE_ES_STUBS=1` writes `unused-rail-placeholder` when env empty).
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
