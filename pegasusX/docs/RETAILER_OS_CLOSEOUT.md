# Retail OS close-out (v1)

**Date:** 2026-08-02

## What “closed” means

| Layer | Status |
|-------|--------|
| Packs CORE → ASSIST + CT pulse | Code complete |
| Spanner Retail OS DDL (SSMR) | Applied |
| Kafka / Redis / WS | Live on SSMR |
| Family → Team migrate | **Prod-ready** (API + desktop/Android/iOS) |
| Auto-order **execution** | **Prod-ready draft** (worker + run/runs + 3 clients) |
| Backend image on SSMR | **Pending deploy** (`nomock4` lacks routes until roll) |
| Offline POS sales | **Shipped** (cash queue + sync; no offline card) |
| Planogram vision | **Won’t** for v1 |

## Family → Team (prod contract)

```
GET  /v1/retailer/family-members
     → { members, family_writes: "open"|"gone", migrate }
POST /v1/retailer/family-members
     body: { name|nickname, phone }
     → 201 | 410 family_writes_gone
POST /v1/retailer/family-members/migrate-to-team
     body: { retailer_role?: "RECEIVER", deactivate_login?, keep_family_rows? }
     → FamilyMigrateResult (temp_password once per migrated row)
```

| Rule | Detail |
|------|--------|
| Authz | `staff.manage` (OWNER/ADMIN/MANAGER) |
| Side effects | Auto-enables TEAM pack; creates org staff; bcrypt temp password |
| Policy | After migrate, family POST → **410**; GET reports `family_writes=gone` |
| Mobile name | Accepts `name` or `nickname` |
| Clients | Desktop Settings → Family; Android Family; iOS Family |

## Auto-order worker draft (prod contract)

```
POST /v1/retailer/settings/auto-order/run?mode=draft|place
GET  /v1/retailer/settings/auto-order/runs
     → { items: AutoOrderRun[] }  // last 20, newest first
```

| Rule | Detail |
|------|--------|
| Authz | `order.place` |
| Enablement | Global or any scoped override must be on |
| Candidates | Test seed → else AI prediction line items aggregated by SKU (+ supplier_id) |
| Idempotency | In-process `retailer\|day\|sku` (pod-local; multi-pod may double-draft rare) |
| Cart | Upserts cart lines when `cartRepo` + supplier present |
| place mode | Still drafts only; message `drafted_place_mode_pending_order_create` |
| Events | `RETAILER_AUTO_ORDER_UPDATED` + owner notification on draft success |
| Clients | Desktop / Android / iOS **Run auto-order now** + **Last runs** |

## Explicitly deferred

### Offline POS

Online-required COMPLETED sales remain. Future: offline counts + parked carts first, not offline card/cash complete sales.

### Planogram vision

SECTIONS aisle/shelf string tags are enough for v1. Photo/CV shelf compliance is v2+.

### Durable multi-pod auto-order audit

Run history + bucketDone are in-process RAM. Fine for draft v1; Spanner run table is a follow-up if multi-pod audit is required.

## Remaining ops to “fully closed” on SSMR

1. Cloud Build + roll `backend-go` / worker with this code  
2. Smoke: migrate-to-team + auto-order/run ≠ 404  
3. Owner: Global Pay + Firebase SMS (prod flip blockers, not Retail OS code)

## Test gate

```bash
cd pegasusX/apps/backend-go
go test ./retailer/ -run 'Family|AutoOrder' -count=1
```
