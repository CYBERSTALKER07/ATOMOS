# Retailer capability packs (Retail OS Phase 0)

**Status:** Implemented foundation (2026-08-02)  
**Clients:** desktop `/settings/capabilities`, Android Profile → Store capabilities, iOS Profile → Store capabilities  
**API:** `GET/POST /v1/retailer/capabilities*`, `GET /v1/retailer/me`

## Principle

Minimum-path first. Every retailer starts with **CORE** only. Larger shops opt into packs. Hard dependencies block enable; soft dependencies warn.

## Pack catalog

| Pack | Default | Hard deps | Soft deps |
|------|---------|-----------|-----------|
| CORE | always on | — | — |
| TEAM | off | — | — |
| LOCATIONS | off | — | TEAM |
| STORE_STOCK | off | — | TEAM |
| SECTIONS | off | STORE_STOCK | TEAM |
| POS | off | STORE_STOCK | TEAM, SHIFTS |
| SHIFTS | off | TEAM | POS |
| REPORTS_PRO | off | — | — |
| CUSTOMER_ASSIST | off | SECTIONS, TEAM | — |

## Auth (v2 JWT)

| Claim | Meaning |
|-------|---------|
| `sub` | `RetailerUserId` (person) |
| `retailer_org_id` | tenant `RetailerId` |
| `retailer_role` | OWNER / ADMIN / … |
| `capability_packs` | snapshot of enabled packs |

Legacy tokens with empty `retailer_org_id` treat `sub` as org id and role as OWNER.

Login bootstraps an **OWNER** row in `RetailerUsers` if missing.

## Durable settings (Phase 0.4)

- Auto-order → `RetailerAutoOrderSettings`
- Favorite suppliers → `RetailerFavoriteSuppliers`

## Migration

`apps/backend-go/schema/migrations/20260802_retail_os_phase0.ddl`  
Also appended to `schema/spanner.ddl`.

Apply with your usual Spanner DDL path (`make spanner-init` / cloud migrate).

## Phase 1 Team (shipped)

| API | Notes |
|-----|--------|
| `GET/POST /v1/retailer/org/members` | Roster + invite (auto-enables TEAM pack) |
| `PATCH/PUT/DELETE …/members/{userID}` | Update / soft-deactivate (owner locked) |
| Login | Staff phone+password; JWT `sub`=user id, `retailer_org_id`=shop |
| FCM fanout | `broadcastRetailer` pushes to all active user ids + org id |

Clients: Settings → **Team** (desktop/Android/iOS). Family contacts remain legacy.

## Phase 2 Locations (shipped)

| API | Notes |
|-----|--------|
| `GET/POST /v1/retailer/locations` | List/create; first primary auto-bootstrapped from `Retailers` |
| `PATCH …/locations/{id}` | Update address/geo/windows |
| `POST …/locations/{id}/set-primary` | Primary mirrors to shop delivery fields |
| `POST /v1/auth/retailer/switch-location` | Re-issues JWT with `active_location_id` |
| `PUT …/org/members/{userID}/locations` | Staff location scope |
| Order create | Optional `location_id` on `CreateRequest` (correlation; lat/lng/h3 still drive routing) |

Clients: Settings → **Locations** (desktop/Android/iOS). Checkout bind uses active branch via switch + primary mirror.

## Phase 3 Store stock (shipped)

See `docs/RETAILER_STORE_STOCK.md`.

| Area | Status |
|------|--------|
| Ledger balances + movements | Spanner + memory tests |

## Phase 4 POS (shipped)

See `docs/RETAILER_POS.md`.

## Phase 5 Shifts & time (shipped)

See `docs/RETAILER_SHIFTS.md`.

| Area | Status |
|------|--------|
| Clock in/out + time entries | Memory + Spanner |
| Shift open/close cash recon | Linked POS session cash tenders |
| POS require clock-in | When SHIFTS pack + config |
| Variance alerts | Owner inbox + outbox event |
| Clients | Desktop `/shifts`, Android/iOS Profile → Shifts |

## Phase 6 Sections + Reports + Assist (shipped)

| Pack | Doc |
|------|-----|
| SECTIONS | `docs/RETAILER_SECTIONS.md` |
| REPORTS_PRO | `docs/RETAILER_REPORTS_PRO.md` |
| CUSTOMER_ASSIST | `docs/RETAILER_ASSIST.md` |

| Area | Status |
|------|--------|
| Sections CRUD + SKU/staff map | Memory + Spanner |
| Unassigned SKU pool | From stock balances |
| Reports summary/sales/inventory/shifts + CSV | Aggregates P3–P5 data |
| Assist ticket lifecycle | OPEN→CLAIMED→DONE + notify |
| Clients | Desktop `/sections` `/reports` `/assist`; mobile Profile |
| Receive from order | LineItemsJson from COMPLETED/ARRIVED-ish |
| Transfer / adjust / count | APIs + UI |
| Reorder `CurrentStock` | Prefers store ledger sum |

## Phase 4 POS (shipped)

See `docs/RETAILER_POS.md`.

Registers → open session → sales (FLOOR stock) → void → close with cash variance.

## Next phases

P5 Shifts (clock + require_shift on POS open), P6 Sections/Reports.
