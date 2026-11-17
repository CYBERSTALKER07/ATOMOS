# Task Plan: ATOMOS Enterprise Intelligence Execution

<!--
  WHAT: Working memory on disk for planning and executing /Users/shakhzod/Desktop/ATOMOS_Enterprise_Intelligence_Plan.md.
  REPO: /Users/shakhzod/Desktop/pegasusx
  WORKFLOW: /planning-with-files + /swarm (following AGENTS.md Laws 0.1-0.5 and Invariants 1-10)
  ARCHIVE: Prior Nov 10 planning files backed up in session scratch/nov10_planning_archive/
-->

## Goal
Verify codebase ground truth against `/Users/shakhzod/Desktop/ATOMOS_Enterprise_Intelligence_Plan.md`, align on user scope and owner decisions (`D1-D5` and open questions), and execute the approved phases cleanly in `/Users/shakhzod/Desktop/pegasusx` via `/swarm` with full live-emulator and CI gate verification.

## Current Phase
Phase 4 — Wave 5 (`G-09` Full Ecosystem Verification & Live HTTP/Playwright Manual QA Audit) — Active (`/goal` Enterprise Production Run)

## Phases

### Phase 0: Ground-Truth Audit & Scope Clarification
- [x] Read `ATOMOS_Enterprise_Intelligence_Plan.md` (§0-§13), `AGENTS.md`, and `scripts/ci_repo_hygiene_gate.sh`
- [x] Back up Nov 10 `task_plan.md`, `findings.md`, `progress.md` to session scratch
- [x] Verify actual status of Gate Zero (`T1-T9`), `SetLotsEnabled`, and `substitution.go` against current repo (`HEAD e25490326` + uncommitted changes)
- [x] Record exact verified status in `findings.md` and `progress.md`
- [x] Confirm target scope, §11 owner decisions, swarm model policy, and mirror parity with user (`ask_question`)
- **Status:** complete

### Phase 1: M0.1 Gate Zero Completion (Remaining Open & Partial Items)
- [x] **T8 (Closed & live-verified):** Made `PaymentLedgerEntries` strictly append-only (`spanner.InsertMap`) across `payment/repository_spanner.go`, `creditnote/repository_spanner.go`, and `cmd/seed-supplier-prodsim/main.go` with safe RW pre-reads, fixed `SaveWebhook` `NULL_FILTERED` index emulator query (`#267`), and verified live `:9010` tests (`#269`)
- [x] **T9 (Closed & live-verified):** Migrated `DraftAdjustments` & `DraftCreditNotes` `ProposedAmount NUMERIC` -> `ProposedAmountMinor INT64 NOT NULL DEFAULT (0)` (`spanner.ddl` + `20261116_financial_drafts_minor_units.ddl`, `ReviewedAt` `allow_commit_timestamp=true`) and enforced four-eyes SoD (`OperatorId != ReviewedBy`) across both draft types (`#254`, `#259`, `#269`)
- [x] **T1 (Closed & live-verified):** Registered `EventInventoryLotReleased = "inventory.lot.released"`, synced 3-way contracts (`events.go`, `types.go`, `contracts/events.schema.json`, `packages/types/src/envelope.ts`), audited all 11 release callers, and wired atomic in-txn outbox emission in `order/inventory_release.go` (`#249`, `#260`, `#269`)
- [x] Offline review & `-race` unit/contract/static gate sweep completed cleanly by `gz-verifier` (`#262`)
- [x] **Live Spanner Emulator Verification Complete (`#269`):** With Docker Spanner (`:9010`), Kafka (`:9092`), and Redis (`:6379`) live, `gz-verifier` verified 13 `backend-go` packages (`930 PASS, 1 pre-existing skip TestHandleCheckout_AirwallexPolicyViolation, 0 FAIL`), `apps/handoff-service` (`5 PASS`), `packages/handoff` (`9 PASS`), `events -args -strict=true` (`12 PASS`), `go vet`, `ci_schema_drift_gate.sh`, `money_path_gate.sh`, `ci_enterprise_gates.sh`, `gen_contracts_gate.sh`, `validate_spanner_stale_reads.sh`, `ci_repo_hygiene_gate.sh`, `role_row_contract_check_full.sh`, and `admin-portal typecheck`.
- **Status:** complete

### Phase 2: Full P0 Core (M0.2-M0.5 — Phased Execution Waves)
- [x] **Wave 2A — M0.2 Append-Only Balanced GL & Dual-Write (`complete`, live-verified `#296`):**
  - DDL & Parity: `GlAccounts`, `JournalEntries` (`EntryHash`/`PrevHash`, `IdempotencyKey` unique), `JournalPostings` (interleaved `ON DELETE NO ACTION`), `GlBalances` (8-sharded), `PostingRules` (`UZ`/`KZ`) in `schema/spanner.ddl` + `schema/migrations/20261116_m02_append_only_gl.ddl` + `schemadrift/parity.go` (`40` required tables)
  - Events & Topic: `TopicLedger = "pegasusx-ledger"`, `gl.entry.posted`, `gl.entry.reversed` 3-way synced (`events/`, `contracts/events.schema.json`, `packages/types/src/envelope.ts`)
  - `gl/` + `glroutes/`: 27-account COA, `PostingRules` engine, SHA-256 hash chain, intra-batch chaining/coalescing, reversal API, `PaymentLedgerEntries` backfill, 6 finance endpoints mounted in `bootstrap/app.go`, `http_router.go`, and `e2ep0/harness.go`
  - `ArLedgerEntries` append-only (`InsertMap` across all 8 call sites) + transactional GL dual-write across `ar`, `migration`, `payment`, `creditnote`, `payout`, `cashrecon`, `claims`, `order` (`order.delivered` + credit-leave + split-tender + cash shortfall escrow release), and `internal/services/billing` (`fee.per_order`)
  - CI Immutable-Table Lint (`scripts/ci_immutable_ledger_gate.sh`, wired into `scripts/money_path_gate.sh` & `Makefile`) passing with `0` violations (`1182` non-test Go files scanned)
- [x] **Wave 2B — M0.3 Four-Way Reconciliation & Finance Portal UI (`complete`, live-verified `#315`, `#319`):**
  - DDL & Events (`#305`, `#307`): `ReconciliationRuns`, `ReconciliationBreaks` + 3 indexes (`spanner.ddl`, `20261116_m03_four_way_reconciliation.ddl`, `schemadrift/parity.go` `42` required tables); `recon.run.completed`, `recon.break.opened`, `recon.break.resolved` 3-way synced (`247` events / `60` payloads)
  - Orphan `StockLots` Fix (`#305`): `stocklots/reconcile.go` updated to `FULL OUTER JOIN` + `StrongRead()` + `HasV2`/`MissingV2` + live `reconcile_spanner_test.go`
  - Core Engine (`recon/`, `reconroutes/`, `controltower/` `recon_exceptions.go`, `bootstrap/`, `http_router.go`, `e2ep0/`; `#310`, `#317`, `#319`): `Orders<->AR`, `Stock<->Lots<->GL`, `Cash<->PSP`, `Tax<->Fiscal`, `GET/POST/PATCH /v1/finance/recon/*`, `sweeper:recon_nightly`
  - Web Portal Surfaces (`#312`, `#315`): `supplier-portal` `/finance/trial-balance`, `/finance/ledger`, `/finance/reconciliation`, upgraded `/reconciliation` (`<Suspense>` wrapped, `0` `@pegasusx/ui` imports); `admin-portal` `/finops` GL & Recon cards; `en/ru/uz` `5267` i18n keys
- [x] **Wave 2C — M0.4 Audit v2 + RBAC/SoD (`complete`, live-verified `#338–#345`):**
  - DDL & Events (`#330`, `#342`): `AuditEvents`, `AuditChainAnchors`, `AuditLegalHolds`, `RolePermissions`, `SupplierUserRoles`, `SodPolicies`, `SodViolations` (`spanner.ddl`, `20261116_m04_audit_v2_rbac_sod.ddl`, `schemadrift/parity.go` `49` required tables); `audit.anchor.written`, `iam.role.changed`, `sod.violation.detected` 3-way (`250` events / `63` payloads); `ci_immutable_ledger_gate.sh` extended (`AuditEvents`, `AuditLog`, `CreditPolicyAudit`, `PlatformAdminAudit`)
  - Audit v2 & Debt Fixes (`audit/`, `auditroutes/`, `foundry/`, `credit/policy.go:629`, `payload/sequence.go`; `#338`): SHA-256 daily hash chain, role-aware PII masking, >32KB diff offload, WORM anchor sink, `AuditMiddleware` + nightly anchor worker, Kinetic action hook, append-only `CreditPolicyAudit`, live `HandleGetLIFOSequence` replacing `TRUCK_123` stub
  - Supplier RBAC & SoD (`auth/`, `iam/`, `iamroutes/`, domain guards, `bootstrap/`, `http_router.go`, `e2ep0/`; `#340`): 23 perm keys, 9 role templates (`AUDITOR` read-only, `AGENT` no-approve), hardened `RequireDeviceCert`, `CheckFourEyes` & static SoD, 100% mutating-route inventory test (`chi.Walk`)
  - Web UI (`#339`): `supplier-portal` `/governance` + `ObjectHistoryPanel`, `admin-portal` `/governance` & `/governance/audit`, `5350` i18n keys (`92/92` supplier static, `21/21` admin static)
- [x] **Wave 2D — M0.5 Data Platform & Ontology Projector (`complete`, live-verified `#359–#370`):**
  - Schemas, Types & Contracts (`#359`, `#361`): 17 v1 schemas in `contracts/ontology/*.v1.schema.json` + `index.json`, `packages/types/src/ontology.ts`, `TopicIntel`/`TopicConnectors`/`ontology.object.changed` 3-way (`251` events / `64` payloads), `BigQuerySinkCheckpoints` (`50` tables)
  - Ontology Projector & Unified EKG (`ontology/`, `ontologyroutes/`, `planning/`, `supplier/`, `foundryroutes/`, `bootstrap/`, `runtime_workers.go`, `http_router.go`, `e2ep0/`; `#369`): monotonic version projection, `<=900`-mutation batching, `MARKING:FINANCE_ONLY`/`PII` masking, `RebuildFromSpanner`, `/v1/ontology/*`, unified `/v1/supplier/knowledge-graph`, `PLATFORM_ADMIN`/connector restriction on `POST /v1/foundry/ontology/objects|links`
  - Infra & Sink (`infra/terraform/modules/{analytics,messaging,storage_security}`, `bqexport/`, `apps/ai-worker/`, `infra/k8s/`; `#363`): BigQuery `raw`/`curated`/`features` + authorized views, Vertex staging, `pegasusx-{ledger,intel,connectors}` topics, WORM GCS bucket, `void-bq-sink`
  - Web UI & Cleanups (`#365`): `supplier-portal` `/analytics/knowledge-graph` interactive ontology UI, `warehouse-portal/app/cycle-counts/page.tsx` stale stub copy removed, `5395` i18n keys
- **Status:** complete

### Phase 3: P1 Intelligence & Decision Workflows (`M1.1–M1.10`)
- [x] **Wave 3A (`complete`, live-verified `#383–#390`) — Control & Accounting Core (`M1.1–M1.4`):**
  - DDL & Events (`#383`): `OriginalPeriodId` on `JournalEntries` + 9 tables (`Periods`, `ApprovalPolicies`, `ApprovalRequests`, `ApprovalDecisions`, `ApprovalDelegations`, `InventoryCostLayers`, `InventoryMovements`, `EsfDocuments`, `EsfStatusHistory`; `59` required tables) + `period.*`, `approval.*`, `inventory.*`, `esf.*` 3-way contracts (`263` events, `71` payloads)
  - `M1.1` Period Close (`periods/`, `periodroutes/`, `gl/`; `#386`, `#390`): local-timezone soft/hard close, live checklist evaluator, sealed WORM/GCS snapshot, 2-approver reopen, late operational event rollover stamping `OriginalPeriodId`
  - `M1.2` CEL Approval Engine (`approvals/`, `approvalroutes/`, `featureflags/`, `order/financial_drafts.go`, `gl/handlers.go`; `#386`, `#390`): zero-dep CEL AST evaluator, pinned policy version, SHA-256 `PayloadHash`, expiring delegations, `AGENT`/`AUDITOR` `403`, SoD, MFA threshold, HMAC Telegram webhook
  - `M1.3` Inventory Valuation + COGS (`valuation/`, `valuationroutes/`, `warehouse/`, `stocklots/`, `recon/`; `#385`, `#390`): FEFO/moving-avg `InventoryCostLayers`, atomic `INVENTORY`/`COGS`/`AP:VENDOR`/`EXP:SHRINK` GL dual-write, `STOCK_GL` `DeltaMinor == 0`
  - `M1.4` First-Class ESF (`esf/`, `esfroutes/`, `fiscal/uzbekistan.go` mock replaced, `order/buyer_acceptance_poller.go`; `#385`, `#390`): outbound/inbound ESF lifecycle, rejection VAT reversal, VAT summary `DiffMinor == 0`, missing IKPU dashboard
  - Web UI (`#384`, `#389`): `supplier-portal` `/finance/close`, `/approvals`, `/fiscal` (`95/95` build+static, `33/129`), `warehouse-portal` `/inventory/valuation` (`52/52`, `20/63`), `5713` i18n keys
- [x] **Wave 3B (`complete`, live-verified `#405–#418`) — Intelligence, Credit, Proposals & ROI (`M1.5–M1.8`):**
  - DDL & Events (`#405`): `RetailerPaymentTerms.ConsentVerified`, `ForecastRuns`, `ForecastValues`, `ForecastExplanations`, `CreditScoreHistory`, `Proposals`, `RoiBaselines`, `RoiEvents` (`66` required tables) + 10 events (`273` events, `78` payloads)
  - `M1.5` Forecast v2 (`forecastv2/`, `forecastroutes/`, `retailer/ai_demand.go`; `#409`): removed `http://localhost:8000` fallback, Champion/Challenger calendar regressors + `GeminiExplainer`, `POST /v1/replenishment/suggestions/{id}/approve`
  - `M1.6` Credit v2 (`credit/`, `creditroutes/`, `bootstrap/wave3b.go`; `#409`, `#414`, `#417`): removed `credit/scoring.go:220` `1_000_000` placeholder, Relationship + Network scoring + `MLScorerClient`, `/v1/credit/{retailers/{id}/score,portfolio,limits/{id}}`, `SetMutationContextHook`
  - `M1.7` & `M1.8` Proposals + ROI (`proposals/`, `proposalroutes/`, `roi/`, `roiroutes/`, `bootstrap/wave3b.go`, `e2ep0/`; `#411`, `#417`): 3 L1 agents (`REORDER_AGENT`, `COLLECTIONS_AGENT`, `EXPIRY_REBALANCE_AGENT`) -> CEL `approvals` -> domain executors + `RoiEvents` (with `HoldoutControl`, `IsEstimate`, QBR JSON/CSV)
  - Web & Desktop UI (`#408`, `#410`, `#415`): `supplier-portal` `/analytics/demand`, `/credit/scores` (`BLOCK`), `/proposals`, `/roi`, dashboard ROI banner (`98/98`, `34/137`); `retailer-app-desktop` Suggested Orders approve (`35/35`, `26/131`); `5809` i18n keys
- [x] **Wave 3C (`complete`, live-verified `#432–#447`) — Connectors & Auditor Pack (`M1.9–M1.10`):**
  - DDL & Events (`#432`, `#435`): `ExternalCredentials`, `ConnectorSyncRuns`, `ConnectorEntityMaps`, `ExportPacks`, `SupplierUserRoles.ExpiresAt` (`70` required tables) + 5 events (`278` events, `82` payloads)
  - `M1.9` Connector Spine & Real `.xlsx` (`connectors/`, `connectorroutes/`, `supplier/`; `#439`, `#444`): common 5-method `Connector` interface, `EnvSecretResolver`, fail-closed `BillzConnector` + `httptest.Server` coverage + `ShadowCompare` (`ConnectorSyncRuns` + `ConnectorEntityMaps` + `TopicConnectors`), `OnecConnector` multi-account, real `excelize/v2` streaming `.xlsx`/`.xls` in `supplier/` (>10,500 rows + `422` encrypted)
  - `M1.10` Auditor Pack & Time-Boxed `AUDITOR` Portal (`partner/`, `exportpacks/`, `exportpackroutes/`, `auth/`, `iam/`, `iamroutes/`, `bootstrap/wave3c.go`, `http_router.go`, `e2ep0/`; `#437`, `#442`, `#444`, `#447`): unlimited streaming & `xlsx` in `partner/`, 15-table OOXML `.xlsx`/`.csv`/1C XML/`.zip` `ExportPacks` w/ SHA-256 + watermark + `410` expiry + `AuditEvents`, time-boxed `AUDITOR` grants/JWT (`401 auditor_access_expired` & `403` read-only across `partner/`, `supplier/`, `connectors/`, `exportpacks/`)
  - Web UI (`#438`, `#443`): `supplier-portal` `/settings/integrations` Connector Spine (`canMutate` safe mode for `AUDITOR`), `/auditor` Read-Only Auditor & Tax Pack Portal (`99/99` build+static, `35/144`), `5979` i18n keys
- **Status:** complete

### Phase 4: P2 Enterprise & Network Scale + Full Ecosystem Verification
- [x] **Wave 4A (`complete`, live-verified `#465–#479`) — P2 Financial & Supply-Chain Depth:**
  - DDL & Events (`#465`): `LegalEntities`, `IntercompanyTransfers`, `FxRevaluationRuns`, `JournalEntries.LegalEntityId`, `ProcessTemplates`, `ProcessInstances`, `PurchaseOrders`, `PurchaseOrderLines`, `GoodsReceipts`, `VendorInvoices` (`79` required tables) + 6 events (`284` events, `86` payloads)
  - Multi-Entity & Consolidation (`gl/`, `glroutes/`, `multientity/`, `multientityroutes/`; `#469`, `#476`): entity `?entity=` scoping, IC transfers (`IC:DUE_FROM`/`IC:DUE_TO`), fail-closed BigInt `1e8` FX revaluation (`FX:ROUNDING`/`FX:UNREALIZED_GAIN_LOSS`), group consolidation w/ IC eliminations
  - P2P Templates & 3-Way Match + Process Mining (`p2p/`, `p2proutes/`; `#467`): `O2C-*`/`P2P-*`/`RETURNS-*` shipped templates, multi-line POs, Receipts (`INVENTORY`/`GRNI`), 3-way match (`GRNI`+`TAX:VAT:INPUT`->`AP:VENDOR`), `OrderStatusTransitions` + P2P bottleneck Process Mining
  - 2-Way Adapters, Bank Pack & Wiring (`connectors/`, `connectorroutes/`, `exportpacks/`, `bootstrap/wave4a.go`, `http_router.go`, `audit_route_inventory_test.go`, `e2ep0/`; `#473`, `#478`, `#479`): Billz/1C/MoySklad/Smartup/iiko pushback gated by `approvals`, `BANK` credit tape (DPD/concentration/loss curves), 100% route inventory & `TestWave4A_*` E2E
  - Web UI (`#468`, `#470`, `#477`): `supplier-portal` `/finance/consolidation`, `/p2p`, upgraded `/settings/integrations` (5 providers, push approval flow) & `/auditor` Bank preview (`101/101`, `36/149`, `6258` i18n keys)
- [x] **Wave 4B (`complete`, live-verified `#491–#496`) — P2 Intelligence, Safety, Identity & P2 UI:**
  - DDL & Events (`#491`): `AnomalyCases`, `WhatIfSimulations`, `CustomRoles`, `ScimIdentities`, `Proposals.AutonomyLevel` (`83` required tables) + 6 events (`290` events, `91` payloads)
  - Anomaly Catalogue & What-If E2E (`anomalies/`, `anomalyroutes/`, `whatif/`, `whatifroutes/`; `#492`): 5 live Spanner detectors (`CASH_GEOFENCE_MISMATCH`, `BASKET_SPIKE_SHARED_INN`, `STOCK_COUNT_VARIANCE`, `PRICING_DISCOUNT_ABUSE`, `MANUAL_JOURNAL_AFTER_HOURS_OR_SPLIT`) + dedup/resolve/escalate; 5-stage `int64` What-If simulator + `commit-proposals` -> L1 proposals
  - HACCP, L2 Soak Thresholds, Custom Roles & SCIM v2 + Wiring (`stocklots/haccp.go`, `proposals/l2_policy.go`, `proposalroutes/`, `iam/`, `iamroutes/`, `bootstrap/wave4b.go`, `http_router.go`, `audit_route_inventory_test.go`, `e2ep0/`; `#495`): `GET /v1/compliance/haccp`, soak-gated `L2` auto-exec blocking high-risk actions, `CustomRoles`, `/scim/v2/{Users,Groups}`, 100% route inventory & `TestWave4B_*`
  - Web UI (`#494`): `supplier-portal` `/control-tower/anomalies`, `/planning/what-if`, `/compliance/haccp`, upgraded `/proposals` L2 soak gate & `/governance` Custom Roles + SCIM v2 (`104/104`, `37/156`, `6436` i18n keys)
- [x] **Wave 5 (`complete`, `G-09`) — Full Ecosystem Verification, Hardening & Live QA:** Code-review security/correctness fixes (`outbox` batch fetch, `SCIM` multi-tenant bearer binding, `connectors` SSRF+pooled client, `exportpacks` formula sanitization, `approvals` CEL caps) + live Playwright-caught `spanner.ErrRowNotFound` / `codes.NotFound` fix across `mfa/`, `orgoidc/`, `supplier/`, `manifest/` (+ regression tests); post-final-edit pass across `backend-go` (`165 ok`, `0` failures, `-race`), `ai-worker`, `handoff-service`, `optimizer-core` (`25/25`), `dispatch-optimizer-py` (`11/11`), `pnpm` workspace tests/typechecks/builds, `ci_macos_control_tower` (`13/13`), `ci_android_unit_tests` (`6/6`), `ci_ios_apps` (`6/6`), Terraform/K8s/CI gates, live HTTP + Playwright QA (`20/20` supplier + `6/6` admin routes), and clean workspace state.
- **Status:** complete

---

## User Confirmed Scope & Decisions (2026-11-16)
1. **Execution Scope (`A1`):** `M0.1 Gate Zero Closure First` — Close remaining `T8`, `T9`, `T1` event + run `T1–T9` live emulator regression before moving to `M0.2+`.
2. **Accounting & Cloud Adapter Scope (`A2`):** Full GL + 1C dual-write, typed BigQuery/Vertex/Billz sink contracts & local/emulator-safe adapters (applies as architecture target for `M0.2+`).
3. **Swarm Model Allocation (`A3`):** All workers on `gemini-4-argon-high` (omit `model` on `swarm add`).
4. **Sibling Parity (`A4`):** Work in `/Users/shakhzod/Desktop/pegasusx` only first (do not auto-sync `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`).

## Decisions Made
| Decision | Rationale | Date |
| :--- | :--- | :--- |
| Archived Nov 10 planning files to session scratch before starting ATOMOS plan | Preserves completed Nov 10 operating-model/CodeGraph record cleanly without repo clutter | 2026-11-16 |
| Verified `T1-T9` against actual repo state instead of assuming `ATOMOS_Enterprise_Intelligence_Plan.md` snapshot (`a813de086`) | Prevented duplicate work: `T2, T3, T4, T5, T6, T7`, `SetLotsEnabled`, and `substitution.go` are already implemented in HEAD `e25490326` + uncommitted remediation changes | 2026-11-16 |
| Omitted `model` on `swarm add` per `/swarm` rule; workers inherited coordinator's active `gemini-4-hydrogen-xhigh` runtime model | Board metadata listed `coordinator` as `gemini-4-argon-high` from an older session, while active runtime inherited `gemini-4-hydrogen-xhigh` (`76%` quota tier) | 2026-11-16 |

## Errors Encountered & Resolutions
| Error | Attempt | Resolution |
| :--- | :--- | :--- |
| `schema/spanner.ddl: No such file or directory` when running at repo root | 1 | Located canonical DDL at `apps/backend-go/schema/spanner.ddl` |
| `~/.gemini/config/skills/swarm/SKILL.md: file does not exist` | 1 | Used `swarm` instructions already injected in session context & planning-with-files skill instructions |
| `zsh: no matches` / exit code 1 in chained verification shell script (`rg` exit 1 on 0 matches) | 1 | Re-ran with explicit subshell guards (`\|\| true`) to inspect all remaining sections cleanly |
| `DraftAdjustments.ReviewedAt` / `DraftCreditNotes.ReviewedAt` missing `allow_commit_timestamp=true` (`#257`) | 1 | `t9-drafts` added `OPTIONS (allow_commit_timestamp=true)` in `spanner.ddl` & `20261116_financial_drafts_minor_units.ddl` (`#259`) |
| Live emulator `#266`: `SaveWebhook` `Idx_PaymentLedgerEntries_GatewayTypeRef` NULL_FILTERED check + `WriteDoorstepShortfall` 3-leg test assertion (`=3` vs `=2`) | 1 | Resolved by `t8-ledger` (`#267`: emulator hint + `ReferenceId IS NOT NULL`, `_p2` cleanup + `3`-row assertion); verified live in `#269` |
| Cloud Spanner reserved keyword `HASH` on `JournalEntries` (`#278`) | 1 | Named column `EntryHash` across `spanner.ddl`, `20261116_m02_append_only_gl.ddl`, and `apps/backend-go/gl` |
| Stale-read allowlist line shifts after `#285` dual-write additions | 1 | `m02-gl-schema-events` ran `--update` (`#287`: 370->370 entries, 0 new `.Single().Query`); `validate_spanner_stale_reads.sh` & `money_path_gate.sh` pass |
| `m02-verifier` `#288` blockers: (1) `ar/` payment mode fell back to `BANK:OPERATING`, (2) doorstep cash shortfall + delivery/credit-leave double-debited `CASH:DRIVER` / stranded `ESCROW:ORDER` | 1 | Resolved by `m02-dual-write` (`#291`: explicit AR mode propagation + `gl:shortfall:` release / credit-leave guard + `ar/service.go` allowlist sync `#293`), live-verified `#296` |
| `m03-verifier` `#315` blockers: (1) `CASH:OFFICE` tie-out + manual test seeding, (2) `AR_ADVANCE` & `LEDGER_ONLY` chargeback scoping, (3) `period_id` validation & `'SUCCESS'` fiscal status | 1 | Resolved by `m03-recon-core` (`#317`: exact cashrecon GL tie-out, real dual-write Spanner tests, `arAdvanceCap` + `LEDGER` exclusion, regex + `'SUCCESS'`), live-verified `#319` |
