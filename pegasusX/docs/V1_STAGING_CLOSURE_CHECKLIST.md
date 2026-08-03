# pegasusX v1 Staging Closure Checklist

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Boss-operated checklist to move from **PROD_CANDIDATE (SSMR)** to **staging-signed** per the v1 closure plan.

**Prerequisites (engineering):**

```bash
cd pegasusX
make px12-preflight    # px12-preflight-ok
make test-ssmr-infra     # __SSMR_OK__ + required PX_E2E_* markers
```

**References:** [`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md), [`CLOUD_CUTOVER_RUNBOOK.md`](./CLOUD_CUTOVER_RUNBOOK.md), [`REAL_WORLD_CASE_MATRIX.md`](./REAL_WORLD_CASE_MATRIX.md)

---

## LC-01 — Terraform + GSM + GKE (Platform)

| Step | Action | Done [ ] |
|------|--------|----------|
| 1 | `terraform apply` in `infra/terraform/` with `monthly_budget_usd = 1500` | |
| 2 | Sync GSM secrets per [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md) | |
| 3 | Deploy `backend-go` (2+ replicas), `ai-worker`, OSRM (`infra/k8s/osrm/`) | |
| 4 | Set `REQUIRE_INFRA_ADAPTERS=true`, `PEGASUSX_ENV=staging` | |
| 5 | Verify boot: Redis idempotency required (no in-memory under strict mode) | |

---

## LC-02 — Global Pay.UZ staging (Finance)

| Step | Action | Done [ ] |
|------|--------|----------|
| 1 | `GLOBAL_PAY_ENV=staging` on staging cluster | |
| 2 | Retailer card checkout on staging `PUBLIC_BASE_URL` | |
| 3 | Webhook replay → payment session settled | |
| 4 | Circuit breaker healthy (`payment/global_pay_executor.go`) | |

---

## LC-03 — Payme/Click sandbox (optional v1)

| Step | Action | Done [ ] |
|------|--------|----------|
| 1 | Sandbox keys in GSM OR document “GlobalPay primary v1” in sign-off | |

---

## LC-04 — Firebase OTP per role (Client + Platform)

| App | `google-services.json` / plist | Real device login | Done [ ] |
|-----|-------------------------------|-------------------|----------|
| driver-app-android / iOS | | | |
| factory-app-* | | | |
| payload-app-* / terminal | | | |
| retailer-app-android / iOS | | | |

Optional markers when test tokens set: `PX_E2E_*_FIREBASE_OTP_OK`

---

## LC-05 — Maps + OSRM (Platform)

| Step | Action | Done [ ] |
|------|--------|----------|
| 1 | `GOOGLE_MAPS_API_KEY` in GSM | |
| 2 | `ROUTING_OSRM_URL` reachable from backend pods | |
| 3 | `GET /v1/fleet/route/{routeID}/geometry` returns polyline | |
| 4 | Dispatch preview uses `OPTIMIZER_BASE_URL` → ai-worker | |

---

## LC-06 — Staging sign-off (Release)

```bash
PUBLIC_BASE_URL=https://api.staging.<domain> bash scripts/validate_staging_credentials.sh
# expect: staging-credentials-ok
```

Complete sign-off table in [`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md) § Sign-off table.

**Do not set `PEGASUSX_ENV=production` until LC-06 complete.**

---

## Phase 4 — Staging proof (after LC-01)

| Step | Command | Evidence | Done [ ] |
|------|---------|----------|----------|
| Cloud smoke | `PUBLIC_BASE_URL=... make cloud-smoke-ssmr` | log archive | |
| Load cert | `PUBLIC_BASE_URL=... make load-cert-cloud` | `artifacts/load/*/LOAD_TEST_REPORT.md` | |
| Multi-pod WS | 2+ backend replicas; dispatch WS on portal pod B | [`P1_PILOT_CHECKLIST.md`](./P1_PILOT_CHECKLIST.md) | |
| Barcode seed | Catalog EAN on pilot SKUs | [`BARCODE_GO_LIVE_CHECKLIST.md`](./BARCODE_GO_LIVE_CHECKLIST.md) | |

---

## Phase 5 — PX-12 boss sign-off

Complete [`docs/qa/PX12_ROLE_ROW_QA.md`](./qa/PX12_ROLE_ROW_QA.md) Phases A–C for environment `staging`.

Pilot week: `bash scripts/p1_pilot_weekly.sh`

---

## War-story manual QA (Phase C)

Run scripts in [`docs/qa/PX12_MANUAL_QA_RUNBOOK.md`](./qa/PX12_MANUAL_QA_RUNBOOK.md#phase-c--war-story-scripts) (~15 min each).

| ID | Story | Boss [ ] |
|----|-------|----------|
| WS-01 | Shop-closed full loop | |
| WS-02 | Concurrent stock reject | |
| WS-03 | Seal → driver gate → delivery | |
| WS-04 | Returns inbound EAN valid/invalid | |
| WS-05 | Replenish TRUCK vs INTERNAL | |
