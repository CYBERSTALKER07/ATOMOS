# Role 1 Specification: Platform Owner & Global Platform Administrator

> **Role Number**: Role 1 of 9  
> **Persona ID**: `ROLE_PLATFORM_ADMIN` / `ROLE_PLATFORM_OWNER`  
> **Document Scope**: Platform Administration, Global Control Tower, Dual-Control Governance, Multi-Tenant Cell Federation, Outbox Ops, Schema Drift Controls & Cross-Tenant Oversight  
> **Repository Root**: `/Users/shakhzod/Desktop/pegasusx`  
> **Key Code Locations**:  
> - Web Console: [`apps/admin-portal`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal)  
> - Backend Platform Admin: [`apps/backend-go/platformadmin`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformadmin)  
> - Backend Platform Routes: [`apps/backend-go/platformroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformroutes)  
> - Backend Control Tower: [`apps/backend-go/controltower`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltower), [`apps/backend-go/controltowerroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltowerroutes)  
> - Backend Infrastructure Probes: [`apps/backend-go/infraroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/infraroutes)  
> - Feature Flag Dual Control: [`apps/backend-go/featureflags`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/featureflags)  
> - Multi-Factor Authentication: [`apps/backend-go/mfa`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/mfa)  
> - Multi-Tenancy & Cells: [`apps/backend-go/auth/tenant.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/tenant.go), [`apps/backend-go/auth/cell_directory.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/cell_directory.go), [`apps/backend-go/auth/market_pack.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/market_pack.go)  
> - Tenant Registration: [`apps/backend-go/tenantreg`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/tenantreg)  
> - Global Products & Entity Resolution: [`apps/backend-go/globalproducts`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/globalproducts), [`apps/backend-go/entityresolutionroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/entityresolutionroutes)  
> - Schema Drift Gate: [`apps/backend-go/schemadrift`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift)  
> - Simulator & Chaos Engine: [`apps/backend-go/simulator`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/simulator)  
> - Outbox Engine: [`apps/backend-go/outbox`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/outbox)  
> - Platform Billing: [`apps/backend-go/internal/services/billing`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/internal/services/billing)  

---

## 1. Executive Summary & Operational Charter

The **Platform Owner & Platform Admin (Role 1)** represents the sovereign governance and observability tier of the PegasusX ecosystem. Unlike tenant-specific personas (Supplier, Retailer, Driver, Warehouse, Factory, Payload) that operate within strictly bounded organizational and regional silos, the Platform Admin operates across the entire federated network.

The Platform Admin role enforces five fundamental pillars:
1. **Multi-Tenant Isolation & Cell Federation**: Enforcing request-scoped tenancy (`TenantContext`), preventing Cross-Tenant IDOR, managing regional cells (`cell-uz`, `cell-eu`, `cell-us`, `cell-kz`), and validating market pack contracts (`UZ`, `EU`, `US`, etc.).
2. **Dual-Control Governance (Four-Eyes Principle)**: Requiring two independent platform administrators to approve high-risk mutations, including money-affecting feature flags (`AR_INVOICES_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`, `FISCAL_PROVIDER`) and tenant KYB state transitions (`PENDING -> APPROVED`). Single-admin authorization is strictly rejected by the database and service layer.
3. **Mandatory Step-Up MFA**: Requiring Time-based One-Time Passwords (TOTP RFC 6238) for all platform administration mutations. Authentication claims must present `mfa_verified: true` before executing governance actions.
4. **Global Control Tower & Scored Exception Engine**: Orchestrating ecosystem-wide telemetry, live EKG network topologies, H3 spatial order density heatmaps, and automated playbook execution against logistics bottlenecks and delivery disputes.
5. **Operational Honesty & Schema Parity**: Continuously monitoring Cloud Spanner outbox publication lag (`Idx_OutboxEvents_Unpublished`), surfacing Spanner dead letters (`OutboxDeadLetters`), providing atomic replay capabilities, and enforcing zero database schema drift across environments.

> [!IMPORTANT]
> Platform Admin JWTs carry `role: PLATFORM_ADMIN` and intentionally **omit** `SupplierId`. Under [`auth.RequireTenant`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/tenant.go#L131-L137), Platform Admin sessions are explicitly exempted from trading-partner scoping to allow cross-tenant inspection and governance while all tenant-facing mutating APIs remain fail-closed.

---

## 2. Platform Admin Metrics & Surface Inventory

### 2.1. Exact Metric Counts

| Metric Category | Exact Count | Codebase Reference & Scope |
|---|:---:|---|
| **Admin Portal Screens / Views** | **16** | [`apps/admin-portal/app/page.tsx`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/app/page.tsx), [`components/`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal/components) |
| **Platform Admin & Control Tower API Endpoints** | **81** | `platformadmin`, `platformroutes`, `featureflags`, `mfa`, `controltowerroutes`, `infraroutes`, `globalproductsroutes`, `partner`, `billing`, `planning`, `fxrates`, `taxroutes`, `ws` |
| **Governance & Control Tower WebSocket/SSE Events** | **21** | `ws.PlatformAdminRoom()`, `telemetry:supplier:{id}`, Kafka outbox event types, simulator streams |
| **Spanner Database Entities Touched** | **34** | 34 tables directly mutated/queried by platform administration out of 232 total tables in [`spanner.ddl`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schema/spanner.ddl) |
| **Multi-Tenant Regional Cells** | **4** | `cell-uz` (Live/Shipped), `cell-eu` (Planned), `cell-us` (Planned), `cell-kz` (Planned) |
| **Catalog Market Packs** | **8** | `UZ` (Shipped), `EU`, `US`, `CA`, `AU`, `GB`, `KZ`, `PK` (Planned) |
| **Money-Affecting Flags Under Dual Control** | **6** | `AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`, `AUTO_ORDER_PLACE_ENABLED`, `AUTO_ORDER_SHADOW_ENABLED`, `AUTO_ORDER_SOAK_GATE_DISABLED`, `FISCAL_PROVIDER` |

---

### 2.2. Admin Portal Views Breakdown (16 Screens / Views)

The Admin Portal ([`apps/admin-portal`](file:///Users/shakhzod/Desktop/pegasusx/apps/admin-portal)) is structured as a reactive single-page command cockpit with step-up security gates and 10 operational console panels:

```
apps/admin-portal
├── Pre-Auth & Authentication Layer
│   ├── [View 01] LoginForm (G4.B Durable Password Authentication + Dev Break-Glass JWT Paste)
│   ├── [View 02] MfaGate: Loading (Initial session & MFA enrollment state discovery)
│   ├── [View 03] MfaGate: Error (Challenge rejection, rate-limit failure & session reset)
│   ├── [View 04] MfaGate: NeedEnroll (Mandatory TOTP authenticator enrollment requirement)
│   ├── [View 05] MfaGate: Enroll (Base32 secret export, otpauth:// URI & 6-digit confirmation)
│   └── [View 06] MfaGate: Verify (6-digit one-time code challenge before console mount)
└── Operational Command Cockpit (AdminConsole Tabs)
    ├── [View 07] CommandBoard (Ecosystem health cards, outbox gauges, queue rollups & tenant table)
    ├── [View 08] TenantsPanel (Multi-tenant directory, KYB legal entity review, cell assignment & dual-control transitions)
    ├── [View 09] FlagsPanel (Dual-control feature flag evaluation, override requests & secondary admin approvals)
    ├── [View 10] OpsPanel (Runtime worker heartbeat, outbox lag monitor, unpublished event stream & Spanner dead-letter replay)
    ├── [View 11] BillingPanel (Platform fee schedules, monthly billing execution & invoice accounts receivable)
    ├── [View 12] AccuracyPanel (AI planning MAPE28/WAPE28 prediction error matrix & automatic demotion gate)
    ├── [View 13] MatchQueuePanel (Entity resolution review queue, GTIN conflicts & candidate deduplication)
    ├── [View 14] PartnerPanel (B2B trading partner API keys, AS2 endpoints, SFTP credentials, COA mapping & manual dunning trigger)
    ├── [View 15] AuditPanel (Immutable governance audit trail: ActorSubject, Action, TenantID, DetailJSON, CreatedAt)
    └── [View 16] OrderStateMachineGraph (Interactive canonical order DAG visualizer across 5 lifecycle phases)
```

---

## 3. Platform Administration API Endpoints (81 Endpoints)

The platform administration backend surface exposes 81 distinct HTTP endpoints mounted across domain routers in [`apps/backend-go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go):

### 3.1. Platform Admin Core & Tenant Lifecycle ([`platformadmin`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformadmin))
1. `POST /v1/auth/platform-admin/login`: Durable bcrypt password verification against `PlatformAdminUsers`, issuing platform-scoped session JWT.
2. `GET /v1/platform-admin/tenants`: List registered supplier/retailer tenants with optional status filter (`PENDING`, `APPROVED`, `SUSPENDED`, `OFFBOARDED`).
3. `GET /v1/platform-admin/tenants/{tenantType}/{tenantID}`: Retrieve single tenant profile with cell mapping and KYB timestamps.
4. `POST /v1/platform-admin/tenants/{tenantType}/{tenantID}/transition`: Request or finalize tenant state transition. Dual-control enforced (`ErrApproverMustDiffer`).
5. `GET /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags`: List tenant-specific feature flag overrides from `FeatureFlagOverrides`.
6. `PUT /v1/platform-admin/tenants/{tenantType}/{tenantID}/flags/{flagKey}`: Set tenant-scoped feature flag override.
7. `GET /v1/platform-admin/audit`: Query immutable governance audit log from `PlatformAdminAudit` ordered by `CreatedAt DESC`.
8. `GET /v1/platform-admin/ws-session`: Mint short-lived (`15m`) ticket JWT (`token_use=ws`) carrying `mfa_verified` for WebSocket connection.
9. `GET /v1/platform-admin/ops/outbox/summary`: Real-time gauge of unpublished events, oldest event age in seconds, and dead-letter totals.
10. `GET /v1/platform-admin/ops/outbox/events`: Sample oldest unpublished events forced via `Idx_OutboxEvents_Unpublished`.
11. `GET /v1/platform-admin/ops/outbox/dead-letters`: List failed events from `OutboxDeadLetters` with attempt count and last error.
12. `POST /v1/platform-admin/ops/outbox/dead-letters/{eventId}/replay`: Atomically delete dead letter and re-insert into `OutboxEvents` for re-dispatch.
13. `GET /v1/platform-admin/ops/disputes`: List orders in `RECONCILIATION_REQUIRED` status resulting from delivery discrepancies.
14. `POST /v1/platform-admin/ops/disputes/{orderId}/resolve`: Platform dispute resolution (`CONFIRM_DELIVERY` -> `AWAITING_PAYMENT` or `CONFIRM_CANCEL` -> `CANCELLED`).
15. `GET /v1/platform-admin/ops/runtime`: Runtime honesty endpoint reporting `run_mode`, worker cluster heartbeat (`pegasusx:runtime:worker:heartbeat`), and outbox relay status.

### 3.2. Feature Flag Dual Control ([`featureflags`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/featureflags))
16. `GET /v1/platform-admin/flags/`: List all feature flag overrides currently in `PENDING` status awaiting secondary approval.
17. `GET /v1/platform-admin/flags/{flagKey}`: Evaluate active flag state for a tenant with source attribution (`ENV`, `DEFAULT`, `OVERRIDE`).
18. `PUT /v1/platform-admin/flags/{flagKey}`: Create or update flag override. Money-affecting flags automatically enter `PENDING` state.
19. `POST /v1/platform-admin/flags/{flagKey}/approve`: Approve pending money-affecting flag override. Rejects if approver equals setter (`ErrApproverMustDiffer`).

### 3.3. Multi-Factor Authentication Gate ([`mfa`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/mfa))
20. `GET /v1/platform-admin/mfa/status`: Retrieve enrollment and verification status of current admin session.
21. `POST /v1/platform-admin/mfa/enroll`: Generate cryptographic TOTP secret and `otpauth://` provisioning URL for authenticator binding.
22. `POST /v1/platform-admin/mfa/confirm`: Confirm newly enrolled TOTP authenticator with a valid 6-digit code, upgrading session token.
23. `POST /v1/platform-admin/mfa/verify`: Challenge 6-digit code on an existing enrolled authenticator, issuing upgraded JWT with `mfa_verified: true`.

### 3.4. Global Control Tower & Playbooks ([`controltower`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltower), [`controltowerroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltowerroutes))
24. `GET /v1/control-tower/exceptions/scored`: Query open scored logistics exceptions sorted by urgency score (`severity * amount_minor / age`).
25. `GET /v1/control-tower/playbooks`: List active and inactive rule-based remediation playbooks.
26. `POST /v1/control-tower/playbooks`: Author new automated or suggested remediation playbook.
27. `PATCH /v1/control-tower/playbooks/{id}`: Update playbook match rules, priorities, or action specifications.
28. `POST /v1/control-tower/playbooks/{id}/deactivate`: Soft-deactivate an active playbook.
29. `GET /v1/control-tower/runs`: List playbook execution runs with outcome status (`SUGGESTED`, `APPROVED`, `EXECUTED`, `FAILED`, `SKIPPED`).
30. `POST /v1/control-tower/runs/{id}/{action}`: Operator action on a suggested run (`approve` to execute remediation, `skip` to dismiss).
31. `POST /v1/control-tower/evaluate`: Manually trigger evaluation engine against specified open exception IDs.
32. `POST /v1/control-tower/manifests/{id}/abort`: Emergency administrative abort of an active truck manifest, unlocking orders and staging.

### 3.5. Platform Routing, Sessions & Cell Directory ([`platformroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformroutes), [`tenantreg`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/tenantreg))
33. `GET /v1/platform/client-policy`: Return required minimum client versions per application (`min_version`, `recommended_version`).
34. `GET /v1/platform/client-config`: Return public runtime feature flags, supported locales, and CDN asset endpoints.
35. `PUT /v1/platform/client-policy`: Admin-gated update of minimum client application version thresholds.
36. `GET /v1/media/upload-ticket`: Generate pre-signed Google Cloud Storage (GCS) PUT ticket for evidence upload (claims, OS&D photos).
37. `POST /v1/user/device-token`: Register APNs / FCM push notification token mapped to authenticated user subject.
38. `POST /v1/auth/refresh`: Refresh expired session token using rotating refresh tokens with replay detection.
39. `POST /v1/auth/logout`: Revoke active JWT by recording `jti` in Redis revocation blacklist until token expiry.
40. `GET /v1/auth/session`: Inspect current claims, tenant context, active market pack, and permitted role scopes.
41. `POST /v1/platform/tenants/register`: Self-serve tenant registration minting new non-seed `SupplierID` with market code and cell association.
42. `GET /v1/platform/cells`: List public regional cell directory (`cell-uz`, `cell-eu`, `cell-us`, `cell-kz`) with API/WS hostnames and live status.
43. `GET /v1/platform/market-packs`: List catalog of supported market packs with legal currency, fiscal adapters, and PSP configurations.
44. `GET /v1/platform/market-packs/{code}`: Fetch specific market pack contract definition by ISO code.
45. `GET /v1/platform/partner-dialects`: Enumerate supported B2B partner EDI/IDoc/AS2 dialects.

### 3.6. Infrastructure Probes & Observability ([`infraroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/infraroutes))
46. `GET /healthz`: Kubernetes liveness probe (HTTP 200 OK).
47. `GET /ready`: Kubernetes readiness probe querying health checkers for Spanner, Redis, Kafka, and GCS.
48. `GET /metrics`: Prometheus metric scrape endpoint mounting default registry (SLO gauges, outbox lag, HTTP latency).
49. `GET /v1/health`: Legacy health check alias.
50. `GET /v1/health/capabilities`: Public capabilities honesty endpoint declaring active `run_mode` (`all`, `api`, `worker`) and outbox relay presence.
51. `GET /debug/infra/redis`: Platform Admin-gated inspection of Redis connection pool statistics (hits, misses, timeouts, total connections).

### 3.7. Global Products & Entity Resolution ([`globalproductsroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/globalproductsroutes), [`entityresolutionroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/entityresolutionroutes))
52. `GET /v1/admin/product-match-queue`: List pending entity resolution matches generated by fuzzy deduplication worker.
53. `POST /v1/admin/product-match-queue/{id}/resolve`: Resolve match item (`ACCEPT` links supplier SKU to GlobalProduct, `REJECT` treats as novel SKU).
54. `GET /v1/global-products/{id}`: Inspect master canonical product record keyed by normalized GTIN.
55. `GET /v1/global-products/{id}/offers`: Compare wholesale offers for a canonical GTIN across multiple competing suppliers.
56. `POST /v1/supplier/products/{productId}/link-global`: Explicit manual linkage between supplier SKU and master GTIN.
57. `POST /v1/supplier/entity-resolution/resolve`: Run deterministic and probabilistic matching pipeline against raw catalog string.
58. `POST /v1/supplier/entity-resolution/explain`: Return vector similarity and token match score explanation for candidate match.

### 3.8. B2B Partner Integration Governance ([`partner`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/partner))
59. `POST /v1/admin/partner-keys`: Issue scoped B2B API key for a specified tenant organization.
60. `GET /v1/admin/partner-keys`: List issued partner keys for a tenant (`key_prefix`, `status`, `scopes`, `created_at`).
61. `POST /v1/admin/partner-keys/{keyID}/revoke`: Revoke active partner API key with mandatory tenant-scoping verification.
62. `GET /v1/admin/partner-sftp`: Retrieve SFTP ingest directory configuration for trading partner.
63. `PUT /v1/admin/partner-sftp`: Update SFTP ingest directory credentials and host mapping.
64. `GET /v1/admin/partner-coa`: Retrieve Chart of Accounts (COA) accounting mapping rules for enterprise ERP sync.
65. `PUT /v1/admin/partner-coa`: Update Chart of Accounts (COA) general ledger account mapping.
66. `GET /v1/admin/partner-as2`: Fetch Drummond-certified AS2 EDI profile and cryptographic certificate fingerprint.
67. `PUT /v1/admin/partner-as2`: Configure AS2 station identifier, partner URL, and signing certificate.

### 3.9. Platform Billing & Platform Invoices ([`internal/services/billing`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/internal/services/billing))
68. `POST /v1/admin/billing/run-monthly`: Trigger monthly SaaS billing aggregation for a specified `YYYY-MM` cycle.
69. `GET /v1/admin/billing/invoices`: List platform commission invoices issued to supplier organizations.
70. `GET /v1/admin/billing/fee-schedules`: Inspect active basis-point fee schedules and tier thresholds.

### 3.10. AI Planning Accuracy & Safety Stock Replay ([`planning`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/planning))
71. `GET /v1/admin/planning/accuracy`: Fetch SKU-level MAPE28 / WAPE28 forecasting accuracy data and demotion status.
72. `POST /v1/admin/planning/accuracy/run-once`: Manually trigger evaluation of demand prediction errors.
73. `POST /v1/admin/planning/forecast/run-once`: Force execution of the 90-day seasonal demand forecasting engine.
74. `POST /v1/admin/planning/safety-stock/replay`: Replay safety stock fill-rate simulations across historical order waves.

### 3.11. Foreign Exchange, Accounts Receivable & Tax ([`fxrates`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/fxrates), [`creditroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/creditroutes), [`taxroutes`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/taxroutes))
75. `GET /v1/admin/fx-rates`: Query active multi-currency foreign exchange rates against baseline currency.
76. `PUT /v1/admin/fx-rates`: Publish updated spot exchange rates with effective timestamp.
77. `POST /v1/admin/ar/dunning/run-once`: Manually trigger evaluation and notification cycle of overdue credit invoices.
78. `POST /v1/admin/tax-regimes/`: Create new tax regime version (VAT rate schedules, MXIK exemptions).
79. `GET /v1/admin/tax-regimes/`: List historical and active tax regime definitions.
80. `GET /v1/admin/tax-regimes/{regimeID}`: Retrieve single tax regime specification with rate brackets.

### 3.12. Real-Time Streaming Gateway ([`ws`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/ws))
81. `GET /v1/ws` & `GET /v1/events`: Dual WebSocket / Server-Sent Events gateway providing bi-directional governance dispatch and multi-room event broadcasts.

---

## 4. WebSocket & Real-Time Event Architecture (21 Event Types)

PegasusX deploys an enterprise event-driven architecture combining Kafka outbox events, Redis Pub/Sub channels, and persistent WebSocket connections managed by [`ws.Hub`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/ws/hub.go).

### 4.1. WebSocket Rooms & Hub Topology

When a Platform Admin connects to `/v1/ws?token={ticket_jwt}`, [`ws.subscribeIdentityRooms`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/ws/handler.go#L208-L235) registers the connection to dedicated governance channels:

```
+---------------------------------------------------------------------------------------------------+
|                                   WEBSOCKET DISPATCH TOPOLOGY                                     |
+---------------------------------------------------------------------------------------------------+
|  Room Identifier                      | Target Role Audience  | Payload Scope                     |
+---------------------------------------+-----------------------+-----------------------------------+
|  "platform-admin"                     | RolePlatformAdmin     | Global Governance, Audit, Flags   |
|  "telemetry:supplier:{supplier_id}"   | SupplierAdmin, Ops    | Real-Time Fleet GPS, Heatmaps     |
|  "cmd:driver:{driver_id}"             | Driver                | Turn-by-Turn Dynamic Reroutes     |
|  "tracking:retailer:{retailer_id}"    | Retailer              | Live Stop ETA & Approaching Van   |
|  "supplier-promo:{supplier_id}"       | Retailers in Cart     | Dynamic Price Drops & Flash Sales |
|  "warehouse:{warehouse_id}"           | Warehouse Ops         | Wave Picking Progress, Staging    |
|  "factory:{factory_id}"               | Factory Ops           | Machine Telemetry, Transfers      |
|  "payload:{payload_id}"               | Dock Payload Crew     | Manifest Loading, Seal Check      |
+---------------------------------------+-----------------------+-----------------------------------+
```

### 4.2. Exact Governance & Control Tower Event Types (21 Types)

1. `PLATFORM_ADMIN_AUDIT`: Broadcast on `platform-admin` room whenever a governance action occurs. Actions include:
   - `TENANT_PENDING`: New tenant registration staged for review.
   - `TENANT_APPROVE_REQUESTED`: First administrator initiated KYB approval.
   - `TENANT_APPROVED`: Second administrator completed dual-control approval.
   - `TENANT_SUSPENDED`: Tenant trading rights frozen due to compliance/delinquency.
   - `TENANT_OFFBOARDED`: Tenant organization permanently decommissioned.
   - `FLAG_OVERRIDE_SET`: Money-affecting feature flag override submitted to `PENDING`.
   - `FLAG_OVERRIDE_APPROVE`: Secondary admin approved flag override, setting `ACTIVE`.
   - `FLAG_AUTO_ORDER_PLACE`: Specific place-flip evidence trail approved.
   - `FLAG_AUTO_ORDER_SOAK_GATE`: Break-glass soak bypass authorized.
2. `control_tower_network`: Live EKG network graph nodes (warehouses, retailers, drivers) and links with status (`active`, `idle`, `busy`, `offline`).
3. `control_tower_h3`: Spatial geospatial density array mapping order and vehicle volumes to H3 hexagonal indices.
4. `CONTROL_TOWER_PLAYBOOK_CHANGED`: Triggered when an automated remediation rule is authored, updated, or deactivated.
5. `CONTROL_TOWER_RUN_CREATED`: Staged playbook remediation proposal awaiting operator review.
6. `CONTROL_TOWER_RUN_UPDATED`: Automated remediation step executed (`APPROVED`, `FAILED`, `SKIPPED`).
7. `CLAIM_FILED`: Logistics claim registered by driver or retailer during delivery.
8. `CLAIM_UNDER_REVIEW`: Claim escalated to financial dispute team.
9. `CLAIM_RESOLVED`: Claim resolved with compensatory credit note or rejection.
10. `DELIVERY_DISPUTED`: Delivery failure marked `RECONCILIATION_REQUIRED` due to missing goods or uncollected cash.
11. `DELIVERY_SESSION_UPDATED`: Live adjustments made to stop delivery sequence.
12. `CASH_SHORTFALL`: Physical cash collection deficiency detected against invoice total.
13. `CASH_OVERAGE`: Cash collection surplus recorded during driver hand-in.
14. `AR_INVOICE_DUNNED`: Progressive dunning step executed against delinquent retailer credit invoice.
15. `FISCAL_RECEIPT_REQUESTED`: Order payment confirmed; Soliq EHF generation triggered.
16. `FISCAL_RECEIPT_SUCCEEDED`: OFD registered fiscal document; QR URL and fiscal sign stored.
17. `FISCAL_RECEIPT_FAILED`: Tax service reject; order transitioned to administrative retry queue.
18. `ORDER_STATUS_CHANGED`: Canonical state transition broadcast across all subscribed hubs.
19. `MANIFEST_ABORTED`: High-priority operational broadcast invalidating loading manifest.
20. `SUPPLIER_REGISTERED`: New trading partner onboarded into ecosystem.
21. `SYSTEM_ALERT`: Platform-wide infrastructure alert (outbox lag threshold breached, database circuit open).

---

## 5. Cloud Spanner Database Entities Touched (34 Tables)

Out of 232 total tables declared in [`apps/backend-go/schema/spanner.ddl`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schema/spanner.ddl), the Platform Admin & Control Tower role directly interacts with 34 core tables:

```
Cloud Spanner Tables Touched by Platform Administration
├── Identity, Governance & Dual-Control
│   ├── PlatformAdminUsers (Subject, PasswordHash, Email, Enabled, CreatedAt, UpdatedAt)
│   ├── PlatformAdminAudit (AuditId, ActorSubject, Action, TenantType, TenantId, DetailJson, CreatedAt)
│   ├── PlatformAdminMFA (Subject, SecretKey, Enrolled, EnrolledAt, LastVerifiedAt)
│   ├── PlatformTenants (TenantType, TenantId, Status, DisplayName, KybNotes, MarketCode, HomeCell, RequestedBy, ApprovedBy, Timestamps)
│   └── FeatureFlagOverrides (FlagKey, TenantType, TenantId, Enabled, Status, Reason, UpdatedBy, ApprovedBy, Timestamps)
├── Multi-Tenancy Core & Organization
│   ├── Suppliers (SupplierId, LegalName, Phone, Status, MarketCode, HomeCell, CreatedAt)
│   ├── Retailers (RetailerId, LegalName, Status, MarketCode, HomeCell, CreatedAt)
│   ├── ClientVersionPolicies (Platform, MinVersion, RecommendedVersion, EnforceAbove, UpdatedAt)
│   └── DeviceTokens (UserId, Platform, DeviceToken, UpdatedAt)
├── Outbox, Dead Letters & Migrations
│   ├── OutboxEvents (EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, PublishedAt, ClaimedBy, PublishAttempts, SupplierId)
│   ├── OutboxDeadLetters (EventId, AggregateType, AggregateId, TopicName, Payload, CreatedAt, DeadLetteredAt, Attempts, LastError, SupplierId)
│   └── SchemaMigrations (Version, AppliedAt)
├── Orders, Sagas & Dispute Resolution
│   ├── Orders (OrderId, SupplierId, RetailerId, DriverId, Status, TotalMinor, Currency, ParentOrderId, ShopClosedAt, Timestamps)
│   ├── ParentOrders (ParentOrderId, RetailerId, SagaState, ChildCount, ExpectedChildCount, LeaseExpiresAt, CreatedAt)
│   ├── OrderStatusTransitions (TransitionId, OrderId, FromStatus, ToStatus, TriggeredBy, Reason, CreatedAt)
│   └── OrderShopClosedLog (LogId, OrderId, AttemptNumber, ProximityMeters, PhotoEvidenceUrl, GraceExpiry, CreatedAt)
├── Global Master Products & Entity Resolution
│   ├── GlobalProducts (GlobalProductId, Gtin, CanonicalName, BrandId, UomId, CreatedAt)
│   ├── GlobalBrands (BrandId, NormalizedName, CreatedAt)
│   ├── UnitsOfMeasure (UomId, Code, Category, Factor)
│   ├── SupplierProductOffers (SupplierId, ProductId, GlobalProductId, MatchType, Status, UpdatedAt)
│   └── ProductMatchQueue (MatchId, SupplierId, ProductId, RawTitle, NormalizedTitle, MatchedGlobalId, Confidence, Status, CreatedAt)
├── B2B Partner Integrations
│   ├── PartnerApiKeys (KeyId, TenantType, TenantId, KeyPrefix, SecretHash, Status, Scopes, CreatedAt)
│   ├── PartnerSftpConfigs (TenantType, TenantId, Host, Port, Username, RemoteDir, UpdatedAt)
│   ├── PartnerAs2Configs (TenantType, TenantId, As2Id, PartnerUrl, CertificateThumbprint, UpdatedAt)
│   └── PartnerCoaMaps (TenantType, TenantId, PegasusAccount, ErpAccount, UpdatedAt)
├── Fiscalization & Tax
│   ├── OrderFiscalReceipts (ReceiptId, OrderId, Provider, EhfId, FiscalSign, QrUrl, StatusCode, CreatedAt)
│   ├── TaxRegimeVersions (RegimeId, Version, EffectiveFrom, VatRateBasisPoints, MxikExemptions, CreatedAt)
│   └── OrderLineFiscalSnapshots (SnapshotId, OrderId, LineNumber, MxikCode, PackageCode, VatRateBasisPoints, TotalMinor)
├── Financial Ledgers, Billing & Settlement
│   ├── PaymentLedgerEntries (EntryId, SessionId, OrderId, SupplierId, Gateway, EntryType, AmountMinor, Currency, ReferenceId, CreatedAt)
│   ├── ArInvoices (InvoiceId, SupplierId, RetailerId, OrderId, PrincipalMinor, BalanceMinor, TermsDays, AgingBucket, LastDunnedAt)
│   ├── ArLedgerEntries (EntryId, InvoiceId, EntryType, AmountMinor, CreatedAt)
│   ├── BillingFeeSchedules (ScheduleId, SupplierId, BasisPoints, FixedFeeMinor, MinimumMonthlyMinor, CreatedAt)
│   └── PayoutBatches (BatchId, SupplierId, Status, TotalAmountMinor, Currency, RailReference, CreatedAt)
└── Control Tower & Exception Remediation
    ├── ControlTowerPlaybooks (PlaybookId, SupplierId, Name, Priority, MatchRules, Actions, AutoExecute, IsActive, CreatedAt)
    └── ControlTowerPlaybookRuns (RunID, PlaybookID, ExceptionID, SupplierID, Mode, Status, ActionsResult, CreatedAt)
```

---

## 6. Comprehensive Feature-by-Feature Deep Dive

### 6.1. Global Control Tower: Ecosystem Real-Time Pulse & Spatial Heatmaps

The Global Control Tower operates as an intelligent nervous system orchestrating physical fleet assets, warehouse throughput, and customer store sites across regional cells:

#### 1. Live EKG Network Graph (`control_tower_network`)
The control tower maintains a real-time topology of network nodes:
- **Warehouses**: Active loading dock capacity, staged pick waves, outbox queue health.
- **Drivers**: Live transit status (`active`, `idle`, `busy`, `offline`), vehicle load percentages, route progress.
- **Retailers**: Open store hours, pending delivery windows, geofence status.
Connections (`NetworkLink`) visualize live active order flows and fulfillment commitments between nodes.

#### 2. Spatial Geospatial Density Maps (`control_tower_h3`)
Using Uber's **H3 discrete global grid system** ([`@uber/h3-go/v4`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/simulator/control_tower.go#L11-L46)), the control tower aggregates:
- Hexagonal order demand density across urban centers (e.g. Tashkent, Samarkand).
- Fleet vehicle positioning and transit density.
- Delivery latency heatmaps identifying congested delivery zones for dynamic routing intervention.

#### 3. Scored Exception Engine ([`controltower/score.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltower/score.go))
Logistics exceptions are prioritized via an algorithmic scoring formula:
$$\text{Score} = \text{SeverityWeight} \times \left( \frac{\text{AmountMinor}}{100{,}000} \right) \times \left( 1 + \frac{\text{AgeMinutes}}{60} \right)$$
Exceptions with the highest score bubble to the top of the operator console:
- `STOCK_SHORTAGE`: Picking wave inventory discrepancy.
- `VEHICLE_BREAKDOWN`: Driver vehicle disabled in transit.
- `SHOP_CLOSED_PENDING`: Retailer unavailable during handoff.
- `CASH_SHORTFALL`: Divergence in collected banknotes.

#### 4. Automated & Suggested Playbook Engine ([`controltower/engine.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/controltower/engine.go))
Operators configure declarative JSON playbooks (`ControlTowerPlaybooks`):
- **Match Rules**: Exception types, minimum monetary threshold, retailer commercial segment, exception age range.
- **Execution Mode**:
  - `SUGGESTED`: Generates a proposal requiring manual operator confirmation (`POST /v1/control-tower/runs/{id}/approve`).
  - `AUTO`: Evaluates and executes immediately via background workers.
- **Actions**:
  - `ABORT_MANIFEST`: Cancels truck manifest and rolls orders back to staging.
  - `NOTIFY_DISPATCHER`: Pushes high-priority alerts to warehouse supervisors.
  - `AUTO_APPROVE_CREDIT_NOTE`: Approves compensatory credit up to maximum policy limits.

---

### 6.2. Multi-Tenant SSMR Onboarding & Cell Federation

PegasusX architecture enforces strict multi-tenancy and cell federation under **Gate 5 / §8.10**:

```mermaid
flowchart TD
    Client["Client Request (Web / Mobile / B2B)"] --> Ingress["Ingress Gateway (TLS Termination)"]
    Ingress --> TenantMW["auth.RequireTenant Middleware"]
    
    subgraph CellRouting ["Cell Federation & Tenant Scoping"]
        TenantMW --> CheckRole{Role == PLATFORM_ADMIN?}
        CheckRole -- Yes --> BypassScope["Exempt from SupplierID Scope<br/>(Cross-Tenant Governance)"]
        CheckRole -- No --> ExtractTenant["Extract TenantContext.SupplierID<br/>from JWT Claims or API Key"]
        ExtractTenant --> CheckEnforced{TenantContextEnforced()?}
        CheckEnforced -- Missing & Enforced --> FailClosed["HTTP 401 Unauthorized<br/>TENANT_REQUIRED"]
        CheckEnforced -- Valid --> AttachContext["WithTenant(ctx, TenantContext)"]
    end
    
    BypassScope --> PlatformHandlers["Platform Admin Console Handlers"]
    AttachContext --> DomainServices["Domain Repositories (Orders, Cart, Inventory)"]
    DomainServices --> SpannerQuery["Spanner Query: WHERE SupplierId = @tenantId"]
```

#### 1. Request-Scoped Tenancy (`TenantContext`)
- Every domain database query enforces `WHERE SupplierId = @tenantId`.
- Seed-fallback injection ([`auth.PreferTenantSupplierID`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/tenant.go#L97-L113)) is strictly disabled in sandbox, staging, and production (`SeedFallbackAllowed() == false`). Requests lacking valid tenant attribution fail closed.

#### 2. Cell Directory & DNS Federation ([`auth/cell_directory.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/cell_directory.go))
The system declares a cell catalog matching Terraform deployment modules:
- `cell-uz`: Production cell serving Central Asia (`api.pegasusx.app`, `wss://api.pegasusx.app`). Shipped and live.
- `cell-eu`: European Union cell (`api-eu.pegasusx.app`). Planned PEPPOL-compliant cell.
- `cell-us`: US East cell (`api-us.pegasusx.app`). Planned commercial cell.
- `cell-kz`: Kazakhstan cell (`api-kz.pegasusx.app`). Planned regional cell.

#### 3. Market Pack Versioned Contracts ([`auth/market_pack.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/auth/market_pack.go))
Tenants are bound to localized market packs that configure regional behavior:
- **Currency & Precision**: Uzbek Sum (`UZS`, 0 decimals, integer Tiyin), Euro (`EUR`, 2 decimals), US Dollar (`USD`, 2 decimals).
- **Fiscal Adapters**: `MY_SOLIQ` (Uzbekistan State Tax Committee), `PEPPOL` (European e-Invoicing), `COMMERCIAL` (Standard B2B invoice).
- **Payment Rails**: `OCTO`, `CLICK`, `PAYME`, `CASH`, `SEPA`, `ACH`.
- **Logistics Parameters**: Geofence breach radius (e.g. 150m), shop-closed grace window (e.g. 15m), driver cash custody threshold (e.g. 24h).

---

### 6.3. Dual-Control Governance & Money-Affecting Flags

To eliminate catastrophic single-operator errors or insider threats, PegasusX enforces the **Four-Eyes Principle** on high-risk mutations:

#### 1. Tenant Approval Dual Control ([`platformadmin/service.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformadmin/service.go#L230-L299))
- **Step 1 (Request)**: First platform admin reviews KYB documents and triggers `POST /v1/platform-admin/tenants/{type}/{id}/transition` with `status: APPROVED`. The service records `RequestedBy: admin_1`, sets `TENANT_APPROVE_REQUESTED`, and leaves the tenant in `PENDING` state.
- **Step 2 (Approve)**: Second platform admin reviews the request. If `Actor == RequestedBy`, the transaction fails with `409 Conflict: ErrApproverMustDiffer`. When a different administrator executes the transition, `ApprovedBy: admin_2` and `ApprovedAt` are stamped, activating the tenant.

#### 2. Money-Affecting Feature Flags ([`featureflags/handlers.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/featureflags/handlers.go#L115-L161))
Six specific feature flags directly alter financial flows or autonomous order placement:
- `AR_INVOICES_ENABLED`: Enables automated issuance of accounts receivable invoices.
- `AR_DUNNING_ENABLED`: Enables automated dunning notifications and credit locking.
- `AUTO_ORDER_PLACE_ENABLED`: Authorizes AI worker to place live binding purchase orders without human checkout.
- `AUTO_ORDER_SHADOW_ENABLED`: Operates AI auto-ordering in shadow simulation mode.
- `AUTO_ORDER_SOAK_GATE_DISABLED`: Break-glass bypass of the mandatory 14-day zero-defect soak gate.
- `FISCAL_PROVIDER`: Selects active tax fiscalization provider (`MY_SOLIQ`, `FAKE`, `HARD_FAIL`).

When a setter alters any of these flags, the record enters `status: PENDING`. It remains inactive until a different administrator calls `POST /v1/platform-admin/flags/{flagKey}/approve`. If audit persistence fails, the activation is rolled back to `PENDING` (fail-closed compensation).

---

### 6.4. Operational Resilience, Circuit Breakers & Dead-Letter Replay

#### 1. Cloud Spanner Outbox Lag SLO Monitoring ([`platformadmin/ops.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformadmin/ops.go#L32-L80))
The transactional outbox pattern guarantees at-least-once message delivery to Kafka. The platform admin console monitors:
- **Unpublished Count**: Number of staged events awaiting Kafka relay.
- **Oldest Event Age**: Measured in seconds via index `Idx_OutboxEvents_Unpublished`.
- **SLO Alert Threshold**: If oldest event age exceeds **120 seconds**, the console trips the `lagging: true` alert, signaling worker pool starvation.

#### 2. Spanner Dead-Letter Queue & Atomic Replay ([`platformadmin/ops.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/platformadmin/ops.go#L311-L372))
When Kafka publication permanently fails after max retry attempts, events move to `OutboxDeadLetters`. The platform admin can inspect payloads and trigger atomic replay:
```go
_, err = ops.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
    outboxMutation := spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
        "EventId": eid, "AggregateType": at, "AggregateId": aid,
        "TopicName": topic, "Payload": payload, "CreatedAt": spanner.CommitTimestamp,
        "PublishedAt": nil, "PublishAttempts": int64(0), "SupplierId": supplierCol,
    })
    deleteMutation := spanner.Delete("OutboxDeadLetters", spanner.Key{eid})
    return txn.BufferWrite([]*spanner.Mutation{outboxMutation, deleteMutation})
})
```

#### 3. Outbound Circuit Breakers ([`pkg/circuit/breaker.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/pkg/circuit/breaker.go))
Outbound integrations (Soliq OFD, SMS gateways, payment providers) are protected by finite-state circuit breakers:
- **StateClosed**: Normal traffic flow; tracks upstream errors.
- **StateOpen**: Tripped after 5 consecutive failures within a 30-second window; immediately rejects outbound calls with `ErrUpstreamUnavailable` for 60 seconds without consuming socket descriptors.
- **StateHalfOpen**: Allows a single canary probe to test upstream recovery before re-closing the circuit.

---

### 6.5. Fiscal Soliq Integration Health & Clearing Status

Under **ADR-009**, Uzbekistan fiscalization is a hard legal gate on order finalization:
1. **Fiscal Hard-Gate**: Upon order handoff and payment capture, the order enters `FISCALIZING`. It cannot transition to `COMPLETED` until Soliq OFD confirms receipt.
2. **Circuit Breaker Protection**: The Soliq HTTP client wraps calls in `circuit.Breaker`. Upstream downtime trips the breaker, preventing worker thread pool exhaustion.
3. **Automated Buyer Acceptance**: In Uzbekistan wholesale, the retailer must digitally countersign the electronic invoice (E-Imzo). The background daemon `BuyerAcceptancePoller` polls the tax portal hourly to record countersignature status in `OrderFiscalReceipts`.
4. **Audit Immutability**: All Soliq requests, raw responses, and QR URLs are archived permanently in Spanner with commit timestamps.

---

### 6.6. Double-Entry Financial Ledger Audit & Payout Settlements

PegasusX eliminates rounding discrepancies by executing all financial transactions in integer minor units (Tiyin):
1. **Append-Only Ledger**: The `PaymentLedgerEntries` table records every debit and credit:
   - Unique Index: `(Gateway, EntryType, ReferenceId)` guarantees idempotency against duplicate payment webhooks.
2. **Cash Reconciliation & Shortfall Escalation**:
   - Drivers collecting banknotes record collected amounts stop-by-stop.
   - At end-of-shift, physical cash is deposited at the warehouse cash office.
   - Discrepancies generate `CASH_SHORTFALL` or `CASH_OVERAGE` events and adjust driver dispatch eligibility.
3. **Platform Fee Invoicing**:
   - The platform billing engine ([`internal/services/billing/invoice_worker.go`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/internal/services/billing/invoice_worker.go)) aggregates monthly Gross Merchandise Value (GMV) per supplier.
   - Computes platform commissions based on `BillingFeeSchedules` and issues `MasterInvoices`.

---

### 6.7. Simulation & Chaos Injection Engine

Located in [`apps/backend-go/simulator`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/simulator), the simulation engine generates realistic synthetic traffic for staging and demonstration:
- **Network EKG Generation**: Generates synthetic node and link states simulating active warehouse queues, retailer orders, and moving delivery vans.
- **H3 Geospatial Telemetry**: Simulates driver GPS breadcrumbs clustered across commercial districts in Tashkent.
- **Production Guard**: Defense-in-depth safety checks strictly prevent the simulator from running in production:
  ```go
  if auth.IsSandbox() || auth.IsProduction() || auth.IsStaging() {
      return
  }
  ```

---

### 6.8. Schema Drift Validation & Database Migration Controls

Located in [`apps/backend-go/schemadrift`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift), the schema drift engine prevents database runtime failures:
1. **Greenfield DDL Parity**: [`AssertMigrationTableParity`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift/parity.go#L79-L105) asserts that every `CREATE TABLE` statement inside `schema/migrations/*.ddl` exists verbatim within the greenfield base schema `schema/spanner.ddl`.
2. **Required Product Tables Check**: [`AssertRequiredTables`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift/required_tables.go#L12-L33) queries Spanner `INFORMATION_SCHEMA.TABLES` to ensure critical runtime tables (`RouteTwins`, `StopTwins`, `VehicleInventory`, `DriverScores`, `PlanningScenarios`, etc.) exist before the server accepts traffic.
3. **Column Drift Check**: [`AssertShopClosedSchema`](file:///Users/shakhzod/Desktop/pegasusx/apps/backend-go/schemadrift/shop_closed.go#L29-L63) validates that mandatory operational columns (`ShopClosedAt`, `ProximityUnlockedAt`, `PartialDelivery`) exist on the `Orders` table, failing closed at process startup if migrations are missing.

---

## 7. Security, Compliance & Audit Trail

1. **Zero-Trust Access**: Platform Admin accounts cannot be created via public sign-up. They are seeded into `PlatformAdminUsers` via secure deployment pipelines with bcrypt hashes (work factor 12).
2. **Session Lifetimes**: Platform admin sessions are capped at 8 hours; WebSocket connection tickets are capped at 15 minutes.
3. **Audit Immutability**: Every mutation executes within a Spanner `ReadWriteTransaction` that buffers writes to `PlatformAdminAudit` alongside the domain mutation. If the audit insert fails, the domain change is rolled back.
4. **Token Revocation**: Platform admin logouts immediately write the token's unique `jti` to the Redis revocation blacklist, terminating active WebSocket connections within 2 seconds.


---

## 8. Foundry Surface Capabilities & Pre-Execution Edge-Case Guards

1. **Propagating Ontology Security Markings (`apps/backend-go/foundry/ontology.go`)**:
   - Mandatory governance markings (`MARKING:AML_INVESTIGATION`, `MARKING:DISCREPANT_QUARANTINE`, `MARKING:THERMAL_QUARANTINE_SPOILED`, `MARKING:MICRO_SKIMMING_SUSPECT`, `MARKING:LOT_RECALL_QUARANTINE`) automatically propagate across `OntologyGraphEdges` to downstream settlement slices, cashier sessions, and audit dossiers.
2. **Kinetic Pre-Execution Rubric & Audit Ledger (`POST /v1/foundry/kinetic/actions/execute`)**:
   - Every high-risk state transition evaluates registered `PreconditionEvaluator` rubrics, enforces idempotency via `IdempotencyKey`, writes immutable before/after diffs to `KineticActionExecutions`, and buffers transactional `OutboxEvents` in Cloud Spanner.
3. **Two-Phase Soliq Fiscal Payout Gate (`POST /v1/foundry/guards/slice-payout-eligibility`)**:
   - Blocks `InvoiceSettlementSlices` from transitioning to `BATCHED` until `FiscalState == REGISTERED_SOLIQ`.
