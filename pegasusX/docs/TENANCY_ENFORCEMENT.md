# Tenancy enforcement (G4.A)

## PreferTenant

`auth.PreferTenantSupplierID(ctx, fallback)`:

1. TenantContext supplier  
2. JWT claim supplier  
3. Seed fallback **only if** `SeedFallbackAllowed()`

## Fail-closed defaults

| Env | Tenant enforced | Seed fallback |
|-----|-----------------|---------------|
| `PEGASUSX_ENV=production` | yes | **no** unless `ALLOW_SEED_FALLBACK=true` |
| `PEGASUSX_ENV=ssmr` | yes | **no** unless break-glass |
| local/dev | no (unless `TENANT_CONTEXT_ENFORCED`) | yes |

Flags:

- `TENANT_CONTEXT_ENFORCED` — force on/off  
- `ALLOW_SEED_FALLBACK` — break-glass demo seed  

Authenticated callers without supplier never receive seed when enforced.

## Body scope

`auth.RejectBodyScopeOverrides` rejects mutator bodies that override JWT
`supplier_id` / home-node fields.

## Memory repos

Production never allows in-memory domain repos (`ALLOW_MEMORY_FALLBACK` only
local/SSMR with `REQUIRE_INFRA_ADAPTERS=false`).
