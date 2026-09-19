# 4. Alignment with Systems Big Retailers and Suppliers Already Run

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


A mid-size Uzbek/CIS distributor runs **1C** for accounting and often stock; a retail chain runs 1C or SAP plus a POS plus possibly a WMS. They will not re-key orders, will not run their inventory in someone else's database, and will not accept a system whose numbers their accountant cannot recognize. This section assesses the machine-to-machine reality from code.

## 4.1 Current state of the machine-to-machine surface

| Channel | Status | Evidence | Honest limitation |
|---|---|---|---|
| Partner REST API | **WIRED-LIVE** — all 20 documented endpoints implemented: orders create/get, catalog, availability, webhooks ×6, exports ×3, CoA ×2, AS2 ×3, EDI docs ×3 | `apps/backend-go/partner/routes.go:17-46`; `contracts/partner.openapi.yaml` | **Order create ignores its contract-required `Idempotency-Key`** (`partner/handlers.go:37-63` vs `partner.openapi.yaml:494-499`) — ERP retries can double-create orders. No amend/cancel endpoints. Catalog is read-only for machines |
| Machine auth | **WIRED-LIVE** — bcrypt `pxk_` API keys + OAuth2 `client_credentials` issuing 15-min HS256 JWTs, scope intersection, live revoke, per-key rate-limit actor | `partner/keys.go:17-35`; `partner/oauth_jwt.go:100-133`; `partner/auth.go:111-127` | HS256 shared-secret only; no mTLS/RS256/JWKS; `RateLimitClass` column exists but is never read |
| Outbound webhooks | **WIRED-LIVE** — HMAC-SHA256 signed, 8-attempt backoff, dead-letter + replay, portal self-service | `partner/delivery.go:34-98` | **Only 4 of 155 event types exposed**: `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `CLAIM_FILED`, `PAYMENT_CLEARED` (`partner/kafka_handler.go:26-31`). No DELIVERED/DESADV/return/invoice events |
| EDI | **WIRED-LIVE ("EDI-lite")** — real segment codec (UNA dialect), ORDERS inbound parser, ORDRSP/DESADV/INVOIC outbound builders, idempotent ingest on `(tenant, direction, type, external id)`, event-driven outbound | `partner/edi/segment.go:27-145`; `partner/edi_inbound.go:228-265`; `partner/edi_outbound.go:358-389` | Self-declared non-certified dialect; **no CONTRL/APERAK functional acknowledgment** (one-legged loop); zero X12 |
| AS2 | **PARTIAL** — real PKCS#7 sign/encrypt + sync MDN, per-tenant station config, certs via SecretRef | `partner/as2/crypto.go:76-142`; `partner/as2_receive.go:14-88` | Not Drummond-certified; sync MDN only; **default off and absent from shipped manifests** (`partner/as2_flags.go:8-12`; no `PARTNER_*` keys in `kustomize.yaml` configmap) |
| Bulk export / SFTP | **WIRED-LIVE (export)** / **FLAG-GATED (SFTP off)** — CSV/JSON/XML async jobs → GCS signed URL or SFTP push | `partner/export_worker.go:230-249`; `partner/sftp.go` | SFTP default off; host-key verification disabled (`ssh.InsecureIgnoreHostKey()`, `partner/sftp.go:67,136`) |
| 1C accounting | **PARTIAL** — double-entry journal export merging AR + payment ledgers, 1C-style default accounts (Dt 62.01/Kt 90.01; Dt 51.01/Kt 62.01), per-tenant configurable CoA, XML `dialect="1c"` | `partner/export_journals.go:32-57,90-261`; `partner/coa.go:13-17` | No CommerceML 2.x exchange package; only AR-open and cash-settle legs (no VAT breakout, no multi-leg, no COGS); a 1C scheduler cannot consume it without an import script |
| GS1 | **WIRED-LIVE** — GTIN-8/12/13/14 + GLN + SSCC-18 check-digit validation, SSCC minting at seal, ZPL GS1-128 labels, DESADV GIN+BJ from real ship units | `gs1/checkdigit.go:76-171`; `gs1/zpl.go:20-56` | **No DataMatrix** (critical for UZ/CIS marking regimes); no EPCIS; no GS1 registry sync |
| Master-data import | **WIRED-LIVE but human-only** — 9-state import wizard: signed-URL upload, xlsx/csv/tsv, column auto-discovery, mapping, staging, error summaries | `supplier/import_sessions_handlers.go:39-100`; `supplier/import_async.go:331` | **No machine-import endpoint** — a chain cannot push 50k SKUs programmatically |
| Compliance export | **WIRED-LIVE** — CSV export for admin/supplier JWT | `orderroutes/routes.go:92`; `supplierroutes/routes.go:162` | CSV only; no BI sink, no BigQuery (zero references anywhere) |
| Event backbone | **WIRED-LIVE** — transactional outbox → leased relay → Kafka (`RequiredAcks=all`, no auto-create), consumer dedup, DLQ + replay tooling | `outbox/relay.go:155-203`; `outbox/kafka_publisher.go:79-90`; `kafka/dlq_writer.go` | **Kafka is a single broker, replication-factor 1** (`infra/k8s/kafka.yaml:42-47`) — the entire integration backbone dies with one pod; outbox relay has no DLQ of its own |
| SAP / 1C / Odoo / NetSuite connectors | **ABSENT** — zero integration code (grep-verified) | — | Only indirect interchange via files (SFTP/AS2/CSV/XML) |

## 4.2 What must exist for big players to adopt without re-keying

Ranked by blocker severity, each grounded in a verified absence:

**P0 — adoption blockers**
1. **Idempotent REST order create.** Honor `Idempotency-Key` on `POST /partner/v1/orders` with a durable store (the EDI path already has this via unique index; the REST path has nothing).
2. **Master-data sync API.** Partner-key upsert endpoints for products/prices/stock (or EDI PRICAT/INVRPT inbound). Today machines can read catalog but cannot write it.
3. **Webhook event coverage.** Expose the delivery/invoice/manifest/return lifecycle (a configurable per-subscription event filter over the 155-type catalog, most of which already flows through the same outbox→Kafka substrate the webhooks consume).
4. **EDI functional acknowledgments.** CONTRL/APERAK outbound and inbound ORDRSP/INVOIC handling — the loop is currently one-legged.
5. **Enable the transports in shipped manifests.** AS2/SFTP default off and are missing from the configmap and `.env.k8s.example`; the integration layer is invisible in rendered deploys.
6. **Kafka HA.** Three brokers, RF=3, before any enterprise signs against webhook/EDI delivery semantics.

**P1 — enterprise credibility**
7. Certified EDIFACT envelope compliance (UNB/UNZ interchange control) and/or X12; today's dialect is proprietary.
8. 1C CommerceML 2.x exchange package + richer journals (VAT lines, multi-leg, returns/credit-note postings).
9. DataMatrix generation (UZ marking), EPCIS events, GS1 registry sync.
10. M2M auth hardening: mTLS or RS256/JWKS, IP allowlists, wire the unused per-key `RateLimitClass`.
11. Partner order amend/cancel/status-transition endpoints.
12. POS sales ingestion as a demand signal (the auto-order chain's weakest input; weather is wired, POS is explicitly residual).

**P2 — maturity**
13. Async MDN + CEM; Drummond certification.
14. SFTP host-key pinning.
15. Excel exports + BI sink (BigQuery or parquet feed).
16. Partner sandbox + self-serve key onboarding (keys currently require a human JWT session, `partner/routes.go:50-57`).
17. Resolve the 7 declared-but-never-emitted event types (`ALLOCATION_FAIR_SHARE_APPLIED`, `INVENTORY_IMPORT_STATUS_UPDATE`, `RETAILER_CLOCK_IN/OUT`, `RETAILER_SHIFT_OPENED/CLOSED`, `STORE_STOCK_CLAIM_HOLD`).
