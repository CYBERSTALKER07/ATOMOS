# CommerceML 2.x exchange package (Phase 2 design)

**Status:** Design + reference import path (2026-08-11). Not a certified 1C exchange package.
Runtime integration today remains partner REST + EDI-lite + journals CSV/XML
([`PARTNER_API.md`](./PARTNER_API.md), [`PARTNER_EDI.md`](./PARTNER_EDI.md),
[`PARTNER_JOURNALS_1C.md`](./PARTNER_JOURNALS_1C.md)).

## Goal

Let a retail chain running 1C:Enterprise push catalog/prices/stock and pull
commercial journals without re-keying into the supplier portal.

## Package layout (CommerceML 2.x-shaped)

```
commerceml/
  classifier.xml          # categories (optional)
  import.xml              # goods / offers (products + prices)
  offers.xml              # prices only (delta)
  rests.xml               # stock balances
  orders.xml              # optional outbound ORDERS mirror
  documents/
    journals.csv|xml      # PegasusX journals export (1C CoA mapped)
```

Mapping to partner machine API (preferred over raw file drop for idempotency):

| CommerceML artifact | Partner API |
|---------------------|-------------|
| `import.xml` / goods | `PUT /partner/v1/catalog/products` |
| `offers.xml` | `PUT /partner/v1/catalog/prices` |
| `rests.xml` | `PUT /partner/v1/inventory/stock` |
| POS sell-through | `POST /partner/v1/demand/pos-feed` |
| Journals | `POST /partner/v1/exports` `resource=journals` |

All mutating calls require `Idempotency-Key`. External IDs map to
`ProductId == external_id` (import-wizard convention).

## Reference import script

```bash
# From repo root — dry-run parses CommerceML XML and prints partner upsert payloads.
python3 scripts/commerceml_import_ref.py \
  --import path/to/import.xml \
  --offers path/to/offers.xml \
  --rests path/to/rests.xml \
  --out /tmp/partner-upserts.json
```

Apply with curl (or generated SDK):

```bash
curl -X PUT "$API/partner/v1/catalog/products" \
  -H "Authorization: Bearer $PXK" \
  -H "Idempotency-Key: cm-$(date +%s)-products" \
  -H "Content-Type: application/json" \
  --data @/tmp/products.json
```

## Journals enrichment (1C-friendly)

Journal rows now include:

- `vat_minor` — VAT breakout when sourced from credit notes
- `credit_note_id` — returns / corrective legs
- `entry_type=CREDIT_NOTE` with CoA debit revenue / credit AR

## Still open (Phase 6 certified package)

- Full CommerceML bidirectional document cycle inside 1C processing
- Certified EDIFACT + Drummond AS2
- Direct 1C COM/HTTP service posting
