# Cloud Credentials Checklist

## Required GSM secrets (backend / payments)

- `GLOBAL_PAY_USERNAME` / `GLOBAL_PAY_PASSWORD` / `GLOBAL_PAY_WEBHOOK_SECRET` (or `pegasusx-<tenant>-global-pay-*`)
- `JWT_SECRET` (`pegasusx-<tenant>-jwt-secret`)
- **Server Maps key** `pegasusx-<tenant>-google-maps-api-key` → env `GOOGLE_MAPS_API_KEY`
  - APIs: Geocoding, Places, **Routes** (`routes.googleapis.com`)
  - Restrict to GKE NAT / backend egress IPs
  - Used for: `/v1/platform/geocode/*` and route geometry (`ROUTING_PROVIDER=auto|google`)

## Maps Platform — key model (do not share one key everywhere)

| Key | Consumers | Restrictions |
|-----|-----------|----------------|
| Server (`…-google-maps-api-key`) | backend-go | IP / GKE NAT; Geocoding + Places + Routes |
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
- Issue keys via authenticated `POST /v1/admin/partner-keys`; never commit plaintext `pxk_` / `whsec_` secrets
- Apply DDL migration `apps/backend-go/schema/migrations/20260805_partner_integration_layer.ddl`
- Contract: [`PARTNER_API.md`](./PARTNER_API.md), [`contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml)
