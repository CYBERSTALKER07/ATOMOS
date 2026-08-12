---
name: security-tenancy
description: Tenant scope, detail IDORs, PLATFORM_ADMIN break-glass, JWT session revocation gaps.
---

# Security & tenancy

## Tenant law

- `RequireTenant` in enforced envs; `PreferTenantSupplierID` for scoped queries
- `PLATFORM_ADMIN` is cross-tenant break-glass — exempt from RequireTenant (P0-3 intentional)
- Never honor `?supplier_id=` when claim scope is empty (seed fallback anti-pattern)

## Open follow-through (re-verify)

Detail IDORs called out in gap register security section, e.g.:

- `payment.HandleListPayers`
- `driver.HandleGetDriver` / `HandleGetVehicle`
- `warehouse.HandleGetWarehouse`, `factory.HandleGetFactory`
- creditnote / supplier AI seed fallbacks

## Auth gaps

- JWT session revocation / denylist still open (P1-11)
- MFA for PLATFORM_ADMIN still open (P2)

## Evidence

`auth/tenant.go`, `auth/refresh.go`, gap register “Security follow-through”, platformadmin routes
