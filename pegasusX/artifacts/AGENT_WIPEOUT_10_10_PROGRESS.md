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
| **Retailer iOS file claim UI** on COMPLETED orders | Done (photo via HTTPS URL until GCS upload) |
| **Warehouse portal claims** read-only `/claims` | Done |

## Still open for true 10/10

### Vendor / Boss (WS0) — needs APIs/creds (not done here)
- [ ] DNS A + ManagedCert Active  
- [ ] Global Pay merchant password + refund action confirmation  
- [ ] Webhook SUCCESS E2E  
- [ ] `PEGASUSX_ENV=production` flip when gates pass  
- [ ] Kafka RF≥3 (multi-broker)  
- [ ] Soliq/OFD when legally required  

### Software remaining (no vendor, still product work)
- [ ] Retailer Android/desktop claims UI  
- [ ] In-app camera → GCS signed upload (not URL paste)  
- [ ] Driver camera → photo_url  
- [ ] Credit collections desk UI  
- [ ] Warehouse reverse-logistics auto-ticket from Kafka  
- [ ] Full multi-tenant IDOR battery  
- [ ] Full PX_E2E claims marker in smokecheck  

## How to use new supplier queue
1. Log into supplier portal as ADMIN with `supplier_id`  
2. Exceptions → **Claims / chargebacks**  
3. Filter OPEN → Approve + chargeback / Reject  
