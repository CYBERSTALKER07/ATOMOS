# Retailer sell-through flywheel (Next Layer L3)

**Status:** L3.1–L3.2 backend shipped (2026-08-02)  
**Depends:** Retail OS P3 STORE_STOCK + P4 POS  

## Purpose

POS floor sales and voids drive **daily sell-through rollups** and a **`SELL_THROUGH` DemandAdjustments factor**, so reorder suggestions can prefer real store velocity over wholesale history alone.

Warehouse / factory apps do **not** see POS internals — only Kafka events if consumed.

## Wire

### Written on

| Event | Effect |
|-------|--------|
| POS sale COMPLETED | `QtySold += line.qty`; factor `SELL_THROUGH += qty` |
| POS sale VOIDED | `QtyVoided += line.qty`; factor `SELL_THROUGH -= qty` |

Also updates `ReorderSuggestions.CurrentStock` via existing `syncReorderCurrentStock`.

### Table

`RetailerSellThroughDaily` — PK `(RetailerId, LocationId, SkuId, Day)`  
Migration: `schema/migrations/20260802_retail_os_sell_through.ddl`

### API

```
GET /v1/retailer/insights/sell-through?location_id=&from=YYYY-MM-DD&to=YYYY-MM-DD
```

Auth: `pos.sell` | `stock.view` | `reports.view` | `cap.manage`

Response:

```json
{
  "source": "STORE_POS",
  "items": [
    {
      "retailer_id": "…",
      "location_id": "…",
      "sku_id": "SKU-TEA",
      "day": "2026-08-02",
      "qty_sold": 5,
      "qty_voided": 0,
      "net_sold": 5,
      "qty_on_hand_eod": 95,
      "source": "STORE_POS"
    }
  ]
}
```

### Events

`RETAILER_SELL_THROUGH_UPDATED` on sale/void rollup.

### DemandAdjustments

Factor key **`SELL_THROUGH`** (additive units sold net of voids for the calendar day).  
`AdjustedDemand = BaseVelocity + sum(factors)`.

## L3.3 worker merge (shipped)

`replenishment.ReorderSuggestionWorker.RunBatch`:

1. Load today's `DemandAdjustments` (+ `BaseVelocity`, `FactorsJson`)  
2. Strip same-day `SELL_THROUGH` factor from `AdjustedDemand` (avoid double-count with 7d rollup)  
3. Sum last **7 days** `RetailerSellThroughDaily` net sold (org-wide per SKU)  
4. `demand = max(base, sell_through_units/7)` via `MergeDemandVelocities`  
5. Persist suggestion + `SourcesJson`, `SellThroughVel`, `BaseDemand`  
6. Supplier list: `sources[]`, `sell_through_velocity`, `base_demand_per_day`

Migration: `20260802_retail_os_sell_through_sources.ddl`

```bash
cd apps/backend-go && go test ./replenishment/ -count=1
```

## L3.4 UI source chips (enterprise)

| Surface | Where |
|---------|--------|
| Shared | `@pegasusx/ui-kit/portal` `DemandSourceChips` |
| Types | `ReorderSuggestionRow.sources` in `@pegasusx/types` |
| Supplier | Portal `/replenishment/suggestions` — chips + POS/base velocity columns + filter |
| Retailer desktop | Auto-order suggestions (shared chips); Insights sell-through panel |
| API | `GET /v1/retailer/reorder-suggestions` + insights sell-through |
| Mobile | Retailer Android + iOS Auto-order: suggestions list + `DemandSourceChips` + Place now (confirm) |

## L3.5 AutoOrderWorker ← suggestions (shipped)

Candidate priority: **seed → OPEN ReorderSuggestions → AI predictions**.

- `SuggestedQty` becomes draft cart qty  
- Skips `local:` SKUs  
- Run audit field `candidate_source`: `seed` \| `reorder_suggestions` \| `ai_predictions`  
- Supplier ID: last order for SKU, else first favorite  

## Auto-order place mode (enterprise)

| Piece | Detail |
|-------|--------|
| Modes | `draft` (default) \| `place` |
| Settings | `execution_mode` on auto-order settings; PATCH `/global` |
| Run | `POST /v1/retailer/settings/auto-order/run?mode=place` |
| Authz | `order.place` + OWNER/ADMIN/MANAGER |
| Process flag | `AUTO_ORDER_PLACE_ENABLED=true` (default off) |
| Path | `OrderCreator` → `order.Service.Create` with `order_source=AUTO_ORDER` + optional `supplier_id` |
| Geo | Primary location must have 15-char H3 + lat/lng in prod; memory tests use fallback |
| Idempotency | In-process bucket + Spanner `RetailerAutoOrderBucket` (`20260802_retail_os_auto_order_bucket.ddl`) |
| Run audit | Dual-write memory + Spanner `RetailerAutoOrderRuns` (`20260809_retailer_auto_order_runs.ddl`) |
| Source filter | `?source=STORE_POS\|WHOLESALE_HISTORY` on supplier + retailer suggestion lists |
| Desktop | Draft now / Place now + confirm modal + `placed_orders` audit |
| Mobile | Same place confirm UX on Android/iOS Auto-order |

### Ops enable checklist

1. Apply sell-through + sources + auto_order_bucket DDL  
2. Retailer primary location has delivery geo  
3. Set `AUTO_ORDER_PLACE_ENABLED=true` on API pods  
4. Desktop: confirm Place now → order appears supplier-side  
5. Second place same day → `already_processed_bucket`  

## Not yet

- Supplier mobile suggestions surface  
- Offline place  


## Quantity negotiations — product-disabled

Delivery-time driver propose → supplier resolve is **gated off** and is **not** a substitute for:

| Path | When | What it does |
|------|------|----------------|
| **Quantity negotiations (off)** | During delivery, before settle | Change order line qty / payment basis |
| Claims / OS&D | After delivery | Damage/shortage → reverse logistics / quarantine |
| Shop-closed | At stop, retailer unavailable | Exception + resolve path |
| Missing-items / partial offload | Doorstep variance | Different exception codes |

| Piece | Disabled behavior |
|-------|-------------------|
| Gate | `quantityNegotiationDisabled = true` |
| Propose / resolve | `410 feature_disabled` |
| Pending list | Empty `[]` |
| Sweeper | Not started |
| Clients | Stub / empty-state only |
| E2E | `PX_E2E_NEGOTIATION_SKIPPED` |

Implementation remains in `order/negotiation*.go` for a future product re-enable only.

## B4 DEMAND_SIGNAL Kafka (2026-08-02)

Flywheel broadcast for **suppliers** — distinct from planning `DemandSignals` Spanner rows (weather/holiday/promo multipliers via REST `/v1/demand/signals`).

| Piece | Detail |
|-------|--------|
| Event type | `DEMAND_SIGNAL` |
| Payload | `{retailer_id, sku, day, qty_delta, net_sold, source:STORE_POS, kind:sale\|void, supplier_id?}` — no POS session/tender |
| Emit | Per non-`local:` line on sell-through sale/void |
| Topic main | Always written to `KAFKA_TOPIC_MAIN` (default `pegasusx-main`) via outbox |
| Topic demand | `KAFKA_TOPIC_DEMAND` (default `pegasusx-demand`) when `KAFKA_TOPIC_DUAL_WRITE=true` |
| Aggregate | `DemandSignal` / key `retailer\|sku\|day` |
| Guards | `local:` SKUs never emit |
| Contract | `events.DemandSignalEvent`, `contracts/events.schema.json` → `DemandSignalPayload` |

### Planning DemandSignals vs flywheel DEMAND_SIGNAL

| | Planning REST | Flywheel Kafka |
|--|---------------|----------------|
| Storage | Spanner `DemandSignals` | Outbox → Kafka |
| Purpose | Multiplier drivers (HOLIDAY, WEATHER, PROMO) | STORE_POS qty sold/voided |
| API | `GET/POST /v1/demand/signals` | Subscribe `DEMAND_SIGNAL` |
| Supplier UI | `/analytics/demand/signals` form | Consumer/badge later |

### Ops

1. Create Kafka topic `pegasusx-demand` (or env override)  
2. Set `KAFKA_TOPIC_DUAL_WRITE=true` for dual publish  
3. Apply DDL `20260809_flywheel_demand_feed.ddl`  
4. Supplier consumer: filter `type=DEMAND_SIGNAL`, optional `supplier_id`  

### B4.4 Supplier feed UI (2026-08-02)

| Piece | Detail |
|-------|--------|
| Table | `FlywheelDemandFeed` (SignalId PK; index SupplierId+CreatedAt) |
| Write | On sell-through emit, separate Spanner Apply after Kafka outbox |
| API | `GET /v1/supplier/analytics/demand/flywheel?days=7&limit=100&sku=` |
| Portal | `/analytics/demand/flywheel` — “POS flywheel demand” |
| Nav | Analytics → POS flywheel (alongside Demand Forecast + Demand Signals) |
| Scope | Rows filtered to `SupplierId` of session (best-effort from Products/orders at emit) |

Honest empty when no POS yet or DDL not applied (`feed_error: unavailable`).


## Wave A honesty (2026-08-02)

- Control Tower simulator: `CONTROL_TOWER_SIMULATOR_ENABLED=true` only (default off)  
- Family writes gone: durable `RetailerOrgFlags` after migrate-to-team  
- Notif inbox dual-write: org id + staff user subjects  

## B3 Local POS catalog (2026-08-02)

Retailer-owned non-Pegasus goods for POS between wholesales. **Never** feed supplier reorder / DemandAdjustments.

| Piece | Detail |
|-------|--------|
| Table | `RetailerLocalCatalog` PK `(RetailerId, LocalSkuId)` + barcode index |
| DDL | `schema/migrations/20260809_retailer_local_catalog.ddl` (applied SSMR) |
| Namespace | IDs always `local:…` via `NormalizeLocalSKUID` / `IsLocalSKU` |
| API | `GET/POST /v1/retailer/local-skus`, `PATCH …/local-skus/{id}`, `GET /v1/retailer/pos/catalog?q=` |
| POS | Sale lines resolve barcode → catalog; free-form `local:` kept namespaced |
| Guards | Sell-through rollup skips `local:`; auto-order + reorder suggestions skip `local:` |
| Desktop | `/stock/local-skus` — list, quick-add, activate/deactivate |
| Memory | In-process `localCatalog` when Spanner unset (tests / local) |


## Tests

```bash
cd apps/backend-go && go test ./retailer/ -run 'SellThrough|AutoOrder' -count=1
cd apps/backend-go && go test ./replenishment/ -run 'MergeDemand|ComputeSuggested' -count=1
```
