# Partner journals export (1C-friendly)

Gate-3 accounting export via the existing partner export job pipeline.

## Resource

| Field | Value |
|-------|--------|
| `resource` | `journals` |
| `format` | `csv` \| `json` \| `xml` |
| Scope | `exports:read` |
| Window | same as other exports (default 7d, max 90d) |

## Sources

| `source` | Table | Notes |
|----------|--------|--------|
| `ar` | `ArLedgerEntries` ⋈ `ArInvoices` | currency, order_id, aging_bucket |
| `payment` | `PaymentLedgerEntries` | gateway settlement lines |

AR rows are preferred until `MaxExportRows` (50k); payment fills the remainder.

## Chart of accounts

Default 1C-style accounts (when no tenant map):

| Event | Debit | Credit |
|-------|-------|--------|
| AR `OPEN` | `62.01` | `90.01` |
| AR `PAYMENT` | `51.01` | `62.01` |
| Payment capture/settle | `51.01` | `62.01` |
| Payment refund / chargeback / void | `62.01` | `51.01` |

**Configurable (2026-08-06):** per-tenant override in `PartnerCoaMaps` (`account_ar`, `account_revenue`, `account_bank_cash`).

| Method | Path | Auth |
|--------|------|------|
| GET/PUT | `/partner/v1/coa` | Partner key `exports:read` |
| GET/PUT | `/v1/supplier/partner-coa` | Supplier ADMIN JWT |
| GET/PUT | `/v1/admin/partner-coa` | Admin / retailer JWT |

Empty fields on PUT resolve to platform defaults. Optional env defaults: `PARTNER_COA_AR`, `PARTNER_COA_REVENUE`, `PARTNER_COA_BANK`.

Migration: `apps/backend-go/schema/migrations/20260806_partner_coa.ddl`.

Portal: Settings → Integrations → **1C chart of accounts**.

## Columns

`entry_date`, `source`, `entry_id`, `entry_type`, `debit_account`, `credit_account`, `amount_minor`, `currency`, `supplier_id`, `retailer_id`, `invoice_id`, `order_id`, `aging_bucket`, `gateway`, `memo`

## XML shape

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Journal version="1" dialect="1c">
  <Entry entry_date="..." source="ar" debit_account="62.01" credit_account="90.01" amount_minor="..." ... />
</Journal>
```

## Related

- Thin AR dump (no CoA): resource `ledger`
- API overview: [`PARTNER_API.md`](./PARTNER_API.md)
