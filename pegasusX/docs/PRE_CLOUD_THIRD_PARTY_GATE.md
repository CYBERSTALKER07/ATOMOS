# Pre-cloud third-party gate

> **Purpose:** What must be **wired and proven end-to-end on local SSMR / staging fakes** before connecting **real** third-party APIs (payment PSPs, OFD / my.soliq, Firebase production, Maps keys).  
> Code can be green while credentials are still fake — this doc separates **software readiness** from **live credential cutover**.

**Related:**  
[`WIRE_READY_STAGING_RUNBOOK.md`](./WIRE_READY_STAGING_RUNBOOK.md) ·  
[`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md) ·  
[`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md) ·  
[`P0_LAUNCH_CHECKLIST.md`](./P0_LAUNCH_CHECKLIST.md) ·  
[`adr/009-fiscal-hard-gate.md`](./adr/009-fiscal-hard-gate.md) ·  
[`FISCAL_HARD_GATE_STATE_TABLE.md`](./FISCAL_HARD_GATE_STATE_TABLE.md)

Last verified against codebase: **2026-07-20**.

---

## 1. Two layers (do not conflate)

| Layer | Meaning | Gate |
|-------|---------|------|
| **A — Software hard-gate** | Product path works with **fakes** (FAKE OFD, dev webhook secrets, emulator Spanner) | `make wire-ready` + `make test-ssmr-fiscal` + unit money path tests |
| **B — Live credential cutover** | Same paths against **real** Global Pay / Payme / Click / OFD / Firebase / Maps | [`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md) LC-01…LC-06 |

**Do not** set `PEGASUSX_ENV=production` or real merchant keys until **A is green** and **B sign-off** is complete.

---

## 2. Money path — must be green before real PSP / OFD

These are **pay-at-delivery + fiscal hard-gate** requirements (ADR-001 + ADR-009). Proven with **fakes only**.

### 2.1 Order state machine

| Rule | Evidence in code | Local proof |
|------|------------------|-------------|
| No soft `ARRIVED` / `AWAITING_PAYMENT` / `PENDING_CASH_COLLECTION` → `COMPLETED` | `order/state_machine.go` | `go test ./order -run Status\|Fiscal\|Collect` |
| Capture → `FISCALIZING` | `CollectCash`, `SettleExternalPayment`, `CompleteOrder` | unit + SSMR |
| `COMPLETED` only after fiscal `SUCCESS` or audited force | `ApplyFiscalWorkerResult`, `ForceCompleteOrder` | `make test-ssmr-fiscal` |
| Credit leave-behind does **not** fiscalize | `DELIVERED_ON_CREDIT` → fiscal only on settlement | unit SM + service tests |

### 2.2 Async OFD (never client-side)

| Piece | Implementation | Fake / real switch |
|-------|----------------|--------------------|
| Attempt table | `OrderFiscalReceipts` + migration `20260720_order_fiscal_receipts.ddl` | Spanner emulator |
| Outbox | `FISCAL_RECEIPT_REQUESTED` same txn as capture | Kafka outbox relay |
| Worker | `order/consumer.go` → `ApplyFiscalWorkerResult` | **API + worker pods** both required |
| Provider | `order/fiscal_provider.go` | `FISCAL_PROVIDER=FAKE` (default) → `MY_SOLIQ` only with sandbox/prod base URL + API key + TIN |
| Fail hooks (FAKE) | order/retailer id contains `fiscal-fail`, or `amount_minor=13` | SSMR fail/retry path |
| Force | `POST /v1/order/{id}/force-complete` ADMIN + WAREHOUSE_ADMIN + `reason_code` | unit + fiscal e2e |
| Shift freeze | `GET /v1/driver/open-fiscal`; return-complete **409** `open_fiscal_block` | `PX_E2E_FISCAL_SHIFT_FREEZE_OK` |
| Cash variance | `amount_received_minor` + `CASH_SHORTFALL` / `CASH_OVERAGE` outbox + dispatcher fanout | `PX_E2E_FISCAL_SHORTFALL_OK` |

### 2.3 Payment webhooks (before real PSP)

| Gateway | Code path | Pre-cloud proof |
|---------|-----------|-----------------|
| Global Pay | `payment/global_pay_executor.go`, `POST /v1/webhooks/global-pay` | Fixture / SSMR payment smoke; **not** live until LC-02 |
| Payme / Click | webhook handlers + idempotency | `go test ./payment -run Webhook`; sandbox optional LC-03 |
| Card clear → fiscal | `PAYMENT_CLEARED` → order consumer `SettleExternalPayment` → `FISCALIZING` | unit `SettleExternalPayment` + integration |
| Late webhooks | Terminal / `FISCALIZING` / `FISCAL_FAILED` → no-op | unit |

**Integer Tiyin only** — no float money on fiscal or collect-cash.

### 2.4 SSMR markers (money)

```bash
cd pegasusX
make test-ssmr-lifecycle   # spine + PX_E2E_FISCAL_CASH_OK
make test-ssmr-fiscal      # full fiscal suite → __SSMR_FISCAL_OK__
```

| Marker | Meaning |
|--------|---------|
| `PX_E2E_FISCAL_CASH_OK` | Cash → worker SUCCESS → `COMPLETED` |
| `PX_E2E_FISCAL_FAIL_RETRY_OK` | Fail hook + retry accepted |
| `PX_E2E_FISCAL_FORCE_OK` | Force-complete after fail |
| `PX_E2E_FISCAL_SHORTFALL_OK` | Received &lt; expected |
| `PX_E2E_FISCAL_SHIFT_FREEZE_OK` | Return-complete blocked on open fiscal |
| `PX_E2E_FISCAL_ALL_OK` | Umbrella |

`make wire-ready` runs full ecosystem SSMR; **also** run `test-ssmr-fiscal` before any **money** third-party cutover (OFD or card-at-delivery).

---

## 3. Non-money third parties — software vs live

| Integration | Pre-cloud (fake / skip) | Live cutover |
|-------------|-------------------------|--------------|
| Spanner / Redis / Kafka | Docker SSMR | GCP Spanner + Memorystore + managed Kafka |
| Firebase OTP / FCM | Auth bypass / emulators where allowed | Production project + per-app plist/json (LC-04) |
| Google Maps geocode | Optional; topology can use fixed lat/lng in SSMR | `GOOGLE_MAPS_API_KEY` (LC-05) |
| OSRM | Local/docker sidecar | K8s `osrm` sidecar |
| Optimizer | `optimizer-core` container | Same image in GKE |
| Global Pay / Payme / Click | Dev secrets + fixture webhooks | Staging perform + webhook (LC-02/03) |
| OFD / my.soliq | `FISCAL_PROVIDER=FAKE` | `FISCAL_PROVIDER=MY_SOLIQ` + `FISCAL_MY_SOLIQ_*` sandbox then prod |

---

## 4. Deploy topology (required for async fiscal)

Fiscal worker is **not** a separate microservice; it is the **order Kafka consumer** on the **worker** process:

- Deploy **both** `backend-go` (API) and `backend-go-worker` (outbox + consumers including fiscal).
- Worker down ⇒ orders stuck in `FISCALIZING` / cash bag freeze — treat as P0 ops incident.

K8s: `infra/k8s/overlays/staging` or `pilot` — see wire-ready runbook.

---

## 5. Env matrix (software → sandbox → production)

| Env | `FISCAL_PROVIDER` | Payment | Notes |
|-----|-------------------|---------|-------|
| Local SSMR | `FAKE` | dev webhook secrets | Default |
| Staging cloud | `FAKE` until OFD sandbox ready | `GLOBAL_PAY_ENV=staging` | Prove spine first |
| Staging OFD sandbox | `MY_SOLIQ` + sandbox URL/key/TIN | staging PSPs | Only after fiscal SSMR green |
| Production | `MY_SOLIQ` + prod credentials | production PSPs | After LC sign-off |

Misconfigured `MY_SOLIQ` **hard-fails** CreateReceipt (does not invent SUCCESS).

---

## 6. Exit checklist before real third-party keys

### Software (Boss / eng)

- [ ] `make wire-ready` → `wire-ready-ok`
- [ ] `make test-ssmr-fiscal` → `__SSMR_FISCAL_OK__`
- [ ] `make px12-preflight` → `px12-preflight-ok`
- [ ] Fiscal migration applied on target Spanner (`OrderFiscalReceipts` + order fiscal columns)
- [ ] Worker deployment proven (stuck-FISCALIZING drill: kill worker → freeze; restore → complete)
- [ ] No path to `COMPLETED` without SUCCESS or audited FORCE (unit + SSMR)

### Live credentials (Finance / Platform) — separate track

- [ ] Global Pay staging perform + webhook (LC-02)
- [ ] Payme/Click sandbox if in scope (LC-03)
- [ ] OFD sandbox receipt create with real TIN (after software green)
- [ ] Firebase + Maps per [`PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md`](./PRODUCTION_CREDENTIAL_VALIDATION_RUNBOOK.md)
- [ ] Secrets in GSM / External Secrets — never commit

---

## 7. Explicit non-goals for this gate

- Production OFD credentials in local Docker
- Multi-supplier multi-leg fiscal (schema ready; single-supplier today)
- Provisional fiscal on unpaid credit
- Full `make wire-ready` substitute for OFD sandbox — SSMR uses FAKE only

---

## 8. Gap log (pre-cloud money path)

| ID | Item | Status (2026-07-20) |
|----|------|---------------------|
| PC-01 | Fiscal SM + worker + FAKE OFD | **Closed** — SSMR fiscal green |
| PC-02 | Cash shortfall event fanout to WS | **Closed** — dispatcher routes `CASH_*` |
| PC-03 | Shared `EventType` union fiscal + cash variance | **Closed** — `packages/types` |
| PC-04 | `test-ssmr-fiscal` in ops muscle memory | **Closed** — this doc + Makefile target |
| PC-05 | Live OFD / PSP credentials | **Open** — boss LC track, not a code block |
| PC-06 | Double-entry ledger row for shortfall | **Deferred** — outbox + support events; ledger entry optional follow-up |
| PC-07 | Card-at-delivery dedicated SSMR marker | **Open (P2)** — unit path exists; optional `PX_E2E_FISCAL_CARD_OK` later |
| PC-08 | Reconciliation soft COMPLETE without fiscal | **Closed** — force audit path + UpdateStatus gate |

Last full-spine hunt: **2026-07-20** (gap-hunter v2).
