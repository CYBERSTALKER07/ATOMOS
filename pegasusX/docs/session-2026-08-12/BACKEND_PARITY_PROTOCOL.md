# Backend Role Parity Protocol

**Date:** 2026-08-12  
**Tree:** `pegasusX` only  
**Phase:** Backend Class A audit (no UI rewrites)

## Definition of done (backend Class A)

JWT-scoped mutation → Spanner RW txn + in-txn outbox → relay → Kafka → declared consumer(s) → WS hub room and/or FCM and/or partner webhook; cache invalidate after commit; idempotency on mutators; no silent Spanner writes; edge cases covered by tests or documented intentional deferral.

## Role fleet

| Agent | Role | Primary packages | Routes |
|-------|------|------------------|--------|
| A0-Spine | Cross-role bus | order, payment, outbox, kafka, ws, cache, idempotency, ar, payout, fiscal | orderroutes, paymentroutes, webhookroutes |
| A1-Supplier | ADMIN (= SUPPLIER) | supplier, planning, credit, claims (approve), pulse, controltower | supplierroutes, creditroutes, controltowerroutes |
| A2-Retailer | RETAILER | retailer, order create, claims file, ar view/pay | retailerroutes |
| A3-Driver | DRIVER | driver, telemetry, order arrive/cash/shop-closed | driverroutes, deliveryroutes, telemetryroutes |
| A4-Warehouse | WAREHOUSE_ADMIN | warehouse, stocklots, returns, dispatch | warehouseroutes, returnsroutes |
| A5-Factory | FACTORY_ADMIN | factory, manifest | factoryroutes |
| A6-Payload | PAYLOAD | payload, factory loading-bay bridge | payloaderoutes / payloaderoutes |
| A7-Platform | PLATFORM_ADMIN | platformadmin, featureflags, mfa, partner admin, globalproducts | platformroutes |

Client inventory SoT: [`../FEATURES_BY_APP_ROLE.md`](../FEATURES_BY_APP_ROLE.md), [`../ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md).  
Supplier web/desktop = `supplier-portal`. Platform break-glass = `admin-portal` (not supplier).

## Per-mutation checklist

| # | Check | Pass |
|---|--------|------|
| 1 | Auth scope | Role gate; tenant/home-node from claims — never body supplier_id for auth |
| 2 | Idempotency | Guard on public mutators |
| 3 | Spanner RW | Writes in RW txn |
| 4 | Outbox | Emit in same txn for state transitions |
| 5 | Cache | Invalidate after commit |
| 6 | Realtime | Dispatcher + hub/FCM/webhook |
| 7 | Edge cases | Cancel, concurrency, double-submit, permission, payment/fiscal fail |
| 8 | Tests | Unit and/or PX_E2E marker for money/stock/claims |

## Data plane

```
HTTP/Webhook → Service → Spanner + OutboxEvents
  → Relay → Kafka
  → Consumers → WS / Redis pubsub / FCM / partner webhooks
```

Flag: silent mutation, orphan event, wrong room, api-only run-mode hole.

## Report path

`docs/session-2026-08-12/BACKEND_PARITY_<ROLE>.md`

1. Feature inventory (route → service → Class A status)  
2. Gaps P0/P1/P2 with file:line  
3. Event/consumer matrix  
4. Edge-case matrix  
5. Proposed fixes (do not implement in audit phase)

## Severity

- **P0:** money, auth IDOR, silent outbox on state machine, data corruption  
- **P1:** missing realtime fanout, incomplete transitions, platform contract split  
- **P2:** dead code, naming, polish  
