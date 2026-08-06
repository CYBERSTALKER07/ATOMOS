# Partner AS2 transport (§8.9)

**Status:** Wired as a third transport for existing EDI-lite bytes (ORDERS / ORDRSP / DESADV / INVOIC). **Not Drummond-certified.** Certified EDIFACT and the 1C CommerceML exchange package remain open.

Apply migration `apps/backend-go/schema/migrations/20260806_partner_as2.ddl` (`PartnerAs2Configs`).

## What it does

| Direction | Behavior |
|-----------|----------|
| Inbound | `POST /partner/v1/as2` receives AS2 MIME; unwraps PKCS#7 (or plain when insecure); feeds EDI bytes into existing `IngestORDERSBytes` |
| Outbound | After local write (+ optional SFTP), posts the same body to `PartnerUrl` when AS2 is enabled |
| MDN | Sync multipart/report MDN with `Received-Content-MIC` (SHA-256) |

Codecs in `partner/edi/` are unchanged. SFTP / `PARTNER_EDI_LOCAL_ROOT` remain supported.

## Crypto profile

- Sign-then-encrypt via `go.mozilla.org/pkcs7`
- SHA-256 MIC
- Sync MDN only (no async MDN / CEM)

## Config

`PartnerAs2Configs` (PK `TenantType`, `TenantId`):

| Field | Notes |
|-------|--------|
| `As2Enabled` | Gate send/receive for tenant |
| `OurAs2Id` / `PartnerAs2Id` | AS2-To / AS2-From |
| `PartnerUrl` | Outbound HTTPS endpoint |
| `OurCertSecretRef` / `OurKeySecretRef` / `PartnerCertSecretRef` | PEM via `LoadSecretRef` / GSM |
| `SignRequired` / `EncryptRequired` | Defaults true |

## Env

| Env | Default | Meaning |
|-----|---------|---------|
| `PARTNER_AS2_ENABLED` | off | Gate AS2 receive/send |
| `PARTNER_AS2_INSECURE_PLAIN` | off | SSMR only: accept/send unsigned `application/edifact` |
| `PARTNER_AS2_SECRET_<REF>` | — | PEM material for a SecretRef |

Never enable `PARTNER_AS2_INSECURE_PLAIN` on production overlays.

## APIs

| Method | Path | Auth |
|--------|------|------|
| POST | `/partner/v1/as2` | None (AS2-From/To + PKCS7) |
| GET/PUT | `/partner/v1/as2/config` | Partner key `exports:read` |
| GET/PUT | `/v1/supplier/partner-as2` | Supplier ADMIN JWT |
| GET/PUT | `/v1/admin/partner-as2` | Admin / retailer JWT |

Portal: Settings → Integrations → **AS2 transport**.

## SSMR markers

`PX_E2E_PARTNER_AS2_ORDERS_OK` / `_SKIPPED`, `PX_E2E_PARTNER_AS2_ORDRSP_OK` / `_SKIPPED`  
(require `PARTNER_AS2_ENABLED` + `PARTNER_AS2_INSECURE_PLAIN`).

## Related

- EDI-lite messages: [`PARTNER_EDI.md`](./PARTNER_EDI.md)
- Partner overview: [`PARTNER_API.md`](./PARTNER_API.md)
