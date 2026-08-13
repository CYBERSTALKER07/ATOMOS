# Living Gap Ledger — Enterprise 10/10

**SoT program:** [`MASTER_10_10_EXECUTION_PROGRAM.md`](./MASTER_10_10_EXECUTION_PROGRAM.md)  
**Do not implement out of phase order without updating this ledger.**

Status: `OPEN` | `IN_PHASE` | `DONE` | `WONT` (with reason)

---

## G1 — Money & law (P0)

| ID | Gap | Packages / clients | Status | Notes |
|----|-----|-------------------|--------|-------|
| G1-A1 | Cash collect AR pay-down post-commit fail-open | `order/service.go`, `ar/*` | **DONE** | G1.A: `RecordPaymentForOrderInTxn` in CollectCash InTxn |
| G1-A2 | Credit ClearBalance fail-open edges | `credit/*`, order card/cash settle | **DONE** | ClearBalanceInTxn in finalizeCardSettlement + CollectCash; CLEARED idempotency |
| G1-B1 | Tax default still PEGASUS commercial | `order/fiscal_*`, k8s overlays | **DONE** | G1.B: tax-class unset→MY_SOLIQ; prod overlay MY_SOLIQ; PEGASUS needs allow flag |
| G1-B2 | MY_SOLIQ misconfig fail-closed docs/ops | fiscal + deploy | **DONE** | hardFailProvider + production boot validate OFD/EDS |
| G1-C1 | Negotiation UI vs 410 API | order/negotiation*, supplier/driver clients | **DONE** | Keep 410 default; portal residual page honest; no score theatre |
| G1-C2 | Driver PATCH state always 501 | driver clients + mobile_compat | **DONE** | Clients redirected/fail-honest; 501 trap + use_delivery_edges |
| G1-C3 | Credit scores stub empty | `credit/repository.go`, supplier UI | **DONE** | Stub documented; removed notif pref; no invented scores |
| G1-C4 | Mid-delivery update not_implemented | `delivery_handshake.go`, driver apps | **DONE** | Clients stop calling; amend path only; error use_amend_or_partial_offload |
| G1-D1 | Payout bank-file only / ErrNoLiveRail | `payout/rail.go`, supplier portal | **DONE** | RailInfo + no_live_rail 409 + MarkPaid honesty; runbook |
| G1-D2 | FCM silent no-op in some profiles | notifications / bootstrap | **DONE** | push_degraded logs; prod fail-closed unless FCM_ALLOW_NOOP |

## G2 — Physical + autonomy

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G2-A1 | Pick waves / cycle counts default false | **DONE** | Seal-class via Effective* + dual-control overrides; env default stays false |
| G2-A2 | Stocklots remaining silent mutators (density) | **DONE** | Putaway/pick/cycle approve emit outbox; residual low-noise accepted |
| G2-B1 | Payload line-level scan ledger | **DONE** | ManifestLoadLines + APIs + seal gate; client full rewire residual |
| G2-C1 | Cold chain default off for chilled | **DONE** | Seal baseline assert when cold effective + chilled SKUs |
| G2-C2 | Labor-capacity not hard-coupled to dispatch | **DONE** | LABOR_CAPACITY_ENFORCE + warehouse ExecuteDispatch |
| G2-D1 | Dual FactoryTruckManifests vs SupplierTruckManifests | **DONE** | Option B + ManifestDomain; no table merge |
| G2-E1 | Auto-order place fail-closed (no soak flip) | **PARTIAL** | Flip path + dual-control exist; no soak artifact invent — place stays off until ops |

## G3 — Collections + client honesty

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G3-A1 | Dunning transports/flags production | **DONE** | Transports wired; GET `/v1/admin/ar/dunning/status` honesty; dual-control AR_DUNNING already money flag |
| G3-B1 | Credit risk scoring v1 | **DONE** | `credit/scoring.go` g3_v1 weights; desk scores + GET credit-scores; AR metrics soft |
| G3-C1 | Retailer dead settings prefs | **DONE** | Removed settlement-priority theatre; push mute labeled local-only |
| G3-C2 | Tracking GPS fallback honesty | **DONE** | LIVE / LAST_KNOWN / AWAITING_TELEMETRY + last-known coords |
| G3-C3 | POS scan-to-cart | **DONE** | POST `/v1/retailer/pos/scan` barcode → ready sale line |
| G3-D1 | Supplier settlement/earnings honesty | **DONE** | Earnings 503 codes; portal ledger_fallback already labeled |

## G4 — Tenancy + ops

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G4-A1 | Seed fallbacks in enforced envs | **DONE** | SeedFallbackAllowed; PreferTenant never seeds under ssmr/prod by default |
| G4-B1 | Admin token-paste login | **DONE** | Password login primary; paste dev-only |
| G4-B2 | Outbox/DLQ ops visibility | **DONE** | platform-admin ops APIs + Ops tab; DLQ via CLI note |
| G4-C1 | Optimizer prod truth / HEURISTIC labels | **DONE** | optimizer_class HEURISTIC\|OPTIMAL + health/capabilities |

## G5 — Enterprise I/O

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G5-A1 | EDI profile packs per tenant | **DONE** | PartnerEdiProfiles + inbound/outbound gate + API |
| G5-B1 | SAP or certified 1C adapter | **DONE** | 1C-first import adapter + journals; SAP residual README |
| G5-C1 | Master-data sync | **DONE** | Parties/plants + version conflict DLQ |
| G5-D1 | External WMS ASN bidirectional | **DONE** | DESADV out + POST /wms/asn in idempotent |

## G6 — Brain

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G6-A1 | Forecast MAPE publish + demote | **DONE** | MAPE28 + FORECAST_DEMOTE_* + accuracy_demoted |
| G6-A2 | REGION/CITY sensing shortcut | **DONE** | Fail-closed geo match in demand/scope_match.go |
| G6-B1 | MEIO beyond greedy | **DONE** | cost_aware_v2 bang-for-buck + capital (heuristic) |
| G6-C1 | CP_SAT naming honesty | **DONE** | GREEDY_ASSIGN alias; cold score −1; Rust never OPTIMAL |
| G6-D1 | Road-network ETA in dispatch | **DONE** | DISPATCH_SCORE_USE_OSRM + matrix_source honesty |

## G7 — Polish

| ID | Gap | Status | Notes |
|----|-----|--------|-------|
| G7-1 | Factory SLA board | **DONE** | sla-board API + badges + FACTORY_SLA_BREACH worker |
| G7-2 | Client drift matrix close | **DONE** | Dead-letters UI + ROLE_ROW/parity G7 |
| G7-3 | FEATURES_BY_APP_ROLE regen | **DONE** | 2026-08-13 G4–G7 deltas |
| G7-4 | Full Reality Report re-score | **DONE** | SCORECARD + RESIDUAL_REGISTER |

## Already closed (do not re-open without regression)

| ID | Wave | Summary |
|----|------|---------|
| B1–B5 | Waves | Money keys, logistics outbox, retailer org, supplier ops, platform PA |
| B6 | Wave | AR open same-txn credit leave; claim HTTP idem; AR aging outbox |
| B7 | Wave | Stubs 503; reverse pin; stocklots membership; scan outbox; factory setup outbox; payload WH scope |

---

## Rule

Closing a gap requires: code + cross-role alignment + tests + scorecard cell update in the owning phase’s `05_SCORECARD_DELTA.md`.
