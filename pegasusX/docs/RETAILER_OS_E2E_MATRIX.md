# Retail OS E2E / parity matrix (Phase 7)

**Last updated:** 2026-08-02  
**Backend gate:** `cd apps/backend-go && go test ./retailer/ ./retailerroutes/ -count=1`

| Case | Backend test | Desktop | Android | iOS | SSMR |
|------|--------------|---------|---------|-----|------|
| Solo CORE login + procurement path | auth/core handlers | Dashboard/catalog | Tabs | Tabs | Optional smoke |
| Enable TEAM + invite staff | `org_members_test` | Settings → Team | Profile → Team | Profile → Team | |
| LOCATIONS create / switch | `locations_test` | Settings → Locations | Locations | Locations | |
| STORE_STOCK receive → transfer → count | `store_stock_test` | `/stock` | Store stock | Store stock | |
| POS register → sale → void → close | `pos_test` | `/pos` | POS | POS | |
| SHIFTS clock + POS require + variance | `shifts_test` | `/shifts` | Shifts | Shifts | |
| SECTIONS map + unassigned | `phase6_test` | `/sections` | Sections | Sections | |
| REPORTS_PRO summary + CSV | `phase6_test` | `/reports` | Reports Pro | Reports Pro | |
| ASSIST open → claim → complete | `phase6_test` | `/assist` | Floor assist | Floor assist | |
| Pack hard-dep graph | `capability_packs_test` | Capabilities | Capabilities | Capabilities | |
| Control Tower honest pulse | `control_tower_pulse_test` | `/control-tower` | Control Tower | Control Tower | |

## Edge cases (see also REAL_WORLD_CASE_MATRIX)

| Case | Expected |
|------|----------|
| POS without STORE_STOCK (API enable) | Hard-dep BLOCKED or auto-enable stock on first register |
| SHIFTS on, POS open without clock-in | `409 clock_in_required` |
| Cash variance ≥ threshold | Owner notification + event |
| Assist without packs | Auto-enable TEAM/SECTIONS deps or clear error |

## Spanner migrations (ops)

Apply under `apps/backend-go/schema/migrations/`:

- `20260802_retail_os_phase0.ddl` … `phase6_sections_assist.ddl`

Also present in `schema/spanner.ddl`.
