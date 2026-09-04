# Track 3 Audit Report: Supplier, Factory & Catalog Domain

**Audit Date:** 2026-08-30  
**Target Codebase:** `pegasusX/apps/backend-go`  
**Packages Inspected:** `supplier`, `factory`, `catalog`, `pricing`, `globalproducts`, `stocklots`, `tenantreg`, `supplierroutes`, `factoryroutes`, `catalogroutes`, `globalproductsroutes`  
**Auditor:** Teamwork Codebase Explorer (Track 3 Lead)

---

## Executive Summary

Track 3 encompasses the core supply chain backbone of PegasusX: tenant registration, multi-supplier isolation, cell/market partitioning, catalog management, global SKU linkage, dynamic pricing rules, bill of materials (BOM), manufacturing work orders, factory batch scheduling, quality control (QC), and stocklot inventory lifecycle.

While significant domain architecture and schema models are in place, our line-by-line audit revealed **14 critical architectural, transactional, concurrency, and security flaws**. Most notably:
1. **Critical Outbox NOT NULL Constraint Failures:** Multiple background and compensation flows directly insert into `OutboxEvents` without the mandatory `SupplierId` column (required by Spanner schema `spanner.ddl:690`), guaranteeing runtime transaction aborts during dispatch compensation and exception handling.
2. **Phantom Table & Hardcoded Mock BOM in Factory Production:** Production order processing queries non-existent Spanner table `FactoryRawMaterials` and hardcodes a dummy 2x multiplier rather than evaluating dynamic BOM recipes, with actual stock deduction code commented out.
3. **Multi-Tenant User Authentication Collisions:** Supplier and Factory user lookup queries (`Idx_SupplierUsers_ByPhone`) perform non-deterministic `LIMIT 1` queries on non-unique phone numbers, allowing users with identical phone numbers across suppliers to be authenticated into the wrong supplier tenant.
4. **Order Vetting Blocks Valid Cash-on-Delivery (COD) and Credit Orders:** Order approval enforces settled ledger payment entries, blocking valid B2B Credit Term (Net-30/60) and COD orders.
5. **Inventory Import Wipes Out Active Order Stock Reservations:** Applying an inventory spreadsheet import resets `QuantityReserved = 0` across all imported SKUs, wiping out in-flight order allocations and causing severe inventory overselling.
6. **Catastrophic Topology Update Cascades:** Updating supplier topology (`PUT /v1/supplier/topology`) issues raw `DELETE FROM Warehouses` and `DELETE FROM Factories`, orphaning all foreign keys, active orders, stocklots, and vehicle assignments.
7. **Severe Global State Mutation in Factory Service:** `factory/apply.go` reads and writes *every single* manifest and transfer row in the entire factory database on every single mutation, creating an extreme $O(N)$ scalability bottleneck and concurrency race condition across concurrent HTTP requests.

---

## Domain 1: Multi-Supplier Isolation & Cell/Market Partitioning

### Finding 1.1: Multi-Tenant Authentication Leak via Non-Deterministic User Phone Lookups
- **Exact Location:** `pegasusX/apps/backend-go/supplier/repository_spanner_onboarding.go:374`, `pegasusX/apps/backend-go/supplier/repository_spanner.go:713`, `pegasusX/apps/backend-go/factory/auth_login.go:207-214`
- **Flaw Details:**
  In `supplier/repository_spanner_onboarding.go`:
  ```go
  374: if existingSupplierID == supplierID && isActive && userID != "" {
  375:     return false, nil
  376: }
  ```
  `ensureSupplierUserPhoneAvailable` only checks if the phone is active within the *current* supplier. If a phone number is registered across two different suppliers (e.g. multi-tenant contractors, shared drivers, or platform testers), the system allows the insertion.
  However, during login in `supplier/repository_spanner.go`:
  ```go
  712: SQL: `SELECT UserId, SupplierId, Name, Phone, PasswordHash, SupplierRole, IsActive
  713:       FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
  714:       WHERE Phone = @phone AND IsActive = true
  715:       LIMIT 1`,
  ```
  And in `factory/auth_login.go`:
  ```go
  207: SQL: `SELECT UserId, SupplierId, Name, Phone, PasswordHash, SupplierRole, COALESCE(AssignedFactoryId, ''), IsActive
  208:       FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
  209:       WHERE Phone = @phone AND IsActive = true AND SupplierRole IN ('FACTORY', 'FACTORY_ADMIN', 'FACTORY_STAFF')
  210:       LIMIT 1`,
  ```
  Because index `Idx_SupplierUsers_ByPhone` is non-unique and the query specifies `LIMIT 1` with no `ORDER BY` and no `SupplierId` predicate, Spanner returns an arbitrary tenant row. A staff member attempting to log into Supplier A will intermittently be authenticated as Supplier B, gaining unauthorized administrative access to Supplier B's portal and factory operations.
- **Blast Radius:** Critical multi-tenant isolation breach. Complete compromise of tenant confidentiality and administrative control.
- **Recommendation:** Enforce globally unique phone numbers across `SupplierUsers`, or require `supplier_id` / `tenant_id` as part of the login credentials, and add `WHERE SupplierId = @supplier_id` to all phone lookup queries.

---

### Finding 1.2: Cross-Market Contamination in Catalog Discovery
- **Exact Location:** `pegasusX/apps/backend-go/catalog/repository.go:266`
- **Flaw Details:**
  `ListDiscoverableProducts` constructs a discovery query:
  ```go
  266: SQL: `SELECT ProductId, SupplierId, Name, Description, CategoryId, PriceMinor, Currency, ImageUrl, Barcode, Sku, WeightKg, VolumeM3, PackagingUnit, UnitsPerPack, IsActive, CreatedAt, UpdatedAt
  267:       FROM Products
  268:       WHERE IsActive = TRUE`
  ```
  The query lacks filtering by `MarketCode`, `CountryCode`, or `Supplier.PackCountry`. As a result, retailers in Uzbekistan (UZS) will see products published in Kazakhstan (KZT), the European Union (EUR), or the United States (USD). When retailers place orders on these cross-market products, downstream payment gateways fail due to currency mismatches, and proximity routing fails due to incompatible H3 coordinate frames.
- **Blast Radius:** Cross-market leakage across global catalog discovery. Broken order checkout and currency conversion failures.
- **Recommendation:** Join `Products` with `Suppliers` on `SupplierId` and filter by `Suppliers.MarketCode = @market_code` or `Suppliers.CountryCode = @country_code` derived from the retailer's authenticated checkout pack context.

---

### Finding 1.3: Unauthenticated Factory Location and Capacity Tampering
- **Exact Location:** `pegasusX/apps/backend-go/factory/location_ops.go:187-200`
- **Flaw Details:**
  In `scopedFactoryID`:
  ```go
  187: func scopedFactoryID(r *http.Request) string {
  188: 	if claims, ok := auth.FromContext(r.Context()); ok {
  189: 		if fid := strings.TrimSpace(claims.HomeNodeID); fid != "" {
  190: 			return fid
  191: 		}
  192: 	}
  193: 	return strings.TrimSpace(r.URL.Query().Get("factory_id"))
  194: }
  ```
  `HandleOpsLocation` allows updating factory GPS coordinates, address, and daily capacity. If an authenticated user's JWT claims do not contain `HomeNodeID` (such as a generic `RoleAdmin` or supplier staff token), the handler accepts `factory_id` directly from query parameters without verifying that `factory.SupplierId == claims.SupplierId`. Any authenticated supplier user can alter the GPS coordinates and operational status of any other supplier's factory.
- **Blast Radius:** Insecure Direct Object Reference (IDOR) on physical factory infrastructure and GPS routing coordinates.
- **Recommendation:** Query Spanner to verify `WHERE FactoryId = @fid AND SupplierId = @sid` before accepting any mutation in `HandleOpsLocation`.

---

## Domain 2: Catalog, SKU Lifecycle, Pricing Rules & Global Products

### Finding 2.1: Volume Tier Pricing Silently Ignored in Pricing Engine
- **Exact Location:** `pegasusX/apps/backend-go/pricing/repository.go:28-38`, `pegasusX/apps/backend-go/pricing/models.go:19`
- **Flaw Details:**
  `PriceListItem` contains `MinQty *int64` to support tiered volume discounting. However, `GetActiveUnitPriceMinor` is implemented as:
  ```go
  28: SQL: `SELECT pli.PriceMinor
  29:       FROM PriceListItems pli
  30:       JOIN PriceLists pl ON pli.PriceListId = pl.PriceListId
  31:       WHERE pl.SupplierId = @supplierId
  32:         AND pli.ProductId = @productId
  33:         AND pl.IsActive = TRUE
  34:         AND (pl.ValidFrom IS NULL OR pl.ValidFrom <= @now)
  35:         AND (pl.ValidTo IS NULL OR pl.ValidTo >= @now)
  36:       ORDER BY pl.Priority DESC
  37:       LIMIT 1`
  ```
  `GetActiveUnitPriceMinor` does not accept a `quantity` parameter, does not filter by `pli.MinQty <= @quantity`, and performs a raw `LIMIT 1` on `PriceListItems`. If a product has volume tiers (e.g. 1-99 units @ $10, 100+ units @ $8), the database returns whichever row the Spanner index happens to evaluate first, regardless of the retailer's order volume.
- **Blast Radius:** Financial inaccuracies on large-volume B2B orders; either overcharging bulk buyers or undercharging small-quantity buyers.
- **Recommendation:** Update `GetActiveUnitPriceMinor(ctx, supplierID, productID string, quantity int64)` to include `AND (pli.MinQty IS NULL OR pli.MinQty <= @qty)` and order by `pli.MinQty DESC` to match the highest applicable tier.

---

### Finding 2.2: Retailer Pricing Overrides Disregard Currency and PriceList Hierarchy
- **Exact Location:** `pegasusX/apps/backend-go/schema/spanner.ddl:101-115`, `pegasusX/apps/backend-go/supplier/retailer_pricing.go:111-137`
- **Flaw Details:**
  `RetailerPricingOverrides` table stores `CustomPriceMinor INT64 NOT NULL`, but **omits** a `Currency STRING(3)` column.
  When `ApplyPricing` evaluates prices:
  1. It reads `CustomPriceMinor` from `RetailerPricingOverrides`.
  2. If the supplier changes their base catalog currency or if a foreign retailer accesses the product, `CustomPriceMinor` is interpreted directly in the request currency with no currency conversion or validation.
  3. Furthermore, `RetailerPricingOverrides` directly overrides `Products.PriceMinor`, completely bypassing custom `PriceLists` and seasonal promotional rules.
- **Blast Radius:** Currency corruption on retailer-specific negotiated pricing; potential 10,000x pricing discrepancies when currencies differ (e.g. USD vs UZS).
- **Recommendation:** Add `Currency STRING(3) NOT NULL` to `RetailerPricingOverrides` and enforce that `Currency` matches the supplier's active market pack currency upon insertion and price evaluation.

---

### Finding 2.3: Catastrophic $O(N^2)$ Fuzzy Matching and Quadratic Query Spikes
- **Exact Location:** `pegasusX/apps/backend-go/catalog/repository.go:283-345`, `pegasusX/apps/backend-go/globalproducts/service.go:303-340`
- **Flaw Details:**
  `findSimilarProducts` in `catalog/repository.go` and `HandleLinkProduct` in `globalproducts/service.go` attempt to find matching products by loading all products into memory and computing Levenshtein / Jaro-Winkler distances in a nested loop:
  ```go
  for _, p1 := range allProducts {
      for _, p2 := range allProducts {
          sim := jaroWinkler(p1.Name, p2.Name)
          ...
      }
  }
  ```
  In `globalproducts/service.go`, for every candidate product, it fires a separate Spanner query `ReadRow("GlobalProducts", ...)` inside the loop. As catalog size grows beyond 5,000 SKUs, this results in $(5000)^2 = 25,000,000$ string comparisons and thousands of serial database queries, exhausting server CPU and timing out HTTP requests.
- **Blast Radius:** Denial of service (DoS) and extreme latency spikes during catalog sync and SKU creation.
- **Recommendation:** Implement trigram indexing (`pg_trgm` equivalent or pre-computed embedding search) and batch-read global product records using `spanner.KeySet`.

---

## Domain 3: Factory Operations, BOM & Manufacturing Execution

### Finding 3.1: Phantom Table & Mock BOM Hardcoding in Production Execution
- **Exact Location:** `pegasusX/apps/backend-go/factory/bom.go:49-86`
- **Flaw Details:**
  In `ValidateBOMAndStartProduction`:
  ```go
  49: // Mock BOM: Each product requires 2 units of raw material {ProductID}-RAW
  50: rawMaterialID := productID + "-RAW"
  51: needed := requested * 2
  52: 
  53: row, err := txn.ReadRow(ctx, "FactoryRawMaterials",
  54:     spanner.Key{factoryID, rawMaterialID},
  55:     []string{"QuantityOnHand", "QuantityReserved"})
  ```
  1. The code assumes a hardcoded mock recipe (`needed = requested * 2`) and a dummy raw material ID (`productID + "-RAW"`).
  2. Table `FactoryRawMaterials` **does not exist** anywhere in `schema/spanner.ddl` or migration files. When executed against Spanner, `txn.ReadRow` returns `Table not found: FactoryRawMaterials`, forcing the workflow into `EXCEPTION_MATERIAL`.
  3. Furthermore, on lines 76-86, raw material inventory deduction is explicitly commented out:
  ```go
  76: // Material deduction muts would be added here
  77: _ = rawID
  78: _ = needed
  79: _ = muts
  ```
- **Blast Radius:** Factory production work order execution is entirely non-functional in production. Manufacturing batches cannot validate real raw material inventory.
- **Recommendation:** Create `BillOfMaterials` (`ParentProductId`, `ChildMaterialId`, `QuantityPerUnit`) and `FactoryRawInventory` tables in `schema/spanner.ddl`, and implement dynamic multi-tier BOM explosion with atomic inventory reservation.

---

### Finding 3.2: Severe Concurrency Bottleneck in `factory/apply.go`
- **Exact Location:** `pegasusX/apps/backend-go/factory/apply.go:10-58`
- **Flaw Details:**
  The `apply` mutation wrapper executed on every single factory operation (creating transfers, updating manifests, sealing trucks, recording exceptions) does the following:
  ```go
  20: manifests, err = tx.ListManifests(ctx)
  24: transfers, err = tx.ListTransfers(ctx)
  ...
  30: s.manifests = manifests
  31: s.transfers = transfers
  ...
  41: for _, m := range s.manifests {
  42:     if err := tx.SaveManifest(ctx, m); err != nil { return err }
  43: }
  44: for _, t := range s.transfers {
  45:     if err := tx.SaveTransfer(ctx, t); err != nil { return err }
  46: }
  ```
  1. **$O(N)$ Write Explosion:** If the factory has 500 manifests and 2,000 transfers, every status change on 1 manifest causes 2,500 Spanner UPDATE mutations.
  2. **Race Condition & Data Overwrite:** Two concurrent HTTP requests running `apply` overwrite the shared struct slices `s.manifests` and `s.transfers`. The second transaction to commit will overwrite the state changes of the first transaction, causing lost updates.
- **Blast Radius:** Total database write saturation, severe Spanner transaction lock conflicts, and corrupted manifest/transfer state across concurrent factory operators.
- **Recommendation:** Remove full-table reading and bulk saving from `apply.go`. Mutate only the specific `ManifestID` or `TransferID` rows affected by the transaction using targeted `spanner.UpdateMap`.

---

### Finding 3.3: Factory Manifest Completion Does Not Receive Goods at Warehouse
- **Exact Location:** `pegasusX/apps/backend-go/factory/service.go:684-735`
- **Flaw Details:**
  When a factory truck finishes its delivery run and transitions to `COMPLETED` (`transitionManifestLocked`), it marks internal transfers as `COMPLETED`.
  However, it **never**:
  1. Updates the destination `WarehouseSupplyRequests.State` to `RECEIVED` or `DELIVERED`.
  2. Credits `SupplierInventoryV2` or creates `StockLots` at the destination warehouse.
  As a result, factory goods are marked as shipped and delivered, but the receiving warehouse never receives inventory, leaving warehouse stock at zero.
- **Blast Radius:** Broken factory-to-warehouse replenishment pipeline. Physical inventory exists in the warehouse, but digital inventory remains uncredited.
- **Recommendation:** In `transitionManifestLocked`, when moving to `COMPLETED`, query the associated `WarehouseSupplyRequests`, transition them to `RECEIVED`, and invoke `stocklots.PutawayInTxn` / `RollupInventoryV2InTxn` to credit warehouse inventory.

---

### Finding 3.4: QC Inspection Failure Fails to Quarantine Supply Request
- **Exact Location:** `pegasusX/apps/backend-go/factory/qc.go:286-310`
- **Flaw Details:**
  In `UpsertQC`:
  ```go
  286: if req.Result == "FAIL" {
  287:     // Inserts FactorySupplyRequestQC row with Result="FAIL"
  288: }
  ```
  While `FactorySupplyRequestQC` audit record is inserted with `Result: "FAIL"`, `UpsertQC` does **not** update `WarehouseSupplyRequests.State` in Spanner (e.g. from `IN_PRODUCTION` to `QUARANTINE` or `REJECTED`). Downstream batchers and transfer schedulers checking `WarehouseSupplyRequests` continue to treat the failed batch as valid, allowing defective goods to be assigned to truck manifests.
- **Blast Radius:** Defective or contaminated production batches bypass quality control and are shipped to distribution warehouses.
- **Recommendation:** In `UpsertQC`, when `req.Result == "FAIL"`, update `WarehouseSupplyRequests.State = 'QUARANTINE'` and emit an `EventFactorySupplyQCRejected` outbox event.

---

### Finding 3.5: IoT Telemetry Ingest Incurs Silent Data Loss on Redis Restart
- **Exact Location:** `pegasusX/apps/backend-go/factory/iot_ingest.go:92-109`
- **Flaw Details:**
  In `flushBatch`:
  ```go
  102: key := fmt.Sprintf("factory:iot:%s:units", machineID)
  103: pipe.IncrBy(ctx, key, units)
  ```
  Production counts from machine IoT telemetry are aggregated and flushed exclusively to Redis with no database persistence. If the Redis container restarts or evicts keys under memory pressure, all accumulated production metrics and machine run counts are permanently erased.
- **Blast Radius:** Loss of factory operational telemetry, false OEE (Overall Equipment Effectiveness) reporting, and inaccurate manufacturing output tracking.
- **Recommendation:** Write periodic flush batches to a `FactoryMachineTelemetry` Spanner table or append to an outbox topic for durable time-series aggregation.

---

## Domain 4: Transactional Integrity, Concurrency & Outbox Consistency

### Finding 4.1: Missing `SupplierId` on Raw `OutboxEvents` Mutations Causes Spanner NOT NULL Constraint Aborts
- **Exact Location:**
  - `pegasusX/apps/backend-go/supplier/dispatch_execute.go:683-691`, `712-720`, `322-330`
  - `pegasusX/apps/backend-go/payload/exceptions.go:77-84`
  - `pegasusX/apps/backend-go/warehouse/dispatch_rescue.go:201-208`
  - `pegasusX/apps/backend-go/routing/replan.go:59-66`
- **Flaw Details:**
  The `OutboxEvents` table schema in `schema/spanner.ddl:690` explicitly declares:
  ```sql
  CREATE TABLE OutboxEvents (
    EventId        STRING(36)    NOT NULL,
    AggregateType  STRING(64)    NOT NULL,
    AggregateId    STRING(64)    NOT NULL,
    EventType      STRING(64)    NOT NULL,
    SupplierId     STRING(64)    NOT NULL,
    Payload        BYTES(MAX)    NOT NULL,
    CreatedAt      TIMESTAMP     NOT NULL OPTIONS (allow_commit_timestamp=true),
  ) PRIMARY KEY (SupplierId, EventId);
  ```
  Notice that `OutboxEvents` is interleaved/partitioned by `SupplierId`, making `SupplierId` a mandatory NOT NULL primary key column.
  However, in `supplier/dispatch_execute.go:683-691` (inside `compensatePartialDispatch`):
  ```go
  spanner.InsertMap("OutboxEvents", map[string]any{
      "EventId":       uuid.NewString(),
      "AggregateType": events.AggregateManifest,
      "AggregateId":   manifestID,
      "EventType":     "MANIFEST_CANCELLED",
      "Payload":       []byte(fmt.Sprintf(`{"manifest_id":"%s"}`, manifestID)),
      "CreatedAt":     spanner.CommitTimestamp,
  })
  ```
  `SupplierId` is **completely omitted**.
  When `compensatePartialDispatch` runs during a dispatch failure, Spanner rejects the mutation with `Failed to commit: column SupplierId cannot be NULL`. The compensation transaction aborts, leaving zombie manifests in `ASSIGNED` status and trapped inventory locks.
  In `routing/replan.go:59-66`, the mutation also uses incorrect column names `"Topic"` and `"PayloadJson"` which do not exist in `OutboxEvents`.
- **Blast Radius:** Critical transactional failure. Dispatch compensation routines and route replanning fail catastrophically, leaving the system in an inconsistent state.
- **Recommendation:** Always use canonical `outbox.EmitJSON(ctx, txn, ...)` or ensure `SupplierId` is explicitly provided in all raw `spanner.InsertMap("OutboxEvents", ...)` calls.

---

### Finding 4.2: Inventory Import Wipes Out Active Order Stock Reservations
- **Exact Location:** `pegasusX/apps/backend-go/supplier/import_sessions_apply.go:306-314`
- **Flaw Details:**
  When applying an inventory import spreadsheet, `applyImportSessionTxn` writes:
  ```go
  306: mutations = append(mutations, spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
  307:     "SupplierId":       session.SupplierID,
  308:     "WarehouseId":      warehouseID,
  309:     "ProductId":        productID,
  310:     "QuantityOnHand":   row.QoH,
  311:     "QuantityReserved": int64(0),
  312:     "UpdatedAt":        now,
  313: }))
  ```
  By unconditionally writing `QuantityReserved = 0`, any stock currently reserved for pending or in-flight orders is immediately wiped out. Available-To-Promise (ATP) calculations (`QoH - QuantityReserved`) artificially increase, allowing new orders to reserve and claim stock that was already promised to previous customers, leading to double-allocation and severe stockouts.
- **Blast Radius:** Immediate inventory corruption and double allocation during daily supplier stock updates.
- **Recommendation:** Read the existing `QuantityReserved` from `SupplierInventoryV2` within the transaction and preserve it, or use `RollupInventoryV2InTxn` to calculate reserved quantities from active `OrderLotReservations`.

---

### Finding 4.3: Destructive Supplier Topology Updates Orphan Entire Warehouses
- **Exact Location:** `pegasusX/apps/backend-go/supplier/repository_spanner.go:918-940`
- **Flaw Details:**
  When a supplier updates their operating nodes via `PUT /v1/supplier/topology`, `ReplaceTopology` executes:
  ```go
  918: DELETE FROM WarehouseCoverageCells WHERE SupplierId = @supplierId
  924: DELETE FROM WarehouseCoverageCities WHERE SupplierId = @supplierId
  930: DELETE FROM Warehouses WHERE SupplierId = @supplierId
  936: DELETE FROM Factories WHERE SupplierId = @supplierId
  ```
  1. Deleting all warehouses and factories instantly severs foreign key relationships and deletes existing `WarehouseId` references across active `Orders`, `InventoryLevels`, `SupplierInventoryV2`, `StockLots`, and `Vehicles`.
  2. If new IDs are minted using `stableTopologyID`, all existing inventory rows for the old warehouse IDs become orphaned and permanently inaccessible in the UI.
- **Blast Radius:** Complete loss of inventory and broken operational history on warehouse profile updates.
- **Recommendation:** Perform an upsert/diffing reconciliation rather than raw table truncation. Mark decommissioned warehouses as `IsActive = false`.

---

## Domain 5: Cross-Role Parity & Ecosystem Alignment

### Finding 5.1: Order Vetting Blocks Valid Cash-on-Delivery and B2B Credit Orders
- **Exact Location:** `pegasusX/apps/backend-go/supplier/orders_vet.go:128-136`, `239-270`
- **Flaw Details:**
  When a supplier attempts to approve an order via `HandleVetOrder` (`Decision: "APPROVED"`), `VetOrder` requires:
  ```go
  128: if !orderPaymentClearedInTxn(ctx, txn, orderID) {
  129:     return SupplierOrder{}, ErrPaymentNotCleared
  130: }
  ```
  `orderPaymentClearedInTxn` only returns true if there is a settled `PaymentSessions` record (`WEBHOOK_PAID`, `CASH_COLLECTED`, or `SETTLEMENT_CREDIT`).
  However, in wholesale supply chains:
  - **Cash on Delivery (COD):** Cash is collected by the driver at delivery time; no payment session is settled at order review time.
  - **B2B Credit Terms (Net-30/Net-60):** Payment is invoiced and settled post-delivery.
  Because `orderPaymentClearedInTxn` does not check the order's `PaymentType` or `Terms`, it returns `ErrPaymentNotCleared` and aborts approval for all COD and credit orders.
- **Blast Radius:** Inability for suppliers to approve and process Cash-on-Delivery and Trade Credit orders.
- **Recommendation:** In `VetOrder`, inspect `Orders.PaymentMethod` / `PaymentTerms`. If the payment type is `COD` or `SUPPLIER_CREDIT`, bypass the upfront settled payment check.

---

### Finding 5.2: Replenishment Trigger Creates Empty Stub Supply Requests
- **Exact Location:** `pegasusX/apps/backend-go/supplier/portal_admin_ops.go:277-347`
- **Flaw Details:**
  `HandleReplenishmentTrigger` calculates projected replenishment needs outside the transaction, but then writes a `WarehouseSupplyRequests` row with hardcoded `0` projected units and zero line items, followed by emitting an event for an empty request. Downstream factory batchers receiving the event find zero required SKUs and take no action.
- **Blast Radius:** Automated inventory replenishment fails to generate actionable manufacturing work orders.
- **Recommendation:** Pass the computed replenishment line items directly into the transaction and insert corresponding `WarehouseSupplyRequestItems` records.

---

### Finding 5.3: Disjoint Inventory Systems (InventoryLevels vs SupplierInventoryV2)
- **Exact Location:** `supplier/portal_handlers.go:1049-1105`, `supplier/returns.go:281-318`
- **Flaw Details:**
  The codebase contains two competing inventory tables:
  1. `InventoryLevels` (legacy table managed by `inventory.Service`).
  2. `SupplierInventoryV2` / `StockLots` (WMS lot-based inventory managed by `stocklots.Service`).
  Different handlers mutate different tables:
  - `PATCH /v1/supplier/inventory` mutates only `InventoryLevels`.
  - `POST /v1/supplier/returns/resolve` mutates only `SupplierInventoryV2`.
  - `supplier/import_sessions_apply.go` mutates both but resets `QuantityReserved`.
  The two tables continually drift apart, causing discrepancies between the supplier web portal, retailer discovery queries, and driver warehouse pick waves.
- **Blast Radius:** System-wide inventory desynchronization and phantom stockouts.
- **Recommendation:** Standardize on `SupplierInventoryV2` + `StockLots` as the single source of truth across all packages, and deprecate `InventoryLevels`.

---

## Deep Architectural & Edge-Case Open Questions

1. **How should multi-tier Bills of Materials (BOM) handle shared sub-components across parallel production runs?**
   - *Problem:* If Product A and Product B both require Raw Material C, and two production batches are scheduled concurrently, what is the locking and reservation strategy? Currently, without row-level locks or pessimistic queueing, both batches will see sufficient stock and start production, resulting in material exhaustion mid-run.
2. **What is the canonical ledger contract when a supplier accepts a return for a partially damaged lot?**
   - *Problem:* When a return is resolved as `RETURN_TO_STOCK`, the system credits inventory. However, what happens if only 5 out of 20 units are restocked and 15 are written off? How are credit notes, tax reversal adjustments, and retailer refunds coordinated across the financial ledger without partial-state race conditions?
3. **How should cross-cell factory fulfillment operate when the primary factory has a machine jam?**
   - *Problem:* When an IoT telemetry event signals `MACHINE_JAM`, factory planning leaves supply requests in `IN_PRODUCTION`. Should the system dynamically reroute the supply request to a secondary assigned factory along the supply lane, and how does this affect truck manifest load balancing?
4. **How should retailer-specific pricing overrides interact with tiered volume discounts and promotional campaigns?**
   - *Problem:* If a retailer has a negotiated override price of $8/unit, but a global supplier promotion offers $7/unit for purchases over 500 units, which rule takes precedence? The current pricing engine lacks an explicit rule precedence hierarchy (Override vs Promo vs Tier vs Base Markup).
5. **How should lot recall campaigns handle orders that are currently in-transit on delivery trucks?**
   - *Problem:* In `stocklots/recall.go`, initiating a recall flags affected lots as `QUARANTINE`. If an order containing that lot is already loaded onto a delivery truck (`DISPATCHED`), does the system automatically alert the driver app to halt delivery, reject the item at the retailer doorstep, or issue a post-delivery quarantine notice?
6. **How should supplier user identities be federated across multiple suppliers in multi-enterprise organizations?**
   - *Problem:* An enterprise conglomerate may operate multiple supplier entities in different market packs. How should phone-based authentication differentiate between tenants without sacrificing frictionless mobile authentication?

---

## Actionable Recommendations & Prioritized Fix Plan

| Priority | Finding ID | Issue | Target File |
| :--- | :--- | :--- | :--- |
| **P0 (Blocker)** | 4.1 | Missing `SupplierId` in `OutboxEvents` mutations | `supplier/dispatch_execute.go`, `payload/exceptions.go`, `warehouse/dispatch_rescue.go`, `routing/replan.go` |
| **P0 (Blocker)** | 3.1 | Phantom table `FactoryRawMaterials` and mock BOM | `factory/bom.go`, `schema/spanner.ddl` |
| **P0 (Blocker)** | 1.1 | Multi-tenant user login collision via non-unique phone | `supplier/repository_spanner.go`, `factory/auth_login.go` |
| **P0 (Blocker)** | 3.2 | $O(N)$ full table rewrite bottleneck in `factory/apply.go` | `factory/apply.go`, `factory/service.go` |
| **P1 (High)** | 4.2 | Inventory import wiping out `QuantityReserved` | `supplier/import_sessions_apply.go` |
| **P1 (High)** | 5.1 | Order vetting blocking COD and B2B Credit orders | `supplier/orders_vet.go` |
| **P1 (High)** | 4.3 | Destructive warehouse deletion on topology PUT | `supplier/repository_spanner.go` |
| **P1 (High)** | 3.3 | Manifest completion not crediting warehouse inventory | `factory/service.go` |
| **P1 (High)** | 1.2 | Catalog discovery leaking cross-market products | `catalog/repository.go` |
| **P2 (Medium)** | 2.1 | Volume tier pricing ignored in pricing repository | `pricing/repository.go` |
| **P2 (Medium)** | 1.3 | Unauthenticated factory location modification | `factory/location_ops.go` |
| **P2 (Medium)** | 3.4 | QC failure not quarantining supply request | `factory/qc.go` |
| **P2 (Medium)** | 5.3 | Unify `InventoryLevels` and `SupplierInventoryV2` | `supplier/portal_handlers.go`, `supplier/returns.go` |
| **P3 (Low)** | 2.3 | $O(N^2)$ fuzzy matching optimization | `catalog/repository.go`, `globalproducts/service.go` |
| **P3 (Low)** | 3.5 | Persist IoT machine telemetry to durable storage | `factory/iot_ingest.go` |
