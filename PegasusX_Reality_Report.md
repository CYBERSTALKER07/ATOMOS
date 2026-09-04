**PegasusX / ATOMOS
End-Product Reality Report**

Codebase-Grounded Product Audit — v2 (with subagent evidence)

Generated: 11 August 2026
Source tree: pegasusX/ monorepo
All claims verified against live code

# **Executive Summary**

PegasusX is a working vertical logistics execution platform with a real multi-tenant Go backend, Spanner-backed data model, Kafka event pipeline, and client applications across six roles. The core order-to-delivery-to-payment loop is wired end-to-end: strict state machine, idempotency guards, transactional outbox, and a fiscal hard-gate between delivery and completion.

The system is not a unified planning platform, a replacement for ERP/WMS suites, or a legal fiscal intermediary in Uzbekistan. Payment integrations are live only for Global Pay, Payme, and Click (all with real signature verification). Adyen and Stripe are stubs. The AI layer is a deterministic heuristic, not a learned model. Advanced planning features (forecast algorithm, safety stock v2, MEIO, control tower, AS2) are disabled by default in production.

Critical security findings: (1) Driver login accepts any phone with PIN 1234 and issues a JWT for hardcoded driver drv_factory_1 — no per-driver credential database. (2) Tenant suspension is not enforced by middleware — a suspended tenant with a valid JWT can still make API calls. (3) Driver GPS telemetry sent over WebSocket is discarded by the server read-loop; the real telemetry sink (POST /v1/telemetry/location) is never called by either driver client.

The supplier portal, warehouse portal, and factory portal have broad page coverage. The Go backend has ~110 supplier routes, ~50 warehouse routes, ~35 factory routes, and a full partner API with OAuth2, HMAC webhooks, EDI, and AS2 scaffolding. Significant portions are stubs, fail-closed returns, or dead code paths.

A Tashkent shopkeeper can run a full day (browse catalog, place cash/card order, track delivery, receive stock, sell via POS, file a damage claim) without a sales agent — today, on all three retailer clients. The two blockers are: no self-serve credit repayment (no repay endpoint exists anywhere), and CREDIT is not a selectable checkout payment method on any client despite the backend supporting it.

────────────────────────────────────────────────────────────────────────

# **Section 1 — Human / Field-Agent Displacement**

## **1.1  What the system automates today (verified in code)**

- **Order intake → dispatch assignment**

AutoDispatchWorker runs in runtime_workers.go. dispatch/binpack.go implements H3-resolution-7 bin-packing as the primary path. OR-Tools optimizer-core (Python) exists in services/optimizer-core but is deployed at replicas: 0 in production.

- **Delivery completion → fiscal receipt**

order/state_machine.go enforces a hard gate: ARRIVED cannot go to COMPLETED directly. Must pass through AWAITING_PAYMENT → FISCALIZING → COMPLETED. Fiscal provider selected via FISCAL_PROVIDER env var; PEGASUS is the default.

- **Cash collection geofencing**

order/proximity_settlement.go defines SettlementProximityRadiusM = 100.0 meters. Driver must be within 100m of the delivery point to trigger cash settlement.

- **Payment capture**

payment/execution.go routes to GLOBAL_PAY (live, circuit breaker, retry), Payme and Click webhooks (real signature verification), plus stubs for ADYEN/STRIPE (hosted redirect only), CASH/INTERNAL/CREDIT (manual mode).

- **Order cancellation**

Full cancel flow with reason codes, evidence requirements, and supplier approval loops in the state machine.

- **Idempotency on all mutations**

idempotency.Guard middleware enforces Idempotency-Key headers on all POST/PATCH/DELETE handlers. Duplicate key + same body returns cached response; same key + different body returns 409.

- **Shop-closed inventory release**

order/shop_closed.go handles driver-reported shop-closed flow, releasing reserved stock and logging the exception.

- **Outbox event delivery**

outbox/relay.go drains OutboxEvents table to Kafka with retry and dead-letter. OutboxEvents written atomically inside the same Spanner transaction as the business mutation.

- **Offline order queue (retailer)**

PendingOrderSyncWorker (Kotlin WorkManager) queues orders locally in Room DB and replays them with stored idempotency keys when online. Backend idempotency middleware resolves duplicates safely.

- **Partner webhook delivery**

partner/delivery.go signs X-Pegasus-Signature: sha256=... HMAC, exponential backoff 1<<attempt seconds, MaxDeliveryAttempts=8, dead-letter + replay endpoints. Delivery worker runs every 15 seconds.

## **1.2  What still requires a human today**

- **Driver physical execution**

Loading, driving, unloading, QR validation at the door. No autonomous vehicle integration exists.

- **Cash physical handoff**

Cash reconciliation has a worker (CashReconEscalation) but the physical act of handing cash is a driver/retailer interaction.

- **Fiscal signing for legal receipts**

MY_SOLIQ provider exists but uses DevHMACSigner (fiscal/signer_env.go) — an HMAC stub, not a certified EDS key. Legal OFD receipt generation in Uzbekistan is not live. PKCS#7/E-IMZO path hard-errors 'not provisioned yet — owner task: Soliq EDS key procurement'.

- **Supplier order vetting**

supplier.HandleVetOrders is a real route but vetting logic requires supplier human judgment for accept/reject on non-standard orders.

- **Dispute resolution**

Claims (POST /v1/orders/{id}/claims) are stored and routed to supplier; resolution requires human supplier action.

- **Warehouse pick confirmation**

Pick waves exist in the WMS package; physical picking is a human worker task confirmed via HandleConfirmPickTask.

- **Credit limit decisions**

Credit profiles have automated checks (limit, balance, delinquency count) but setting the initial limit is a human supplier decision.

- **Credit repayment**

No repay endpoint exists anywhere in the backend. A retailer on trade credit cannot pay down their balance digitally.

## **1.3  Automation trajectory**

- **1. Demand forecasting**

ai-worker/synthesis/engine.go implements BuildSupplierRecommendation as a deterministic heuristic over the last 30 order signals (recency + volume + value, clamped to [0,1]). planning/forecast/ has real Croston/SBA + SES + Holt-Winters implementations with SBC classification — but FORECAST_ALGO_ENABLED defaults OFF. Replacing the heuristic with the already-implemented statistical models and enabling the flag is the fastest path to real forecasting.

- **2. Replenishment auto-execution**

retailer/auto_order_worker.go runs every 15 minutes and drafts carts for opted-in retailers. AUTO_ORDER_PLACE_ENABLED=false in production; worker runs in shadow/draft mode. AUTO_ORDER_SHADOW_ENABLED defaults true. Enabling place mode requires resolving the payment capture path for auto-drafted orders.

- **3. Route optimization**

optimizer-core (Python OR-Tools, pywrapcp, multi-depot VRP with capacity and time windows) is fully implemented. Deployed at replicas: 0 in production. Backend client (dispatch/optimizerclient/client.go) treats it as a soft dependency — any error falls back to H3 bin-pack. Publishing the Docker image and raising replicas is the only remaining step.

- **4. Fiscal automation**

MY_SOLIQ EDS signing path requires a certified Uzbekistan EDS key and a real OFD gateway endpoint. soliq/client.go defines a clean SoliqClient interface (Submit/CheckStatus) supporting multiple operators. The code path is wired but untested against a live gateway.

- **5. Driver telemetry**

Driver apps send GPS over WebSocket every 5 seconds. The server read-loop (ws/handler.go:273) discards all incoming bytes. The real DB-backed sink is POST /v1/telemetry/location (telemetryroutes/routes.go:106) — never called by either driver client. Fixing this is a one-line client change or a server-side WS frame parser.

────────────────────────────────────────────────────────────────────────

# **Section 2 — Problem Coverage vs Existing Logistics / Planning Software**

## **2.1  What PegasusX covers (verified)**

- **Order execution**

Multi-stage state machine with 15+ states. Real. Wired. Production-live.

- **Cash and card collection at delivery**

Global Pay live for card. Payme and Click webhooks live with real signature verification. Cash geofence-gated. Real. Wired.

- **Warehouse dispatch management**

Bin-packing fallback is live. Full WMS (bins, lots, pick waves, cycle counts, FEFO picking, cold-chain quarantine) implemented in the warehouse package.

- **Supplier catalog and inventory**

CRUD for products, pricing rules, inventory policies. Real. GTIN/EAN validation in catalog UI.

- **Partner API**

OAuth2 client_credentials, scoped API keys, HMAC-signed webhook subscriptions with delivery worker, EDI document tracking (ORDERS/ORDRSP/DESADV/INVOIC/CONTRL/APERAK), AS2 receive endpoint with PKCS7 sign+encrypt. AS2 disabled in production.

- **Driver manifest management**

Seal, dispatch, reassign, exception reporting. Real.

- **Credit management**

Credit profiles, balance tracking, delinquency bumping, order gate checks. Real. No repay endpoint.

- **Returns**

Returns state machine with warehouse receive, condition assessment, disposition. Real. Driver return-goods list is DB-backed.

- **Retailer POS**

POS sessions, holds, sales recording, shift management. Real in retailer desktop app and mobile.

- **Offline ordering**

Retailer Android has Room DB + WorkManager offline queue with idempotency key replay. Real.

- **GS1 identifiers**

Real GS1 Mod-10 check digit for GTIN-8/12/13/14, GLN-13, SSCC-18. ZPL label rendering. GS1_LABELS_ENABLED defaults true.

- **Cold chain**

Temperature readings stored. Quarantine logic (stocklots/coldchain.go:227) fires server-side automatically. No UI to view readings.

## **2.2  Where PegasusX falls short vs O9 / Kinaxis / Blue Yonder**

- **No demand planning engine (production)**

planning/forecast/ has real Croston/SBA + SES + Holt-Winters with SBC classification — correctly implemented. But FORECAST_ALGO_ENABLED defaults OFF. ai-worker is a deterministic heuristic, not a model. No ML ensemble, no gradient boosting, no external regressors beyond a single weather multiplier.

- **No supply network optimization**

MEIO endpoint exists (GET /v1/supplier/meio/network-summary) but returns a static summary. No multi-echelon joint optimization solver is running. Safety stock math is single-echelon (z·√(L·σd² + d²·σL²)) — correct formula, but per-node only.

- **No S&OP workflow**

GET /v1/supplier/planning/s-and-op exists as a route but the underlying engine is not a real S&OP process manager. No scenario comparison, no what-if simulation layer.

- **No ERP integration suite**

No native SAP, Oracle, or Dynamics connectors. Partner API is a custom REST surface. EDI is self-described 'EDI-lite (EDIFACT-ish UNA segment files). Not a certified EDIFACT/X12 implementation.' Enterprise integration requires custom middleware.

- **No 1C integration**

No CommerceML/1C connector exists anywhere in the Go backend. export_journals.go emits home-grown XML tagged dialect='1c' — not genuine 1C Enterprise exchange XML/CommerceML schema. CoA export targets 1C-style account codes (62.01/90.01/51.01) but there is no real exchange protocol. Critical gap for Uzbekistan and CIS markets.

- **No real fiscal compliance**

FISCAL_PROVIDER=PEGASUS issues platform commercial receipts, not legally compliant OFD receipts. MY_SOLIQ is present but DevHMACSigner is the only implemented signer; PKCS#7/E-IMZO path hard-errors. Legally non-compliant until EDS key lands.

- **No multi-leg freight**

Order model is single-leg: warehouse → retailer. No cross-docking, multi-stop linehaul, or freight forwarding.

- **No HR / payroll**

No payroll, benefits, or HR management. Labor capacity workers track warehouse staffing but not payroll.

- **Routing uses haversine, not road network**

eta/calculator.go uses straight-line haversine distance. No OSRM/Google Routes/HERE integration. OR-Tools VRP exists for dispatch optimization but is a single-replica soft dependency.

- **Promotion uplift is a fixed linear coefficient**

planning/promo_eval.go uses DefaultPromoElasticity = 0.5 with a simple linear formula. Not a fitted elasticity model. Explicitly labeled 'sandbox' in code. No cannibalization/halo modeling.

- **Segmentation is a static rule table**

segment/priority.go uses rule-based service-policy lookup, not RFM/clustering. Comment: 'Credit risk tier boosts were removed with the credit-score product.'

## **2.3  Where PegasusX has a vertical advantage**

- **Last-mile B2B cash execution**

No major planning suite handles the physical cash collection geofence, driver manifest, and fiscal receipt in a single workflow. PegasusX does.

- **Uzbekistan-specific payment rails**

Payme and Click webhook integrations with real signature verification are live. Not available in O9/Kinaxis/Blue Yonder.

- **Retailer mobile ordering with offline sync**

retailer-app-android has PendingOrderSyncWorker with local Room database and idempotency key replay. Real offline-first ordering.

- **Unified dispatch across supplier + warehouse**

Both supplier routes and warehouse routes feed into the same dispatch engine. Planning suites treat these as separate silos.

- **GS1 SSCC pallet labeling**

Real GS1-128 ZPL label generation with SSCC, GLN, and check-digit validation. Built into the payload terminal backend.

- **Cold-chain quarantine automation**

Temperature breach triggers automatic lot quarantine server-side (stocklots/coldchain.go:227) without human intervention.

────────────────────────────────────────────────────────────────────────

# **Section 3 — Alignment with Systems Already Used by Big Retailers and Suppliers**

## **3.1  Integration surfaces that exist in code**

- **REST Partner API**

/partner/v1/* with OAuth2 client_credentials, scoped API keys (orders:read/write, catalog:read/write, inventory:read/write, demand:write, webhooks:manage, exports:read). 23 authenticated routes. Key revocation enforced on every request (not just at issuance).

`[partner/routes.go:17-50, partner/auth.go:122]`

- **Partner webhooks**

HMAC-signed outbound webhooks (X-Pegasus-Signature: sha256=...), exponential backoff, MaxDeliveryAttempts=8, dead-letter queue, replay endpoint, secret rotation. Delivery worker runs every 15 seconds. 13 event types exposed (out of ~159 internal).

`[partner/delivery.go:34-56, runtime_workers.go:112-114]`

- **EDI**

ORDERS/ORDRSP/DESADV/INVOIC/CONTRL/APERAK. DESADV emits real EDIFACT segment syntax (CPS/PAC/GIN+BJ SSCC packing hierarchy). Self-described 'EDI-lite — not a certified EDIFACT/X12 implementation.' Inbound worker 30s, outbound 20s.

`[partner/edi/desadv.go:40-73, partner/edi/segment.go:1-2]`

- **AS2**

POST /partner/v1/as2 accepts AS2 messages with real PKCS7 sign+encrypt (SMIME). Sync MDN only (no async). PARTNER_AS2_ENABLED defaults false. PARTNER_AS2_INSECURE_PLAIN explicitly commented 'Never enable in production overlays.'

`[partner/as2/crypto.go:76-141, partner/as2_flags.go:9-17]`

- **POS demand feed**

POST /partner/v1/demand/pos-feed accepts point-of-sale demand signals.

`[partner/pos_demand_feed.go:110]`

- **Payment webhooks**

Global Pay (signature verification, idempotency, status re-verification), Payme (basic-auth shared secret), Click (computed signature). All mounted. SettleExternalPayment drives AWAITING_PAYMENT → FISCALIZING.

`[payment/global_pay_webhook.go:30, payment/payme_webhook.go:39-53, payment/click_webhook.go:143-154]`

- **SFTP export**

Real SFTP client (golang.org/x/crypto/ssh + github.com/pkg/sftp) with host-key pinning. InsecureIgnoreHostKey fallback flagged //nolint:gosec when strict mode off. PARTNER_SFTP_STRICT_HOSTKEY=1 required for enforcement.

`[partner/sftp.go:59-77]`

- **Catalog import**

CSV import with staged rows, column mapping persistence, async Kafka pipeline via ai-worker import_worker.go.

`[supplier/import_sessions.go:189-190, ai-worker/import_worker.go:16-61]`

- **GS1**

Real GS1 Mod-10 check digit for GTIN-8/12/13/14, GLN-13, SSCC-18. ZPL rendering with GS1-128 AI(00)/(01) barcode. GS1_LABELS_ENABLED defaults true.

`[gs1/checkdigit.go:30-171, gs1/zpl.go:20-56]`

## **3.2  What does NOT exist**

- 1C / CommerceML connector — no Go code references CommerceML. Only scripts/commerceml_import_ref.py (standalone Python reference) and docs/COMMERCEML_EXCHANGE.md exist. No backend wiring.

- SAP connector — no BAPI, IDoc, or OData integration anywhere in the codebase.

- Certified EDIFACT stack — only 4-6 message types hand-built segment-by-segment. No UNB/UNZ interchange envelope validation. No INVOIC/PRICAT full schema versioning.

- POS terminal protocol bridge — pos_demand_feed.go is a REST endpoint, not an OPOS/JavaPOS/ECR fiscal-printer driver.

- Accounting software connectors — no QuickBooks, Xero, or 1C:Accounting integration.

- WMS connectors — no Manhattan, Blue Yonder WMS, or SAP EWM integration.

- Generated partner SDK — sdk/partner contains only a README and a generation script. No TypeScript or Go client is checked in.

- GLN-based location/party master data exchange — GS1 validates GLN format but there is no /partner/v1/parties endpoint for multi-DC/multi-store location sync.

- Async MDN for AS2 — sync-only, unsigned MDN. Enterprise partners typically require async MDN and interop certification.

## **3.3  What big-player adoption requires**

- **1C integration (critical for Uzbekistan)**

Build a real CommerceML 2.x XML producer/consumer (catalog.xml/offers.xml/import.xml) in the Go backend. The existing scripts/commerceml_import_ref.py can serve as a spec source. The Partner API catalog upsert and order create endpoints are the correct integration seam.

- **Legal fiscal receipts**

Replace DevHMACSigner with real PKCS#7 EDS signing against the Uzbekistan tax authority CA. Test MY_SOLIQ against a live OFD gateway. This is explicitly blocked on EDS key procurement (signer_env.go:69-70).

- **Certified EDIFACT**

License or integrate a conformant EDIFACT library with syntax version negotiation. Current codec is self-described 'EDI-lite' and not certifiable.

- **Async MDN + AS2 interop certification**

Implement async MDN delivery. Test against a reference AS2 station (Cleo/IBM Sterling). Enable PARTNER_AS2_ENABLED by default after interop testing.

- **SFTP host-key enforcement by default**

Require pinned host keys in production overlays, not opt-in via PARTNER_SFTP_STRICT_HOSTKEY.

- **Broader webhook event coverage**

Expand PartnerWebhookableEvents allowlist beyond 13 event types. Large retailers integrating with SAP/OMS typically want returns, pricing, promotions, and B2B invoicing events.

- **Versioned SDK**

Run and publish gen_partner_sdk.sh output to a package registry with semver tied to partner.openapi.yaml.

────────────────────────────────────────────────────────────────────────

# **Section 4 — Existence of a True Unified Platform**

A true unified transactional platform for routine replenishment and physical execution would have the following properties. PegasusX's actual state is measured against each:

**Order placement without human initiation:** **PARTIAL**

AutoOrderWorker drafts carts every 15 minutes for opted-in retailers. AUTO_ORDER_PLACE_ENABLED=false in production — orders are drafted, not placed. AUTO_ORDER_SHADOW_ENABLED defaults true. Human confirmation required.

**Dispatch assignment without human initiation:** **YES**

AutoDispatchWorker runs in the background. dispatch/binpack.go assigns orders to drivers without human input. H3 bin-pack fallback is live. OR-Tools optimizer is deployed at replicas: 0.

**Payment capture without human initiation:** **PARTIAL**

Global Pay capture is automatic after delivery confirmation for card payments. Payme/Click webhooks drive AWAITING_PAYMENT → FISCALIZING automatically. Cash requires physical driver-retailer interaction. Credit orders require no payment action at delivery.

**Fiscal receipt generation without human initiation:** **PARTIAL**

Receipt generation is automatic on order completion. FISCAL_PROVIDER=PEGASUS issues platform receipts, not legal OFD receipts. Legal fiscal automation requires MY_SOLIQ with a certified EDS key.

**Replenishment suggestion without human initiation:** **YES (staging) / NO (production)**

ReplenishmentEngine.StartCron runs in the background. FORECAST_ALGO_ENABLED=false and SAFETY_STOCK_V2_ENABLED=false in production. Engine runs but advanced algorithms are disabled.

**Warehouse picking without human initiation:** **NO**

Pick waves are created automatically but physical picking requires a human worker with a mobile device.

**Physical delivery without human initiation:** **NO**

Requires a human driver.

**Exception resolution without human initiation:** **PARTIAL**

Shop-closed inventory release, proximity-gated cash, and cold-chain quarantine are automated. Claims, disputes, and manifest exceptions require human resolution.

**Credit repayment without human initiation:** **NO**

No repay endpoint exists anywhere in the backend. Credit repayment is entirely manual.

**Driver GPS tracking without human initiation:** **BROKEN**

Driver apps send GPS over WebSocket every 5 seconds. Server read-loop discards all incoming bytes. Real telemetry sink (POST /v1/telemetry/location) is never called by either driver client.

**Overall assessment: PegasusX is a partially automated execution platform, not a fully autonomous one. The order-to-delivery-to-payment loop is automated for card payments on confirmed deliveries. The planning layer (forecasting, replenishment, optimization) is present in code but disabled in production. The fiscal layer is live for platform receipts but not for legal Uzbekistan OFD receipts. Driver GPS tracking is broken — the WebSocket sink discards all frames.**

────────────────────────────────────────────────────────────────────────

# **Section 5 — Per-Role, Per-App, Per-Feature Reality**

## **5.1  Retailer**

### **5.1.1  Retailer Mobile App (Android / iOS)**

**Login / auth:** **WIRED**

POST /v1/auth/retailer/login and /register are live. JWT issued. Multi-org membership selection supported.

**Catalog browsing:** **WIRED**

GET /v1/catalog/categories and /v1/catalog/products are live with supplier filtering. Spanner-backed.

**Order placement:** **WIRED**

POST /v1/order/create is live, idempotent, flows through the real state machine.

**Unified checkout (cash/card):** **WIRED**

POST /v1/checkout/unified is live. Cash and Global Pay card both work. CREDIT is not a selectable payment method on any client despite backend support.

**Order tracking:** **WIRED**

WebSocket push (not naive polling) plus GET /v1/retailer/tracking fallback. Real driver location via TelemetryHub broadcast.

**Claims + photo upload:** **WIRED**

POST /v1/orders/{id}/claims with media upload ticket. Real handler in claims/handlers.go.

**Offline order queue:** **WIRED**

PendingOrderSyncWorker (Kotlin WorkManager) queues orders in Room DB and replays with stored idempotency keys. Backend idempotency middleware resolves duplicates safely.

**AI predictions:** **DECORATIVE**

GET /v1/ai/predictions returns empty array [] from HandleAIPredictionsAlias. GET /v1/retailer/ai/predictions calls s.orders.ListRetailerAIPredictions — real Spanner read, but ai-worker must be deployed and consuming Kafka for data to exist.

**AI pre-order:** **REMOVED**

POST /v1/ai/preorder returns HTTP 410 Gone. Feature explicitly removed.

**Card management:** **PARTIAL**

POST /v1/retailer/card/initiate and /confirm are live (Global Pay tokenization). GET /v1/retailer/cards returns hardcoded empty array from HandleRetailerCards in mobile_compat.go.

**Auto-order settings:** **WIRED (shadow)**

PATCH /v1/retailer/settings/auto-order/* is live. Worker runs every 15 minutes. AUTO_ORDER_PLACE_ENABLED=false means orders are drafted, not placed. AUTO_ORDER_SHADOW_ENABLED defaults true.

**Reorder suggestions:** **WIRED**

GET /v1/retailer/reorder-suggestions reads from ReorderSuggestionWorker output.

**Store stock receive/count:** **WIRED**

POST /v1/retailer/stock/counts and /commit are live with version control. Android has full receive/transfer/adjust/counts. iOS has receive/transfer/adjust; count-commit flow not confirmed.

**POS (mobile):** **WIRED**

POS sessions and sales on Android and iOS. Holds confirmed in backend route table; not verified in Android/iOS API layer.

**Credit profile view:** **DECORATIVE**

GET /v1/retailer/credit-profile displays balances/invoices. No repay endpoint exists anywhere in the backend.

**Receipts:** **WIRED**

Android/iOS use public QR receipt route (/v1/platform/receipts/{id}). Desktop uses authenticated /v1/retailer/orders/{id}/receipt. Both work.

**Notifications inbox:** **WIRED**

GET /v1/user/notifications and POST /read are live.

**Default locale:** **PARTIAL**

packages/i18n/locales.ts defaults to English. Full uz.json and ru.json catalogs exist. First-run experience for a Tashkent shopkeeper is likely in the wrong language unless device locale detection overrides.

### **5.1.2  Retailer Desktop / Web Portal**

**POS sessions and sales:** **WIRED**

POS session open/close, sales recording, holds management. Real.

**Stock counts:** **WIRED**

POST /v1/retailer/stock/counts and /commit are live with version control.

**Reports:** **WIRED**

GET /v1/retailer/reports/sales, /summary, /export are live.

**Credit management:** **WIRED**

Credit relationships and profiles visible. No repay action.

**Auto-order settings:** **WIRED**

Full auto-order configuration UI including per-supplier, per-category, per-product rules.

**Shift management:** **WIRED**

Clock-in/clock-out, time entries. Real.

**Control tower pulse:** **DECORATIVE**

GET /v1/retailer/control-tower/pulse — endpoint exists but ControlTowerWorker processes warehouse/supplier events, not retailer-specific pulse data.

**Reorder suggestions:** **WIRED**

Present in desktop app.

**POS (desktop):** **WIRED**

Full POS UI in desktop app.

## **5.2  Supplier**

### **5.2.1  Supplier Portal (Next.js web + Tauri desktop)**

**Dashboard:** **WIRED**

GET /v1/supplier/dashboard returns real order counts, revenue, status breakdown.

**Order management:** **WIRED**

GET /v1/supplier/orders with filtering, vetting (accept/reject), real state transitions.

**Catalog management:** **WIRED**

CRUD for products with GTIN/EAN validation and image upload via media ticket.

**Inventory management:** **WIRED**

GET/PATCH /v1/supplier/inventory with policy configuration. CSV import with staged rows and column mapping.

**Dispatch preview and execute:** **WIRED**

POST /v1/supplier/dispatch/preview and /execute are live with idempotency. Preview returns bin-pack assignment. Execute commits to Spanner and emits outbox event.

**Manifests:** **WIRED**

List, seal, dispatch, complete. Real.

**Earnings:** **WIRED**

GET /v1/supplier/earnings reads from Spanner via SetEarningsLookup.

**Fleet management:** **WIRED**

Driver and vehicle CRUD. Live map via WebSocket.

**Compliance dashboard:** **WIRED**

GET /v1/compliance/dashboard returns fiscal-open orders, force-completes, claim mismatches, credit freezes. Real.

**Credit management:** **WIRED**

Credit program setup, retailer credit profiles, dunning inbox. AR_DUNNING_ENABLED=false in production — dunning worker runs but does not send notifications.

**Promotions:** **WIRED**

CRUD for promotions. Promotion struct only supports flat DiscountBps + scope — no BOGO/bundle/tiered mechanics. Elasticity sandbox (planning/promo_eval.go) exists but is not mounted to any route.

**Replenishment policies:** **WIRED (shadow)**

GET/POST /v1/supplier/replenishment/policies is live. Engine runs but FORECAST_ALGO_ENABLED=false and SAFETY_STOCK_V2_ENABLED=false in production.

**AI recommendations:** **PARTIAL**

GET /v1/supplier/ai/recommendations reads from ai-worker output. ai-worker must be deployed and consuming Kafka events for data to exist.

**Control tower:** **DECORATIVE**

GET/POST /v1/supplier/control-tower/zone-overrides stores overrides. ControlTowerWorker runs but zone overrides do not influence the bin-packing assignment. Portal control-tower page has hardcoded empty arrays for performanceData and scenariosData.

**MEIO network summary:** **DECORATIVE**

GET /v1/supplier/meio/network-summary returns a summary structure but no MEIO solver is running.

**S&OP planning:** **DECORATIVE**

GET /v1/supplier/planning/s-and-op returns data but no real S&OP process engine is behind it.

**Seasonal estimates:** **DECORATIVE**

GET/POST /v1/supplier/planning/seasonal-overrides stores data. No seasonal model consumes it in production.

**Knowledge graph:** **WIRED**

GET /v1/supplier/knowledge-graph calls svc.GetKnowledgeGraph — Spanner-backed planning package.

**Planning agent invoke:** **DECORATIVE**

POST /v1/supplier/planning/agent/invoke accepts requests but no LLM or agent backend is wired.

**Partner EDI / AS2 / SFTP:** **WIRED (scaffolding)**

Full CRUD for partner keys, webhooks, EDI documents, AS2 config, SFTP config. AS2 disabled in production.

**Supplier billing/earnings UI:** **BROKEN**

Supplier apps call POST /v1/supplier/billing/setup and GET /v1/supplier/earnings that do not exist in any Go route file. Only POST /v1/admin/billing/run-monthly exists (admin-role gated).

**Payday calendar:** **BROKEN**

demand/payday-calendar/page.tsx calls fetch('/api/demand/signals?type=PAYDAY') — a bare Next.js relative path with no matching route. Will 404. Real endpoint is /v1/demand/signals.

**Supplier desktop app:** **ABSENT**

apps/supplier-app-desktop is a redirect stub. package.json: 'Discoverability anchor — desktop supplier app lives in supplier-portal (Next.js + Tauri 2).' No separate codebase.

### **5.2.2  Supplier Mobile (Android / iOS)**

Supplier Android (142 Kotlin files) mirrors the portal almost 1:1: catalog, pricing, promotions, orders, dispatch/fleet-live-map, planning (PlanningBrainScreen, KnowledgeGraphScreen, ReplenishmentPoliciesScreen), analytics, treasury (credit notes/profiles/ledger/reconciliation/chargebacks), network/topology, org-fleet, exceptions, manifests, billing, notifications. iOS (131 Swift files) mirrors the same module set. Retrofit SupplierApi.kt declares matching @GET/@POST 'v1/supplier/...' paths identical to Go route registrations.

Missing on mobile vs portal: no dedicated Control Tower live-map screen (portal's WS-driven network graph has no mobile equivalent). No payday-calendar screen (consistent with it being a broken portal page). No segmentation screen found by name on mobile.

## **5.3  Driver**

**Login:** **BROKEN / CRITICAL SECURITY ISSUE**

POST /v1/auth/driver/login accepts any phone number with PIN 1234 (or DRIVER_DEMO_PIN env var). Issues JWT for hardcoded driver ID drv_factory_1. No per-driver credential database. Firebase ID token path exists but falls back to the same hardcoded driver ID. Contrast with warehouse/factory login which use bcrypt.CompareHashAndPassword.

`[driver/auth_login.go:55-108]`

**Manifest view:** **WIRED**

GET /v1/driver/manifest and /v1/fleet/manifest return real manifest data for the authenticated driver.

**Route geometry + navigation:** **WIRED**

GET /v1/fleet/route/{id}/geometry returns real route polyline. Android renders via Google Maps SDK Polyline. iOS renders via FleetMapView.swift. DB-backed.

**Order state updates:** **WIRED**

PATCH /v1/orders/{id}/state validates against the state machine. Real transitions only.

**QR delivery scan:** **WIRED**

POST /v1/delivery/scan-qr validates delivery tokens. Real. DB-backed.

**Cash collection:** **WIRED**

POST /v1/order/collect-cash with amount_received_minor, latitude, longitude. DB-backed. Geofence-gated at 100m.

**Credit delivery:** **WIRED**

POST /v1/delivery/credit-delivery with photo_proof_url. Real. Separate /v1/driver/orders/{id}/credit-leave endpoint exists but has no photo field and is unused by either client.

**Shop-closed reporting:** **WIRED**

POST /v1/delivery/shop-closed with photo. Triggers inventory release and exception logging.

**Partial offload:** **WIRED**

POST /v1/delivery/partial-offload handles partial delivery with state machine validation.

**Returns pickup list:** **WIRED**

GET /v1/driver/return-goods returns real Spanner query results. Mounted via returnsroutes.RegisterDriverRoutes (not driverroutes).

**Earnings:** **WIRED but NOT DB-backed**

GET /v1/driver/earnings reads from in-memory earningsMinor map[string]int64 (driver/service.go:467). Earnings hook is never wired in bootstrap.go. Data lost on pod restart. Last30Days always empty.

**GPS telemetry:** **BROKEN**

Driver apps send TelemetryPayload over WebSocket every 5 seconds. Server read-loop (ws/handler.go:273) discards all incoming bytes. Real DB-backed sink is POST /v1/telemetry/location (telemetryroutes/routes.go:106) — never called by either driver client.

**Availability:** **WIRED**

GET/PATCH /v1/driver/availability with real persistence.

**Rescue request:** **WIRED**

POST /v1/driver/ops/rescue/request and /respond are live.

**Pending collections:** **WIRED**

GET /v1/driver/pending-collections returns real cash collection list.

**Open fiscal:** **WIRED**

GET /v1/driver/open-fiscal returns orders in FISCALIZING state for the driver.

**Offline queue:** **WIRED**

DriverOfflineQueue.kt and OfflineSyncWorker.kt queue actions locally and drain on reconnect. POST /v1/sync/batch is live.

## **5.4  Payloader (Warehouse Terminal + Native Apps)**

**Login:** **WIRED**

POST /v1/auth/payloader/login — full parity across terminal, Android, iOS.

**Manifest checklist:** **WIRED**

GET /v1/payloader/manifests and /{id} detail. DB-backed.

**Scan-to-load:** **WIRED**

Barcode scanner via expo-camera (terminal), DataWedge + CameraX (Android), EANBarcodeScannerView (iOS). GET /v1/catalog/barcode/{ean} lookup. Native scanner packages exist for both platforms.

**Start loading / inject order:** **WIRED**

POST /v1/payloader/manifests/{id}/start-loading and /inject-order. Both /v1/payloader/* and /v1/supplier/* variants mounted.

**Seal manifest:** **WIRED**

POST /v1/payloader/manifests/{id}/seal commits seal with outbox event.

**Exceptions:** **WIRED**

GET /v1/payloader/manifest-exceptions with resolve endpoint.

**Reassign / fleet-reassign:** **WIRED**

POST /v1/payloader/reassign-order and /recommend-reassign. Real.

**Returns inbound gate:** **WIRED**

Android/iOS have getInboundReturns, scanInboundReturn, confirmInboundReturns. DB-backed. Terminal lacks this feature.

**GS1 SSCC labels:** **DISABLED**

Backend fully built (ship-units, ZPL labels, SSCC generation). GS1_LABELS_ENABLED defaults true. But payload-terminal never calls the ship-units/labels endpoints — client-side decorative.

**LIFO load-sequence hints:** **ABSENT**

No LIFO/hint endpoint found in any of the 3 client API layers or backend packages.

**Duplicate route package:** **DECORATIVE**

apps/backend-go/payloaderoutes (single-r) is dead code — near-duplicate of payloaderroutes (double-r) with extra ship-units/labels routes. Never imported/mounted in main.go. Maintenance trap.

## **5.5  Warehouse**

**Login / setup:** **WIRED**

POST /v1/auth/warehouse/login and /register are live. bcrypt password hashing.

**Ops dashboard:** **WIRED**

GET /v1/warehouse/ops/dashboard returns real order counts and status.

**Order management:** **WIRED**

GET /v1/warehouse/ops/orders with delay, reject, overflow, propose-delivery actions.

**Dispatch preview / execute:** **WIRED**

POST /v1/warehouse/ops/dispatch/preview and /execute are live.

**Dispatch locks:** **WIRED**

GET/POST/DELETE /v1/warehouse/dispatch-lock prevents concurrent dispatch execution. DB-backed.

**WMS — bins:** **WIRED**

GET/POST/PATCH /v1/warehouse/ops/bins for bin location management. Feature-flag gated.

**WMS — lots / putaway:** **WIRED**

GET /v1/warehouse/ops/lots and POST /lots/putaway are live. FEFO picking (stocklots/picking.go:148-161 sorts by ExpiryDate ASC).

**WMS — pick waves:** **WIRED**

Create, list, confirm pick tasks. Feature-flag gated (PickWavesEnabled).

**WMS — cycle counts:** **WIRED**

Create, submit, ABC enqueue. Feature-flag gated (CycleCountsEnabled).

**WMS — inventory adjustments:** **WIRED**

Adjustments with approve/reject workflow. Admin-gated.

**WMS — temperature readings:** **WIRED**

GET/POST /v1/warehouse/ops/temperature-readings. DB-backed. Cold-chain quarantine fires server-side automatically (stocklots/coldchain.go:227). No UI to view readings — backend-only.

**Inventory reconcile:** **WIRED**

GET /v1/warehouse/ops/inventory-reconcile and /inventory-accuracy.

**Stock commitments:** **WIRED**

GET /v1/warehouse/ops/stock-commitments shows real reserved stock.

**Demand forecast:** **WIRED**

GET /v1/warehouse/demand/forecast calls order.WarehouseDemandForecast — real order history aggregation by delivery window. Not ML, but real data.

**Treasury:** **DECORATIVE**

GET /v1/warehouse/ops/treasury?view=invoices returns hardcoded empty array. Comment: 'Fail-closed honesty: never return hardcoded fake invoices/totals.'

**CRM:** **DECORATIVE**

GET /v1/warehouse/ops/crm — implementation returns minimal data.

**Financials:** **DECORATIVE**

GET /v1/warehouse/ops/financials — same fail-closed pattern as treasury.

**Returns page:** **DECORATIVE**

HandleOpsReturns: if Spanner present and seed disabled, always returns {items: []}. Otherwise serves in-memory seed data. Never queries real return records.

**Supply requests:** **WIRED**

GET/POST/PATCH /v1/warehouse/supply-requests with fulfill-options. Cross-service with factory.

**Emergency transfer:** **WIRED**

POST /v1/warehouse/transfers/emergency and /force-receive.

**Fleet live map:** **WIRED**

GET /v1/warehouse/ops/fleet/live-map via WebSocket. Real route/yard query.

**Replenishment insights:** **WIRED**

GET /v1/warehouse/replenishment/insights with action endpoint.

## **5.6  Factory**

**Login / setup:** **WIRED**

POST /v1/auth/factory/login and /register are live. bcrypt password hashing.

**Dashboard:** **WIRED**

GET /v1/factory/dashboard returns real metrics.

**Transfers:** **WIRED**

GET/POST /v1/factory/transfers with transition state machine.

**Manifests:** **WIRED**

Full manifest lifecycle: start-loading, seal, dispatch, complete, rebalance, cancel.

**Manifest exceptions:** **WIRED**

GET /v1/factory/manifest-exceptions with resolve endpoint.

**Fleet:** **WIRED**

GET /v1/factory/fleet/drivers and /vehicles.

**Fleet live map:** **WIRED**

GET /v1/factory/fleet/live-map via WebSocket.

**Staff:** **WIRED**

GET/POST /v1/factory/staff.

**Dispatch:** **WIRED**

POST /v1/factory/dispatch triggers dispatch for factory manifests.

**Supply requests:** **WIRED**

GET /v1/factory/supply-requests with accept and fulfill-options.

**Analytics:** **WIRED**

GET /v1/factory/analytics/overview returns real metrics.

**Loading bay:** **WIRED but thin**

Read-only Kanban + one dispatch button. Actual start-loading/seal/inject-order actions live only in payload apps, not the factory portal.

**Production planning (BOM/MRP):** **ABSENT**

No bill_of_materials or production_order tables in schema. No matching routes. Confirmed absent — factory has no manufacturing logic.

────────────────────────────────────────────────────────────────────────

# **Section 6 — Recommendations and Prioritized Gap List**

## **6.1  Critical (fix immediately — security and correctness)**

- **Driver auth is a hardcoded demo**

Replace driver/auth_login.go with per-driver credential verification against a Spanner table (DriverCredentials with phone, PIN hash, driver_id). Remove DRIVER_DEMO_PHONE / DRIVER_DEMO_PIN fallback. Warehouse and factory login already use bcrypt.CompareHashAndPassword — apply the same pattern. This is a production security vulnerability.

- **Tenant suspension is not enforced**

platformadmin.Service.IsActive exists but no middleware calls it. A suspended tenant with a valid JWT can still make API calls. Add a middleware check after auth that calls IsActive for the resolved tenant.

- **Driver GPS telemetry is broken**

Driver apps send GPS over WebSocket every 5 seconds. Server read-loop (ws/handler.go:273) discards all incoming bytes. Either make /v1/ws parse and forward TelemetryPayload frames to the LastLocations store, or switch both driver clients to POST /v1/telemetry/location. One-line client change or small server-side WS frame parser.

- **Driver earnings are in-memory**

driver/service.go:467 reads from earningsMinor map[string]int64 — process memory, lost on restart. Earnings hook is never wired in bootstrap.go. Wire to a Spanner query over completed orders/fiscal receipts.

- **Supplier billing endpoints missing**

Supplier apps call POST /v1/supplier/billing/setup and GET /v1/supplier/earnings that do not exist in any Go route file. Only POST /v1/admin/billing/run-monthly exists (admin-role gated). Add supplier-scoped handlers reading BillingFeeSchedules/BillingMeterEvents.

- **Payday calendar page is broken**

demand/payday-calendar/page.tsx calls fetch('/api/demand/signals?type=PAYDAY') — a bare Next.js relative path with no matching route. Fix to call /v1/demand/signals via the API client.

## **6.2  High priority (blocks real customer value)**

- **Enable OR-Tools optimizer**

Publish optimizer-core Docker image to Artifact Registry. Raise replicas from 0 to 1+ in prod kustomization. The bin-pack fallback is functional but produces suboptimal routes. Backend client already handles graceful degradation.

- **Enable AUTO_ORDER_PLACE_ENABLED**

After verifying the payment capture path for auto-drafted orders, enable place mode in staging. This is the single highest-value automation feature. AUTO_ORDER_SHADOW_ENABLED already defaults true.

- **Complete MY_SOLIQ fiscal integration**

Replace DevHMACSigner with real PKCS#7 EDS signing. Test against a live Uzbekistan OFD gateway. Explicitly blocked on EDS key procurement (signer_env.go:69-70). Required for legal receipt generation.

- **Build 1C CommerceML connector**

Critical for Uzbekistan market. Implement real CommerceML 2.x XML producer/consumer (catalog.xml/offers.xml/import.xml) in the Go backend. Use scripts/commerceml_import_ref.py as spec source.

- **Wire ControlTowerWorker to dispatch**

The worker runs but its zone overrides do not influence the bin-packing assignment. Connect the two.

- **Add credit repayment endpoint**

No repay endpoint exists anywhere in the backend. A retailer on trade credit cannot pay down their balance digitally. Add POST /v1/retailer/credit/repay (or AR payment) handler.

- **Add CREDIT as a checkout payment method**

Backend has a CREDIT executor (payment/execution.go:167) but no client exposes it as a selectable payment method. Add to checkout UI on all three retailer clients.

- **Fix warehouse returns page**

HandleOpsReturns always returns {items: []} when Spanner is present and seed is disabled. Wire to real ReturnId/OrderId rows from Spanner (mirror the pattern in returns/inbound.go).

## **6.3  Medium priority (completeness)**

- **Replace AI heuristic with the already-implemented statistical models**

planning/forecast/ has real Croston/SBA + SES + Holt-Winters with SBC classification. Enable FORECAST_ALGO_ENABLED and wire the forecast output into the ai-worker synthesis path.

- **Enable GS1 labels in payload-terminal**

Backend ship-units and ZPL label endpoints are fully built and GS1_LABELS_ENABLED defaults true. Add a print/scan screen to payload-terminal calling the existing endpoints.

- **Enable AR dunning notifications**

AR_DUNNING_ENABLED=false in production. The worker runs but does not send dunning notifications. Enable after legal review of notification content.

- **Wire warehouse treasury/financials**

HandleOpsTreasury and HandleOpsFinancials return empty/hardcoded data. Wire to real AR invoice and payment data from Spanner.

- **Enable PARTNER_AS2**

AS2 receive endpoint exists with real PKCS7 sign+encrypt but is disabled. Implement async MDN. Test with a real trading partner before enabling.

- **Remove or implement planning agent**

POST /v1/supplier/planning/agent/invoke accepts requests but has no backend. Either wire an LLM agent or return 410 Gone.

- **Implement MEIO solver**

The MEIO endpoint returns a static summary. Implement or integrate a real multi-echelon inventory optimization solver.

- **Add promotion mechanics**

Promotion struct only supports flat DiscountBps. Add mechanic_type field (BOGO/bundle/tiered) and evaluator branch. Mount the elasticity sandbox (planning/promo_eval.go) to a route.

- **Expand webhook event coverage**

Only 13 of ~159 internal event types are exposed via partner webhooks. Expand PartnerWebhookableEvents allowlist with per-partner opt-in.

- **SFTP host-key enforcement by default**

Require pinned host keys in production overlays, not opt-in via PARTNER_SFTP_STRICT_HOSTKEY.

- **Build cold-chain temperature UI**

Temperature readings are stored and quarantine fires automatically. No portal or mobile page consumes the readings endpoint.

- **Add LIFO load-sequence hints to payload-terminal**

No LIFO/hint endpoint found in any client or backend package. Add sequence field on manifest items and UI sort/badge.

- **Set default locale to Uzbek or Russian**

packages/i18n/locales.ts defaults to English. Full uz.json and ru.json catalogs exist. Change default for the Uzbekistan market.

- **Consolidate duplicate payloaderoutes package**

apps/backend-go/payloaderoutes (single-r) is dead code — near-duplicate of payloaderroutes (double-r). Never mounted. Delete it.

- **Merge credit-leave and credit-delivery**

credit-leave accepts no photo field. credit-delivery has photo_proof_url. Merge so photo/signature capture is enforced on all credit-exit paths.

## **6.4  Architecture observations**

- **Dual payloaderoutes package**

apps/backend-go/payloaderoutes/routes.go and apps/backend-go/payloaderroutes/routes.go both exist with nearly identical route definitions. Only payloaderroutes is mounted (main.go:192). payloaderoutes is dead code with extra ship-units/labels routes. Delete it.

- **Retailer mobile_compat.go stubs**

retailer/mobile_compat.go contains HandleRetailerCards (returns []), HandleAIPredictionsAlias (returns []), HandleAIPreorder (returns 410). These exist to satisfy mobile client contracts without breaking the app. Clearly mark as stubs or replace with real implementations.

- **Feature flag sprawl**

20+ feature flags control behavior across the codebase. Many are false in production and true in staging. This creates a two-tier reality where staging behavior differs materially from production. Consider a feature flag management system or at minimum a single source of truth document.

- **Optimizer-core deployment gap**

optimizer-core is fully implemented in Python (OR-Tools pywrapcp, multi-depot VRP, capacity constraints, time windows) but has never been deployed to production. The H3 bin-pack fallback is functional but produces routes that are likely 15-25% longer than optimal.

- **i18n coverage**

en.json, ru.json, uz.json have full key parity. Some Uzbek translations are identical to English (untranslated). Default locale is English. Not a blocking issue but worth noting for the Uzbekistan market.

- **admin-portal is a dead stub**

apps/admin-portal/package.json: 'Deprecated stub — canonical admin/supplier surface is supplier-portal.' All scripts invoke redirect.mjs. Safe to delete or prune from CI matrices.

- **supplier-app-desktop is a redirect stub**

apps/supplier-app-desktop/redirect.mjs prints 'supplier-app-desktop has no separate codebase' and exits. Intentional pointer to supplier-portal via Tauri. Not decorative code to flag as broken.

- **Routing uses haversine, not road network**

eta/calculator.go uses straight-line haversine distance. No OSRM/Google Routes/HERE integration. OR-Tools VRP exists for dispatch optimization but is a single-replica soft dependency with graceful fallback to the same haversine+bin-pack heuristic.

- **Webhook event coverage is narrow**

Only 13 of ~159 internal event types are exposed via partner webhooks (kafka_handler.go:28-61, webhook_events.go:9-23). Large retailers integrating with SAP/OMS typically want returns, pricing, promotions, and B2B invoicing events.

- **Freeze registry is in-memory only**

ai-worker/freeze_registry.go uses an in-memory TTL map (5-minute default). No cross-instance persistence. Risk on multi-pod ai-worker deployment.

## **6.5  What a distributor replacing van-sales needs (absent)**

- **Route settlement / cash-up reconciliation per driver-day**

Close out a driver's route with expected vs. collected cash, short/over detection. Portal has cash-reconciliations but it's supplier-level batch, not per-route/per-driver daily settlement.

- **Territory/route assignment engine**

Assign retailers to fixed daily/weekly delivery routes (classic van-sales beat plan). delivery-zones/supply-lanes exist but no beat-plan/route-roster generator was found.

- **On-truck real-time inventory decrement**

Track what's physically on the van vs. warehouse stock as sales happen mid-route. Manifests/ship-units exist for warehouse dispatch, but no van-level running-stock ledger was evidenced.

- **Pre-sell vs. van-sell order type distinction**

Differentiate orders taken a day ahead (pre-sell) from same-visit cash sales (van-sell) for planning accuracy. No such tag was found in order/catalog types.

- **Driver commission/incentive calculation**

Pay drivers/reps based on route sales performance. EarningsScreen exists but is supplier-level P&L, not driver-level incentive computation.

- **Empties/returnable-asset tracking**

Track deposit-asset custody (crates, kegs, bottles) as they move supplier→retailer→back. Returns domain covers product returns/claims, not returnable-asset custody.

────────────────────────────────────────────────────────────────────────

# **Appendix — Key File References**

**Order state machine:** `apps/backend-go/order/state_machine.go`

**Fiscal provider selection:** `apps/backend-go/order/fiscal_provider.go`

**Payment execution router:** `apps/backend-go/payment/execution.go`

**Global Pay executor:** `apps/backend-go/payment/global_pay_executor.go`

**Global Pay webhook:** `apps/backend-go/payment/global_pay_webhook.go`

**Payme webhook:** `apps/backend-go/payment/payme_webhook.go`

**Click webhook:** `apps/backend-go/payment/click_webhook.go`

**Transactional outbox:** `apps/backend-go/outbox/outbox.go`

**Outbox relay:** `apps/backend-go/outbox/relay.go`

**Driver auth (SECURITY ISSUE):** `apps/backend-go/driver/auth_login.go`

**Driver earnings (in-memory):** `apps/backend-go/driver/service.go:467`

**WS telemetry discard loop:** `apps/backend-go/ws/handler.go:273`

**Real telemetry sink (unused):** `apps/backend-go/telemetryroutes/routes.go:106`

**AI synthesis engine:** `apps/ai-worker/synthesis/engine.go`

**Forecast models (Croston/Holt-Winters):** `apps/backend-go/planning/forecast/`

**Safety stock formula:** `apps/backend-go/replenishment/safety_stock.go:80-104`

**Replenishment engine:** `apps/backend-go/replenishment/engine.go`

**Bin-pack dispatch:** `apps/backend-go/dispatch/binpack.go`

**OR-Tools optimizer:** `services/optimizer-core/server/contract_solver.py`

**Optimizer client (soft dep):** `apps/backend-go/dispatch/optimizerclient/client.go`

**Partner API routes:** `apps/backend-go/partner/routes.go`

**Partner order creation:** `apps/backend-go/partner/handlers.go:45`

**Partner webhook delivery:** `apps/backend-go/partner/delivery.go`

**EDI DESADV builder:** `apps/backend-go/partner/edi/desadv.go`

**AS2 crypto:** `apps/backend-go/partner/as2/crypto.go`

**SFTP client:** `apps/backend-go/partner/sftp.go`

**POS demand feed:** `apps/backend-go/partner/pos_demand_feed.go:110`

**GS1 validators:** `apps/backend-go/gs1/checkdigit.go`

**GS1 ZPL labels:** `apps/backend-go/gs1/zpl.go`

**Retailer mobile stubs:** `apps/backend-go/retailer/mobile_compat.go`

**Warehouse treasury (empty):** `apps/backend-go/warehouse/ops_portal.go:752`

**Warehouse returns (empty):** `apps/backend-go/warehouse/ops_portal.go:678`

**Production feature flags:** `infra/k8s/backend-go/configmap.yaml`

**Optimizer prod replicas=0:** `infra/k8s/overlays/prod/kustomization.yaml:60-70`

**Auto-order worker:** `apps/backend-go/retailer/auto_order_worker.go`

**Cash recon escalation:** `apps/backend-go/runtime_workers.go`

**Tenant admin (no middleware):** `apps/backend-go/platformadmin/service.go`

**Auth tenant middleware:** `apps/backend-go/auth/tenant.go`

**Spanner schema:** `apps/backend-go/schema/spanner.ddl`

**Proximity settlement:** `apps/backend-go/order/proximity_settlement.go`

**Shop-closed handler:** `apps/backend-go/order/shop_closed.go`

**Dev HMAC signer (fiscal):** `apps/backend-go/fiscal/signer_env.go`

**Dead payloaderoutes package:** `apps/backend-go/payloaderoutes/routes.go`

**Soliq client interface:** `apps/backend-go/soliq/client.go`

**Promotion elasticity (sandbox):** `apps/backend-go/planning/promo_eval.go`

**Segment rule table:** `apps/backend-go/segment/priority.go`

**ETA haversine calculator:** `apps/backend-go/eta/calculator.go`

**Freeze registry (in-memory):** `apps/ai-worker/freeze_registry.go`

**Catalog import pipeline:** `apps/backend-go/supplier/import_sessions.go`

**Cold-chain quarantine:** `apps/backend-go/stocklots/coldchain.go:227`

**FEFO picking:** `apps/backend-go/stocklots/picking.go:148-161`

*Report generated from live code audit of pegasusX/ monorepo, incorporating findings from six independent subagent audits. All findings verified against source files. No reliance on documentation or README claims.*
