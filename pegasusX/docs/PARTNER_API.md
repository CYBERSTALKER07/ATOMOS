# Partner Integration Layer (Gate 3 / §8.9)

**Status:** Wave 1 + Wave 2A wired in monorepo (2026-08-06). Apply migrations:

- `20260805_partner_integration_layer.ddl` (keys + webhooks)
- `20260806_partner_exports.ddl` (export jobs + SFTP configs)

Otherwise e2e may print `PX_E2E_PARTNER_SKIPPED` / `PX_E2E_PARTNER_EXPORT_SKIPPED`.

**OpenAPI (partner):** [`contracts/partner.openapi.yaml`](../contracts/partner.openapi.yaml) · **OpenAPI (human JWT core):** [`contracts/jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml) — [`JWT_CORE_OPENAPI.md`](./JWT_CORE_OPENAPI.md)

## Machine identity

- Keys: `pxk_<prefix>_<secret>` — bcrypt hashed at rest (`PartnerApiKeys`)
- Issue (human JWT): `POST /v1/admin/partner-keys` or `POST /v1/supplier/partner-keys`
- Auth: `Authorization: Bearer pxk_…` on `/partner/v1/*`
- Rate limit actor: `partner:<KeyId>` (prefix before verify)
- `LOAD_BOOTSTRAP_SECRET` never exempts `/partner/*`
- Default scopes include `exports:read` for supplier/retailer keys

## Partner HTTP

| Method | Path | Scope |
|--------|------|-------|
| POST | `/partner/v1/orders` | `orders:write` + Idempotency-Key |
| GET | `/partner/v1/orders/{id}` | `orders:read` (IDOR fail-closed) |
| GET | `/partner/v1/catalog` | `catalog:read` |
| GET | `/partner/v1/inventory/availability` | `inventory:read` |
| GET | `/partner/v1/webhooks` | `webhooks:manage` |
| POST | `/partner/v1/webhooks` | `webhooks:manage` |
| DELETE | `/partner/v1/webhooks/{id}` | `webhooks:manage` |
| POST | `/partner/v1/webhooks/{id}/ping` | `webhooks:manage` |
| GET | `/partner/v1/webhooks/dead-letter` | `webhooks:manage` |
| POST | `/partner/v1/webhooks/dead-letter/{attemptID}/replay` | `webhooks:manage` |
| POST | `/partner/v1/exports` | `exports:read` |
| GET | `/partner/v1/exports` | `exports:read` |
| GET | `/partner/v1/exports/{jobID}` | `exports:read` |

Orders go through `order.Service.Create` (reserve / credit / pricing). Tenant is bound on the key (`RETAILER` or `SUPPLIER`), not the seed constructor.

## Outbound webhooks

- Kafka consumer `void-partner-webhooks` enqueues allowlisted events (`ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `CLAIM_FILED`, `PAYMENT_CLEARED`)
- Delivery worker HMAC-SHA256: `X-Pegasus-Signature: sha256=<hex>` over `{timestamp}.{body}`
- Headers: `X-Pegasus-Timestamp`, `X-Pegasus-Event-Id`, `X-Pegasus-Event-Type`
- Idempotent on `(SubscriptionId, EventId)`; dead-letter after 8 attempts
- Replay resets `AttemptCount` to 0 and sets `Status=PENDING` with `NextRetryAt=now`
- List responses omit signing secret (show `secret_prefix` only)

## Bulk export + SFTP (Wave 2A)

- Resources: `orders` | `invoices` | `inventory` | `ledger` | **`journals`** (1C-friendly AR + payment ledger)
- Formats: `csv` | `json` | **`xml`** (journals flat `<Journal>` dialect; other resources still use csv/json primarily)
- Caps: 50k rows, max 90-day window; tenant filter fail-closed
- Object path: `partner-exports/{tenantType}/{tenantId}/{jobId}.{ext}`
- Storage: GCS when configured, else `PARTNER_EXPORT_LOCAL_ROOT`, else temp dir
- `GET` succeeded jobs include a short-lived `download_url`
- Optional SFTP push when `PARTNER_SFTP_ENABLED=true` and active `PartnerSftpConfigs` (secret via GSM `SecretRef`, never plaintext in Spanner)
- Without SFTP: job still `SUCCEEDED` with `SftpStatus=SKIPPED`

### Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `PARTNER_EXPORTS_ENABLED` | on | Gate export APIs + worker |
| `PARTNER_EXPORT_LOCAL_ROOT` | empty | Local filesystem root for objects |
| `PARTNER_SFTP_ENABLED` | off | Upload after successful write |
| `PARTNER_SFTP_SECRET_<REF>` | — | Dev/SSMR secret material for a `SecretRef` |

### JWT convenience (supplier portal)

| Method | Path |
|--------|------|
| GET/POST | `/v1/supplier/partner-keys` |
| POST | `/v1/supplier/partner-keys/{keyID}/revoke` |
| GET/POST/DELETE | `/v1/supplier/partner-webhooks…` |
| GET | `/v1/supplier/partner-webhooks/dead-letter` |
| POST | `/v1/supplier/partner-webhooks/dead-letter/{attemptID}/replay` |
| GET/POST | `/v1/supplier/partner-exports…` |
| GET/PUT | `/v1/supplier/partner-sftp` |

Portal UI: Settings → Integrations.

## SSMR markers

Wave 1: `PX_E2E_PARTNER_KEY_AUTH_OK`, `PX_E2E_PARTNER_ORDER_CREATE_OK`, `PX_E2E_PARTNER_IDOR_DENIED`, `PX_E2E_WEBHOOK_DELIVERED_OK` (or `PX_E2E_PARTNER_SKIPPED` if tables missing).

Wave 2A: `PX_E2E_PARTNER_EXPORT_OK` / `_SKIPPED`, `PX_E2E_WEBHOOK_REPLAY_OK` / `_SKIPPED`.

**1C journals:** resource `journals` maps `ArLedgerEntries` + `PaymentLedgerEntries` to debit/credit lines (CSV/JSON/XML). See [`PARTNER_JOURNALS_1C.md`](./PARTNER_JOURNALS_1C.md). Marker: `PX_E2E_PARTNER_JOURNALS_OK` / `_SKIPPED`.

## EDI-lite (Wave 2B)

ORDERS inbound + ORDRSP/DESADV/INVOIC outbound over SFTP/local root. See [`PARTNER_EDI.md`](./PARTNER_EDI.md).

| Method | Path | Scope |
|--------|------|-------|
| GET | `/partner/v1/edi/documents` | `exports:read` |
| GET | `/partner/v1/edi/documents/{id}` | `exports:read` |
| POST | `/partner/v1/edi/documents/{id}/replay` | `exports:read` |

Markers: `PX_E2E_PARTNER_EDI_ORDERS_OK` / `_SKIPPED`, `PX_E2E_PARTNER_EDI_ORDRSP_OK` / `_SKIPPED`.

## Still open

AS2, certified EDIFACT, configurable 1C chart-of-accounts, certified 1C exchange package. JWT **core** OpenAPI is WIRED ([`jwt-core.openapi.yaml`](../contracts/jwt-core.openapi.yaml)); residual is full-platform coverage + SDK replace of `@pegasusx/api-client`.
