# PegasusX — Enterprise global plan (backend + infra)

**Date:** 2026-08-16  
**Method:** Four parallel code audits (auth/tenancy, terraform/k8s, fiscal/PSP, per-role blockers) + live FEATURES inventory.  
**Companions:** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) (product phases) · [`GLOBAL_SCALE_BACKEND_FEATURES.md`](./GLOBAL_SCALE_BACKEND_FEATURES.md) (every backend feature tagged) · [Part I modules × exits](./GLOBAL_SCALE_BACKEND_FEATURES.md#part-i--phased-modular-plan) · [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) · [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md) · [`MULTI_TENANCY_GATE5_PHASE1.md`](./MULTI_TENANCY_GATE5_PHASE1.md)

**Honesty:** This is **not** a go-live certificate and **not** an apply order. T1 mints a non-seed UUID + shipped pack. T2 freezes old seed register. T3: missing KYB row ≠ active; APPROVE is dual-control pack+cell (`system:` seed one-step). T4: retailer register requires `supplier_id` or invite; seed attach forbidden outside ssmr; demo `"1234"` only in ssmr. T5: factory/warehouse register is invite or ADMIN+node; bcrypt only; driver/payload login from table rows, not `+998` defaults. **M1:** checkout reads shipped-pack currency + PSP. **M2:** fiscal fail-closes planned/PEPPOL/FAKE-in-prod; receipts stamp pack adapter. **M3:** one pack `breach_radius_meters` (UZ 150; 500 deleted). **M4:** shop-closed / labor / weather / factory SLA / calendar TZ from shipped pack. **M5:** empty order/payment currency from shipped pack (NewService / refund no longer invent `UZS`). **M6:** pack `payout_rail` (UZ `bank-file`; unknown+live → `no_live_rail`; no invented PSP). **M7:** tax stamp from pack country (`countryFromCurrency` deleted). **C1–C5:** same TF root is cell-safe; project factory declares `pegasusx-cell-eu` / `europe-west1` (plan only). Empty adapters OK (commercial fiscal, new JWT). Same DDL, no UZ restore. C4 isolation proof written. C5: global DNS/AR (`infra/terraform/global/`, prefix `pegasusx/global`) + session `api_url` from `home_cell`. **GS-I:** per-supplier OIDC (`orgoidc`); process-global Auth0 wrap removed. `checkout_reads_this` stays false. Live infra is still **one project / one region** (`pegasus-503013` / `asia-south1`). EU and global projects are not applied.

**North star:** **Global multi-supplier** — many companies register (minted `SupplierId` + shipped `market_code` + `home_cell`) → each runs Class A in that cell → retailers attach many suppliers → money/law from the pack. Same code, **cloned cells**, not a fork. Isolation key stays `SupplierId`.

**Do not:** second tenant key; merge factory/payload tables; share Spanner/Kafka/JWT across cells; default `FISCAL_PROVIDER=MY_SOLIQ` on a non-UZ cell; flip factory planning / auto-order place globally; treat country picker or `planned` packs as live markets.

---

## 0. Verdict from the four audits

| Audit | Finding |
|-------|---------|
| **Auth / tenancy** | Gate 5 `SupplierId` isolation is real. JWT **stamps** `market_code`/`home_cell` on every `Issue` (A1). A2 persists nullable `Suppliers.MarketCode`/`HomeCell`; session `source: claim\|profile\|env`. T1–T5 done. **GS-I:** OIDC per supplier org; AUTH0_DOMAIN wrap not mounted. **M1–M7:** checkout + fiscal + radius + TZ + currency + payout_rail + tax country read shipped pack; flag still false. |
| **Infra** | **C1–C5 done** (plan only): isolation proof + global DNS/AR plan + session `api_url`. `cells/eu/project/` declares `pegasusx-cell-eu`. Live objects still one project / `pegasusx/ssmr`. |
| **Fiscal / PSP** | **M2:** fiscalize/retry read shipped `fiscal_adapter` (planned/PEPPOL fail-closed). Cell `FISCAL_PROVIDER` remains SSMR default (PEGASUS/FAKE). Prod: MY_SOLIQ needs EDS; FAKE forbidden; planned `DEFAULT_MARKET_CODE` fails boot. Buyer poller only if UZ MY_SOLIQ. Flag still false. Stripe/Adyen checkout-init is `catalogHonestyExecutor` (`adapter_planned` → not a redirect). Live Stripe/Adyen charge is DEFER until a sold legal entity. |
| **Roles** | Class A loops (dispatch, WMS, doorstep, seal) **KEEP**. T5 closed +998 login defaults. M1–M7 closed pack readers. **GS-P** dialect gates shipped. Remaining blockers: flag, one cell, extra PSPs / live PEPPOL. |

---

## 1. Architecture (locked)

```
                 ┌─ control (optional later): DNS, image mirror, signing keys
                 │
Company register ┴─► Tenant row: SupplierId + market_code + home_cell
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
         cell-uz          cell-eu          cell-us
         asia-south1      europe-west1     us-central1
         own project      own project      own project
         Spanner+Kafka    Spanner+Kafka    Spanner+Kafka
         Redis+GKE        Redis+GKE        Redis+GKE
         JWT secret       JWT secret       JWT secret
         pack UZ          pack EU/planned  pack US/planned
```

- **Tenant key stays `SupplierId`.** Pack and cell are **attributes**, not a second RLS key.
- **One cell = one GCP project + region + full stack.** Same Terraform root, **new state prefix**.
- **Cross-cell = partner API / EDI**, never a shared write DB.
- **HS256 stays** until cell-local secrets exist; do not share the JWT secret.

---

## 2. Backend program (modules)

### GS-A — Auth session (A0–A2 shipped)

| ID | Module | Work | Exit |
|----|--------|------|------|
| **A0** | Catalog + session | Done: `auth/market_pack.go`, `GET /v1/auth/session`, `GET /v1/platform/market-packs` | UZ shipped; EU/US/KZ planned; checkout_reads_this false |
| **A1** | Stamp pack on Issue | **Done** — `StampMarketClaims` inside every `auth.Issue` (claim → profile lookup → env). Refresh copies claims | JWT round-trip from live login; `go test ./auth/` |
| **A2** | Persist on `Suppliers` | **Done** — nullable `MarketCode` / `HomeCell`. `WireMarketProfileLookup`. Session `source: claim\|profile\|env` | Empty row ≠ silent UZ-as-chosen |
| **A3** | Align catalogs | countrycfg is AUTH_COUNTRIES in-memory (UZ,KZ,KG,TJ,TM,AE,TR,RU,US,GB) — **not** a `CountryConfigs` table. MarketPack is product law; countrycfg ops-only until GS-M. Unknown pack 404. Do not add a CountryConfigs Spanner table | One law: MarketPack |

### GS-T — Self-serve tenant register

| ID | Module | Work | Exit |
|----|--------|------|------|
| **T1** | `tenantreg` package | **Done** — `POST /v1/platform/tenants/register` — legal name, phone, password, **shipped** `market_code`. Mint **new** UUID. Never seed-overwrite register | New supplier; 404 planned pack |
| **T2** | Freeze old register | **Done** — `/v1/auth/supplier/register` 409 + Location T1 once seed registered. ALLOW_MULTI ignored | No overwrite of registered seed |
| **T3** | KYB | **Done** — `EnsurePending`; missing row ≠ active; dual-control APPROVE pack+cell. `system:` seed one-step | Two actors + shipped pack |
| **T4** | Retailer attach | **Done** — `supplier_id` or HMAC invite required. Seed attach forbidden outside ssmr. Demo `"1234"` only in ssmr | No shop on seed |
| **T5** | Staff invite only | **Done** — `POST /v1/supplier/staff-invites`. Public factory/warehouse register needs invite or ADMIN+node. bcrypt only. Driver/payload login from table rows; `+998` not a default | Workers are rows, not env phones |

**Auth security must-fix inside T (same PR family):**

- Retailer demo password `"1234"` — **T4 done:** master key only when `PEGASUSX_ENV=ssmr`.
- Warehouse secret: **T5 done** — bcrypt only (plaintext compare removed).
- Partner `withPartnerTenant`: do not stuff retailer `TenantID` into `SupplierId`.
- Production `RequireTenant` — **LB-1 done:** cfg default is `auth.TenantContextEnforced()` (sandbox|production), not `IsSandbox()` only.

### GS-M — Pack is law (checkout / fiscal / proximity)

| ID | Reader | Must consume | Fail-closed |
|----|--------|--------------|-------------|
| **M1** | Checkout | **Done** — reads `status=shipped`, `currency_code`, `psp_adapters`. Flag stays false | Planned pack; unknown gateway; currency mismatch |
| **M2** | Fiscal | **Done** — reads shipped `fiscal_adapter`. Env is cell default only (SSMR PEGASUS/FAKE). Flag stays false | MY_SOLIQ without EDS; PEPPOL unimplemented; FAKE in prod; planned pack |
| **M3** | Proximity | **Done** — consume pack `breach_radius_meters` (UZ 150). Dual 150 vs 500 deleted. Settlement stays 100 m. Flag stays false | Planned pack / radius ≤ 0 |
| **M4** | Shop-closed / labor / credit TZ | **Done** — pack timezone + grace + weather scope + factory SLA hours + labor max shift. Flag stays false | Planned pack; empty TZ / grace / weather scope |
| **M5** | Order/payment defaults | **Done** — empty currency from shipped pack in `order.NewService` / picker / refunds / `payment.NewService` | Empty currency → pack, not UZS |
| **M6** | Payout | **Done** — pack `payout_rail` (UZ `bank-file`). Unknown name does not fall through to bank-file. Unknown + `live` → `no_live_rail` | Planned pack; unknown rail + live |
| **M7** | Tax regime stamp | **Done** — pack country, not `countryFromCurrency` KZT→KZ else UZ | Missing regime fails txn; planned pack 404 |

**Do not flip `checkout_reads_this` until M1–M2 agree with UZ pack** (catalog says MY_SOLIQ; SSMR runtime is often PEGASUS).

**Adapters (LOCAL, not forks):**

| Pack | Fiscal | PSP | SMS | Payout |
|------|--------|-----|-----|--------|
| UZ shipped | MY_SOLIQ + EDS | GLOBAL_PAY + CASH | PlayMobile | Bank-file |
| EU planned | PEPPOL or COMMERCIAL | Stripe (real executor later; today `catalogHonestyExecutor`) | Twilio | SEPA file |
| US planned | COMMERCIAL | Stripe | Twilio | ACH file |
| KZ planned | TBD | CASH first | local | Bank-file |

One new country = pack + 1–3 adapters.

### GS-I — Enterprise identity (done)

- **Done.** OIDC **per supplier org** (`SupplierOIDC` + `/v1/auth/oidc/{discovery,exchange}` + `/v1/supplier/oidc`). Process-global `AUTH0_DOMAIN` wrap is not mounted (it would 401 native HS256).
- Staff/driver stay JWT; buyer admin may SSO. Portal: Login with IdP + Integrations attach card.
- No `client_secret` in Spanner. SAML/SCIM later.

### Per-role backend change list (first PR only)

| Role | First backend change | Tag |
|------|----------------------|-----|
| All | A1 stamp JWT pack (**done**) | ADAPT |
| Supplier | T1–T5 (**done**); GS-I OIDC attach (**done**) | KEEP |
| Retailer | T4 attach partner **done**; M1 currency/PSP | ADAPT + LOCAL |
| Warehouse | T5 staff bcrypt **done**; M4 labor TZ | ADAPT |
| Factory | T5 invite-only **done**; pack SLA hours | ADAPT |
| Payload | T5 staff-row login **done** | KEEP |
| Driver | T5 `Drivers` table login **done**; Tashkent coords dropped from login | KEEP |
| Platform admin | T3 pack+cell dual-control; never default Soliq on non-UZ cell | ADAPT |

**KEEP without geo work:** WMS, freeze, seal/inject/seal-all, doorstep state machine, H3 dispatch + factory solver adapter (`FactoryTruckManifests` only), MEIO heuristic, dual manifest planes, CRM/entity-resolution, loyalty earn/tier (burn out of scope), 410 theatre (cards/negotiate).

**DEFER:** factory planning on, auto-order place, saved cards, negotiations, marketplace.

---

## 3. Infra program (GS-C)

### Reality

| Fact | Evidence |
|------|----------|
| One project `pegasus-503013`, region `asia-south1` | `infra/terraform/staging.tfvars` |
| Live state prefix `pegasusx/ssmr` | `backend-ssmr.hcl` only — **not** hardcoded in `backend.gcs.tf` (C1) |
| GSM `auto {}` still live default | `gsm_regional_only=false` on ssmr/staging so apply does not ForceNew secrets. EU must set true (check) |
| Spanner/GSM IAM instance-scoped when `cell_scoped_iam` | Redis leftover: Memorystore BASIC has no instance IAM |
| Env overlays remain | `overlays/{prod,ssmr,staging}` plus named `overlays/cells/uz` (C1) |
| `HOME_CELL=cell-uz` | ConfigMap + `overlays/cells/uz` |
| `cell_id` / `api_hostname` / `k8s_namespace` | `infra/terraform/cell.tf` |

`tenant_slug` **namespaces names**. It is **not** a cell.

### Must be cell-local (never share)

Spanner, Kafka+topics+DLQ, Redis, GKE+WI, VPC/NAT, **JWT secret**, PSP/fiscal/SMS/Maps keys, Ingress+cert+`PUBLIC_BASE_URL`, OSRM extract, outbox, GSM **regional** replication, budget/alerts.

**May share later (global project):** container images, DNS zone (records per cell), desktop signing keys, billing account.

### Phases (design → plan; **no apply** from this catalog)

| ID | Work | Exit |
|----|------|------|
| **C0** | Cell = project+region+stack. Inventory live `pegasus-503013` objects. Fix README link to missing Phase 0 runbook. | Written contract |
| **C1** | Cell-safe **same** TF root: per-cell `backend.hcl`; vars `cell_id`, `api_hostname`, `k8s_namespace`; Kafka topics from `cell_id`; GSM user-managed in `var.region` only; WI from namespace; instance-level IAM; custom-mode VPC one subnet; `overlays/cells/uz` | **Done** (plan only) — `europe-west1` cannot use `pegasusx/ssmr`. `make cell-backend-guard` |
| **C2** | `infra/terraform/cells/{uz,eu}/` tfvars + backend.hcl; `make cell-plan CELL=eu` | **Done** (plan only). Isolated data dir; EU prefix ≠ `pegasusx/ssmr`; EU project ≠ `pegasus-503013` |
| **C3** | New project `pegasusx-cell-eu`, `europe-west1`, own Spanner (same DDL), Redis, Kafka, GKE, GSM-EU, `api-eu.pegasusx.app`, **FISCAL commercial/none**, **new JWT**, no UZ backup restore | **Done** (plan only) — empty adapters OK. Project not applied |
| **C4** | Isolation proof: EU GSA denied UZ Spanner/GSM; UZ JWT 401 on EU API; Kafka cross-bootstrap fails; GSM locations EU-only | **Done** (written) — `make cell-isolation-proof`. Live gcloud deny waits for apply |
| **C5** | Optional modules/ + global AR/DNS; clients use session `home_cell` for API URL | **Done** (plan only) — `make global-plan` + `make cell-api-proof`. Native pin closed in GS-R |

**Reject:** two cells in `pegasus-503013`; Terraform workspaces; one global Spanner; `auto {}` GSM for EU.

### K8s overlay shape

```
infra/k8s/overlays/cells/uz/   # HOME_CELL=cell-uz, DEFAULT_MARKET_CODE=UZ, FISCAL as today
infra/k8s/overlays/cells/eu/   # HOME_CELL=cell-eu, pack EU/planned, FISCAL_ALLOW_COMMERCIAL, no Soliq secrets
```

Copy staging merge pattern. Do not fork `base/`. Prod must stop pointing ExternalSecrets at `pegasusx-ssmr-*` once uz cell is named.

---

## 4. Math / logic / edge cases (do not break)

| Invariant | Global rule |
|-----------|-------------|
| Money integer minor | Pack `decimal_places`; **never** float |
| Credit leave + AR same txn | Unchanged |
| Fiscal missing keys | hard-fail / no COMPLETED OFD |
| FAKE fiscal | forbidden in production |
| Seed | fail-closed ssmr/prod |
| Dual manifests | FACTORY vs SUPPLIER tables |
| Geofence | **Done (M3)** — one pack `breach_radius_meters` (UZ 150; 500 deleted). Settlement stays 100 m |
| Optimizer | HEURISTIC vs OPTIMAL honesty; sidecar URL per cell |
| Planning / place | flag-off per tenant |

---

## 5. Non-technical (regulation + local services)

| Concern | UZ cell | Other cell |
|---------|---------|------------|
| Tax | MY_SOLIQ + EDS (owner keys) | PEPPOL or commercial + `FISCAL_ALLOW_COMMERCIAL_RECEIPTS` |
| PSP | Global Pay | Real Stripe/Adyen **or** cash+credit only — not static redirect |
| KYC | STIR/INN in setup | Pack required tax-id type |
| Privacy | — | GDPR = **home cell project**; DPA before EU self-serve |
| Comms | PlayMobile | Twilio / email |
| Labor | Tashkent calendar | Pack TZ |
| Contracts | — | DPA + SLA + status page before EU/US register |

---

## 6. Platform parity (GS-R bind done)

Every pack-visible field (currency, TZ, fiscal label, maps) on **all clients that role uses**. **Bind done:** session splash + native pin. No web-only currency helper. Deep POS UZS leftover continuous.

| Role | Clients |
|------|---------|
| Supplier | portal + Android + iOS |
| Retailer | desktop + Android + iOS |
| Warehouse / factory | portal + Android + iOS |
| Payload | terminal + Android + iOS |
| Driver | Android + iOS |
| PA | web only |

First UI **shipped:** session `pack` on login splash (currency + “receipts: Soliq / commercial”). Deep POS/fiscal screens follow M2. Native pin closed.

---

## 7. Sequence (backend ∥ infra)

```
A1 stamp JWT ──► A2 persist pack ──► T1 register ──► T3–T5 identity
                         │
                         ├── M1–M7 pack is law (do not claim second country before)
                         │
C0–C1 cell-safe TF ──► C2 repo layout ──► C3 EU plan ──► C4 isolation ──► C5 DNS / home_cell URL
                         │
                         └── I OIDC after T
```

**Legal single-cell (UZ) launch** still follows [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) R1–R2 (Soliq/GP keys). Global **register** is T+A, not more features.

**Second-country claim** requires M + C4. Not a country picker.

---

## 8. Explicit non-goals

- World marketplace  
- Multi-region Spanner writes  
- Saved cards / negotiations / loyalty **burn** (earn/tier are live KEEP)  
- Factory planning default on  
- Process-global Auth0 wrapping all JWT  
- Applying EU Terraform before C1 state split  
- Adding a `CountryConfigs` Spanner table (AUTH_COUNTRIES stays in-memory until MarketPack is law)  

---

## 9. Next implementation slice (approved order)

1. **A1** — **Done.** `StampMarketClaims` on every `auth.Issue`  
2. **A2** — **Done.** `Suppliers.MarketCode` / `HomeCell` + `WireMarketProfileLookup`  
3. **T1** — **Done.** `POST /v1/platform/tenants/register`  
4. **T2** — **Done.** Freeze `/v1/auth/supplier/register`  
5. **T3** — **Done.** KYB dual-control; missing row ≠ active  
6. **T4** — **Done.** Retailer attach (invite / trading-partner id)  
7. **T5** — **Done.** Invite-only staff; bcrypt; kill +998 demo defaults  
8. **C1** — **Done** (plan only). Per-cell `backend-*.hcl` + `cell_id` vars. No apply.  
9. **C2** — **Done** (plan only). `make cell-plan CELL=eu`. No apply.  
10. **C3** — **Done** (plan only). Project factory + empty adapters. No apply.  
11. **C4** — **Done** (written). `make cell-isolation-proof`. No apply.
12. **C5** — **Done** (plan only). `make global-plan` + `make cell-api-proof`. No apply.
13. **GS-I** — **Done.** Per-supplier OIDC; AUTH0 wrap removed. SAML/SCIM later.  
14. **GS-R** — **Done (bind).** `make pack-client-proof`. Native pin + session splash. Deep UZS leftover continuous.  
15. **GS-P** — **Done.** `GET /v1/platform/partner-dialects` + pack gates. `make partner-dialect-proof`. Never blocked register. No next phase from this catalog.  

Agents used: auth/tenancy explore, terraform/k8s explore, fiscal/PSP explore, per-role blockers explore.
