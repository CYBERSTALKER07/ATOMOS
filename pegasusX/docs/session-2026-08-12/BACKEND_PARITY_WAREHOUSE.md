# Backend Parity — A4 WAREHOUSE (Class A audit)

**Agent:** A4-WAREHOUSE  
**Date:** 2026-08-12  
**Tree:** `pegasusX` only  
**Phase:** AUDIT ONLY (no code changes)  
**Protocol:** [`BACKEND_PARITY_PROTOCOL.md`](./BACKEND_PARITY_PROTOCOL.md)  
**Clients (later UI):** warehouse-portal, warehouse-android, warehouse-ios  
**SoT clients:** [`../FEATURES_BY_APP_ROLE.md`](../FEATURES_BY_APP_ROLE.md) §3, [`../ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md)

---

## Scope packages

| Package | Role | Notes |
|---------|------|-------|
| `warehouseroutes` | HTTP mount | Primary WH surface; `RequireRole(WAREHOUSE\|WAREHOUSE_ADMIN\|ADMIN)` + **`RequireWarehouseOpsScope`** |
| `warehouse` | Service + ops + dispatch WH side | Spanner + outbox on many paths; portal seed fallbacks |
| `stocklots` | WMS bins/lots/pick/cycle/cold | **No outbox package usage at all** |
| `returns` + `returnsroutes` | Reverse-logistics inbound gate | Shared with PAYLOAD; confirm has outbox |
| `creditnoteroutes` reverse-logistics | Credit-note reverse tasks | Role gate = **WAREHOUSE only** (not ADMIN) |
| `order` (warehouse handlers) | delay/reject/overflow/preorder/return-policy | Mounted under warehouseroutes |
| `payload` (reassign/labels) | Reassign + ship-units | Mounted under warehouseroutes |
| `dispatch` | Shared planner | Called from warehouse dispatch execute/preview |
| `inventory` | Shared stock credit helper | **Not WH-owned HTTP**; used by returns/creditnote/warehouse receive |
| `warehouseops` | Alias facade | `Service = warehouse.Service` |
| `laborcapacityroutes` | `/v1/labor-capacity/*` | Supplier+warehouse JWT; **no home-node pin middleware** |

**Scope middleware naming (important):**  
Warehouse **ops** staff are pinned by `auth.RequireWarehouseOpsScope` (`auth/warehouse_ops_scope.go:26–80`), not by `RequireWarehouseScope`.  
`RequireWarehouseScope` (`auth/warehouse_scope.go:32–127`) is the **supplier-portal** warehouse filter (ADMIN + WAREHOUSE_ADMIN home-node / FACTORY_ADMIN linked warehouses). Protocol checklist item “home-node scope RequireWarehouseScope” maps operationally to **OpsScope for WH role row**.

---

## 1. Feature inventory (route → service → Class A)

Class A columns: Auth | Idem | Spanner RW | Outbox same-txn | Cache invalidate | Realtime (hub/FCM via dispatcher) | Edge | Tests  
Legend: ✅ pass · ⚠️ partial · ❌ fail · — n/a (read)

### 1.1 Auth / setup

| Method | Route | Handler | Auth | Idem | RW | Outbox | Cache | RT | Notes | Class A |
|--------|-------|---------|------|------|----|--------|-------|----|-------|---------|
| POST | `/v1/auth/warehouse/login` | `HandleWarehouseLogin` | public | — | — | — | — | — | JWT mint | n/a |
| POST | `/v1/auth/warehouse/register` | `HandleWarehouseRegister` | public | — | ⚠️ | ⚠️ | — | — | setup path | partial |
| POST | `/v1/auth/warehouse/refresh` | `HandleWarehouseRefresh` | public | — | — | — | — | — | | n/a |
| POST | `/v1/warehouse/setup` | `HandleWarehouseSetup` | public | ⚠️ | ⚠️ | ⚠️ | — | — | bootstrap | partial |
| GET | `/v1/warehouse/ws-session` | `WSSessionHandler` | role | — | — | — | — | hub | `warehouseroutes/routes.go:180–183` | n/a |

### 1.2 Ecosystem CRUD / transfers / supply

| Method | Route | Handler | Auth | Idem | RW | Outbox | Cache | RT | Class A |
|--------|-------|---------|------|------|----|--------|-------|----|---------|
| POST | `/v1/warehouses` | `HandleCreateWarehouse` | ops+role | ⚠️ optional | ✅ | ✅ EmitJSON | ✅ | via Kafka | ⚠️ |
| PUT | `/v1/warehouses/{id}` | `HandleUpdateWarehouse` | ops+role | ⚠️ | ✅ | ✅ | ✅ | via Kafka | ⚠️ |
| GET | `/v1/warehouses*` | list/get | ops+role | — | — | — | — | — | — |
| POST | `/v1/warehouse/transfers/emergency` | `HandleEmergencyTransfer` | ops+role | ✅ guard | ✅ | ✅ TRANSFER_CREATED | ⚠️ | Kafka+hub path | ⚠️ |
| POST | `/v1/warehouse/transfers/force-receive` | `HandleForceReceive` | ops+role | ✅ | ✅ | ✅ | ⚠️ | Kafka | ⚠️ |
| POST | `/v1/warehouse/transfers/{id}/receive` | `HandleReceiveTransfer` | ops+role | ✅ | ✅ | ✅ RECEIVED | ⚠️ | Kafka | ⚠️ |
| GET/POST | `/v1/warehouse/supply-requests` | `HandleSupplyRequests` | ops+role | ⚠️ | ✅ | ✅ OPENED | ✅ | Kafka | ⚠️ |
| PATCH | `/v1/warehouse/supply-requests/*` | `HandleSupplyRequestByID` | ops+role | ⚠️ | ✅ | ✅ UPDATE | ✅ | Kafka | ⚠️ |
| POST | replenishment insight action | `HandleReplenishmentInsightAction` | insights scope | ⚠️ | ✅ | ✅ | ⚠️ | ⚠️ | ⚠️ |

Evidence: transfers `warehouse/transfers.go:89–97`, `:218–261`; supply `warehouse/service.go:445–494`, `supply_request_body.go:171–172`; CRUD `crud_handlers.go:58–59`, `:137–138`.

### 1.3 Inventory (V2 absolute) + policy + settings

| Method | Route | Handler | Auth | Idem | RW | Outbox | Cache | RT | Class A |
|--------|-------|---------|------|------|----|--------|-------|----|---------|
| GET/PATCH | `/v1/warehouse/ops/inventory` | `HandleInventory` | ops | PATCH: optional guard | ✅ | **❌ emit nil** | ❌ | ❌ | **FAIL** |
| PATCH | `/v1/warehouse/ops/inventory/{productID}/policy` | `HandleInventoryPolicy` | ops | optional guard | ✅ | **❌ emit nil** | ❌ | ❌ | **FAIL** |
| GET/PATCH | `/v1/warehouse/ops/settings` | `HandleOpsSettings` | ops | optional guard | ✅ Spanner UpdateMap | **❌ none** | ❌ | ❌ | **FAIL** |
| GET/PATCH | `/v1/warehouse/ops/location` | `HandleOpsLocation` | ops | optional guard | ✅ | ✅ LOCATION_UPDATED | ✅ plan cache | Kafka hub | ✅/⚠️ |
| GET/PATCH | `/v1/warehouse/ops/dispatch/settings` | `HandleDispatchSettings` | ops | **❌ no key require** | ✅ | ✅ SETTINGS_UPDATED | ⚠️ | hub direct + outbox | ⚠️ |

**Silent stock write (P0):**

```340:342:apps/backend-go/warehouse/ops_portal.go
		err := s.repo.UpdateInventoryQuantity(r.Context(), whID, patchBody.ProductID, patchBody.Quantity, func(buf outbox.TxnBuffer) error {
			return nil
		})
```

**Silent policy write:**

```73:75:apps/backend-go/warehouse/ops_inventory_policy.go
	err := s.repo.UpdateInventoryPolicy(r.Context(), whID, productID, policy, req.ReorderThreshold, func(buf outbox.TxnBuffer) error {
		return nil
	})
```

Repo still supports outbox in-txn (`repository_spanner.go:651–709`) — callers deliberately pass no-op emit.  
When `WMS_LOTS_ENABLED`, absolute set is rejected (`repository_spanner.go:652–654`) — good fail-closed for lots mode; non-lots mode remains silent.

**Settings silent Spanner:**

```195:197:apps/backend-go/warehouse/ops_settings.go
	_, err := s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("Warehouses", update)})
	})
```

### 1.4 WMS — bins / lots / pick / cycle / cold (`stocklots`)

Mounted: `warehouseroutes/routes.go:57–84`. Handler uses `auth.EffectiveWarehouseID` (`stocklots/handlers.go:28–33`) under OpsScope middleware for protected group.

| Method | Route | Mutates stock? | Idem | Outbox | Hub/RT | Class A |
|--------|-------|----------------|------|--------|--------|---------|
| POST | `/ops/bins` | locations only | ❌ | ❌ | ❌ | **FAIL** (silent layout) |
| PATCH | `/ops/bins/{locationID}` | locations | ❌ | ❌ | ❌ | **FAIL** |
| POST | `/ops/lots/putaway` | **StockLots + V2 rollup** | client lot_id only | **❌** | **❌** | **FAIL P0** |
| POST | `/ops/pick-waves` | reservations + lots reserved | ❌ | **❌** | **❌** | **FAIL P0** |
| POST | `.../tasks/{taskID}/confirm` | **QoH/reserved pick** | ❌ | **❌** | **❌** | **FAIL P0** |
| POST | `.../waive-shorts` | task status | ❌ | ❌ | ❌ | FAIL |
| POST | `/ops/cycle-counts` | count rows | ❌ | ❌ | ❌ | FAIL |
| POST | `.../submit` | count + PENDING adj | ❌ | ❌ | ❌ | FAIL |
| POST | `/ops/inventory-adjustments/{id}/approve` | **lot delta + V2** | ❌ | **❌** | **❌** | **FAIL P0** |
| POST | `/ops/inventory-adjustments/{id}/reject` | adj status | ❌ | ❌ | ❌ | FAIL |
| POST | `/ops/cycle-counts/enqueue-abc` | count rows | ❌ | ❌ | ❌ | FAIL |
| POST | `/ops/temperature-readings` | readings; may **quarantine lots** | ❌ | **❌** | **❌** | **FAIL P0** |
| GET | lots/waves/counts/accuracy/reconcile | read | — | — | — | — |

**Package-level outbox grep: zero matches in `stocklots/`.** All mutators use `spannerutils.RunReadWriteTransaction` + `BufferWrite` only (e.g. putaway `lots.go:41–177`, pick confirm `picking.go:283–363`, adjust `counting.go:193–317`, temp `coldchain.go` quarantine `227–275`).

**IDOR / resource-scope holes (P0/P1):**

| Handler | Issue | Evidence |
|---------|-------|----------|
| `HandleLotByID` | No warehouse check on lot | `handlers.go:174–191` |
| `HandlePickWaveByID` | No warehouse check | `handlers.go:308–324` |
| `HandleConfirmPickTask` | No wave warehouse vs ops check | `handlers.go:327–358` |
| `HandleApprove/RejectInventoryAdjustment` | No warehouse match on adj | `handlers.go:530–583` + `counting.go:193–209` |
| `HandleCycleCountByID` / submit | No warehouse match on count | `handlers.go:444–500` |
| Temperature GET/POST | manifest-scoped; no ops warehouse assert | `handlers.go:662–713` |

### 1.5 Dispatch (warehouse side)

| Method | Route | Handler | Auth | Idem | RW | Outbox | Cache | RT | Class A |
|--------|-------|---------|------|------|----|--------|-------|----|---------|
| GET/POST | `/ops/dispatch/preview` | `HandleDispatchPreview` | ops | — | — | — | plan cache | — | — |
| POST | `/ops/dispatch/execute` | `HandleDispatchExecute` | ops | **✅ required key** | ✅ CommitSupplier | ✅ Order/Route/Manifest + SplitShipment | plan invalidation path | Kafka → hub | **✅** (strongest WH mutator) |
| POST | rescue preview/propose | rescue handlers | ops | ⚠️ | ✅ propose | ⚠️ InsertMap OutboxEvents | ⚠️ | ⚠️ | ⚠️ |
| GET | tracking / runs | read | ops | — | — | — | — | — | — |
| GET/POST/DELETE | dispatch-lock(s) | `HandleDispatchLock*` | ops | ✅ guard | ✅ | ✅ lock + freeze topics | ✅ | Kafka | ✅/⚠️ |

Evidence: idem require `ops_dispatch_handlers.go:214–216`; commit+outbox `dispatch_execute.go:324–377`; freeze dual-topic `service.go:957–1001`.

Also: supplier portal mounts **same** execute/preview under `/v1/supplier/dispatch/*` with **`RequireWarehouseScope`** (`supplierroutes/routes.go:73–114`) — dual entry, one service.

### 1.6 Orders / preorders / reassign

| Method | Route | Package | Auth | Idem | Outbox | Class A |
|--------|-------|---------|------|------|--------|---------|
| POST | `/ops/orders/{id}/delay` | order | ops | ✅ guard | ✅ ORDER_STATUS_CHANGED | ✅ |
| POST | `/ops/orders/{id}/reject` | order | ops | ✅ | ✅ | ✅ |
| POST | `/ops/orders/{id}/overflow` | order | ops | ✅ | ✅ | ✅ |
| POST | propose-delivery | order | ops | ⚠️ | ⚠️ | ⚠️ |
| POST | preorder edit/reject | order | ops | ⚠️ | ⚠️ | ⚠️ |
| GET/PUT | `/return-policy` | order | **WarehouseAdmin/Admin only** | ⚠️ | ⚠️ | ⚠️ |
| POST | `/reassign-order` | payload | ops | ⚠️ | ⚠️ (payload Class A) | ⚠️ |

Evidence: `order/warehouse_ops.go:135–154` (outbox in `warehouseTransition`); handlers `:401–521`.

### 1.7 Fleet / staff / broadcast / payment

| Method | Route | Outbox | Idem | Notes | Class A |
|--------|-------|--------|------|-------|---------|
| POST/PATCH | drivers/vehicles | ✅ Driver/Vehicle events (`fleet_ops.go`) | ✅ guard | | ⚠️ |
| POST | `/ops/staff` | **❌** | optional | `Apply` insert SupplierUsers only `ops_portal.go:592–602` | **FAIL silent** |
| POST | broadcast / templates | hub direct + ⚠️ | guard | `ops_broadcast.go:451–452` hub; may skip full bus | ⚠️ |
| POST | payment-config | ⚠️ | optional key | | ⚠️ |

### 1.8 Reverse logistics

#### A) returnsroutes (inbound gate)

| Method | Route | Auth middleware | Scope | Idem | Outbox | Cache | Class A |
|--------|-------|-----------------|-------|------|--------|-------|---------|
| GET | `/v1/returns/inbound` | Role only (no OpsScope) | home-node in handler | — | — | — | — |
| POST | `/v1/returns/inbound/sessions` | Role | home-node | ❌ | **❌ silent session** | ❌ | FAIL |
| POST | `/v1/returns/inbound/scan` | Role | home-node + row WH | optional | **❌ silent ReceivedQty** | ❌ | **FAIL P1** |
| POST | `/v1/returns/inbound/confirm` | Role | home-node | optional | **✅ RETURN_RECEIVED** + stock credit | ✅ inventory key | ⚠️ (idem optional) |
| GET | history / barcode | Role | scoped | — | — | — | — |

Confirm outbox: `returns/inbound.go:540–565`. Scan write without outbox: `:371–378`.  
`resolveGateWarehouseID`: `inbound.go:778–794` (ops scope OR claims home node).  
**returnsroutes does not mount `RequireWarehouseOpsScope`** (`returnsroutes/routes.go:32–44`).

#### B) creditnote reverse-logistics

| Method | Route | Auth | Scope | Outbox | Stock | Class A |
|--------|-------|------|-------|--------|-------|---------|
| GET | `/v1/warehouse/reverse-logistics` | **RoleWarehouse only** | home-node vs query ✅ | — | — | ⚠️ role split |
| POST | `.../{taskId}/receive` | **RoleWarehouse only** | **body warehouse_id; NO home-node check** | ✅ EventReverseLogisticsReceived | putaway/V2 in txn | **FAIL P0 IDOR** |

Evidence:

```123:147:apps/backend-go/creditnote/handlers.go
func (h *Handlers) HandleReceiveReverse(...) {
	// RoleWarehouse only — WAREHOUSE_ADMIN rejected
	// body.WarehouseID used with no claims.HomeNodeID equality check
	h.Svc.ReceiveReverseTask(r.Context(), taskID, wh, body.ReceivedQty, claims.Subject)
}
```

List path correctly pins home node (`handlers.go:223–228`). Receive path does not.  
Outbox+stock: `creditnote/repository_spanner.go:534–597`.

### 1.9 Reads / analytics / pulse (non-Class-A)

Dashboard, board, exceptions, manifests, CRM, treasury, financials, analytics, demand forecast, stock-commitments, fleet live-map, pulse (`pulseroutes`), order receipt (`orderroutes`) — mostly GET; auth via OpsScope where under warehouseroutes. Not stock state machines.

### 1.10 Shared helpers (not WH HTTP-owned)

| Package | Use | Outbox |
|---------|-----|--------|
| `inventory.CreditSupplierInventoryV2InTxn` | returns/creditnote/receive when lots off | caller-owned |
| `inventory.Service.AdjustStock` | generic InventoryLevels | **no outbox** — library risk if exposed |
| `laborcapacityroutes` | WH clients call `/v1/labor-capacity/*` | availability POST: no outbox, no warehouse pin |

---

## 2. WarehouseHub & realtime plane

| Component | Location | Status |
|-----------|----------|--------|
| Hub construct | `bootstrap/bootstrap.go:701` `ws.NewHub("warehouse", ...)` | ✅ |
| Injected into warehouse.Service | `warehouse/service.go:136,233` | ✅ |
| Injected into returns.Service | `returns/service.go:22,59` | ✅ |
| WS role routing | `ws/handler.go:275–276` Warehouse + WarehouseAdmin → warehouse + supplier + telemetry hubs | ✅ |
| Kafka fanout room | `kafka/notification_dispatcher.go:754–762` `warehouse:{id}` + FCM + inbox | ✅ |
| Direct hub (bypass bus) | broadcast `ops_broadcast.go:451–452`; supply transfer approach `supply_transfer_driver.go:111–120`; returns approach `lifecycle.go:169–173` | ⚠️ dual path |
| Warehouse Kafka consumer | `warehouse/consumer.go:33–36` only `SUPPLY_REQUEST_ACCEPTED` | narrow |

Declared dispatcher event families for WH:

| Event family | Dispatcher case | Room |
|--------------|-----------------|------|
| WAREHOUSE_CREATED / supply / lock / split / capacity | `notification_dispatcher.go:107–109`, `:188–192` | warehouse + supplier |
| WAREHOUSE_LOCATION_UPDATED | `:180–181` | warehouse |
| TRANSFER_* / SUPPLY_TRANSFER_* | `:188–190` | warehouse |
| RETURN_RECEIVED / driver approach | `:167–169`, `handleReturnGateEvent:787–807` | warehouse + payload room + supplier |
| ORDER_* from WH delay/reject | order handlers | multi-party |
| **WMS putaway/pick/adjust/temp** | **none declared** | **none** |
| **INVENTORY quantity/policy** | **none** | **none** |
| **OPS settings** | **none** | **none** |

---

## 3. Gaps (P0 / P1 / P2)

### P0 — money / stock / auth IDOR / silent state machine

| ID | Gap | Evidence | Proposed fix (audit only) |
|----|-----|----------|---------------------------|
| WH-P0-1 | **All stocklots stock mutations silent** (putaway, pick reserve/confirm, adjust approve, cold quarantine) — no OutboxEvents, no hub, no cache | package-wide no `outbox` import; e.g. `lots.go:122–160`, `picking.go:181–331`, `counting.go:209–316`, `coldchain.go:272–275` | Define `EventStockLotPutaway`, `EventPickTaskConfirmed`, `EventInventoryAdjusted`, `EventLotQuarantined`; emit in same RW txn; fanout warehouse room; invalidate `warehouse:inventory:{id}` |
| WH-P0-2 | **PATCH inventory quantity silent** (non-lots mode) | `ops_portal.go:340–342` emit nil; `repository_spanner.go:651–678` | Emit `STORE_STOCK_ADJUSTED` or WH inventory event; require idempotency key; broadcast + cache |
| WH-P0-3 | **Reverse-logistics receive IDOR** — body `warehouse_id` not pinned to home node; `WAREHOUSE_ADMIN` cannot call (role-only WAREHOUSE) | `creditnote/handlers.go:123–147` vs list scope `:223–228`; routes `creditnoteroutes/routes.go:30–31` | Pin `wh == claims.HomeNodeID`; allow WarehouseAdmin; reject body override |
| WH-P0-4 | **WMS resource IDOR** — confirm pick / approve adj / get lot by id without warehouse membership | `stocklots/handlers.go:174–191`, `:327–358`, `:530–555` | Load resource warehouse; assert `== EffectiveWarehouseID` |
| WH-P0-5 | **Inbound scan silent physical qty** | `returns/inbound.go:371–378` | Emit `RETURN_SCAN_PROGRESS` or defer until confirm only (document intentional) + optional WS progress |

### P1 — realtime / incomplete Class A / contract split

| ID | Gap | Evidence | Proposed fix |
|----|-----|----------|--------------|
| WH-P1-1 | Inventory policy silent | `ops_inventory_policy.go:73–75` | Outbox + retailer-visible policy fanout if needed |
| WH-P1-2 | Ops settings silent (OOS policy, express, fees) | `ops_settings.go:195–197` | Emit warehouse settings event; invalidate catalog/policy caches |
| WH-P1-3 | Staff create silent Spanner Apply | `ops_portal.go:592–602` | Outbox member-added + hub; never hardcode pin in prod response (`"pin":"5678"`) |
| WH-P1-4 | stocklots: zero idempotency guards on public mutators | handlers POST paths | Reuse warehouse `guardMutationReplay` or shared middleware |
| WH-P1-5 | returns routes lack OpsScope middleware; PAYLOAD shares gate | `returnsroutes/routes.go:32–44` | Mount `RequireWarehouseOpsScope` for WH roles; keep payload home-node |
| WH-P1-6 | Idempotency mostly **optional** except dispatch execute | only `requireMutationIdempotencyKey` on execute | Require keys on putaway, pick confirm, adjust approve, inbound confirm, transfers |
| WH-P1-7 | Dual reverse-logistics surfaces (returns inbound vs creditnote reverse-tasks) | FEATURES + both packages | Document single SoT; align role gates |
| WH-P1-8 | Reverse-logistics role excludes WAREHOUSE_ADMIN | `creditnoteroutes/routes.go:30–31` | Include RoleWarehouseAdmin |
| WH-P1-9 | `WAREHOUSE_DISPATCH_SETTINGS_UPDATED` not in events.go constants / may hit parity default | `ops_dispatch_handlers.go:332` string literal | Register constant + dispatcher case (ops event already partially covered via WarehouseEvent decode) |
| WH-P1-10 | Direct hub broadcast without outbox (approach, some broadcast) | `lifecycle.go:169–173` | Prefer outbox → Kafka → dispatcher for multi-pod |
| WH-P1-11 | labor-capacity POST no warehouse scope / outbox | `laborcapacityroutes/routes.go:22–30` | Pin home node; emit availability event |
| WH-P1-12 | Portal vs mobile product surface split (CT portal-primary; pick/cycle nested) | FEATURES §3 | Backend OK if same routes; ensure mobile uses same paths |

### P2 — polish / dead paths

| ID | Gap | Evidence |
|----|-----|----------|
| WH-P2-1 | Protocol name `RequireWarehouseScope` vs ops `RequireWarehouseOpsScope` | confuse audits |
| WH-P2-2 | Portal seed / memory fallbacks in ops reads | `ops_portal.go` ensurePortalSeed |
| WH-P2-3 | warehouse consumer only supply-accepted | `consumer.go:33–36` |
| WH-P2-4 | inventory package doc claims InventoryLevels; WH uses SupplierInventoryV2 | `inventory/doc.go` vs replenish.go |
| WH-P2-5 | Hardcoded staff PIN | `ops_portal.go:610` |

---

## 4. Event / consumer matrix (warehouse-relevant)

| Event | Producer (txn?) | Relay/Kafka | Consumer / fanout | WS room |
|-------|-----------------|-------------|-------------------|---------|
| WAREHOUSE_CREATED | warehouse CRUD ✅ | TopicMain | dispatcher handleWarehouseCreated | warehouse |
| WAREHOUSE_LOCATION_UPDATED | location_ops ✅ | TopicMain | handleWarehouseLocationUpdated | warehouse |
| WAREHOUSE_DISPATCH_LOCK_CHANGED + FREEZE_* | dispatch locks ✅ | TopicMain + FreezeLocks | operational event | warehouse |
| WAREHOUSE_SUPPLY_REQUEST_OPENED / SUPPLY_REQUEST_* | supply handlers ✅ | TopicMain | supply request + warehouse consumer (accepted) | warehouse + factory |
| WAREHOUSE_TRANSFER_* | transfers ✅ | TopicMain | handleTransferEvent | warehouse |
| MANIFEST_DRAFT_CREATED / ROUTE_CREATED / ORDER_ASSIGNED | dispatch execute ✅ | TopicMain | manifest/route/order handlers | multi |
| SPLIT_SHIPMENT_CREATED | dispatch execute ✅ | TopicMain | operational | warehouse |
| ORDER_STATUS_CHANGED | warehouse delay/reject/overflow ✅ | TopicMain | order event | multi |
| RETURN_RECEIVED_AT_WAREHOUSE | returns confirm ✅ | TopicMain | handleReturnGateEvent | warehouse+payload |
| DRIVER_RETURN_APPROACHING | lifecycle (often **direct hub**, not outbox) | optional | hubs direct | warehouse |
| REVERSE_LOGISTICS_RECEIVED (creditnote) | ReceiveReverse ✅ | TopicMain | ⚠️ verify dispatcher case | ? |
| **Stock lot / pick / cycle / temp quarantine** | **none** | — | — | — |
| **Inventory qty / policy / settings** | **none** | — | — | — |

Partner webhook allowlist includes `EventReturnReceivedAtWarehouse` (`partner/webhook_events.go:20`).

---

## 5. Edge-case matrix

| Scenario | Behavior | Gap? |
|----------|----------|------|
| Double dispatch execute | Idempotency key required + guard | Covered tests `ops_dispatch_test.go` |
| Double putaway | Optional client `lot_id`; no HTTP idem key | P1 |
| Double pick confirm | Spanner task status only; no idem | P1 race |
| Adjust approve twice | `adjustment_not_pending` | OK concurrency-ish |
| Lots mode absolute inventory PATCH | Fail-closed error | Good |
| Cross-warehouse query override | OpsScope rejects qs ≠ home node | ✅ `warehouse_ops_scope.go:58–63` |
| Cross-warehouse body on reverse receive | **Accepted** | P0 IDOR |
| Cross-warehouse pick task ID | **Accepted if task exists** | P0 IDOR |
| Cancel / concurrency on order reject | Status transition errors mapped | ✅ |
| Global ADMIN CEO on ops routes | OpsScope pass-through without pin | Intentional `warehouse_ops_scope.go:35–39` — service must resolve order WH |
| PAYLOAD on returns inbound | Allowed by role | Shared gate by design |
| WAREHOUSE_ADMIN on reverse-logistics creditnote | **403** | P1 contract |

---

## 6. One API contract: portal + mobile

Per FEATURES_BY_APP_ROLE §3 and ROLE_ROW matrix: clients share **same** `/v1/warehouse/*` and `/v1/returns/*` JWT routes.

| Surface | Portal | Android / iOS | Backend contract |
|---------|--------|---------------|------------------|
| Dispatch execute/preview | yes | yes | single warehouseroutes |
| Bins / pick-waves / cycle / cold | dedicated nav | Transfer Actions nested | **same** stocklots routes |
| Labor capacity | yes | yes | `/v1/labor-capacity` shared |
| Control tower | portal-primary | absent enums | not a backend split |
| Returns inbound | yes | yes | returnsroutes |
| Reverse logistics creditnote | yes | yes | role gate may break ADMIN tokens |
| Return policy | settings embed | RETURN_POLICY | GET/PUT same |

**Backend does not intentionally fork portal vs mobile URLs.** Gaps are **Class A completeness** and **auth role inconsistencies**, not dual OpenAPI.

---

## 7. Class A scorecard (summary)

| Domain | Overall | Bottleneck |
|--------|---------|------------|
| Dispatch execute + locks | **Strong** | Model for other mutators |
| Order delay/reject/overflow | **Strong** | |
| Transfers / supply / location | **Good** | |
| Fleet create | **Good** | |
| **WMS stocklots** | **Fail** | silent stock + no idem + resource scope |
| **Inventory PATCH/policy** | **Fail** | silent |
| **Ops settings / staff** | **Fail** | silent |
| Returns confirm | **Partial** | good outbox; scan silent; soft idem |
| Creditnote reverse receive | **Partial+IDOR** | outbox+stock OK; scope broken |
| WarehouseHub wiring | **Present** | starved by missing events |

**Definition of done (protocol):**  
JWT-scoped mutation → Spanner RW + in-txn outbox → relay → Kafka → consumer → WS hub/FCM — **not met for primary WMS stock paths**.

---

## 8. Proposed fix order (do not implement in audit)

1. **P0 stock event bus for stocklots** — putaway, pick confirm, adjustment approve, cold quarantine (same txn as lot write + V2 rollup).  
2. **P0 scope asserts** on all WMS by-id mutators + reverse receive home-node pin; include WAREHOUSE_ADMIN on reverse routes.  
3. **P0 inventory PATCH** — emit outbox or delete path when lots enabled (already blocked); when lots disabled emit + hub.  
4. **P1** require Idempotency-Key on all stock mutators; mount OpsScope on returns for WH roles.  
5. **P1** settings/policy/staff outbox + stop hardcoded PIN.  
6. **P2** align naming (OpsScope vs Scope), expand warehouse consumer if needed, document intentional scan-without-event.

---

## 9. Test evidence (existing)

| Area | Tests |
|------|-------|
| OpsScope / WarehouseScope | `auth/warehouse_scope_test.go` |
| Dispatch execute outbox | `warehouse/ops_dispatch_test.go` |
| Dispatch lock outbox | `warehouse/service_test.go` |
| Location / broadcast idem | `location_ops_test.go`, `ops_broadcast_test.go` |
| Returns inbound idem | `returns/inbound_idempotency_test.go` |
| Stocklots unit | fefo/picking/counting/coldchain tests (domain logic; **not Class A outbox**) |
| E2E stock policy | smokecheck `PX_E2E_WAREHOUSE_STOCK_POLICY_OK` |

**Missing tests:** outbox presence for putaway/pick/adjust; reverse receive home-node denial; cross-warehouse pick confirm denial.

---

## 10. File index (primary evidence)

| Path | Why |
|------|-----|
| `apps/backend-go/warehouseroutes/routes.go` | Full route inventory + OpsScope |
| `apps/backend-go/returnsroutes/routes.go` | Returns gate mount |
| `apps/backend-go/creditnoteroutes/routes.go` | Reverse logistics role split |
| `apps/backend-go/auth/warehouse_ops_scope.go` | Home-node pin for ops |
| `apps/backend-go/auth/warehouse_scope.go` | Supplier-portal WH filter |
| `apps/backend-go/stocklots/*.go` | Silent WMS |
| `apps/backend-go/warehouse/ops_portal.go` | Silent inventory PATCH |
| `apps/backend-go/warehouse/ops_inventory_policy.go` | Silent policy |
| `apps/backend-go/warehouse/ops_settings.go` | Silent settings |
| `apps/backend-go/warehouse/dispatch_execute.go` | Class A gold path |
| `apps/backend-go/warehouse/mutation_idempotency.go` | Idem helpers |
| `apps/backend-go/returns/inbound.go` | Scan silent / confirm outbox |
| `apps/backend-go/creditnote/handlers.go` | Reverse receive IDOR |
| `apps/backend-go/kafka/notification_dispatcher.go` | Hub fanout |
| `apps/backend-go/ws/handler.go` | Hub subscription by role |
| `apps/backend-go/events/events.go` | Declared WH event types (no WMS stock events) |

---

**Audit complete.** No code changes made. Next agent phase: implement P0 fixes only after multi-role audit merge.
