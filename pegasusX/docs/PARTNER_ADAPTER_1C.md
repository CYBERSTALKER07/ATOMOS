# 1C adapter pack (G5.B)

**Package:** `apps/backend-go/partner/adapters/onec`  
**Status:** Subset import + journals dialect — **not** a certified 1C partner product.

## Import

`POST /partner/v1/adapters/onec/import` (`catalog:write`)

| Content-Type | Body |
|--------------|------|
| `application/json` | `{ "external_doc_id", "products":[{external_id,name,barcode,price_minor,currency}] }` |
| `application/xml` | Minimal CommerceML-like `<Каталог><Товары><Товар>…` |

Idempotent on `(tenant, external_doc_id)`. Maps to existing product/price upsert.

## Export

Use existing journals export:

```http
POST /partner/v1/exports
{ "resource": "journals", "format": "xml" }
```

Dialect label `1c` + CoA maps: [`PARTNER_JOURNALS_1C.md`](./PARTNER_JOURNALS_1C.md).

## SAP residual

See [`apps/backend-go/partner/adapters/sap/README.md`](../apps/backend-go/partner/adapters/sap/README.md).
