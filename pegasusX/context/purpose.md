# pegasusX Purpose

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


## Mission
Operate a single-supplier logistics ecosystem at high reliability for thousands of retailers, while preserving full architectural compatibility with the Pegasus multi-supplier reference.

## Role Alignment
- **SUPPLIER (`role=ADMIN`)** — operates the supplier portal. Default deploy uses one seeded supplier; up to `MAX_SUPPLIERS` (default 10) via registration cap.
- **RETAILER (`role=RETAILER`)** — self-registered ordering customer.
- **DRIVER (`role=DRIVER`)** — route execution, geofenced actions, delivery verification. Scoped to a Home Node (Warehouse OR Factory).
- **PAYLOAD (`role=PAYLOADER`)** — loading, offloading, manifest confirmation.
- **FACTORY_ADMIN / WAREHOUSE_ADMIN** — node-scoped administration.

## Invariants
1. `SupplierId` is required on every supplier-owned row, claim, and event. Single-tenant is enforced by seed + policy, not by removing the field.
2. Double-entry ledger writes pair debit/credit atomically with the business mutation, inside the same Spanner read-write transaction.
3. State-changing events are written via transactional outbox in the same transaction as the row mutation. Direct producer writes are reserved for telemetry only.
4. Role scopes are derived from JWT claims, never from request bodies (`supplier_id`, `warehouse_id`, `factory_id`, `home_node_id`).
5. Cache invalidation runs post-commit via Redis Pub/Sub. TTL is a safety net, not a correctness mechanism.
6. Driver and vehicle home node is `(HomeNodeType, HomeNodeId)`, supporting any mix of warehouse-local, factory-local, and remote topologies.

## Migration Compatibility
- JSON keys, event names, claim fields, state enums, and route families align with Pegasus where possible.
- pegasusX changes that diverge from Pegasus must be documented in [parity-ledger.md](parity-ledger.md).
