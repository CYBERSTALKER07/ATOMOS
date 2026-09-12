# Phase 2 completion — Enterprise integration (2026-08-11)

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**Gate:** `make phase2-gate` → **`phase2-gate-ok`** (re-proved 2026-08-11, Spanner emulator `localhost:9010`)  
**Status:** Wired (backend / OpenAPI / codecs); Phase 6 cert items remain residuals

## Proof (2026-08-11)

| Step | Result |
|------|--------|
| [1/6] Phase-1 regression | `phase1-gate-ok` |
| [2/6] Partner OpenAPI (master-data + idempotency) | `partner-openapi-gate-ok` |
| [3/6] EDI codecs (CONTRL/APERAK/ORDRSP/INVOIC) | PASS |
| [4/6] GS1 DataMatrix scaffolding | PASS |
| [5/6] SFTP host-key + webhook rotate | PASS |
| [6/6] CommerceML reference converter | `commerceml-ref-ok` |

## Shipped

| Area | What |
|------|------|
| Master-data API | `PUT /partner/v1/catalog/products`, `/catalog/prices`, `/inventory/stock` + scopes + idempotency |
| Webhooks | Expanded allowlist; `POST …/rotate-secret` |
| POS demand | `POST /partner/v1/demand/pos-feed` |
| EDI ACKs | CONTRL + APERAK builders; enqueue on inbound ORDERS/ORDRSP/INVOIC |
| Inbound EDI | ORDRSP + INVOIC parse/record; filename + UNH routing |
| SFTP | `HostKeySHA256` persistence + strict pin callback |
| GS1 DataMatrix | AI element string + placeholder modules + ZPL `^BX` |
| Journals | `vat_minor`, `credit_note_id`, credit-note legs |
| OpenAPI | `contracts/partner.openapi.yaml` v1.5.0 |
| SDK | `scripts/gen_partner_sdk.sh` + `sdk/partner/README.md` |
| CommerceML | `docs/COMMERCEML_EXCHANGE.md` + `scripts/commerceml_import_ref.py` |
| Manifests | PARTNER_* in base configmap; staging SFTP/AS2 on + strict hostkey; SSMR AS2/SFTP on |

## Explicit non-goals still open (Phase 6 / cert)

- Certified ECC200 DataMatrix library (placeholder modules remain)
- Drummond AS2 / certified EDIFACT
- Full CommerceML bidirectional 1C package
- Synthetic 5k-SKU SSMR chain e2e script (gate covers unit/OpenAPI/codec path)
- Owner partner sandbox keys

## Next deep-dive

**Phase 6** (decision-gated marketplace/cert) or analytics column tenancy / client residuals.
