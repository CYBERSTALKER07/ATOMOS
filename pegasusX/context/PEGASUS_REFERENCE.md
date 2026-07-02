# Pegasus Multi-Supplier Reference Architecture

This document holds the architectural context for the **Pegasus** reference implementation. It serves as a strict boundary definition against **PegasusX**.

## 1. What is Pegasus?
Pegasus is the multi-supplier, multi-tenant federation backend and platform. 
It differs from PegasusX (which is single-supplier execution & planning) by introducing:
- Tenant isolation.
- Cross-supplier collaboration.
- Federated Data (EKG federation).
- Full Integrated Business Planning (IBP).

## 2. Platform Commitments
Pegasus adds tenant isolation and federation *without* reshaping the core PegasusX Spanner tables. Instead, it layers admin-portal routing and tenant separation on top.

## 3. Multi-Supplier Parity Track
*(Migrated from PlanDigitalBrain.md)*

| Phase | Scope | PegasusX | Pegasus | Status |
|---|---|---|---|---|
| **P1** | Federation read APIs + admin control tower | Emits events pegasus subscribes to | `admin/planning_federation.go`, `GET /v1/admin/planning/*`, `/planning` page | **shipped** |
| **P2** | Tenant supplier planning UI | Supplier portal + native (full PX91) | `admin-portal/app/supplier/*` — MEIO, PlanningBrain, confidence, seasonal, EKG | **pending** |
| **P3** | Platform IBP + cross-supplier collaboration | N/A (deferred) | Executive scenario library, tenant rollup | **pending** |
| **P4** | Federated EKG + downstream role parity | Warehouse/factory/retailer confidence wired | Same surfaces per tenant in admin-portal | **pending** |

**P2 recommended next slice:** Port pegasusX supplier planning APIs (`/meio`, `/planning/*`, `/knowledge-graph`) to pegasus `supplierplanningroutes` + admin-portal supplier planning screens. Driver/payload rows remain execution-only.

**Pegasus backend (P1 shipped):**
- `GET /v1/admin/planning/baseline` — federated baseline rollup
- `GET /v1/admin/planning/meio` — MEIO stub per tenant
- `GET /v1/admin/planning/knowledge-graph` — EKG federation
- `GET /v1/admin/planning/control-tower` — zone override rollup

## 4. Contract Compatibility
PegasusX maintains structural parity with the Pegasus reference so that JSON keys, event names, claim fields, state enums, and route families stay migratable. Any divergence must be tracked in `parity-ledger.md`.
