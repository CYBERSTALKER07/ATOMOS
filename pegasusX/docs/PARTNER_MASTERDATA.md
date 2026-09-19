# Partner master-data sync (G5.C)

## Existing

| Path | Entity |
|------|--------|
| `PUT /partner/v1/catalog/products` | Products (`external_id`, barcode≈GTIN) |
| `PUT /partner/v1/catalog/prices` | Prices |
| `PUT /partner/v1/inventory/stock` | Stock on-hand |

## G5 additions

| Path | Entity |
|------|--------|
| `PUT /partner/v1/masterdata/parties` | Parties (role, legal_name, GLN) + version conflict |
| `PUT /partner/v1/masterdata/plants` | External plant → warehouse map |
| `GET /partner/v1/masterdata/dlq` | Failed/conflict rows |

### Party conflict

If stored `version` > request `version` → `action=conflict` and DLQ row.

### GTIN

Product `barcode` field is the GTIN surface; `ValidGTIN` helper accepts EAN-8/12/13/14 digits.
