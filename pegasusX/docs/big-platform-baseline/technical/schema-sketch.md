# Schema Sketch (Spanner)

Additive sketches — refine in DDL PRs. Prefer new tables over overloading JSON blobs when queried.

## Planning / graph

| Table | Purpose |
|-------|---------|
| KnowledgeEdges | from_type, from_id, edge_type, to_type, to_id, supplier_id, attrs JSON |
| PlanScenarios | scenario_id, horizon, parent_id, status, metrics JSON |
| PlanDecisions | scenario_id, decision_type, payload JSON, version |
| DemandSignals | warehouse_id, sku_id, h3, p10/p50/p90, confidence, contributions JSON, as_of |
| CapacitySlots | node_type, node_id, skill, start_at, end_at, capacity_units, reserved_units |
| TaxRegimeVersions | id, effective_from/to, rules JSON |
| OrderLineFiscalSnapshots | order_id, sku, regime_version_id, taxable/vat/total minor |
| CartSessions | parent multi-supplier checkout |
| CartSessionOrders | cart_session_id → order_id |
| PlaybookVersions | id, trigger, actions JSON, success metrics |
| PlaybookRuns | playbook_id, started_at, status, audit |

## Execution extensions

| Table | Purpose |
|-------|---------|
| WarehouseTasks | pick/pack/putaway state machines |
| DockAppointments | yard/dock schedule |
| RouteETA | order_id/stop, eta_p50, eta_p90, confidence |
| DriverScores | rolling metrics |
| SettlementUnlock | order_id, unlocked_at, geofence_method |

## Existing tables to keep authoritative

Orders, Inventory, Claims, ClaimEvidences, OrderFiscalReceipts, RetailerCreditProfiles, Manifests, OutboxEvents, Ledger/Chargebacks.

## Migration discipline

- One additive migration per feature slice  
- Indexes for supplier_id + time filters  
- No destructive drops without dual-write window  
