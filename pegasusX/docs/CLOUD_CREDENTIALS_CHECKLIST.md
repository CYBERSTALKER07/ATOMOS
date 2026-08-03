# pegasusX Cloud Credentials Checklist (Boss Handoff)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



> Budget envelope: **$1,500/mo** — see [`CLOUD_BUDGET_MODEL.md`](./CLOUD_BUDGET_MODEL.md).  
> Enterprise readiness plan: phased verification in [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md).

## Not required for local Docker / SSMR green

- Spanner, Redis, Kafka cloud endpoints
- Google Maps keys (maps show placeholder without keys)
- Live Global Pay.UZ credentials (dev webhook secrets in `.env.ssmr.example`)
- Terraform apply

---

## Required for staging / production (GCP core)

| Service | Purpose | Tier / budget note |
|---------|---------|-------------------|
| **Cloud Spanner** | System of record | Regional 100 PU (~$650–900/mo) |
| **GKE Autopilot** | `backend-go` + `ai-worker` | 2+1 pods (~$180–280/mo) |
| **Memorystore Redis** | Cache, idempotency, WS relay | Basic 1 GB (~$35–55/mo) |
| **Managed Kafka** | Outbox → notification dispatcher | Basic cluster (~$120–200/mo) |
| **Cloud Run** | supplier / warehouse / factory portals | min-instances=0 |
| **GCS** | Catalog images, CSV import, app update manifests | Standard |
| **Secret Manager** | JWT, webhooks, Global Pay API creds | Via External Secrets |
| **Cloud Monitoring** | Budget alerts 80%/100% | In envelope |

**Terraform:** `infra/terraform/` — set `monthly_budget_usd = 1500`, `billing_account_id`.

**K8s manifests:** `infra/k8s/backend-go/` (note: app reads **`HTTP_PORT`**, not `PORT`), `infra/k8s/osrm/` (route geometry sidecar).

---

## Complete production API & service inventory

### Google Cloud Platform

| API / service | Used by | Provision |
|---------------|---------|-----------|
| Spanner API | `backend-go` | Instance + DB + workload identity |
| Memorystore Redis | cache, idempotency, WS Pub/Sub | VPC-attached |
| Kafka (Confluent or compatible) | outbox relay, freeze locks, consumers | `KAFKA_BROKERS`, topics |
| Cloud Storage | catalog upload tickets, import CSV, Tauri updates | `GCS_BUCKET_NAME` |
| Secret Manager | all production secrets | External Secrets operator |
| Artifact Registry | container images | CI push |
| Cloud Monitoring / Logging | SLO + budget alerts | `infra/terraform/observability.tf` |

### Firebase (Auth + FCM)

| Product | Clients | Env |
|---------|---------|-----|
| Firebase Auth (phone OTP) | factory/warehouse portals, driver, factory/warehouse/payload Android+iOS, payload-terminal | `NEXT_PUBLIC_FIREBASE_*`, `google-services.json` |
| Firebase Admin SDK | `backend-go` token verify + FCM | `FIREBASE_CREDENTIALS_PATH`, `FIREBASE_AUTH_ENABLED=true` |
| FCM | driver Android, payload Android | Free at launch scale |

**Supplier** uses JWT/cookie OTP — no Firebase client.

**Retailer iOS:** Firebase SPM wired (`FirebaseAuth` + `FirebaseCore`) for custom-token exchange.

### Maps & routing

| Need | Apps | API to enable | Cost |
|------|------|---------------|------|
| Fleet/dispatch basemap | supplier/warehouse portals, supplier/warehouse Android | **MapLibre + Carto** (`basemaps.cartocdn.com`) | Free |
| Maps on iOS | supplier, warehouse, retailer, driver iOS | **Apple MapKit** | Free (Apple Dev $99/yr) |
| Driver + retailer Android maps | driver-app-android, retailer-app-android | **Maps SDK for Android** | GCP $200/mo Maps credit |
| Geocoding API | topology address → lat/lng | Backend `GOOGLE_MAPS_API_KEY` via GSM | Same credit |
| Places API | retailer location autocomplete | Backend `GOOGLE_MAPS_API_KEY` via GSM | Same credit |
| Route polylines | backend-go | **Self-hosted OSRM** (`ROUTING_OSRM_URL`) | ~$20–40 sidecar |
| VRP optimization | Go ai-worker (Clarke-Wright) | `OPTIMIZER_BASE_URL` → ai-worker:8081 + `INTERNAL_API_KEY` | GKE pod CPU (~1–2 vCPU) |

**Do not enable:** Maps JavaScript API (portals use MapLibre), Directions API (OSRM handles geometry).

### Barcode scanning (no cloud API)

- Android: ML Kit via `packages/mobile-android-barcode-scanner`
- iOS: `packages/mobile-ios-barcode` (AVFoundation)
- payload-terminal: `expo-camera`

### Payments — Global Pay.UZ (primary)

Boss provides API credentials. Wire into Secret Manager + backend env:

| Secret / env | Purpose |
|--------------|---------|
| `GLOBAL_PAY_ENV` | `production` or `staging` (disables `/sim/globalpay` dev simulator) |
| `GLOBAL_PAY_SERVICE_ID` | Merchant service ID |
| `GLOBAL_PAY_USERNAME` | Checkout API auth |
| `GLOBAL_PAY_PASSWORD` | Checkout API auth |
| `GLOBAL_PAY_WEBHOOK_SECRET` | `POST /v1/webhooks/global-pay` HMAC |

**Outbound hosts:** `checkout-api.globalpay.uz`, `backoffice-api.globalpay.uz` (staging variants for pre-prod).

**Optional gateways (webhook-only today):** `ADYEN_WEBHOOK_SECRET`, `STRIPE_WEBHOOK_SECRET`, `PAYME_WEBHOOK_SECRET`, `CLICK_WEBHOOK_SECRET`. Set `AIRWALLEX_DIRECT_EXECUTION_ENABLED=true` only if Airwallex goes live.

### Redis (Memorystore)

- Idempotency keys, cache, WS pub/sub invalidation channel.
- Supplier-scoped keys use hash tags `{sup:<supplierId>}:...` for future Redis Cluster slot safety (see `cache/keys.go`).

### Realtime (self-hosted — no extra SaaS)

| Mechanism | Config |
|-----------|--------|
| WebSocket 7 hubs | `WS_ALLOWED_ORIGINS` for browser portals |
| Kafka → WS dispatcher | `KAFKA_TOPIC_MAIN` |
| Telemetry HTTP | `POST /v1/telemetry/location` (not Kafka) |
| Freeze locks | `KAFKA_TOPIC_FREEZE_LOCKS` → ai-worker consumer |

### Observability

| Option | When |
|--------|------|
| **Cloud Monitoring** (recommended) | Default for $1,500 budget |
| Prometheus `/metrics` | GKE scrape |
| Datadog APM | Optional — `DD_AGENT_HOST` (extra cost) |

### App distribution (non-API)

- Apple Developer Program (5 iOS apps + APNs)
- Google Play Console (6 Android apps)
- **Tauri desktop** (retailer, supplier, warehouse, factory):
  - **Updater signing:** minisign keypair via `tauri signer generate`. Commit **public** key only (`contracts/desktop-updater/dev.pub` for dev; production pubkey injected at release). Private key in GSM `PEGASUSX_TAURI_SIGNING_PRIVATE_KEY` → CI `TAURI_SIGNING_PRIVATE_KEY`.
  - **Apply pubkey before build:** `bash scripts/apply_desktop_updater_pubkey.sh` (or set `TAURI_UPDATER_PUBKEY`).
  - **Windows Authenticode:** EV code-signing cert for `.msi`/`.exe` (optional secret `WINDOWS_CODESIGN_CERT` + `WINDOWS_CODESIGN_PASSWORD` on release workflow). Timestamp server required.
  - **macOS:** Developer ID Application cert + `notarytool` staple for `.dmg` (document in release runbook; secrets `APPLE_SIGNING_IDENTITY`, `APPLE_NOTARIZE_*`).
  - **Updater CDN:** `gs://pegasusx-ssmr-app-updates/{app}-desktop/{target}/{arch}/updater.json` — see `contracts/desktop-updater/README.md`.
- Expo EAS (payload-terminal) if OTA required

---

## Secret Manager keys (minimum)

| GSM secret | Maps to env |
|------------|-------------|
| `PEGASUSX_JWT_SECRET` | `JWT_SECRET` |
| `PEGASUSX_GLOBAL_PAY_WEBHOOK` | `GLOBAL_PAY_WEBHOOK_SECRET` |
| `PEGASUSX_GLOBAL_PAY_SERVICE_ID` | `GLOBAL_PAY_SERVICE_ID` |
| `PEGASUSX_GLOBAL_PAY_USERNAME` | `GLOBAL_PAY_USERNAME` |
| `PEGASUSX_GLOBAL_PAY_PASSWORD` | `GLOBAL_PAY_PASSWORD` |
| `PEGASUSX_INTERNAL_API_KEY` | `INTERNAL_API_KEY` |
| `pegasusx-<tenant>-google-maps-api-key` | `GOOGLE_MAPS_API_KEY` (Geocoding + Places; restrict Android Maps SDK keys per app) |
| `pegasusx-<tenant>-kafka-bootstrap-servers` | `KAFKA_BROKERS` (Confluent Cloud Basic bootstrap) |
| `PEGASUSX_TAURI_SIGNING_PRIVATE_KEY` | `TAURI_SIGNING_PRIVATE_KEY` (desktop updater bundle signing) |
| `pegasusx-<tenant>-tauri-updater-pubkey` | `TAURI_UPDATER_PUBKEY` (build-time pubkey injection) |
| `pegasusx-<tenant>-windows-codesign-pfx` | `WINDOWS_CODESIGN_PFX_B64` (Authenticode) |
| `pegasusx-<tenant>-windows-codesign-password` | `WINDOWS_CODESIGN_PASSWORD` |
| `pegasusx-<tenant>-apple-notarize-*` | `APPLE_NOTARIZE_*` (notarytool) |
| Adyen / Stripe / Payme / Click webhook secrets | as enabled |

---

## Terraform inputs (minimum)

| Variable | Purpose |
|----------|---------|
| `project_id` | GCP project |
| `tenant_slug` | Resource namespace |
| `region` | Primary region (default `asia-south1`) |
| `monthly_budget_usd` | `1500` |
| `billing_account_id` | Enables budget alerts |
| `kafka_bootstrap_servers` | Confluent Cloud bootstrap (sensitive) |
| `google_maps_api_key` | Geocoding + Places server-side key (sensitive) |
| `ai_worker_monitoring_host` | Optional uptime checks |

## Post-provision commands

```bash
cd pegasusX/infra/terraform && terraform init && terraform apply
cd pegasusX && go run ./apps/backend-go/cmd/setup
cd pegasusX && make validate-launch-readiness
cd pegasusX && make test-ssmr-infra
```
