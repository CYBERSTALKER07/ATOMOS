# Living Gap Ledger — Enterprise 10/10

**SoT program:** [`MASTER_10_10_EXECUTION_PROGRAM.md`](./MASTER_10_10_EXECUTION_PROGRAM.md)  
**Do not implement out of phase order without updating this ledger.**

Status: `OPEN` | `IN_PHASE` | `DONE` | `WONT` (with reason)

---

## G1 — Money & law (P0)

| ID | Gap | Packages / clients | Status | Notes |
|----|-----|-------------------|--------|-------|
| G1-A1 | Cash collect AR pay-down post-commit fail-open | `order/service.go`, `ar/*` | OPEN | Credit-leave open co-atomic (B6) |
| G1-A2 | Credit ClearBalance fail-open edges | `credit/*`, claims settle | OPEN | |
| G1-B1 | Tax default still PEGASUS commercial | `order/fiscal_*`, k8s overlays | OPEN | Code complete path; profile flip |
| G1-B2 | MY_SOLIQ misconfig fail-closed docs/ops | fiscal + deploy | OPEN | |
| G1-C1 | Negotiation UI vs 410 API | order/negotiation*, supplier/driver clients | OPEN | Ship or delete |
| G1-C2 | Driver PATCH state always 501 | driver clients + mobile_compat | OPEN | Remove client calls |
| G1-C3 | Credit scores stub empty | `credit/repository.go`, supplier UI | OPEN | Score v1 or remove UI |
| G1-C4 | Mid-delivery update not_implemented | `delivery_handshake.go`, driver apps | OPEN | Implement or remove |
| G1-D1 | Payout bank-file only / ErrNoLiveRail | `payout/rail.go`, supplier portal | OPEN | Honesty UX or live rail |
| G1-D2 | FCM silent no-op in some profiles | notifications / bootstrap | OPEN | Fail-loud + metrics |

## G2 — Physical + autonomy

| ID | Gap | Status |
|----|-----|--------|
| G2-A1 | Pick waves / cycle counts default false | OPEN |
| G2-A2 | Stocklots remaining silent mutators (density) | OPEN |
| G2-B1 | Payload line-level scan ledger | OPEN |
| G2-C1 | Cold chain default off for chilled | OPEN |
| G2-C2 | Labor-capacity not hard-coupled to dispatch | OPEN |
| G2-D1 | Dual FactoryTruckManifests vs SupplierTruckManifests | OPEN |
| G2-E1 | Auto-order place fail-closed (no soak flip) | OPEN |

## G3 — Collections + client honesty

| ID | Gap | Status |
|----|-----|--------|
| G3-A1 | Dunning transports/flags production | OPEN |
| G3-B1 | Credit risk scoring v1 | OPEN (or G1-C3 WONT score) |
| G3-C1 | Retailer dead settings prefs | OPEN |
| G3-C2 | Tracking GPS fallback honesty | OPEN |
| G3-C3 | POS scan-to-cart | OPEN |
| G3-D1 | Supplier settlement/earnings honesty | OPEN |

## G4 — Tenancy + ops

| ID | Gap | Status |
|----|-----|--------|
| G4-A1 | Seed fallbacks in enforced envs | OPEN |
| G4-B1 | Admin token-paste login | OPEN |
| G4-B2 | Outbox/DLQ ops visibility | OPEN |
| G4-C1 | Optimizer prod truth / HEURISTIC labels | OPEN |

## G5 — Enterprise I/O

| ID | Gap | Status |
|----|-----|--------|
| G5-A1 | EDI profile packs per tenant | OPEN |
| G5-B1 | SAP or certified 1C adapter | OPEN |
| G5-C1 | Master-data sync | OPEN |
| G5-D1 | External WMS ASN bidirectional | OPEN |

## G6 — Brain

| ID | Gap | Status |
|----|-----|--------|
| G6-A1 | Forecast MAPE publish + demote | OPEN |
| G6-A2 | REGION/CITY sensing shortcut | OPEN |
| G6-B1 | MEIO beyond greedy | OPEN |
| G6-C1 | CP_SAT naming honesty | OPEN |
| G6-D1 | Road-network ETA in dispatch | OPEN |

## G7 — Polish

| ID | Gap | Status |
|----|-----|--------|
| G7-1 | Factory SLA board | OPEN |
| G7-2 | Client drift matrix close | OPEN |
| G7-3 | FEATURES_BY_APP_ROLE regen | OPEN |
| G7-4 | Full Reality Report re-score | OPEN |

## Already closed (do not re-open without regression)

| ID | Wave | Summary |
|----|------|---------|
| B1–B5 | Waves | Money keys, logistics outbox, retailer org, supplier ops, platform PA |
| B6 | Wave | AR open same-txn credit leave; claim HTTP idem; AR aging outbox |
| B7 | Wave | Stubs 503; reverse pin; stocklots membership; scan outbox; factory setup outbox; payload WH scope |

---

## Rule

Closing a gap requires: code + cross-role alignment + tests + scorecard cell update in the owning phase’s `05_SCORECARD_DELTA.md`.
