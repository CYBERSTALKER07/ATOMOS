# Phase 1 (3–6 months) — Regulatory + Differentiation Must-Haves

## Scope

1. **Full Soliq EHF + versioned tax engine** (`regulatory/soliq-ehf-integration.md`, `tax-regime-versioning.md`)  
2. **Enhanced shop-closed + partial offload + proximity settlement** (`last-mile/`)  
3. **Causal demand sensing (basic external signals)** (`planning/2.1`)  
4. **Labor capacity model + driver score** (`foundations/1.4`, `execution/3.3`)  
5. **Predictive ETAs** (`execution/3.2`)  
6. **Compliance / fiscal audit dashboard** (`regulatory/compliance-audit-dashboard.md`)  

## Dependencies / unblock first

| Item | Why |
|------|-----|
| Fiscal worker reliability | Live smoke saw FISCALIZING stick; fix before Soliq |
| Real GCS signBlob | Claims photos production-grade |
| DNS/TLS | Public apps + Soliq callbacks |
| GP password | Card settlement paths |

## Phase 1 DoD

- [ ] EHF path green in sandbox (or PEGASUS+regime snapshot if Soliq delayed)  
- [x] Shop-closed economic options + timeout escalate *(backend matrix + retailer codes + supplier queue; live DDL apply pending)*  
- [x] Partial offload API + offline merge tests *(API + qty unit tests + return path; full offline queue UI follow-up)*  
- [x] Proximity unlock server check *(handler + cash/credit gates + driver unlock call)*  
- [ ] Demand signals table + one UI surface  
- [ ] Driver score computed nightly  
- [ ] ETA endpoint with confidence band  
- [ ] Compliance export for open fiscal + force-completes  
- [ ] All money int64; role-row or deferred  

## Out of scope for Phase 1

Full MEIO solver, robotics, multi-modal tendering, carbon product UI, full playbook product.
