# Domain 6.2 — Per-Role Tenant Wiring Audit

Date: 2026-08-12 (UTC+5) · Read-only audit + targeted fail-closed hardening.

## Reference pattern
`auth/tenant.go` `PreferTenantSupplierID` + the canonical fail-closed list in
`analytics/handlers.go` (`routePerfListStmt` injects `WHERE FALSE` on empty
supplier). `main.go` mounts `auth.RequireTenant(cfg.TenantContextEnforced)` —
when ON (ssmr/production) tenant-less authenticated requests are 401'd at
middleware, which masks many handler-level gaps. The gaps below are live when
enforcement is off (dev/staging/default), when a handler discards request scope,
or when a route is mounted without auth.

## Summary
~130 list/detail handlers audited across 13 packages:
- ~75 OK (fail-closed via `PreferTenantSupplierID` or equivalent actor/org/warehouse scoping)
- ~53 GAP
- 2 N/A (intentionally public)

## Systemic root causes (fix these to clear whole classes)
1. `supplier/session_scope.go` — `scopedSupplierID` falls back to static seed
   `s.supplierID` when claims/TenantContext lack supplier → defeats downstream
   fail-closed checks (~15 handlers).
2. `payment/service.go` `resolvePaymentSupplierScope` — accepts `?supplier_id=`
   when resolved scope is empty; ledger repos drop the WHERE clause on empty
   (4 handlers).
3. `payload/repository_spanner.go` — `Hydrate` ignores request tenant; `RunTx`
   uses static construction-time supplierID (2+ handlers).
4. `auth/warehouse_ops_scope.go` / `auth/factory_scope.go` — RoleAdmin passthrough
   lets supplier-less admins read any node via query/path params.
5. `order` admin-bypass idiom `claims.SupplierID == "" ||` in
   `assertTimelineAccess`/`authorizeReceiptParty`/`reporterAuthorizedForOrder`
   (6 handlers).
6. `factory` in-memory demo state shared across all tenants (7 handlers).
7. `demandroutes` mounted without auth on the root router; seed-tenant fallback
   (3 handlers).

## Highest-severity individual gaps (no tenant filter at all)
- `payment.HandleListPayers` — cross-tenant payer list even under enforcement.
- `driver.HandleOrderGet` — no identity/tenant check (`WHERE OrderId=@oid` only).
- `driver.HandleGetDriver` / `HandleGetVehicle`, `warehouse.HandleGetWarehouse`,
  `factory.HandleGetFactory` — CRUD detail IDORs.
- `creditnote.HandleOrderLines` / `HandleListReverseTasks` — no tenant filter;
  client-supplied warehouse param.
- `supplier.HandleAIRecommendations` — hardcoded seed supplier.

## Fixes applied this pass
(See commit + the per-file diffs. Targeted the contained, high-severity,
low-risk gaps; systemic auth-middleware and demo-state refactors are tracked as
follow-ups because they require careful rollout behind TenantContextEnforced.)
