# Agent wipe-out 10/10 — implementation progress

**Updated:** 2026-07-28  
**Program plan:** session plan.md (WS0–WS7)

## Shipped this session (software)

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

## Still open for true 10/10

### Vendor / Boss (WS0)
- [ ] DNS A + ManagedCert Active  
- [ ] Global Pay merchant password + refund action confirmation  
- [ ] Webhook SUCCESS E2E  
- [ ] `PEGASUSX_ENV=production` flip when gates pass  
- [ ] Kafka RF≥3 (multi-broker)  
- [ ] Soliq/OFD when legally required  

### Software remaining
- [ ] Retailer claims UI (iOS/Android/desktop) + photo upload GCS  
- [ ] Driver camera → upload photo_url  
- [ ] Credit collections desk UI  
- [ ] Warehouse reverse-logistics queue linked to claims  
- [ ] Multi-tenant IDOR battery (automated)  
- [ ] Refund remaining-after-prior-chargebacks ledger sum  
- [ ] Full PX_E2E claims marker in smokecheck  

## How to use new supplier queue
1. Log into supplier portal as ADMIN with `supplier_id`  
2. Exceptions → **Claims / chargebacks**  
3. Filter OPEN → Approve + chargeback / Reject  
