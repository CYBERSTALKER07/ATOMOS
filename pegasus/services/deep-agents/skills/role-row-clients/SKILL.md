---
name: role-row-clients
description: Per-role features, business duties, and app parity across Android/iOS/portal/desktop/terminal.
---

# Role-row clients + feature parity

**SoT:** `docs/FEATURES_BY_APP_ROLE.md`, `docs/ROLE_ROW_PARITY_MATRIX.md`,
`docs/ECOSYSTEM_FEATURES_BY_ROLE.md`, gap register P1-16..18

## Matrix (required surfaces)

| Role | Clients | Business job |
|------|---------|--------------|
| Supplier | portal (web+Tauri), Android, iOS | Sell, credit, dispatch, settle, integrations |
| Retailer | desktop (Next+Tauri), Android, iOS | Buy, receive, Retail OS, AR/HQ |
| Driver | Android, iOS | Last mile + doorstep money |
| Warehouse | portal (web+Tauri), Android, iOS | Pick/pack/dispatch/WMS |
| Factory | portal (web+Tauri), Android, iOS | Load outbound → payload |
| Payload | terminal (Expo primary), Android, iOS | Seal / inject / exceptions |
| Platform admin | admin-portal (web, thin) | Tenants, flags, audit |

JWT roles (`auth/claims.go`): ADMIN, RETAILER, DRIVER, PAYLOAD, FACTORY,
FACTORY_ADMIN, FACTORY_DRIVER, WAREHOUSE_ADMIN, WAREHOUSE (+ PLATFORM_ADMIN).

## Parity rules

1. Shared contracts first: `packages/types`, `packages/api-client`, then clients.
2. Feature not done until **all row clients** use the same contract (or deferred in gap register).
3. Desktop stack: **Next + Tauri 2** — do not recommend Electron without hard blocker.
4. Payload natives historically thin — terminal is SoT until natives catch up.
5. Cross-role loops count: factory→payload, supplier credit→retailer, driver location→dispatch.

## Per-role feature track (audit against shells + routes)

### Retailer
- Must: catalog, cart, multi-supplier checkout, orders, tracking, credit profile, POS/stock/shifts, claims
- Parity holes: **AR invoices + HQ multi-store desktop-only** (P1-17); `/hq`/`/credit` weak on mobile
- Nav: `RetailerShell.tsx`, `RetailerNavigation.kt`, iOS `Screens/`

### Supplier
- Must: orders/vet, dispatch/fleet, catalog/pricing, credit program, treasury/ledger, claims, integrations
- Parity holes: control-tower / planning / tax-regimes / playbooks often portal-only
- Nav: `SupplierShell.tsx`, `SupplierSection.kt`

### Warehouse
- Must: dispatch, manifests, WMS (lots/waves/counts when flags on), returns/claims, replenishment
- Parity holes: Control Tower portal-only; some floor via transfer actions vs dedicated pick-wave nav
- Nav: `WarehouseShell.tsx`, `WarehouseSection.kt`

### Driver
- Must: manifest, arrive, QR, partial offload, shop-closed, cash, credit leave, fiscal retry, offline sync
- Parity: Android ↔ iOS; **no portal** (by design)
- Nav: `DriverRoutes`, iOS driver Views

### Factory
- Must: loading bay, seal/dispatch, transfers, supply requests, fleet/staff
- **P1-18:** factory loading bay → payload terminal loop broken (payload ignores factory manifests)
- Nav: `FactoryShell.tsx`, `FactorySection.kt`

### Payload
- Must: trucks/orders/manifests board, seal, inject, reassign, inbound returns
- Parity: terminal SoT; natives thin; must see factory+supplier manifests for Class A loop
- Nav: `payload-terminal`, `PayloadRoot.kt`, iOS inbound

### Platform admin
- Must: tenants, money flags dual-control, audit
- **P1-16:** missing UI for product-match, partner keys/AS2/SFTP, dunning, billing, analytics

## How to score a finding

- Cite **route package** + **missing client nav/screen** (shell href / Section enum / route id)
- Attach `gap_id` when known (P1-16, P1-17, P1-18)
- Class C if UI island; Class B if API without clients; Class A only if all row clients consume

## Nav evidence

- Shells: `*Shell.tsx`
- Android: `*Section.kt`, `*Navigation.kt`, `DriverRoutes`
- iOS: `ContentView` / `Screens/`
