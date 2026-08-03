# Privacy & Multi-Tenant Isolation

## Rules

| Viewer | Scope |
|--------|--------|
| Supplier | Own supplier_id only |
| Warehouse/Factory staff | Home node + supplier |
| Retailer | Own retailer_id |
| Driver | Assigned manifests/orders |
| Payload | Warehouse-scoped loads |
| Platform admin | Cross-tenant with explicit audit |

## Control tower

Role-scoped views of inventory, exceptions, fiscal, credit exposure. No cross-supplier leakage in JWT claims or list queries (IDOR tests mandatory).
