# pegasusX Real-World Case Matrix

> **Purpose:** Map operational edge cases to role surfaces, backend guards, and owner SOPs.  
> **Screen routes:** [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md)  
> **Ecosystem spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md)

Last updated: 2026-06-29.

## How to read this matrix

| Column | Meaning |
|--------|---------|
| **Lifecycle** | Order/supply stage |
| **War story** | What happens in the field |
| **Roles** | Who must act |
| **Pages** | pegasusX routes/screens (not Pegasus reference) |
| **Technical guard** | Backend enforcement |
| **SOP** | Human playbook |
| **Educate** | Who needs training before go-live |

---

## Order lifecycle cases

| Lifecycle | War story | Roles | Key pages | Technical guard | SOP | Educate |
|-----------|-----------|-------|-----------|-----------------|-----|---------|
| Browse | Retailer compares suppliers; stale price shown | RETAILER | desktop `/catalog`, mobile Catalog | Catalog cache + WS refresh | — | Retailer procurement |
| Checkout preview | Line limit / delivery fee surprise | RETAILER, WAREHOUSE | checkout modal, `/dispatch-settings` | `orderable_quantities`, fee preview API | [`RETAILER_RECEIVING_WINDOWS_GUIDE.md`](./RETAILER_RECEIVING_WINDOWS_GUIDE.md) | Retailer + warehouse ops |
| Create | New shop outside delivery zone | RETAILER, SUPPLIER | checkout error, `/topology` | H3 `zone_miss` | [`ZONE_MISS_COMMUNICATION_POLICY.md`](./ZONE_MISS_COMMUNICATION_POLICY.md) | Supplier topology team |
| Create | Two shops order last unit | RETAILER, WAREHOUSE | checkout, dispatch settings | Atomic stock reservation (`REJECT` policy) | — | Warehouse stock policy |
| Preorder | Date rejected / AI wrong qty | RETAILER, WAREHOUSE | orders, `/preorders` | `PRE_ORDER_*` events | PX12 manual G + I | Account managers |
| Delivery proposal | Supplier proposes different date | RETAILER | tracking, proposal review | `PX_E2E_DELIVERY_PROPOSAL_OK` | — | Retailer managers |
| Cancel | Before load vs after load | RETAILER, WAREHOUSE | order detail | State machine + inventory release | — | Retailer cutoff policy |
| Dispatch | Truck too small | WAREHOUSE | `/dispatch` capacity modal | Binpack overflow, force audit | [`WAREHOUSE_EXCEPTION_SOP.md`](./WAREHOUSE_EXCEPTION_SOP.md) | Dispatchers |
| Dispatch | Partial batch commit | WAREHOUSE | `/dispatch` | `dispatch_partial_commit` | [`PARTIAL_DISPATCH_RECOVERY_SOP.md`](./PARTIAL_DISPATCH_RECOVERY_SOP.md) | Warehouse lead |
| Seal | Wrong truck sealed | PAYLOAD, DRIVER | terminal seal, driver manifest | Per-truck seal + driver gate | [`REASSIGNMENT_SUPPORT_PLAYBOOK.md`](./REASSIGNMENT_SUPPORT_PLAYBOOK.md) | Yard supervisor |
| Reassign | Driver sick mid-load | PAYLOAD, FACTORY | reassign, payload override | Capacity + idempotency replay | [`REASSIGNMENT_SUPPORT_PLAYBOOK.md`](./REASSIGNMENT_SUPPORT_PLAYBOOK.md) | Payload lead |
| Transit | “Where is my truck?” | RETAILER, SUPPLIER, DRIVER | tracking, fleet live map | Telemetry HTTP (loss-tolerant) | [`LIVE_TRACKING_EXPECTATIONS.md`](./LIVE_TRACKING_EXPECTATIONS.md) | Support + retailers |
| Arrive | GPS wrong building | DRIVER, RETAILER | mission detail, settings profile | `geofence_violation` | [`DRIVER_SUPPORT_PLAYBOOK.md`](./DRIVER_SUPPORT_PLAYBOOK.md) | Driver + retailer coords |
| Shop closed | Shop shut when truck arrives | DRIVER, RETAILER, SUPPLIER | driver ShopClosed, supplier deep link | `SHOP_CLOSED_GRACE_MINUTES` | [`SHOP_CLOSED_E2E_SOP.md`](./SHOP_CLOSED_E2E_SOP.md) | All three roles |
| Pay | Cash at door | DRIVER, RETAILER | cash screen, pending payments | Geofenced `collect-cash` | [`PAYMENT_EXCEPTION_SOP.md`](./PAYMENT_EXCEPTION_SOP.md) | Drivers |
| Pay | Card declined / webhook replay | RETAILER, SUPPLIER | treasury, pending payments | Webhook txn idempotency | [`PAYMENT_EXCEPTION_SOP.md`](./PAYMENT_EXCEPTION_SOP.md) | Finance |
| Complete | Dispute weeks later | RETAILER, SUPPLIER | tracking receipts, earnings | Ledger + receipt dossier | [`FINANCE_SUPPORT_WORKFLOW.md`](./FINANCE_SUPPORT_WORKFLOW.md) | Finance |
| Returns | Wrong barcode at gate | PAYLOAD, WAREHOUSE, SUPPLIER | inbound scan | EAN match `Products.Barcode` | [`BARCODE_GO_LIVE_CHECKLIST.md`](./BARCODE_GO_LIVE_CHECKLIST.md) | Catalog team |
| Pricing override | Custom price for one chain | SUPPLIER, RETAILER | `/pricing/retailer-overrides`, notifications | `RETAILER_PRICE_OVERRIDE` inbox | — | Supplier pricing ops |

---

## Replenishment cases

| Lifecycle | War story | Roles | Key pages | Technical guard | SOP | Educate |
|-----------|-----------|-------|-----------|-----------------|-----|---------|
| Supply create | Factory late on production | WAREHOUSE, FACTORY | `/supply-requests`, factory supply queue | Supply state machine + cancel | [`TRANSFER_CANCELLATION_RUNBOOK.md`](./TRANSFER_CANCELLATION_RUNBOOK.md) | Factory admin |
| FULFILL TRUCK | Separate sites; driver haul | FACTORY, WAREHOUSE, DRIVER, PAYLOAD | loading bay, transfers receive | `transfer_mode=TRUCK` | — | Factory + warehouse |
| FULFILL INTERNAL | Co-located factory/warehouse | FACTORY, WAREHOUSE | topology, supply FULFILL | Auto-receive on INTERNAL | SSMR `PX_E2E_REPLENISH_COLOCATE_OK` | Supplier topology |

---

## Cross-role war stories (staging QA scripts)

| ID | Script | SSMR marker | Runbook |
|----|--------|-------------|---------|
| WS-01 | Shop-closed full loop | `PX_E2E_SHOP_CLOSED_OK` | [`PX12_MANUAL_QA_RUNBOOK.md`](./qa/PX12_MANUAL_QA_RUNBOOK.md#phase-c--war-story-scripts) |
| WS-02 | Concurrent stock reject | `PX_E2E_CONCURRENT_STOCK_REJECT_OK` | same |
| WS-03 | Seal → driver gate → delivery | `PX_E2E_PAYLOAD_*`, `PX_E2E_DELIVERY_OK` | same |
| WS-04 | Returns inbound EAN valid/invalid | `PX_E2E_RETURN_GATE_RECEIVE_OK` | same |
| WS-05 | Replenish TRUCK vs INTERNAL | `PX_E2E_REPLENISH_OK`, `PX_E2E_REPLENISH_COLOCATE_OK` | same |

---

## Intentional v1 exclusions (document only)

| Case | Status |
|------|--------|
| Quantity negotiation | 410 ecosystem-wide (`negotiation_disabled`) |
| Supplier broadcast on native | Portal-primary |
| Retailer dock on warehouse | Retailer-only by design |
| Pegasus CRM/staff depth | P2 out of scope |
