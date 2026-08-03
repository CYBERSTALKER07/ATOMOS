# Role-Row Parity Obligations

| Role | Clients | Notes |
|------|---------|-------|
| Supplier | portal, Android, iOS | Planning desk may be portal-first with mobile read |
| Retailer | desktop, Android, iOS | Cart session must all three |
| Warehouse | portal, Android, iOS | Dispatch + returns |
| Factory | portal, Android, iOS | Loading bay |
| Driver | Android, iOS | No portal |
| Payload | terminal, Android, iOS | Seal/inject |

## Deferral rule

If a client is deferred: write `context/parity-ledger.md` + ROLE_ROW matrix row with reason and owner.

## Contracts first

`packages/types` → `packages/api-client` → clients. Native Generated stubs if Quicktype wired.
