# Warehouse Analytics — Native vs Portal Parity Audit

**Scope:** pegasusX `warehouse-portal` · `warehouse-app-android` · `warehouse-app-ios`  
**Date:** 2026-06-14 (updated)  
**Parent:** `WAREHOUSE_PHASE.md` · `VEGETABLE_PLAN.md` §2.2

---

## Executive summary

| Surface | Analytics | Demand forecast | Replenishment insights |
|---------|-----------|-----------------|------------------------|
| **warehouse-portal** | Reference UI | Reference UI + insights fallback | Reference + OPEN-gated actions |
| **warehouse-app-android** | **Parity** — daily chart, 30d default, import cards | **Ahead** (Series tab) | **Parity** |
| **warehouse-app-ios** | **Parity** — daily chart, 30d default, import cards | **Ahead** (Series tab) | **Parity** |

**Verdict:** WH-4 (P0/P1) and WH-5 (P2 UI depth) are **CLOSED** across all three warehouse surfaces. `daily_breakdown` is live on the wire; portal Recharts and native bar charts consume it.

**SSMR:** `PX_E2E_WAREHOUSE_ANALYTICS_OK` exercises `GET /v1/warehouse/ops/analytics?period=7d|30d` and decodes `daily_breakdown` + import card shapes.

---

## 1. Analytics (`GET /v1/warehouse/ops/analytics?period=`)

### Portal (`app/analytics/page.tsx`)

| UI block | API fields consumed |
|----------|---------------------|
| Period toggle 7d / 30d (default **30d**) | `period` |
| KPI: Revenue, Orders, Avg order, Fleet % | `total_revenue`, `total_orders`, `avg_order_value`, `fleet_utilization` / `fleet_utilization_pct` |
| Import freshness card | `import_freshness.*` |
| Import anomaly queue card | `import_anomaly_queue.*` |
| Daily revenue bar chart (Recharts) | `daily_breakdown` or `daily` |
| Top products table | `top_products[]` |

### Android (`AnalyticsScreen.kt`)

| UI block | Status vs portal |
|----------|------------------|
| Period 7d / 30d (default **30d**) | **Parity** |
| KPI grid + import freshness/anomaly | **Parity** |
| Top products list | **Parity** |
| Daily chart | **Parity** — `chartDaily` |
| Import meta cards (last session / anomaly detail) | **Parity** |

### iOS (`AnalyticsView.swift`)

| UI block | Status vs portal |
|----------|------------------|
| Period segmented 7d / 30d (default **30d**) | **Parity** |
| KPI grid + import cards | **Parity** |
| Daily chart | **Parity** — `chartDaily` |
| Pull-to-refresh | **Ahead** |

### Backend reality (`warehouse/analytics_spanner.go`)

| Field | Populated today? | Notes |
|-------|------------------|-------|
| `total_orders`, `total_revenue`, `completed_orders`, `cancelled_orders` | **Yes** | Period-filtered |
| `period` filter | **Yes** | 7d / 30d |
| `top_products` | **Yes** | From completed order line items |
| `daily_breakdown` / `daily` | **Yes** | Grouped revenue by day |
| `fleet_utilization` | **Yes** | Home-node drivers + active assignments |
| `import_freshness` | **Partial** | Proxy from `SupplierInventoryV2` (not session-based) |
| `import_anomaly_queue` | **Read path live** | Scans `SupplierImportStagedRows`. **Write path live** via `POST /v1/supplier/inventory/import` → `inventory_import_staging.go`. Counts reflect staged `validation_errors` (warehouse-scoped). `import_freshness` remains `SupplierInventoryV2` proxy. |

---

## 2. Demand forecast & replenishment (out of analytics scope)

Demand forecast and replenishment parity are tracked in `WAREHOUSE_PHASE.md` (WH-1, WH-2). Native **Series** tab and insights fallback are **ahead** of portal; approve/dismiss is **parity** on all three surfaces.

---

## 3. Prioritized remediation — **CLOSED**

### P0 — Backend — **CLOSED**

| ID | Item | Status |
|----|------|--------|
| WH-A1 | Honor `period` in analytics Spanner query | **Done** |
| WH-A2 | Populate `top_products`, `daily_breakdown`, `fleet_utilization`, import fields | **Done** |
| WH-A3 | SSMR `PX_E2E_WAREHOUSE_ANALYTICS_OK` | **Done** |

### P1 — Native correctness — **CLOSED**

| ID | Item | Android | iOS |
|----|------|---------|-----|
| WH-N1 | Gate approve/dismiss on `status == "OPEN"` | **Done** | **Done** |
| WH-N2 | Surface `transfer_id` from approve response | **Done** | **Done** |
| WH-N3 | Demand forecast insights fallback | **Done** | **Done** |

### P2 — UI depth — **CLOSED**

| ID | Item | Status |
|----|------|--------|
| WH-U1 | Daily revenue chart (portal + native) | **Done** — `daily_breakdown` on wire |
| WH-U2 | Full import freshness / anomaly cards | **Done** — DDL + `loadAnalyticsImportAnomalyQueue` |
| WH-U3 | Default analytics period 30d | **Done** — all three surfaces |

---

## 4. Cross-role sync checklist

| Check | Status |
|-------|--------|
| API client knows endpoints | **Green** |
| Shared types aligned | **Green** |
| Spanner authority for insights | **Green** |
| Replenishment actions on native | **Green** |
| Analytics E2E marker | **Green** |
| Backend analytics completeness | **Green** — anomaly read + CSV import staging write path |

---

## Verification commands

```bash
curl -H "Cookie: …" "$BASE/v1/warehouse/ops/analytics?period=30d"
cd pegasusX && make test-ssmr-infra   # PX_E2E_WAREHOUSE_ANALYTICS_OK
```
