# Wave B3 — Retailer multi-user + parent order bus
> **POINT-IN-TIME SNAPSHOT (2026-08-12) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Master:** [`BACKEND_PARITY_MASTER.md`](./BACKEND_PARITY_MASTER.md)

## Changes

### M-P0-4 ResolveRetailerOrgID on order money paths
- `order.HandleCreate` / `HandleUnifiedCheckout` / `HandleCheckoutPreview` use `auth.ResolveRetailerOrgID` (org tenant), not `claims.Subject`
- Cancel ownership (`HandleRetailerCancel` + `UpdateStatus` retailer gate) compares org to `order.RetailerID`
- Card + cash checkout (`payment/retailer_checkout.go`) and `HandleRetailerConfirmCash` org-scoped; actor uses `ResolveRetailerUserID`
- Parent GET, QR payload ownership, condition report, receipt party copy aligned to org

### M-P0-6 ParentOrders outbox
- New events: `PARENT_ORDER_CREATED`, `PARENT_ORDER_UPDATED`
- Aggregate: `ParentOrder`
- `insertParentOrder` / `updateParentOrderTotals` use RW txn + in-txn outbox (no bare `Apply`)
- Dispatcher `handleParentOrderEvent` → `retailer:{orgId}`

### M-P1-3 Cart / POS bus
- Cart `UpsertItems` / `ClearCart` / `ClearCartAll` emit `CART_SYNC_UPDATED` in same RW txn (producer was missing)
- POS + store-stock already outbox’d; dispatcher now routes `POS_*` / `STORE_STOCK_*` → RetailerHub (`handleRetailerOpsEvent`)

## Verification

```bash
cd apps/backend-go
go test ./order/ ./payment/ ./retailer/ ./kafka/ ./events/ ./notifications/ -count=1
go build -o /tmp/pegasusx-backend .
```
