# Retailer Sections (Retail OS Phase 6)

Departments/shelves for store ops: SKU maps, staff assignment, assist routing.

## Dependencies

- Hard: **STORE_STOCK**
- Soft: TEAM

First section create auto-enables STORE_STOCK + SECTIONS.

## APIs

| Method | Path |
|--------|------|
| GET/POST | `/v1/retailer/sections` |
| GET/PATCH/DELETE | `/v1/retailer/sections/{id}` |
| GET/PUT | `/v1/retailer/sections/{id}/skus` |
| GET/PUT | `/v1/retailer/sections/{id}/staff` |
| GET | `/v1/retailer/sections/unassigned-skus` |
| GET | `/v1/retailer/me/sections` |

SKU multi-map allowed (endcaps). Permission: `section.manage` for mutations; `stock.view` for list.

## Schema

`schema/migrations/20260802_retail_os_phase6_sections_assist.ddl`

## Clients

Desktop `/sections` · Android/iOS Profile → **Sections**
