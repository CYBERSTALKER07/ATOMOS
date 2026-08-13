# Residual register — deploy-time only (not open code gaps)

**As of:** 2026-08-13 G7 close. These items are **not** reopened in `GAP_LEDGER`.

| Residual | Why not a code gap | Owner |
|----------|--------------------|--------|
| Live Soliq/EDS secrets + OFD cutover | Fiscal default + fail-closed boot already in G1; product-deferred Soliq UI | Ops / legal |
| OR-Tools optimizer-core prod replicas | Code honesty HEURISTIC vs OPTIMAL + OSRM flag; pods are deploy | Ops |
| Auto-order **place** soak flip | Dual-control + soak flags exist (G2.E); no invented flip | Product + ops |
| OIDC / external IdP | PreferTenant fail-closed; OIDC is optional adapter | Security |
| Drummond AS2 / SAP IDoc vendor cert | 1C + EDI-lite + profile packs wired (G5) | Partner |
| FCM / APNs / SMS provider credentials | Dispatcher + device-token paths exist; silent without keys | Ops |
| Human Substance Gate UI walk | Code READY_FOR_WALK; human PASS/FAIL | QA |
| Draft i18n linguistic review | Keys generated; not product-linguistic Done | Localization |

Code-complete program: G1–G7. Scorecard 10s cite this register for remaining 0.5 footnotes.
