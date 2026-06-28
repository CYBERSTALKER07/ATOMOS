# Production Credential Validation Runbook

> **Purpose:** pegasusX code and SSMR gates can be green without live cloud credentials. This runbook proves each external integration works against **staging** (then production) before `PEGASUSX_ENV=production` cutover.
>
> **Inventory reference:** [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md)  
> **Gap ledger:** [`DEPLOYMENT_READINESS_GAP_LEDGER.md`](./DEPLOYMENT_READINESS_GAP_LEDGER.md)

Last updated: 2026-06-28.

---

## Prerequisites

1. Terraform applied (`infra/terraform/`) with `monthly_budget_usd = 1500`.
2. GKE workloads deployed: `backend-go`, `ai-worker`, `osrm` sidecar.
3. External Secrets synced from GSM (see checklist § Secret Manager keys).
4. Staging `PUBLIC_BASE_URL` reachable (Ingress or Cloud Run).

Automated pre-check (non-secret env names only):

```bash
cd pegasusX
PUBLIC_BASE_URL=https://staging.example.com bash scripts/validate_staging_credentials.sh
```

Expected terminal line: `staging-credentials-ok`.

---

## Boss-action inventory (must exist before validation)

| Item | Owner | GSM / artifact | Blocks |
|------|-------|----------------|--------|
| Global Pay.UZ prod/staging API | Finance | `GLOBAL_PAY_*` secrets | Checkout + webhook |
| Payme merchant webhook secret | Finance | `PAYME_WEBHOOK_SECRET` | Payme webhook |
| Click merchant webhook secret | Finance | `CLICK_WEBHOOK_SECRET` | Click webhook |
| Firebase Admin credentials | Platform | `FIREBASE_CREDENTIALS_PATH` | OTP verify + FCM |
| Firebase client configs | Client | `google-services.json` / `GoogleService-Info.plist` per app | Phone OTP on device |
| Google Maps API key | Platform | `GOOGLE_MAPS_API_KEY` | Geocode + Android maps |
| Maps SDK keys (per Android app) | Client | Play Console / GCP restrictions | Driver + retailer maps |
| JWT + internal API key | Platform | `JWT_SECRET`, `INTERNAL_API_KEY` | Auth + optimizer |
| Kafka bootstrap | Platform | `KAFKA_BROKERS` | Outbox relay |
| Store signing (Apple / Google / Tauri) | Client | Certs outside repo | App distribution |

---

## Per-service validation steps

### 1. Core infra (Spanner, Redis, Kafka)

**Local proof (no cloud):**

```bash
cd pegasusX
make test-ssmr-infra
```

Expect `__SSMR_OK__` and required `PX_E2E_*` markers in log.

**Staging proof:**

```bash
PUBLIC_BASE_URL=https://staging.example.com make cloud-smoke-ssmr
```

| Check | Pass criteria |
|-------|---------------|
| Health | `GET /v1/health` → 200 |
| Client policy | `GET /v1/client-policy` → 200 with version fields |
| Spanner | Order create in SSMR or staging smoke log |

---

### 2. Global Pay.UZ

**Code path:** `payment/global_pay_executor.go`, `POST /v1/webhooks/global-pay`.

**Staging validation:**

1. Set `GLOBAL_PAY_ENV=staging` (not `production` until final cutover).
2. Create checkout session via retailer checkout API.
3. Complete perform call (or use staging backoffice).
4. Replay webhook with valid HMAC (mirror `cmd/ssmr-smokecheck/e2e_payment.go`).

```bash
# After checkout session exists:
curl -sS -X POST "$PUBLIC_BASE_URL/v1/webhooks/global-pay" \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: <hmac>" \
  -d @fixtures/global_pay_webhook.json
```

| Pass criteria | Evidence |
|---------------|----------|
| Session → `PAID` | Payment row + order status transition |
| Idempotent replay | Second POST returns same outcome, no double charge |

---

### 3. Payme / Click (sandbox)

**Handlers:** `payment/payme_webhook.go`, `payment/click_webhook.go`.

**Fixture tests (CI, no live keys):**

```bash
cd pegasusX/apps/backend-go
go test ./payment/... -run 'TestPaymeWebhook|TestClickWebhook' -count=1
```

**Staging validation (when sandbox credentials available):**

```bash
# Payme — Basic auth per PAYME_WEBHOOK_SECRET
curl -sS -X POST "$PUBLIC_BASE_URL/v1/webhooks/payme" \
  -H "Content-Type: application/json" \
  -H "Authorization: Basic <base64>" \
  -d @fixtures/payme_perform.json

# Click — MD5 signature in body
curl -sS -X POST "$PUBLIC_BASE_URL/v1/webhooks/click" \
  -H "Content-Type: application/json" \
  -d @fixtures/click_prepare.json
```

| Pass criteria | Evidence |
|---------------|----------|
| Signature verify | 401 without valid auth |
| State transition | Session status matches gateway state |
| Replay | Same transaction id → idempotent response |

Optional SSMR (env-gated):

- `PAYME_TEST_WEBHOOK_BODY` + `PAYME_WEBHOOK_SECRET` → `PX_E2E_PAYME_WEBHOOK_OK`
- `CLICK_TEST_WEBHOOK_BODY` + `CLICK_WEBHOOK_SECRET` → `PX_E2E_CLICK_WEBHOOK_OK`

---

### 4. Firebase (phone OTP)

| Role row | Client artifact | Validation |
|----------|-----------------|------------|
| Driver | Android + iOS | Send OTP on physical device; login posts `id_token` |
| Factory | portal + Android + iOS | Same |
| Warehouse | portal + Android + iOS | Same |
| Payload | terminal + Android + iOS | Same |
| Retailer iOS | `GoogleService-Info.plist` | Custom token exchange login |

**Backend:** `FIREBASE_AUTH_ENABLED=true`, credentials mounted.

| Pass criteria | Evidence |
|---------------|----------|
| Token verify | `POST /v1/auth/<role>/login` with real `id_token` → 200 + JWT |
| Emulator not used in staging | No `FIREBASE_AUTH_EMULATOR_HOST` in prod configmap |

---

### 5. Maps, geocoding, OSRM

| Surface | Config | Validation |
|---------|--------|------------|
| Geocode | `GOOGLE_MAPS_API_KEY` | `PUT /v1/supplier/topology` with new address → lat/lng populated |
| OSRM geometry | `ROUTING_OSRM_URL` | Dispatch preview returns route polyline |
| Android maps | Per-app Maps SDK key | Map renders on driver/retailer device |

```bash
curl -sS "$PUBLIC_BASE_URL/v1/health"  # backend up
# Topology edit via supplier session — see SSMR PX_E2E_TOPOLOGY_EDIT_OK
```

---

### 6. Optimizer (ai-worker)

**Runtime:** `OPTIMIZER_BASE_URL=http://ai-worker:8081` (not optimizer-core).

```bash
kubectl -n pegasusx get pods -l app=ai-worker
curl -sS "http://ai-worker.pegasusx.svc.cluster.local:8081/health"
```

Dispatch execute should log optimizer attempt or binpack fallback (never hard-fail).

---

## Sign-off table

| Integration | Env | Owner | Date | Evidence (link / log) | Signed |
|-------------|-----|-------|------|-------------------------|--------|
| SSMR / core infra | local | | | `make test-ssmr-infra` log | |
| Staging health | staging | | | `cloud-smoke-ssmr` | |
| Global Pay | staging | | | webhook + session id | |
| Payme | staging | | | fixture or sandbox POST | |
| Click | staging | | | fixture or sandbox POST | |
| Firebase OTP | staging | | | device login per role | |
| Maps / OSRM | staging | | | topology + dispatch | |
| ai-worker | staging | | | health + dispatch log | |
| Production cutover | production | Release owner | | All rows above repeated | |

---

## Exit criteria

Launch to **production** only when:

1. Every sign-off row is complete for **staging**.
2. `make p0-preflight` and `make validate-launch-readiness` pass against staging manifests.
3. `PEGASUSX_ENV=production` config validated (`bootstrap/config_validate.go` rules).
4. Production secrets rotated (not reused from staging).
5. Release owner records evidence bundle in release notes.

---

## Rollback

If any live credential validation fails post-cutover:

1. Revert `GLOBAL_PAY_ENV` to staging or disable checkout.
2. Preserve webhook logs and payment session ids.
3. Follow [`LAUNCH_READINESS_RUNBOOK.md`](./LAUNCH_READINESS_RUNBOOK.md) rollback section.
