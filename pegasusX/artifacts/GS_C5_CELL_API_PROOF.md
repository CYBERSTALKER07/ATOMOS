# GS-C5 cell API / global DNS proof

**Date:** 2026-08-16
**Method:** structural + `go test ./auth/` + portal greps. No terraform apply. No live EU/global GCP.

| Claim | Evidence | Result |
|-------|----------|--------|
| Global AR/DNS separate from cells | `infra/terraform/global/` prefix `pegasusx/global`; project `pegasusx-global`; modules `global_dns` + `global_ar` | PASS (plan-only) |
| Session `home_cell` → API URL | `GET /v1/auth/session` returns `api_url`/`ws_url`; `GET /v1/platform/cells` | PASS (unit) |
| Clients use session home_cell | `pinApiBaseUrl` in `@pegasusx/api-client`; supplier/warehouse/factory/retailer-desktop auth | PASS (web/desktop). Native BuildConfig leftover GS-R |
| Live cell stack not extracted | `modules/` is DNS+AR only | PASS |

Live leftover: `pegasusx-global` and `pegasusx-cell-eu` are not applied. DNS IPs are RFC 5737 placeholders. Native apps still use BuildConfig / PEGASUSX_API_BASE_URL as bootstrap.
