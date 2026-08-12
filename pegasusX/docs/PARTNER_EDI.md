# Partner EDI-lite (Gate 3 / §8.9 Wave 2B)

**Status:** Wired over SFTP / local root / **AS2** (2026-08-06). Not a certified EDIFACT implementation. AS2 is transport-only and **not Drummond-certified**.

**Dialect:** UNA-style segment files (`UNA:+.? '`) with `UNB` / `UNH` / `BGM` / `NAD` / `LIN` / `QTY` / `UNT` / `UNZ`. UTF-8, one message per file.

Apply migration `20260806_partner_edi.ddl` (extends `PartnerSftpConfigs`, adds `PartnerEdiDocuments`).

## Messages

| DocType | Direction | Trigger |
|---------|-----------|---------|
| ORDERS | IN | Drop file in inbound dir |
| ORDRSP | IN + OUT | Inbound: partner response recorded; Outbound: `ORDER_CREATED` / status transitions |
| DESADV | OUT | Status → `LOADED` or `IN_TRANSIT` — includes **CPS/PAC/GIN SSCC** from `ManifestShipUnits` when sealed |
| INVOIC | IN + OUT | Inbound: commercial invoice recorded; Outbound: `DELIVERED_ON_CREDIT` / `PAYMENT_CLEARED` |
| CONTRL | OUT | Syntax ACK after inbound ORDERS/ORDRSP/INVOIC (action 7/4) |
| APERAK | OUT | Application ACK after inbound processing (27/29 + optional FTX reason) |
| PRICAT | IN | Price catalog → partner price upsert |
| INVRPT | IN | Inventory report → partner stock upsert |
| SLSRPT | IN | Sales report — ledger + ACK |
| RECADV | IN | Receiving advice — ledger + ACK |
| ORDCHG | IN | Order change — ledger + ACK |
| DELFOR | IN | Delivery forecast — ledger + ACK |
| REMADV | IN | Remittance advice — ledger + ACK |

Inbound ORDERS maps to `order.Service.Create` with `order_source=PARTNER_EDI`. Geo from `LOC+DEL+lat:lng:h3` or retailer primary location. Successful/failed ORDERS emit CONTRL+APERAK via the outbound worker (local root or SFTP/AS2).

### DESADV SSCC packing (EDI-lite)

When `ManifestShipUnits` exist for the order (minted at payload seal):

| Segment | Meaning |
|---------|---------|
| `RFF+PK:{manifestId}` | Manifest / packing reference |
| `CPS+1` / `PAC+n++CT` | Root packing node + unit count |
| `CPS+{i}+1` / `PAC+1++CT` / `GIN+BJ:{sscc}` | Child logistics unit (BJ = SSCC-18) |
| `GIN+BN:{gtin}` | Optional GTIN when present on the ship unit |

Still not certified EDIFACT. **AS2 transport is Wired** — see [`PARTNER_AS2.md`](./PARTNER_AS2.md) (not Drummond-certified).

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
| `PARTNER_SFTP_ENABLED` | off (SSMR on) | Live SFTP list/download/upload |
| `PARTNER_SFTP_STRICT_HOSTKEY` | off (staging on) | Require `HostKeySHA256` pin |
| `PARTNER_AS2_ENABLED` | off (SSMR/staging on) | AS2 receive/send |

## APIs

- Partner key: `GET /partner/v1/edi/documents`, `GET …/{id}`, `POST …/{id}/replay` (`exports:read`)
- Supplier JWT: `/v1/supplier/partner-edi/documents…`
- SFTP PUT accepts `edi_enabled`, `inbound_dir`, `outbound_dir`, `archive_dir`, `host_key_sha256`

Portal: Settings → Integrations (EDI toggle + recent documents).

## SSMR markers

`PX_E2E_PARTNER_EDI_ORDERS_OK` / `_SKIPPED`, `PX_E2E_PARTNER_EDI_ORDRSP_OK` / `_SKIPPED`.

## Still open

Full EDIFACT / Drummond certification and certified 1C CommerceML package (Phase 6). EDI-lite breadth (PRICAT…REMADV) and AS2 MDN/MIC verify are Wired (W5).
