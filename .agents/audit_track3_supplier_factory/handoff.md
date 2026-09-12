# Track 3 Audit Handoff Report: Supplier, Factory & Catalog Domain

**Type:** Hard Handoff  
**Date:** 2026-08-30  
**Domain Scope:** Track 3 (Supplier Portal, Factory Operations, Catalog Management, Pricing Engine, BOM/Manufacturing, StockLots & Inventory)  
**Target Repository:** `pegasusX/apps/backend-go`  

---

## 1. Observation

Direct line-by-line inspection of backend Go code and Spanner DDL identified the following verbatim facts:

1. **Schema Requirement on OutboxEvents:**
   - File: `pegasusX/apps/backend-go/schema/spanner.ddl:690-698`
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
2. **Missing `SupplierId` in Outbox Mutations:**
   - File: `pegasusX/apps/backend-go/supplier/dispatch_execute.go:683-691`, `712-720`
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
   - File: `pegasusX/apps/backend-go/payload/exceptions.go:77-84`
   - File: `pegasusX/apps/backend-go/warehouse/dispatch_rescue.go:201-208`
   - File: `pegasusX/apps/backend-go/routing/replan.go:59-66` (uses wrong columns `"Topic"`, `"PayloadJson"`)

3. **Phantom Table & Mock BOM:**
   - File: `pegasusX/apps/backend-go/factory/bom.go:49-86`
   ```go
   rawMaterialID := productID + "-RAW"
   needed := requested * 2
   row, err := txn.ReadRow(ctx, "FactoryRawMaterials", spanner.Key{factoryID, rawMaterialID}, []string{"QuantityOnHand", "QuantityReserved"})
   ```
   Table `FactoryRawMaterials` does not exist in `schema/spanner.ddl`. Lines 76-86 comment out raw material deductions (`_ = rawID; _ = needed; _ = muts`).

4. **Multi-Tenant User Phone Collision:**
   - File: `pegasusX/apps/backend-go/supplier/repository_spanner_onboarding.go:374`
   ```go
   if existingSupplierID == supplierID && isActive && userID != "" {
       return false, nil
   }
   ```
   - File: `pegasusX/apps/backend-go/supplier/repository_spanner.go:713`
   ```go
   SQL: `SELECT UserId, SupplierId, Name, Phone, PasswordHash, SupplierRole, IsActive
         FROM SupplierUsers@{FORCE_INDEX=Idx_SupplierUsers_ByPhone}
         WHERE Phone = @phone AND IsActive = true
         LIMIT 1`
   ```
   - File: `pegasusX/apps/backend-go/factory/auth_login.go:207-212`

5. **Order Vetting Blocks Valid Cash-on-Delivery and B2B Credit:**
   - File: `pegasusX/apps/backend-go/supplier/orders_vet.go:128-136`, `240-270`
   ```go
   if !orderPaymentClearedInTxn(ctx, txn, orderID) {
       return SupplierOrder{}, ErrPaymentNotCleared
   }
   ```

6. **Inventory Import Wiping Reservations:**
   - File: `pegasusX/apps/backend-go/supplier/import_sessions_apply.go:306-314`
   ```go
   mutations = append(mutations, spanner.InsertOrUpdateMap("SupplierInventoryV2", map[string]any{
       "SupplierId":       session.SupplierID,
       "WarehouseId":      warehouseID,
       "ProductId":        productID,
       "QuantityOnHand":   row.QoH,
       "QuantityReserved": int64(0),
       "UpdatedAt":        now,
   }))
   ```

7. **Destructive Warehouse Deletion on Topology PUT:**
   - File: `pegasusX/apps/backend-go/supplier/repository_spanner.go:930-937`
   ```go
   DELETE FROM Warehouses WHERE SupplierId = @supplierId
   DELETE FROM Factories WHERE SupplierId = @supplierId
   ```

8. **Full Database Table Rewrite in Factory Service:**
   - File: `pegasusX/apps/backend-go/factory/apply.go:20-56`
   ```go
   manifests, err = tx.ListManifests(ctx)
   transfers, err = tx.ListTransfers(ctx)
   ...
   for _, m := range s.manifests { tx.SaveManifest(ctx, m) }
   for _, t := range s.transfers { tx.SaveTransfer(ctx, t) }
   ```

---

## 2. Logic Chain

1. **Outbox Integrity:**
   - Fact: `OutboxEvents` has `PRIMARY KEY (SupplierId, EventId)` and `SupplierId NOT NULL`.
   - Fact: `dispatch_execute.go:683` inserts into `OutboxEvents` without `SupplierId`.
   - Deduction: Spanner will reject the commit with a `NULL value in NOT NULL column SupplierId` error, breaking the compensation routine and leaving orphan manifests and locked stock.

2. **Manufacturing Work Order Viability:**
   - Fact: `ValidateBOMAndStartProduction` queries `FactoryRawMaterials`.
   - Fact: `FactoryRawMaterials` is not defined in `schema/spanner.ddl`.
   - Deduction: Work orders cannot transition to production or validate raw material availability against Spanner; production initiation either crashes or defaults to `EXCEPTION_MATERIAL`.

3. **Tenant Isolation:**
   - Fact: `Idx_SupplierUsers_ByPhone` is a non-unique index and registration allows the same phone across different suppliers.
   - Fact: User login executes `SELECT ... WHERE Phone = @phone LIMIT 1` without filtering by `SupplierId`.
   - Deduction: Two users in different suppliers with the same phone will be nondeterministically authenticated into the wrong tenant.

4. **Inventory Reservation Safety:**
   - Fact: Available-to-promise (ATP) equals `QuantityOnHand - QuantityReserved`.
   - Fact: `import_sessions_apply.go` hardcodes `QuantityReserved = 0` on every imported product.
   - Deduction: Active order reservations are wiped, allowing the same inventory units to be sold again to subsequent buyers (double-selling).

---

## 3. Caveats

1. **Mock Data Seeding in Tests:** Unit tests in `supplier_test.go` and `factory_test.go` frequently run in memory with mock repositories, which masked the Spanner NOT NULL schema errors and the missing `FactoryRawMaterials` table.
2. **SSMR Markers:** SSMR smoke checks (`cmd/ssmr-smokecheck/e2e_check.go`) verify happy-path order creation and dispatch, but did not assert dispatch failure compensation or partial order lot recall recovery.

---

## 4. Conclusion

Track 3 has a complete REST and WebSocket interface surface matching the role-row specifications for Supplier and Factory roles. However, **core transactional safety, multi-tenant authentication boundaries, and inventory consistency have critical P0 bugs** that must be resolved prior to production traffic:
1. Fix all `OutboxEvents` mutations to include `SupplierId`.
2. Add real `BillOfMaterials` schema and dynamic raw material deduction.
3. Enforce tenant-scoped user authentication.
4. Protect `QuantityReserved` during inventory imports.
5. Replace full-table re-saves in `factory/apply.go` with targeted row updates.

---

## 5. Verification Method

To independently verify these findings:

1. **Verify Outbox Schema Constraint:**
   - Inspect `pegasusX/apps/backend-go/schema/spanner.ddl:690-698` vs `pegasusX/apps/backend-go/supplier/dispatch_execute.go:683-691`.
2. **Verify Missing BOM Table:**
   - Grep `FactoryRawMaterials` in `pegasusX/apps/backend-go/schema/spanner.ddl` (returns 0 matches).
3. **Verify User Phone Collision Query:**
   - Inspect `pegasusX/apps/backend-go/supplier/repository_spanner.go:712-716`.
4. **Run Touched Package Tests:**
   ```bash
   cd pegasusX/apps/backend-go
   go test ./supplier ./factory ./catalog ./pricing ./stocklots ./tenantreg
   ```
