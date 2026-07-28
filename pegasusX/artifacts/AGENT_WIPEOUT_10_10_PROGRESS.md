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
- [ ] Credit collections desk UI  
- [x] Warehouse reverse-logistics auto-ticket (claim → SupplierReturns; dock OPEN filter)  
- [ ] (Optional) Chargeback netting on supplier payout UI  
- [ ] (Optional) Auto-approve under threshold / store-credit  


## How to use new supplier queue
1. Log into supplier portal as ADMIN with `supplier_id`  
2. Exceptions → **Claims / chargebacks**  
3. Filter OPEN → Approve + chargeback / Reject  
