# Next-Layer Remaining — Baseline (2026-08-04)

## Preflight

| Check | Result |
|-------|--------|
| `https://api-ssmr.pegasusx.app/healthz` | OK |
| Backend image | `…/backend-go:ssmr-wave-c-52de1e9b` |
| Spanner: `RetailerSellThroughDaily` | Present |
| Spanner: `RetailerLocalCatalog` | Present |
| Spanner: `RetailerPosHolds` | Present |
| Spanner: `RetailerStockLocationVersions` + ForceAudits | Present |
| Spanner: `SlaBreachedAt` | Present |

## Flags observed on `backend-go` (SSMR)

| Flag | Value |
|------|-------|
| `OFFLINE_COUNT_ENABLED` | `true` |
| `ASSIST_SLA_ENABLED` | `true` |
| `ASSIST_SLA_MINUTES` | `15` |
| `AUTO_ORDER_PLACE_ENABLED` | `true` (plan default: prefer draft; will set worker draft + place off unless intentional) |
| `MULTI_ORG_LOGIN_ENABLED` | unset (off) |
| `HQ_ANALYTICS_ENABLED` | unset (off) |
| `FIREBASE_AUTH_ENABLED` | unset / secret-path wiring present |
| `QUANTITY_NEGOTIATION_ENABLED` | unset (compile gate still `true`) |
| `GLOBAL_PAY_*` | from secret (password still owner-gated for SUCCESS) |

## Already shipped in tree (verify only)

L2 family migrate, AutoOrderWorker, CT pulse, L3 sell-through, L6 local SKUs, L7 claim quarantine port, Wave C holds/count/assist/HQ/multi-org.

## Remaining program

L1 card SUCCESS marker + release checklist; L2 leftovers; L3/L6/L7 e2e+docs+mobile local SKU; L4 env gate; L5 Soliq docs; Wave C pilot hygiene + remove 59MB `ssmr-smokecheck` binary.
