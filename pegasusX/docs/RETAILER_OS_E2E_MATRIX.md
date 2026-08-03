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
| Family → Team migrate | `TestFamilyMigrate*` | Settings → Family | Family Members | Family Members | After image deploy |
| Auto-order draft worker | `TestAutoOrder*` | Draft now + runs | Auto-Order runs (read) | Auto-Order runs (read) | After image deploy |
| Auto-order **place** | `TestAutoOrderWorkerPlaceMode` | Place now + confirm + placed_orders | Place now + confirm | Place now + confirm | Flag `AUTO_ORDER_PLACE_ENABLED` + geo + bucket DDL |
| Durable AutoOrder runs | dual-write + list | Last runs (multi-pod) | Last runs | Last runs | Apply `20260809_retailer_auto_order_runs.ddl` |
| Offline POS cash + sync | `TestPOSOffline*` | `/pos` offline queue | POS offline queue | POS offline queue | After image + DDL |
| Sell-through flywheel L3 | `TestSellThrough*` | Insights sell-through panel + chips | Auto-order chips list | Auto-order chips list | Apply sell_through DDL |
| Source chips enterprise | merge + API | ui-kit chips; supplier portal + server `?source=` | DemandSourceChips | DemandSourceChips | sources DDL |

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
- `20260802_retail_os_sell_through.ddl` + `20260802_retail_os_sell_through_sources.ddl`
- `20260802_retail_os_auto_order_bucket.ddl` (multi-pod place idempotency)
- `20260809_retailer_auto_order_runs.ddl` (durable run audit)

Also present in `schema/spanner.ddl` where merged.

### Place enable (SSMR)

1. Primary location has lat/lng + 15-char H3  
2. `AUTO_ORDER_PLACE_ENABLED=true`  
3. Desktop **Place now** → confirm → supplier sees `AUTO_ORDER`  
4. Second place same day → `already_processed_bucket`
