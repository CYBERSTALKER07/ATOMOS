# SAP adapter residual (G5.B)

**Status:** Design residual — G5 implements **1C-first** (`partner/adapters/onec`).

## Intended mapping (not implemented)

| Direction | SAP artifact | Pegasus target |
|-----------|--------------|----------------|
| IN | IDoc `ORDERS05` | `order.Service.Create` + external_doc_id |
| OUT | IDoc `ORDRSP` / `DESADV` / `INVOIC` | Partner EDI outbound |
| Master | MATMAS / DEBMAS | `/partner/v1/catalog/*` + parties |

## Why deferred

- UZ market priority is 1C CommerceML + journals CoA (already wired).
- Full IDoc/OData certification is a multi-quarter program (Gate residual).

## Extension point

Implement `adapters/sap` mirroring `onec.ImportBatch` → partner UpsertProducts /
order create, with `PartnerExternalDocuments` idempotency key = IDOCDOCNUM.
