# Project: Pegasus Full-Stack Hardening, UX/A11y Remediation & Domain Parity

## Architecture
- **Dual-Engine Core Ecosystem**:
  - `pegasusX` (Global Multi-Tenant Cloud): Google Cloud Spanner (interleaved tables, multi-tenant partitioning by `SupplierId`), Apache Kafka event bus (Strimzi HA, per-entity hashing, fair interleaving), double-entry ledger idempotency with atomic Spanner transactions and 64-bit integer tiyin minor units.
  - `pegasus.x` (Sovereign National Core): PostgreSQL 16 (`pgx/v5`, 78 transactional migrations), Redis 7 (Streams outbox relay with `FOR UPDATE SKIP LOCKED` polling, Pub/Sub WebSocket fanout), strict non-contamination (ZERO Spanner, ZERO Kafka).
- **16 Desktop & Web Frontend Applications**:
  - `pegasus` (5 apps): `admin-portal`, `factory-portal`, `payload-terminal`, `retailer-app-desktop`, `warehouse-portal`
  - `pegasus.x` (5 apps): `payloader-tablet`, `retailer-desktop`, `supplier-desktop`, `telegram-miniapp`, `warehouse-desktop`
  - `pegasusX` (6 apps): `admin-portal`, `factory-portal`, `payload-terminal`, `retailer-app-desktop`, `supplier-portal`, `warehouse-portal`
- **Design Tokens & UI Standards**:
  - V.O.I.D Tactical Control Tower design system.
  - Lucide SVG icons (`lucide-react`) for standard UI controls.
  - Fluid responsive layouts (elimination of fixed `max-w-[1600px]` constraints).
  - Semantic `<button type="button">` with native keyboard accessibility (Tab, Enter, Space).
  - Explicit `<input>` labeling (`id` and `aria-label` leading attributes).

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | F1: Form Input Label Pairing | Add explicit `id` and `aria-label` to all 304 unlabeled `<input>` elements across 16 desktop/web apps | M1 | Survey 1 (UX/A11y) |
| 2 | F2: Accessible Interactive Controls | Convert 36 clickable `<div>` elements across 29 files into semantic `<button type="button">` with keyboard navigation | M1 | Survey 1 (UX/A11y) |
| 3 | F3: Lucide SVG Icon Migration | Replace raw unicode emoji glyphs across 75 files with standard `lucide-react` SVG icon components | M1 | Survey 1 (UX/A11y) |
| 4 | F4: Fluid Responsive Containers | Replace fixed pixel containers (`max-w-[1600px]`) with responsive `max-w-7xl` layout wrappers | M1 | Survey 1 (UX/A11y) |
| 5 | F5: Image Alt Attribute | Fix missing `alt` attribute on image components identified in audit | M1 | Survey 1 (UX/A11y) |
| 6 | F6: pegasus.x Non-Contamination | Enforce and certify zero references to `cloud.google.com/go/spanner` and `kafka-go` inside `pegasus.x/` | M2 | Survey 2 (Arch Boundary) |
| 7 | F7: pegasusX Spanner & Kafka Verification | Certify Spanner DDL compliance (19 interleaved tables, `SupplierId` partitioning) and Strimzi Kafka HA event bus | M2 | Survey 2 (Arch Boundary) |
| 8 | F8: Financial Ledger Idempotency | Certify 64-bit integer tiyin minor unit arithmetic and unique index idempotency across both payment engines | M2 | Survey 2 (Arch Boundary) |
| 9 | F9: Field Sales Role & Payload Reconciliation | Reconcile `apps/field-sales-mobile` in `pegasus.x` (`RoleFieldSales`, order payload `{ sku_id, ordered_qty, list_price_minor }`, cash payment endpoint) | M3 | Survey 3 (Domain Parity) |
| 10 | F10: Outbox DLQ Management in Sovereign Core | Implement Outbox Dead-Letter Queue inspection and replay API in `pegasus.x/backend` for operational symmetry with `pegasusX` | M3 | Survey 3 (Domain Parity) |
| 11 | F11: Order State Machine UI Harmonization | Harmonize order status mapping and badge presentation across desktop apps for both 18-state and 12-state flows | M3 | Survey 3 (Domain Parity) |
| 12 | F12: Cross-Role Operational Parity Certification | Formally document and certify state machine and workflow parity across all 8 user roles | M3 | Survey 3 (Domain Parity) |
| 13 | F13: UX Audit Health Score Verification | Re-execute `audit_scanner.py` and `generate_report.py`; verify score $\ge 92/100$ (target 95/100) with 0 critical findings | M4 | Survey 1 & Verification |
| 14 | F14: TypeScript & Full-Stack Zero-Regression Gate | Run `tsc --noEmit` / `pnpm check-types` and backend Go test suites with zero failures | M4 | Verification |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Desktop & Web UX Remediation & A11y Hardening | F1, F2, F3, F4, F5: Remediate all 418 UX/a11y defects across all 16 desktop/web apps | None | DONE |
| M2 | Architectural Boundary & Data Engine Verification | F6, F7, F8: Verify and enforce strict non-contamination, Spanner/Kafka compliance, Postgres/Redis outbox, and financial idempotency | None | DONE |
| M3 | Cross-Role Domain Parity & Operational Reconciliation | F9, F10, F11, F12: Reconcile Field Sales role, Outbox DLQ replay API, and order status machine parity across 8 roles | M2 | DONE |
| M4 | Comprehensive Full-Stack Verification & UX Score Gate | F13, F14: Re-scan UX audit report, run TypeScript check, Go tests, static linting, and final gate | M1, M2, M3 | DONE |

## Interface Contracts
### Order Creation Payload Contract
- `CreateOrderRequest`:
  - `retailer_id`: string (UUID)
  - `supplier_id`: string (UUID)
  - `items`: array of `{ sku_id: string, ordered_qty: int, list_price_minor: int64 }`
  - `payment_method`: enum (`CASH`, `BANK_TRANSFER`, `CREDIT_LINE`)

### Field Sales Authentication & Claims
- `RoleFieldSales`: string = `"field_sales"`
- Claims: `{ user_id, supplier_id, role: "field_sales", agent_id }`

### Outbox DLQ Admin Endpoint Contract
- `GET /v1/admin/ops/dead-letters` -> list of dead letter events
- `POST /v1/admin/ops/dead-letters/replay` -> `{ dead_letter_ids: string[] }` -> `{ replayed_count: int }`

## Code Layout
- `pegasus/apps/`: `admin-portal`, `factory-portal`, `payload-terminal`, `retailer-app-desktop`, `warehouse-portal`
- `pegasus.x/apps/`: `payloader-tablet`, `retailer-desktop`, `supplier-desktop`, `telegram-miniapp`, `warehouse-desktop`, `field-sales-mobile`
- `pegasus.x/backend/`: `internal/api/`, `internal/db/`, `internal/outbox/`, `internal/order/`, `internal/fleet/`, `internal/payload/`
- `pegasusX/apps/`: `admin-portal`, `factory-portal`, `payload-terminal`, `retailer-app-desktop`, `supplier-portal`, `warehouse-portal`
- `pegasusX/apps/backend-go/`: `schema/spanner.ddl`, `outbox/`, `order/`, `payment/`, `ar/`, `factory/`
