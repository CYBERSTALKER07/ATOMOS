# Partner EDI-lite (Gate 3 / §8.9 Wave 2B)

**Status:** Wired over SFTP / local root (2026-08-06). Not a certified EDIFACT or AS2 implementation.

**Dialect:** UNA-style segment files (`UNA:+.? '`) with `UNB` / `UNH` / `BGM` / `NAD` / `LIN` / `QTY` / `UNT` / `UNZ`. UTF-8, one message per file.

Apply migration `20260806_partner_edi.ddl` (extends `PartnerSftpConfigs`, adds `PartnerEdiDocuments`).

## Messages

| DocType | Direction | Trigger |
|---------|-----------|---------|
| ORDERS | IN | Drop file in inbound dir |
| ORDRSP | OUT | `ORDER_CREATED`; status → CONFIRMED / CANCELLED / REJECTED / BACKORDERED / SCHEDULED / … |
| DESADV | OUT | Status → `LOADED` or `IN_TRANSIT` — includes **CPS/PAC/GIN SSCC** from `ManifestShipUnits` when sealed |
| INVOIC | OUT | Status → `DELIVERED_ON_CREDIT`; also `PAYMENT_CLEARED` |

Inbound ORDERS maps to `order.Service.Create` with `order_source=PARTNER_EDI`. Geo from `LOC+DEL+lat:lng:h3` or retailer primary location.

### DESADV SSCC packing (EDI-lite)

When `ManifestShipUnits` exist for the order (minted at payload seal):

| Segment | Meaning |
|---------|---------|
| `RFF+PK:{manifestId}` | Manifest / packing reference |
| `CPS+1` / `PAC+n++CT` | Root packing node + unit count |
| `CPS+{i}+1` / `PAC+1++CT` / `GIN+BJ:{sscc}` | Child logistics unit (BJ = SSCC-18) |
| `GIN+BN:{gtin}` | Optional GTIN when present on the ship unit |

Still not certified EDIFACT / AS2.

## File naming

- Inbound: `ORDERS_{externalDocId}.edi` (or `*.ORDERS` / name containing `ORDERS` + `.edi`)
- Outbound: `{DocType}_{externalDocId}_{unix}.edi`
- After ingest, inbound files move to `archive/`

## Transport

Uses `PartnerSftpConfigs` (Wave 2A) with:

- `EdiEnabled`
- `InboundDir` / `OutboundDir` / `ArchiveDir` (defaults `inbound` / `outbound` / `archive`)

Live SFTP when `PARTNER_SFTP_ENABLED=true`. Local mode: `PARTNER_EDI_LOCAL_ROOT/{supplier|retailer}/{tenantId}/{inbound|outbound|archive}`.

## Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `PARTNER_EDI_ENABLED` | on | Gate EDI workers/APIs |
| `PARTNER_EDI_LOCAL_ROOT` | empty | Local drop/pickup root |
| `PARTNER_SFTP_ENABLED` | off | Live SFTP list/download/upload |

## APIs

- Partner key: `GET /partner/v1/edi/documents`, `GET …/{id}`, `POST …/{id}/replay` (`exports:read`)
- Supplier JWT: `/v1/supplier/partner-edi/documents…`
- SFTP PUT accepts `edi_enabled`, `inbound_dir`, `outbound_dir`, `archive_dir`

Portal: Settings → Integrations (EDI toggle + recent documents).

## SSMR markers

`PX_E2E_PARTNER_EDI_ORDERS_OK` / `_SKIPPED`, `PX_E2E_PARTNER_EDI_ORDRSP_OK` / `_SKIPPED`.

## Still open

AS2, full EDIFACT certification. GLN/SSCC/ZPL and DESADV GIN+BJ are wired — see [`GS1_LABELS.md`](./GS1_LABELS.md). 1C journals — see [`PARTNER_JOURNALS_1C.md`](./PARTNER_JOURNALS_1C.md).
