# PegasusX Migration & Staging Status

> **Re-aligned 2026-08-20** against live codebase. Prefer [`docs/DOCS_SOURCE_OF_TRUTH.md`](../docs/DOCS_SOURCE_OF_TRUTH.md) + [`docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md) when this snapshot conflicts.

*Last Synchronized: 2026-08-20 (Full 29-Package Route Alignment + Spanner DDL 3,648 lines + Outbox DeadLetters DLQ + Gate 5 Multi-Tenancy + Theatre #13 FX + WMS Gate 4 + Partner API & AS2 + SSMR Smokecheck Green)*

---

## 1. Code Completeness & Verified Backend State

1. **Gate 5 / §8.10 Multi-Tenancy**:
   - **Phase 1 (Wired)**: `TenantContext`, `RequireTenant`, `PreferTenantSupplierID`, outbox `SupplierId` partitioning, and per-tenant rate limits.
   - **Phase 2 (Wired)**: `ParentOrders` multi-supplier checkout splitting (`schema/spanner.ddl:221-239`).
   - **Phase 3 (Wired)**: `GlobalProducts` marketplace master catalog indexing and match queue deduplication.
   - Documentation: [`MULTI_TENANCY_GATE5_PHASE1.md`](../docs/MULTI_TENANCY_GATE5_PHASE1.md), [`PHASE2`](../docs/MULTI_TENANCY_GATE5_PHASE2.md), [`PHASE3`](../docs/MULTI_TENANCY_GATE5_PHASE3.md).

2. **Theatre #13 Multi-Currency FX Rates**:
   - Live FX conversions via `FxRates` and `fxrates.ConvertMinor` (fail-closed).
   - Order currency picker enabled via `ORDER_CURRENCY_PICKER_ENABLED` and allowlist (`GET /v1/order/currencies`).
   - Billing GMV conversion to operating currency; settlement authority `operating_currency_total_minor`.
   - Admin & Supplier FX rate management (`GET/PUT /v1/admin/fx-rates`, `GET /v1/supplier/fx-rates`).
   - Documentation: [`docs/FX_RATES.md`](../docs/FX_RATES.md).

3. **Theatre #8 Seasonality & Demand Sensing**:
   - `SeasonalTemplateOverrides.Multiplier` persisted (clamped [0.5, 2.5]) eliminating read-time hardcoding.
   - Shared `seasonalcore` built-ins for planning and replenishment.
   - YoY / monthly estimate drafts via `POST /v1/supplier/planning/seasonal-estimate`.
   - Point-of-sale demand signals ingested via `demandroutes` (`/v1/demand/*`).

4. **WMS & Inventory Hardening (WMS Gate 4)**:
   - Stock lot tracking & FEFO picking algorithms in `stocklots/`.
   - Pick wave creation, execution, and pick task assignment.
   - Cycle counting with approve-and-apply inventory adjustments and accuracy metrics.
   - Cold-chain temperature logging and breach alerts.

5. **B2B Partner API & AS2 Integration (Gate 3)**:
   - Machine API keys (`pxk_*`) and OAuth2 `client_credentials` (`POST /partner/v1/oauth/token`).
   - RFC 7807 problem details (`application/problem+json`) on error responses.
   - AS2 transport with synchronous MDN document exchange (`POST /partner/v1/as2`).
   - EDI-lite DESADV / ORDERS support with GS1 GLN/SSCC/ZPL labeling.
   - Configurable 1C Chart of Accounts mapping (`PartnerCoaMaps`).

6. **Retail OS (Phases 0–7)**:
   - Capability Packs 0–6 implemented across backend and 3 client platforms (Desktop, Android, iOS).
   - Store stock management, POS sales, holds, and shift management.
   - Floor assist tickets and shadow auto-order replenishment.

7. **Transactional Outbox Engine & Dead-Letter Queue**:
   - `OutboxEvents` table (`spanner.ddl:679-697`) with atomic `TxnBuffer` writes inside Spanner ReadWrite transactions.
   - `OutboxDeadLetters` table (`spanner.ddl:698-709`) capturing poison messages with retry counts and error payloads.
   - Background polling relay with at-least-once delivery to Apache Kafka.

---

## 2. SSMR Cloud Staging Reality (`pegasus-503013`)

| Infrastructure Component | Staging Reality |
|---|---|
| **GKE Cluster** | `pegasusx-ssmr-gke` (namespace `pegasusx-ssmr`) active and running. |
| **Spanner Migrations** | Full schema (`schema/spanner.ddl`, 3,648 lines) applied and active. |
| **Backend Runtime** | API and Worker tiers deployed with `REQUIRE_INFRA_ADAPTERS=true`. |
| **Ingress & TLS** | Ingress LB `api-ssmr.pegasusx.app` active with Google Trust Services ManagedCertificate. |
| **Health Probes** | `/healthz`, `/ready`, `/v1/health` returning HTTP 200 OK. |
| **SSMR E2E Smokecheck** | `cmd/ssmr-smokecheck/e2e_check.go` passes **80+ multi-role verification steps** and emits 115+ assertion markers (`PX_E2E_*`). |

---

## 3. Layer B External Credentials & Operations Checklist

The remaining operational steps prior to production traffic are restricted to third-party merchant onboarding and owner secret provisioning:
1. **GlobalPay Acquiring Credentials**: Provision live merchant password in Secret Manager to transition card payments from cash fallback to live card capture.
2. **Firebase Auth & SMS Delivery**: Configure production SMS quota in Firebase console and verify SHA-1 release fingerprints.
3. **Fiscal OFD Signing Secrets**: Mount E-IMZO PKCS#12 certificate volume for live Soliq receipt registration.
