---
name: regulatory-gov
description: Government and trade-compliance APIs — Soliq OFD/EHF, GS1, AS2/EDI, settlement rails.
---

# Regulatory & gov / trade APIs

Deep Agents **must track** whether legal and partner-compliance rails are wired,
proven, or still Class D (flag/cert blocked). Do not invent “connected to gov”
without code + env proof.

## Government (Uzbekistan fiscal)

| API | Purpose | Required behavior | Status track |
|-----|---------|-------------------|--------------|
| **MySoliq / Soliq OFD** (`FISCAL_PROVIDER=MY_SOLIQ`) | Legal tax receipt submit+poll | Fail closed on misconfig; OFD timeout; FISCAL_FAILED retry | Adapter real; **EDS/E-IMZO still P1-7** (dev-HMAC not legal) |
| **Buyer EHF clearance** | Buyer accept/reject (~10d) | PENDING on MySoliq SUCCESS; REJECT→credit note default ON | P1-6 ✅ — re-verify poller |
| PEGASUS / FAKE receipts | Commercial / non-legal | Must **not** stamp buyer acceptance | Code: stamp only MY_SOLIQ |

**Ops gate for legal sales:** procure EDS → pkcs12/E-IMZO sign → sandbox SUCCESS → prod `MY_SOLIQ`.

Evidence: `order/fiscal.go`, `fiscal/signer_env.go`, `order/buyer_acceptance_poller.go`,
`cmd/ssmr-smokecheck/e2e_soliq.go`

## Trade compliance / partner (not tax authority, still regulatory-adjacent)

| Integration | Purpose | Track |
|-------------|---------|-------|
| GS1 GLN / SSCC / ZPL | Legal marking / labels | Wired; DataMatrix **non-conformant** (P1-13) |
| AS2 + EDI-lite | ORDERS/ORDRSP/DESADV/INVOIC | Wired; outbound MDN MIC unverified (P1-14); not Drummond |
| 1C journals export | Local ERP accounting | Wired — verify CoA config |
| Partner webhooks HMAC | Outbound enterprise notify | Wired — allowlist/SSRF still P2 |

Evidence: `gs1/`, `partner/as2/`, `docs/GS1_LABELS.md`, `docs/PARTNER_AS2.md`, partner OpenAPI

## Settlement (money movement, not gov)

| Rail | Track |
|------|-------|
| PSP (Global Pay, Payme, Click, …) | Webhooks + reconciler (P1-8); live merchant keys ops |
| Supplier payout | File rail today; live bank rail missing; live=true fail-closed (P0-2) |
| AR dunning | Flag `AR_DUNNING_ENABLED` |

## Auditor rules

1. Separate findings: **gov fiscal** vs **trade GS1/AS2** vs **PSP/payout**.
2. Never reopen P1-6 as “buyer acceptance missing” without code regression.
3. P1-7 stays open until live sign→submit→poll SUCCESS evidence exists.
4. Do not recommend inventing extra gov APIs beyond Soliq for core order→cash
   unless a SKU-specific marking/traceability mandate appears in product docs.


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
