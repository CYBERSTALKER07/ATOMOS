# GS-P partner dialect proof

**Date:** 2026-08-16
**Method:** structural greps + `go test ./partner/ ./partner/adapters/onec/ ./tenantreg/ ./platformroutes/`. No checkout_reads_this flip. No terraform apply. No live PEPPOL/VAN/SAP.

| Claim | Evidence | Result |
|-------|----------|--------|
| Dialect catalog | `GET /v1/platform/partner-dialects` + `AllowPartnerDialect` | PASS |
| 1C CIS only | UZ/KZ live; EU 422 `dialect_not_for_pack` | PASS |
| PEPPOL planned | EU PUT 422 `dialect_not_live` | PASS |
| X12/SAP sold_only | US 422 | PASS |
| Empty 1C currency | parser no longer invents UZS; fill from pack | PASS |
| Register unblocked | tenantreg has no dialect import | PASS |
| Flag | `checkout_reads_this` still false | PASS |

Leftover (not a second-country claim): flip `checkout_reads_this`; terraform/kubectl apply; live PEPPOL AP; live Stripe/Adyen executor; SAML/SCIM; EDI codec empty-currency defaults; deep UZS screens.
