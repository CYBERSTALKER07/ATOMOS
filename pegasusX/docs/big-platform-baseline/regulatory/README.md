# Regulatory & Compliance Side

> **PLANNING BASELINE** — not living runtime status. Prefer [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md) and code for what is shipped.


Legal, tax, audit, and policy constraints for Uzbekistan-first and multi-regime expansion.

| Doc | Topic |
|-----|--------|
| [soliq-ehf-integration.md](./soliq-ehf-integration.md) | Full Soliq / EHF lifecycle |
| [tax-regime-versioning.md](./tax-regime-versioning.md) | Versioned tax & regime engine |
| [credit-engine-compliance.md](./credit-engine-compliance.md) | Credit limits, freezes, delinquency |
| [claim-settlement-insurance.md](./claim-settlement-insurance.md) | Claims + insurance triggers |
| [integer-money-guarantees.md](./integer-money-guarantees.md) | Zero-leak money law |
| [compliance-audit-dashboard.md](./compliance-audit-dashboard.md) | Audit export surfaces |
| [labor-hours-prep.md](./labor-hours-prep.md) | Fatigue / hours regulation prep |
| [privacy-multi-tenant.md](./privacy-multi-tenant.md) | Supplier isolation, platform view |

## Current production posture (SSMR)

- `FISCAL_PROVIDER=PEGASUS` — platform commercial receipts  
- Soliq / MY_SOLIQ **deferred** until sandbox credentials  
- Force-complete exists and must remain audited  

## Phase 1 regulatory must-haves

1. Soliq EHF + clearance tracking  
2. Tax regime versioning on COMPLETED  
3. Compliance dashboard (open fiscal, force-completes, claim mismatches)  
