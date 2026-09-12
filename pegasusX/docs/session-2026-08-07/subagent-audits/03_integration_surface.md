# 03 Integration Surface

> **HISTORICAL / FROZEN — do not plan from this file alone.**
> Current SoT: [`PROD_READINESS_SEQUENCE.md`](../../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](../ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`FEATURES_BY_APP_ROLE.md`](../../FEATURES_BY_APP_ROLE.md).
> Body is a point-in-time snapshot; re-verify any claim against code before acting.


_Source: subagent `caba0852-3519-4fd8-bdfd-d0e441c23759` from End-Product Reality Report session (2026-08-07)._

All evidence gathered. Here is the report.

---

# PegasusX / ATOMOS — End-Product Reality Report: Machine-to-Machine Integration Surface

**Ground truth method:** code + config only. Every claim carries `file:line`. Note up front: `PLATFORM_AUDIT.md:21` claims "no machine-to-machine integration surface at all" — that is **stale** (pre-Gate-3). The integration layer now exists and is substantially real; the gaps are specific and enumerated below.

---

## 1. PARTNER API — Verdict: **LIVE** (all 20 documented endpoints implemented; one contract violation)

**Contract:** `contracts/partner.openapi.yaml` (781 lines, v1.4.0) documents:
`/partner/v1/oauth/token`, `POST/GET /partner/v1/orders`, `GET /partner/v1/catalog`, `GET /partner/v1/inventory/availability`, 6 webhook ops, 3 export ops, `GET/PUT /partner/v1/coa`, `POST /partner/v1/as2`, `GET/PUT /partner/v1/as2/config`, 3 EDI document ops, plus human-JWT `/v1/admin/partner-keys`.

**Implementation:** every single documented path is mounted in `apps/backend-go/partner/routes.go:17-46` behind `AuthMiddlewareOpts` + per-route `RequirePartner(scope)`:

```25:44:apps/backend-go/partner/routes.go
ar.With(RequirePartner(ScopeOrdersWrite)).Post("/orders", h.HandleCreateOrder)
ar.With(RequirePartner(ScopeOrdersRead)).Get("/orders/{orderID}", h.HandleGetOrder)
ar.With(RequirePartner(ScopeCatalogRead)).Get("/catalog", h.HandleCatalog)
```

Code additionally implements endpoints **beyond** the contract: key revoke + supplier-portal aliases (`routes.go:50-85`).

**Auth — real, two schemes:**
- API keys `pxk_<prefix>_<secret>`, bcrypt-hashed at rest (`partner/keys.go:17-35`, `keys.go:52-55`), prefix lookup → bcrypt verify → status/expiry check → `TouchLastUsed` (`partner/auth.go:70-95`). Table `PartnerApiKeys` with unique prefix index (`schema/migrations/20260805_partner_integration_layer.ddl`).
- OAuth2 `client_credentials` issuing short-lived HS256 JWT (`token_use=partner_access`, 15m default / 60m cap — `partner/oauth_jwt.go:20-24,100-133`), form+JSON+Basic accepted (`partner/oauth.go:144-182`), scope intersection enforced (`oauth_jwt.go:190-209`), **live revoke**: every access-token call re-checks key status in the repo (`auth.go:111-127`). Partner token secret is derived-but-distinct from human JWT secret (`oauth_jwt.go:52-66`).
- No mTLS. No asymmetric keys/JWKS.

**Tenant isolation:** IDOR fail-closed — cross-tenant `GetOrder` returns 404 (`partner/service.go:176-191`).

**Rate limiting:** actor key `partner:<KeyId>` (`partner/routes.go:88-93`); `/partner/*` explicitly never exempt from limiting (`bootstrap/reliability_middleware.go:274-277`); Redis-backed fixed window (`bootstrap/redis_rate_limiter.go:44-69`); default class 240 req/min (`reliability_middleware.go:68-70`). Note: `PartnerApiKeys.RateLimitClass` column exists in DDL but is never read in code — per-key limits not wired.

**Contract violation found:** OpenAPI marks `Idempotency-Key` header **required** on `POST /partner/v1/orders` (`contracts/partner.openapi.yaml:494-499`), but `HandleCreateOrder` (`partner/handlers.go:37-63`) never reads it, and a grep of the whole `partner` package finds no idempotency store usage. Human-facing mutation endpoints do honor it (`supplier/service.go:767`, `retailer/service.go:672`). **The REST order-create path is not idempotent; only the EDI path is** (see §2).

---

## 2. EDI — Verdict: **LIVE** (working "EDI-lite" engine; explicitly not certified EDIFACT; zero X12)

Real code, not placeholder:
- **Segment codec** `partner/edi/segment.go:27-145` — UNA service-string parsing, release-char escaping, UNA read/write round-trip. Package doc is honest: "Not a certified EDIFACT/X12 implementation" (`segment.go:1-2`).
- **ORDERS inbound parser** `partner/edi/orders.go:28-124` — UNH type check, BGM doc id, NAD+BY/SU, LOC geo, DTM delivery date, LIN/QTY lines with validation errors (`missing_bgm`, `no_lines`, `invalid_line`).
- **Outbound builders:** ORDRSP with response codes 27/29/4 (`edi/ordrsp.go:33-67`), DESADV with CPS/PAC/GIN packing hierarchy and SSCC-18 (`edi/desadv.go:12-73`), INVOIC with MOA amount/currency (`edi/invoic.go:18-66`).
- **Inbound worker** `partner/edi_inbound.go` — polls SFTP (`processTenant`, :158-196) and local root (`:110-156`); sha256 payload hash; **idempotent** on `(TenantType, TenantId, Direction, DocType, ExternalDocId)` via unique index (`edi_inbound.go:228-265`; `schema/migrations/20260806_partner_edi.ddl:25-26`); maps to `order.Service.Create` with `order_source=PARTNER_EDI` (`:295`), geo resolution from retailer primary location (`bootstrap/bootstrap.go:1324-1330`).
- **Outbound worker** `partner/edi_outbound.go:130-203` — event-driven: `ORDER_CREATED`/status changes → ORDRSP, `LOADED`/`IN_TRANSIT` → DESADV, `DELIVERED_ON_CREDIT`/`PAYMENT_CLEARED` → INVOIC (`MapEventToOutboundDocs`, `:358-389`), fed from Kafka via `partner/kafka_handler.go:66` → `service.go:603-617`. DESADV loads SSCC rows from `ManifestShipUnits` with real Spanner SQL (`edi_outbound.go:282-326`).
- Document ledger APIs + replay implemented (`handlers.go`, `service.go:565-601`).

**Message types in code: ORDERS (IN), ORDRSP / DESADV / INVOIC (OUT).** No inbound ORDRSP/INVOIC, no APERAK/CONTRL functional acknowledgment, no X12 anywhere (grep: zero hits in Go code).

Flags: `PARTNER_EDI_ENABLED` default **on** (`edi_inbound.go:20-26`); `PARTNER_SFTP_ENABLED` default **off** (`export_worker.go:31-34`); local root mode `PARTNER_EDI_LOCAL_ROOT`.

---

## 3. AS2 — Verdict: **PARTIAL-LIVE** (functional RFC 4130 transport with real crypto; uncertified; dark in shipped manifests)

- Real PKCS#7 sign/encrypt/decrypt/verify via `go.mozilla.org/pkcs7` (`partner/as2/crypto.go:76-142`), SHA-256 MIC (`crypto.go:69-73`), PEM cert/key loaders (`crypto.go:27-67`).
- Sync MDN `multipart/report` with `Received-Content-MIC`, disposition notification (`as2/mdn.go:21-87`).
- Outbound HTTP client, TLS 1.2+, AS2 headers, 2MiB MDN read cap (`as2/client.go:21-105`).
- Inbound `POST /partner/v1/as2` — unauthenticated per RFC 4130, identity via AS2-From/To + station lookup (`partner/as2_receive.go:14-88`), feeds `IngestORDERSBytes`. **Only ORDERS inbound accepted** (`as2_receive.go:78-81,90-97`).
- Per-tenant station config in `PartnerAs2Configs` with cert material by SecretRef only, never in Spanner (`schema/migrations/20260806_partner_as2.ddl`; `service.go:485-510`).

Limitations (self-declared and verified): sync MDN only, no async MDN, no CEM, **not Drummond-certified** (`docs/PARTNER_AS2.md:3,21`). `PARTNER_AS2_ENABLED` defaults **off** (`partner/as2_flags.go:8-12`) — endpoint returns `503 as2_disabled` (`as2_receive.go:15-18`). `PARTNER_AS2_INSECURE_PLAIN` dev mode exists (`as2_flags.go:14-19`; doc warns "Never enable… on production overlays", `PARTNER_AS2.md:43`).

---

## 4. 1C / ACCOUNTING — Verdict: **PARTIAL** (real double-entry journal export; no 1C connector, no CommerceML)

- `resource=journals` export merges AR ledger (`ArLedgerEntries ⋈ ArInvoices`) and payment gateway ledger (`PaymentLedgerEntries`) into debit/credit rows (`partner/export_journals.go:90-261`), AR preferred up to 50k rows.
- **Double-entry mapping exists**: AR OPEN → Dt `62.01` / Kt `90.01`; PAYMENT → Dt `51.01` / Kt `62.01`; refund/chargeback/void reversed (`export_journals.go:32-57`). Defaults are 1C-style accounts (`partner/coa.go:13-17`).
- **Per-tenant configurable CoA** via `GET/PUT /partner/v1/coa` (`service.go:520-563`), validated account shape (`coa.go:95-111`), migration `20260806_partner_coa.ddl`, env defaults `PARTNER_COA_*`.
- Formats: CSV / JSON / **XML `<Journal version="1" dialect="1c">`** (`export_worker.go:319-340`). Async job → GCS object → 15-min signed URL (`export_worker.go:230-249`) or SFTP push. Caps: 50k rows / 90-day window (`partner/types.go:50-51`).

**Absent:** no direct 1C posting, no CommerceML 2.x exchange package (explicitly open: `docs/PARTNER_AS2.md:3`, `docs/PARTNER_EDI.md:70`), no VAT/tax line breakout, no multi-leg entries, no COGS/inventory accounting — only AR-open and cash-settle legs. Delivery model is pull (create job → poll → download) or SFTP push; nothing a 1C scheduler can consume natively without an import script.

---

## 5. GS1 / BARCODES — Verdict: **LIVE** (validation + SSCC mint + ZPL real; DataMatrix/EPCIS/registry absent)

- `apps/backend-go/gs1/checkdigit.go` — GS1 mod-10 validation for **GTIN-8/12/13/14** (`:76-90`), **GLN-13** (`:99-111`), **SSCC-18** (`:120-138`), plus **GenerateSSCC** with 7–10-digit company prefix (`:142-171`). Unit-tested (`checkdigit_test.go`).
- `gs1/zpl.go` — Zebra ZPL GS1-128 `(00)` SSCC label generation (`:20-56`).
- SSCC minted at payload seal into `ManifestShipUnits`, idempotent on `(ManifestId, OrderId)` (`docs/GS1_LABELS.md:39-44`), and DESADV `GIN+BJ` segments are built from those rows (§2).
- Flag `GS1_LABELS_ENABLED` default **on** (`checkdigit.go:12-18`).

**Absent:** no DataMatrix encoder (critical for UZ/CIS marking regimes), no EPCIS, no GS1 registry sync (`GS1_LABELS.md:14` lists these as open).

---

## 6. WEBHOOKS / EVENT BUS — Verdict: **LIVE** (real outbox→Kafka→signed-webhook pipeline; narrow partner event surface; Kafka not HA)

- **Transactional outbox** on Spanner with at-least-once relay to Kafka (`outbox/outbox.go:1-8`); publisher: `RequiredAcks=all`, sync writes, hash balancer, no auto-topic-creation (`outbox/kafka_publisher.go:50-92`); consumer-side dedup via Redis (`bootstrap/bootstrap.go:1347-1352`).
- **Event catalog:** `contracts/events.schema.json` declares **155 event types**; `events/events.go` defines exactly the same 155 constants; **148 are referenced from real code outside the events package** (verified by scripted cross-walk). Only 7 are declared-but-never-referenced in Go: `ALLOCATION_FAIR_SHARE_APPLIED`, `INVENTORY_IMPORT_STATUS_UPDATE`, `RETAILER_CLOCK_IN`, `RETAILER_CLOCK_OUT`, `RETAILER_SHIFT_OPENED`, `RETAILER_SHIFT_CLOSED`, `STORE_STOCK_CLAIM_HOLD`. Schema↔code drift is zero in both directions.
- **Kafka in real deploys:** Strimzi CRDs (`infra/k8s/kafka.yaml` — Kafka 4.3.0 KRaft, **1 replica, replication-factor 1** `:42-47`), topic CRDs (`infra/k8s/kafka-topics.yaml`), staging configmap points at `kafka.pegasusx.svc.cluster.local:9092` with dual-write + domain-topic consumption on (`kustomize.yaml:51-60`). In-memory fallbacks exist but staging sets `REQUIRE_INFRA_ADAPTERS=true` (`kustomize.yaml:69`) making missing infra fail-fast; `kafkaEnabled` only set when the publisher connects (`bootstrap/bootstrap.go:474-488`).
- **Partner outbound webhooks:** consumer group `void-partner-webhooks` on orders+exceptions topics (`bootstrap.go:1421-1430`), but the **allowlist is only 4 of 155 event types**: `ORDER_CREATED`, `ORDER_STATUS_CHANGED`, `CLAIM_FILED`, `PAYMENT_CLEARED` (`partner/kafka_handler.go:26-31`). Delivery: HMAC-SHA256 over `{timestamp}.{body}`, `X-Pegasus-Signature` headers, 8 attempts with capped exponential backoff, DEAD → dead-letter list + replay API (`partner/delivery.go:34-98`; `partner/types.go:29`; idempotent per `(SubscriptionId, EventId)` unique index).
- `webhookroutes/routes.go:23-27` is **inbound-only** payment-gateway webhooks (GlobalPay/Adyen/Stripe/Payme/Click) — not a partner surface.

---

## 7. PUBLIC API FOR RETAILER IT (beyond partner API) — Verdict: **PARTIAL**

- The partner API itself accepts `RETAILER` keys (orders, catalog, availability, exports) — that is the intended machine surface.
- Additionally: **compliance CSV export** `GET /v1/compliance/export` (admin JWT: `orderroutes/routes.go:92`; supplier JWT: `supplierroutes/routes.go:162`; handler `compliance/handler.go:97`, service `compliance/service.go:38`).
- **Bulk master-data import exists but is human-portal only**: supplier import sessions accept `xlsx|xls|csv|tsv` via GCS upload ticket (`supplier/import_sessions_handlers.go:39-100,751`; `import_async.go:331`), JWT-gated, idempotency-guarded (`:64-71`). **No partner-key-accessible import endpoint exists** — a chain's ERP cannot push products/prices/stock programmatically.
- No Excel/report download endpoints in Go beyond the two CSV routes (grep over `*routes/routes.go` for export/download/report: only those two). No OData/BI/BigQuery feed.

---

## 8. AUTH FOR M2M — Verdict: **LIVE**

- Human JWT spine: `contracts/jwt-core.openapi.yaml` (1290 lines, ~45 ops, `BearerJWT` only, login/refresh per role); HS256 issue/parse with role/org/location claims (`auth/jwt.go:63-120`). Hand-written, stdlib HMAC — noted as scaffold in the file header (`auth/jwt.go:1-5`).
- M2M: partner API keys + OAuth2 client_credentials as in §1 — real, with live-revoke, scope intersection, per-key rate-limit actor, and load-bootstrap exemption explicitly denied for `/partner/*` (`reliability_middleware.go:274-277`).
- **Missing for enterprise M2M:** mTLS, asymmetric signing (JWKS/RS256) — all tokens are HS256 shared-secret; IP allowlisting; wired per-key rate-limit classes (DDL column present, unread).

---

## 9. SAP / 1C CONNECTORS — Verdict: **ABSENT**

Grep for `SAP`, `CommerceML`, `connector`, `Odoo`, `NetSuite` across `*.{go,ts,tsx,yaml}`: **zero integration code**. Only doc references (`docs/PARTNER_AS2.md:3` admitting CommerceML is open) and narrative (`PLATFORM_AUDIT.md:219`). Interchange with SAP/1C is only possible indirectly via the file transports (SFTP/AS2 EDI-lite, CSV/XML exports).

---

## 10. INFRA REALITY — Verdict: **GKE real; integration transports dark in shipped manifests**

- **Topology:** GKE (`infra/terraform/gke.tf`), Cloud Spanner, Redis, in-cluster Strimzi Kafka (**single broker, RF=1** — `infra/k8s/kafka.yaml:42-47`), OSRM, optimizer-core, ai-worker, ExternalSecrets→GSM with Workload Identity (`kustomize.yaml:766-832`), GCE Ingress `api.pegasusx.app` with TLS + HTTPS redirect (`kustomize.yaml:898-933`), HPA 3–12 (`:675-697`), separate worker Deployment (`PEGASUSX_RUN_MODE=worker`, `:420-421`) that runs the outbox relay, Kafka consumers, webhook delivery, export and EDI workers (`runtime_workers.go:15-120`).
- **Build:** `cloudbuild.backend.yaml` is build-only (docker → Artifact Registry); no deploy step.
- **Feature flags in the shipped staging configmap** (`kustomize.yaml:43-75`): Kafka/Spanner/Redis/reliability set — but **no `PARTNER_*` / `GS1_*` / `AS2_*` keys at all**. Defaults therefore apply: EDI workers ON, exports ON, GS1 ON — but **SFTP push OFF and AS2 OFF** (`PARTNER_SFTP_ENABLED`/`PARTNER_AS2_ENABLED` default false: `export_worker.go:31-34`, `as2_flags.go:8-12`). `.env.k8s.example:18-24` likewise omits them. Net: the code is live-capable, but the AS2 endpoint 503s and SFTP never fires in the rendered deploys without a manifest change.

---

# WHAT MUST EXIST FOR BIG-PLAYER ADOPTION

Ranked gaps, each grounded in a verified absence:

**P0 — adoption blockers**
1. **Idempotent REST order create.** Contract requires `Idempotency-Key` (`contracts/partner.openapi.yaml:494-499`); handler ignores it (`partner/handlers.go:37-63`). ERP retries will double-create orders. (EDI path is safe; REST is not.)
2. **Master-data sync API.** No partner-key endpoint to upsert products/prices/stock — catalog is read-only for machines (`routes.go:27`); the only import is the human xlsx wizard (`supplier/import_sessions_handlers.go`). A chain cannot onboard 50k SKUs without re-keying.
3. **Webhook event coverage.** 4 of 155 event types reach partner webhooks (`partner/kafka_handler.go:26-31`). No DELIVERED/INVOIC/manifest/return events over webhooks, even though DESADV/INVOIC exist as EDI files.
4. **Inbound EDI acknowledgments.** No CONTRL/APERAK, no inbound ORDRSP/INVOIC — the EDI loop is one-legged (ORDERS in, files out; no functional ACK).
5. **Enable transports in deploy manifests.** AS2/SFTP default off and absent from `kustomize.yaml` configmap and `.env.k8s.example` — integration is invisible in real deploys today.
6. **Kafka HA.** Single broker, RF=1 (`infra/k8s/kafka.yaml:9-19,42-47`): the entire event/webhook/EDI backbone dies with one pod.

**P1 — enterprise credibility**
7. Certified EDIFACT envelope compliance (UNB/UNZ interchange control validation) and/or X12; current dialect is proprietary "EDI-lite" (`edi/segment.go:1-2`).
8. 1C CommerceML 2.x exchange package + richer journals (VAT lines, multi-leg, returns/credit-note postings); current export covers only AR-open/cash-settle (`export_journals.go:32-57`).
9. DataMatrix generation (UZ/CIS marking), EPCIS events, GS1 registry sync (`docs/GS1_LABELS.md:14`).
10. M2M auth hardening: mTLS or RS256/JWKS (all JWTs are HS256 shared-secret — `auth/jwt.go`, `partner/oauth_jwt.go`), IP allowlists, wire the unused `RateLimitClass` per-key limits.
11. Partner order amend/cancel/status-transition endpoints (only create+get exist — `routes.go:25-26`).
12. Inbound inventory/price feed acceptance over partner API or EDI (INVRPT/PRICAT) — partners can only *read* availability today.

**P2 — maturity**
13. Async MDN + CEM; Drummond certification for AS2 (`docs/PARTNER_AS2.md:21`).
14. SFTP host-key pinning — currently `ssh.InsecureIgnoreHostKey()` (`partner/sftp.go:67,136`).
15. Excel exports + BI sink (BigQuery/data-freshness feed); only CSV/JSON/XML exist.
16. Emit or remove the 7 declared-but-dead event types (§6) and extend webhook self-service filters beyond event-type lists.
17. Partner sandbox environment + self-serve key onboarding (keys today require a human JWT session — `routes.go:50-57`).

**Bottom line:** the M2M surface is real code with tests, migrations, and workers — not vaporware. But for a 1C/SAP-run retail chain it is today a *file-drop and pull-export* integration with a narrow webhook stream, an unidempotent REST write path, no master-data push channel, and transports that ship disabled.

I have everything I need. Let me also quickly confirm bootstrap.go:474-488 context for kafkaEnabled (newKafkaRuntimePublisher success). Already saw line 476: `if kafkaPublisher, err := newKafkaRuntimePublisher(...)` — presumably sets kafkaEnabled=true on success. Good enough; note in-memory fallbacks are guarded by AllowMemoryFallback and REQUIRE_INFRA_ADAPTERS=true in staging configmap (kustomize.yaml line 69).

Now write the report. It's a long dense markdown report with 10 sections, verdicts, evidence with file:line, and gap list P0/P1/P2.

Key facts to include:

**1. PARTNER API — verdict LIVE (with one contract violation: Idempotency-Key not honored)**
- Contract: contracts/partner.openapi.yaml (781 lines, v1.4.0). Endpoints: oauth/token, orders POST/GET, catalog, inventory/availability, webhooks CRUD+ping+dead-letter+replay, exports POST/GET/GET, coa GET/PUT, as2 POST + as2/config GET/PUT, edi/documents GET/GET/replay, admin partner-keys POST/GET (+ revoke & supplier aliases in code beyond contract).
- Implementation: apps/backend-go/partner/routes.go:17-46 — every path mounted. Handlers in handlers.go (788 lines), service.go (618 lines).
- Auth: auth.go — Bearer pxk_ keys (bcrypt at rest, keys.go:17-35 GenerateAPIKey, :52-55 VerifyAPIKey), OAuth2 client_credentials → HS256 JWT token_use=partner_access (oauth.go, oauth_jwt.go:100-133), live revoke check on every call (auth.go:111-127), scopes enforced via RequirePartner (auth.go:135-152). Key issuance human-JWT gated (routes.go:50-57). Rate limit actor partner:<KeyId> (routes.go:88-93; reliability_middleware.go:274-277 never exempt).
- IDOR fail-closed (service.go:176-191).
- GAP: OpenAPI requires Idempotency-Key header on POST /orders (partner.openapi.yaml:494-499) but HandleCreateOrder (handlers.go:37-63) never reads it — grep shows no idempotency handling in partner package; supplier/retailer human endpoints do have it (supplier/service.go:767, retailer/service.go:672). EDI inbound IS idempotent via unique index (edi_inbound.go:228-265, migration Idx_PartnerEdiDocuments_Idempotency).

**2. EDI — verdict LIVE (EDI-lite dialect, not certified EDIFACT)**
- Real segment codec: partner/edi/segment.go (UNA:+.? ', escape, UNA header parsing).
- ORDERS parser orders.go:28-124 (UNH/BGM/NAD/LOC/DTM/LIN/QTY), builders ordrsp.go, desadv.go (CPS/PAC/GIN SSCC), invoic.go.
- Inbound worker edi_inbound.go: SFTP poll + local root, sha256 payload hash, idempotent via ExternalDocId unique index, maps to order.Service.Create with source PARTNER_EDI (edi_inbound.go:295).
- Outbound worker edi_outbound.go:130-203 emits ORDRSP/DESADV/INVOIC on order events via kafka_handler.go:66 → EnqueueEdiFromEvent (service.go:603-617) → MapEventToOutboundDocs (edi_outbound.go:358-389). DESADV ship units loaded from ManifestShipUnits via raw Spanner SQL (edi_outbound.go:282-326).
- Transports: SFTP (real ssh/sftp, sftp.go), local root, AS2.
- Flag: PARTNER_EDI_ENABLED default ON (edi_inbound.go:20-26); PARTNER_SFTP_ENABLED default OFF (export_worker.go:31-34).
- Not certified EDIFACT: stated in docs/PARTNER_EDI.md:3 and package comment segment.go:1-2. No X12 anywhere (grep found no X12 code).

**3. AS2 — verdict LIVE-but-uncertified (skeleton-adjacent but functional); default OFF**
- Real PKCS7 sign/encrypt/decrypt/verify via go.mozilla.org/pkcs7 (as2/crypto.go:76-142), MIC SHA-256 (crypto.go:69-73), sync MDN multipart/report (mdn.go:21-87), outbound HTTP client TLS1.2+ (client.go:21-105).
- Receive endpoint unauthenticated RFC4130 identity via AS2-From/To + cert (as2_receive.go:14-88), feeds IngestORDERSBytes.
- Limits: sync MDN only, no async MDN/CEM, not Drummond-certified (docs/PARTNER_AS2.md:3,21). PARTNER_AS2_ENABLED default off (as2_flags.go:8-12), insecure plain dev mode exists (as2_flags.go:14-19) — doc warns never prod (PARTNER_AS2.md:43).
- Only ORDERS inbound supported (as2_receive.go:78-81,90-97).

**4. 1C / ACCOUNTING — verdict PARTIAL (real double-entry CSV/XML export; no 1C connector, no CommerceML)**
- journals resource: export_journals.go — AR (ArLedgerEntries ⋈ ArInvoices) + payment (PaymentLedgerEntries) mapped to debit/credit via CoA; defaults 62.01/90.01/51.01 (coa.go:13-17); per-tenant CoA configurable GET/PUT /partner/v1/coa (service.go:520-563, migration 20260806_partner_coa.ddl).
- Formats CSV/JSON/XML with `<Journal dialect="1c">` (export_worker.go:319-340). Async job → GCS signed URL 15min (export_worker.go:230-249) or SFTP push. Caps 50k rows/90d (types.go:50-51).
- Manual-ish: pull model (create job, poll, download). No direct 1C posting, no CommerceML (docs PARTNER_AS2.md:3 "1C CommerceML exchange package remain open"; PARTNER_EDI.md:70 "Certified 1C exchange package still open").
- mapping coverage: only OPEN/PAYMENT/refund-chargeback-void (export_journals.go:32-57); no VAT lines, no cost of goods, no inventory accounting entries.

**5. GS1 — verdict LIVE (validation/generation real; DataMatrix absent)**
- gs1/checkdigit.go: GTIN-8/12/13/14, GLN-13, SSCC-18 mod-10 validate; GenerateSSCC (company prefix 7-10 digits).
- gs1/zpl.go: ZPL GS1-128 (00) SSCC label gen.
- SSCC minted at payload seal into ManifestShipUnits (docs/GS1_LABELS.md:39-44), DESADV CPS/PAC/GIN+BJ wired (desadv.go:48-73).
- GS1_LABELS_ENABLED default on (checkdigit.go:12-18).
- No DataMatrix encoder (no datamatrix code), no GS1 registry sync (GS1_LABELS.md:14), no EPCIS.

**6. WEBHOOKS / EVENT BUS — verdict LIVE (but narrow partner surface)**
- Transactional outbox on Spanner + relay to Kafka at-least-once, RequiredAcks=all (outbox/outbox.go:1-8, kafka_publisher.go:50-92).
- 155 event types declared in contracts/events.schema.json = 155 Go constants (events/events.go); 148 referenced from real code outside events pkg; 7 declared-but-never-referenced: ALLOCATION_FAIR_SHARE_APPLIED, INVENTORY_IMPORT_STATUS_UPDATE, RETAILER_CLOCK_IN/OUT, RETAILER_SHIFT_OPENED/CLOSED, STORE_STOCK_CLAIM_HOLD.
- Kafka real in deploy: Strimzi CRDs infra/k8s/kafka.yaml (Kafka 4.3.0, kraft, 1 replica, RF=1 — single point), topics infra/k8s/kafka-topics.yaml; kustomize.yaml:51-60 configmap KAFKA_BROKERS kafka.pegasusx.svc.cluster.local:9092, dual-write+consume-domain true. bootstrap falls back to in-memory only when Kafka init fails AND RequireInfraAdapters false (staging sets REQUIRE_INFRA_ADAPTERS=true, kustomize.yaml:69 → fail-fast).
- Partner webhooks: allowlist of only 4 events (kafka_handler.go:26-31: ORDER_CREATED, ORDER_STATUS_CHANGED, CLAIM_FILED, PAYMENT_CLEARED); HMAC-SHA256 signed delivery, 8 attempts, dead-letter+replay (delivery.go:34-98, types.go:29). Consumer group void-partner-webhooks on orders+exceptions topics (bootstrap.go:1421-1430).
- webhookroutes/routes.go = INBOUND payment gateway webhooks (global-pay, adyen, stripe, payme, click) — not partner-facing.

**7. PUBLIC API for retailer IT — verdict PARTIAL**
- Partner API covers retailer keys too (orders, catalog, availability, exports). Beyond that: compliance CSV export /v1/compliance/export (orderroutes/routes.go:92 admin; supplierroutes/routes.go:162; compliance/handler.go:97, service.go:38). No Excel export endpoints in Go (xlsx only for import). No OData/BI feed; no BigQuery sink (PLATFORM_AUDIT.md:21 historical note, now stale for API but still true for analytics sink).
- Bulk master-data INBOUND: supplier import sessions xlsx/xls/csv/tsv via GCS ticket (supplier/import_sessions_handlers.go:39-100, :751; import_async.go:331) — human-portal initiated, JWT-gated, not partner-key accessible (no /partner/v1 import endpoint).

**8. AUTH M2M — verdict LIVE**
- jwt-core.openapi.yaml 1290 lines ~45 ops BearerJWT; auth/jwt.go HS256 issue/parse with roles; partner OAuth as above. Rate limiting: fixed-window/Redis, classes, partner never exempt, actor partner:<KeyId> (reliability_middleware.go:274-277; routes.go:88-93). Scopes: 6 OAuth scopes (partner.openapi.yaml:482-488) + wildcard. Key rotation: revoke endpoint + expiry field (migration 20260805).
- Missing: mTLS, JWKS/asymmetric keys (HS256 shared secret), IP allowlists, per-endpoint rate-limit classes (RateLimitClass column exists but unused).

**9. SAP/1C connectors — verdict ABSENT**
- Zero connector code: grep SAP/1C/connector/Odoo/NetSuite found none in code (only docs + PLATFORM_AUDIT.md:219 narrative). No CommerceML. Only file-based interchange (SFTP/AS2/exports).

**10. INFRA REALITY — verdict: GKE real; integration flags partially dark in manifests**
- GKE (infra/terraform/gke.tf), Spanner, Redis, in-cluster Strimzi Kafka (1 broker, RF=1), OSRM, ExternalSecrets→GSM, GCE ingress api.pegasusx.app, HPA 3-12, worker deployment run-mode worker.
- kustomize.yaml configmap: NO PARTNER_* env at all → defaults apply: EDI on, exports on, GS1 on; SFTP off, AS2 off. So in the shipped staging/prod manifest, AS2 endpoint returns 503 as2_disabled (as2_receive.go:15-18) and SFTP push skipped.
- .env.k8s.example lacks partner flags entirely (lines 18-24).
- Kafka single replica RF=1 (infra/k8s/kafka.yaml:42-47) — not prod-grade HA.
- cloudbuild*.yaml build-only (no deploy step) — image to Artifact Registry.

Gap list P0/P1/P2 grounded in verified absences:

P0:
- Idempotency-Key required by contract but ignored by POST /partner/v1/orders (partner.openapi.yaml:494-499 vs handlers.go:37-63). EDI path idempotent; REST path not.
- No master-data sync endpoints: no partner product/price/stock upsert API (only read catalog/availability). Products import only via human portal xlsx wizard (supplier/import_sessions_handlers.go), not partner-key accessible.
- INVOIC is outbound-only; no inbound INVOIC/APERAK/CONTRL; no functional acknowledgment loop.
- AS2/SFTP dark in deploy manifests (no env in kustomize.yaml configmap); enabling requires manifest change.
- Webhook event coverage only 4 of 155 event types (kafka_handler.go:26-31) — no shipment/delivery/invoice webhook events for partners despite DESADV/INVOIC existing as files.
- Kafka RF=1 single broker (infra/k8s/kafka.yaml) — integration backbone not HA.

P1:
- Certified EDIFACT subset (real UNB/UNZ envelope validation, segment directory compliance), X12 absent entirely.
- 1C CommerceML package + direct 1C enterprise posting; journals lack VAT/tax lines & multi-leg entries.
- DataMatrix (UZ/CRPT marking) absent; EPCIS absent; GS1 registry sync absent.
- mTLS / asymmetric JWT (JWKS) for M2M; per-key rate limit classes wired (RateLimitClass column unused).
- Partner-facing order amend/cancel endpoints; only create+get.
- No inbound inventory/price feed acceptance (partner can't push stock; only read availability).

P2:
- Async MDN, CEM, Drummond certification.
- Excel export for business users; BI sink (BigQuery) feed.
- Partner sandbox/self-serve onboarding; scoped test keys.
- Webhook event-type self-service expansion + per-subscription filters beyond event types.
- SFTP host key pinning (ssh.InsecureIgnoreHostKey at sftp.go:67,136).

Also mention PLATFORM_AUDIT.md:21 is stale.

Verdicts summary. Write it dense. Cite file:line throughout. Since the report goes to the parent agent/user in CLI, markdown is fine.

# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
