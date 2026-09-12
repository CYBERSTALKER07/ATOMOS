# Agent wipe-out 10/10 — implementation progress

**Updated:** 2026-07-28  
**Program plan:** session plan.md (WS0–WS7)

## Shipped this session (software — no external APIs)

| Item | Status |
|------|--------|
| Claims pricing math + cumulative caps | Done (backend) |
| Idempotent approve / deterministic chargeback id | Done |
| Driver missing-items dual wire | Done + live |
| Retailer list claims ownership | Done |
| **Supplier list claims API** `GET /v1/supplier/claims` | Done |
| **Supplier-scoped approve/reject** | Done |
| **CAS status transitions** OPEN→UNDER_REVIEW→RESOLVED | Done |
| **Supplier portal claims queue** `/exceptions/claims` | Done (UI) |
| Session amount cap on chargeback | Done |
| **Retailer iOS file claim UI** on COMPLETED orders | Done (+ camera/GCS upload) |
| **Retailer Android file claim UI** | Done (+ camera/GCS upload) |
| **Retailer desktop file claim UI** | Done (`FileClaimPanel` + upload-ticket) |
| **Driver iOS/Android OS&D photo** | Done (exception-report + photo_url) |
| **GCS media upload ticket** live | Done |
| **Warehouse portal claims** read-only `/claims` | Done |
| **Claims IDOR unit battery** | Done (expanded service tests) |
| **ssmr-smokecheck claims** | Done (`go run ./cmd/ssmr-smokecheck claims`) |

## Still open for true 10/10

### Vendor / Boss (WS0) — needs APIs/creds (not done here)
- [ ] DNS A + ManagedCert Active  
- [ ] Global Pay merchant password + refund action confirmation  
- [ ] Webhook SUCCESS E2E  
- [ ] `PEGASUSX_ENV=production` flip when gates pass  
- [ ] Kafka RF≥3 (multi-broker)  
- [ ] Soliq/OFD when legally required  

### Software remaining (no vendor, still product work)
- [x] Credit collections desk UI (`GET /v1/supplier/credit-profiles` + portal + retailer card)  
- [x] Warehouse reverse-logistics auto-ticket (claim → SupplierReturns; dock OPEN filter)  
- [ ] (Optional) Chargeback netting on supplier payout UI  
- [ ] (Optional) Auto-approve under threshold / store-credit  


## How to use new supplier queue
1. Log into supplier portal as ADMIN with `supplier_id`  
2. Exceptions → **Claims / chargebacks**  
3. Filter OPEN → Approve + chargeback / Reject  


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
