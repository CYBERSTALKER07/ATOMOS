# PegasusX — Enterprise global plan (backend + infra)

**Date:** 2026-08-14  
**Method:** Four parallel code audits (auth/tenancy, terraform/k8s, fiscal/PSP, per-role blockers) + live FEATURES inventory.  
**Companions:** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) (product phases) · [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) · [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md) · [`MULTI_TENANCY_GATE5_PHASE1.md`](./MULTI_TENANCY_GATE5_PHASE1.md)

**Honesty:** This is **not** a go-live certificate and **not** an apply order. Companies cannot self-register into a home cell today. GS-A **advertises** a MarketPack; checkout/fiscal/proximity **do not read it**. Infra is **one project / one region** (`pegasus-503013` / `asia-south1`).

**North star:** A company registers → minted non-seed `SupplierId` + shipped `market_code` + `home_cell` → Class A loop in that cell → money/law adapters from the pack. Same code, **cloned cells**, not a fork.

**Do not:** second tenant key; merge factory/payload tables; share Spanner/Kafka/JWT across cells; default `FISCAL_PROVIDER=MY_SOLIQ` on a non-UZ cell; flip factory planning / auto-order place globally; treat country picker or `planned` packs as live markets.

---

## 0. Verdict from the four audits

| Audit | Finding |
|-------|---------|
| **Auth / tenancy** | Gate 5 `SupplierId` isolation is real. JWT **can** carry `market_code`/`home_cell`; **no login issuer stamps them**. Register overwrites **seed** or 409 `supplier_cap_reached`. Public factory/warehouse register + driver/payload **+998 demo** leak seed. `POST /v1/platform/tenants/register` **does not exist**. |
| **Infra** | Flat TF root, one GCS state prefix `pegasusx/ssmr`, GSM `replication { auto {} }`, project-wide IAM. Overlays = **env** (prod/ssmr/staging), not **cells**. `HOME_CELL=cell-uz` is ConfigMap only. |
| **Fiscal / PSP** | Fiscal = **process env**. Unset prod/staging → MY_SOLIQ. Checkout = supplier gateway policy, default **GLOBAL_PAY**, currency default **UZS**. `countrycfg` and MarketPack both `checkout_reads_this: false`. Stripe/Adyen executors are **theatre redirects**. |
| **Roles** | Class A loops (dispatch, WMS, doorstep, seal) **KEEP**. Blockers are UZS, Asia/Tashkent, +998 demo logins, 150 vs 500 m geofence, Soliq default, one cell. |

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

### GS-A — Auth session (partially shipped)

| ID | Module | Work | Exit |
|----|--------|------|------|
| **A0** | Catalog + session | Done: `auth/market_pack.go`, `GET /v1/auth/session`, `GET /v1/platform/market-packs` | UZ shipped; EU/US/KZ planned; checkout_reads_this false |
| **A1** | Stamp pack on Issue | Helper used by **every** `auth.Issue` (supplier/retailer/factory/warehouse/driver/payload/refresh). Today fields exist, issuers ignore them | JWT round-trip from live login, not only unit test |
| **A2** | Persist on `Suppliers` | Columns `MarketCode`, `HomeCell`. Login reads row; empty → env default. Session `source: claim\|profile\|env` | No silent UZ-as-chosen |
| **A3** | Align catalogs | FEATURES still says countrycfg UZ-only 404; live catalog is UZ+KZ+… stubs. Mark stubs **planned** or 404 unknown | One law: MarketPack |

### GS-T — Self-serve tenant register

| ID | Module | Work | Exit |
|----|--------|------|------|
| **T1** | `tenantreg` package | `POST /v1/platform/tenants/register` — legal name, phone, password, **shipped** `market_code`. Mint **new** UUID. **Never** `resolveRegistrationSupplierID` / seed | New supplier; 404 planned pack |
| **T2** | Freeze old register | `/v1/auth/supplier/register` rejects or redirects once seed registered | No overwrite of `sup_61d8…` |
| **T3** | KYB | `platformadmin.EnsurePending`; **missing row ≠ active** (today missing = active) | Dual-control APPROVE |
| **T4** | Retailer attach | Public retailer register requires invite / trading partner id — **not** PreferTenant(seed) | No shop on seed |
| **T5** | Staff invite only | Close public factory/warehouse register **or** require supplier ADMIN + node ownership. Hash factory register passwords (today plaintext). Kill driver/payload +998 demo as default login | Workers are rows, not env phones |

**Auth security must-fix inside T (same PR family):**

- Retailer demo password `"1234"` is a **master key** if phone exists — disable outside seed/ssmr.
- Warehouse secret: bcrypt **or plaintext** — bcrypt only.
- Partner `withPartnerTenant`: do not stuff retailer `TenantID` into `SupplierId`.
- Production `RequireTenant` must use the **same** rule as `auth.TenantContextEnforced()` (cfg default is ssmr-only today).

### GS-M — Pack is law (checkout / fiscal / proximity)

| ID | Reader | Must consume | Fail-closed |
|----|--------|--------------|-------------|
| **M1** | Checkout | `status=shipped`, `currency_code`, `psp_adapters`, `checkout_reads_this=true` | Planned pack; unknown gateway |
| **M2** | Fiscal | `fiscal_adapter` **overrides** process `FISCAL_PROVIDER` (env = cell default only) | MY_SOLIQ without EDS; PEPPOL unimplemented; FAKE in prod |
| **M3** | Proximity | `breach_radius_meters` — **delete dual 150 vs 500** | — |
| **M4** | Shop-closed / labor / credit TZ | pack timezone + grace; stop `UZDefault()` and `FixedZone("Asia/Tashkent")` | — |
| **M5** | Order/payment defaults | Stop `Currency="UZS"` in `order.NewService` / refund empty | Empty currency → pack, not UZS |
| **M6** | Payout | Add `payout_rail` on pack; unknown + `live` → `no_live_rail` | — |
| **M7** | Tax regime stamp | Stop `countryFromCurrency` KZT→KZ else UZ | Missing regime fails txn (already) using pack country |

**Do not flip `checkout_reads_this` until M1–M2 agree with UZ pack** (catalog says MY_SOLIQ; SSMR runtime is often PEGASUS).

**Adapters (LOCAL, not forks):**

| Pack | Fiscal | PSP | SMS | Payout |
|------|--------|-----|-----|--------|
| UZ shipped | MY_SOLIQ + EDS | GLOBAL_PAY + CASH | PlayMobile | Bank-file |
| EU planned | PEPPOL or COMMERCIAL | Stripe (real executor, not `staticProviderExecutor`) | Twilio | SEPA file |
| US planned | COMMERCIAL | Stripe | Twilio | ACH file |
| KZ planned | TBD | CASH first | local | Bank-file |

One new country = pack + 1–3 adapters.

### GS-I — Enterprise identity (after T)

- OIDC **per supplier org**, not process-global `AUTH0_DOMAIN` wrapping all routes (that would 401 native HS256).
- Staff/driver stay JWT; buyer admin may SSO.
- SCIM later.

### Per-role backend change list (first PR only)

| Role | First backend change | Tag |
|------|----------------------|-----|
| All | A1 stamp JWT pack | ADAPT |
| Supplier | T1 register mint + pack persist | RESHAPE |
| Retailer | T4 attach partner; M1 currency/PSP | ADAPT + LOCAL |
| Warehouse | M4 labor TZ | ADAPT |
| Factory | T5 invite-only; pack SLA hours | RESHAPE + ADAPT |
| Payload | T5 staff lookup not `+998901110022` | RESHAPE |
| Driver | T5 `Drivers` table login; drop Tashkent coords | RESHAPE |
| Platform admin | T3 pack+cell dual-control; never default Soliq on non-UZ cell | ADAPT |

**KEEP without geo work:** WMS, freeze, seal/inject/seal-all, doorstep state machine, H3 dispatch, MEIO heuristic, dual manifest planes, 410 theatre (cards/loyalty/negotiate).

**DEFER:** factory planning on, auto-order place, saved cards, negotiations, marketplace.

---

## 3. Infra program (GS-C)

### Reality

| Fact | Evidence |
|------|----------|
| One project `pegasus-503013`, region `asia-south1` | `infra/terraform/staging.tfvars` |
| One TF state prefix `pegasusx/ssmr` | `backend.gcs.tf` |
| GSM `replication { auto {} }` | all-region copy of JWT/PSP |
| Project-wide `roles/spanner.databaseUser` | EU GSA in same project could read UZ |
| Overlays = env not cell | `overlays/{prod,ssmr,staging}` |
| `HOME_CELL=cell-uz` | ConfigMap only |
| No `cells/` tree, no `cell_id` var | terraform README vs GLOBAL_SCALE |

`tenant_slug` **namespaces names**. It is **not** a cell.

### Must be cell-local (never share)

Spanner, Kafka+topics+DLQ, Redis, GKE+WI, VPC/NAT, **JWT secret**, PSP/fiscal/SMS/Maps keys, Ingress+cert+`PUBLIC_BASE_URL`, OSRM extract, outbox, GSM **regional** replication, budget/alerts.

**May share later (global project):** container images, DNS zone (records per cell), desktop signing keys, billing account.

### Phases (design → plan; **no apply** until C1)

| ID | Work | Exit |
|----|------|------|
| **C0** | Cell = project+region+stack. Inventory live `pegasus-503013` objects. Fix README link to missing Phase 0 runbook. | Written contract |
| **C1** | Cell-safe **same** TF root: per-cell `backend.hcl`; vars `cell_id`, `api_hostname`, `k8s_namespace`; Kafka topics from `cell_id`; GSM user-managed in `var.region` only; WI from namespace; instance-level IAM; custom-mode VPC one subnet; `overlays/cells/uz` | Applying `region=europe-west1` **cannot** mutate `pegasusx/ssmr` state |
| **C2** | `infra/terraform/cells/{uz,eu}/` tfvars + backend.hcl; `make cell-plan CELL=eu` | Plan only |
| **C3** | New project `pegasusx-cell-eu`, `europe-west1`, own Spanner (same DDL), Redis, Kafka, GKE, GSM-EU, `api-eu.pegasusx.app`, **FISCAL commercial/none**, **new JWT**, no UZ backup restore | Empty adapters OK |
| **C4** | Isolation proof: EU GSA denied UZ Spanner/GSM; UZ JWT 401 on EU API; Kafka cross-bootstrap fails; GSM locations EU-only | Written evidence |
| **C5** | Optional modules/ + global AR/DNS; clients use session `home_cell` for API URL | After two cells exist |

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
| Geofence | **one** pack radius (drop 150 vs 500 split) |
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

## 6. Platform parity (GS-R, after M)

Every pack-visible field (currency, TZ, fiscal label, maps) on **all clients that role uses**. No web-only currency for retailer/supplier.

| Role | Clients |
|------|---------|
| Supplier | portal + Android + iOS |
| Retailer | desktop + Android + iOS |
| Warehouse / factory | portal + Android + iOS |
| Payload | terminal + Android + iOS |
| Driver | Android + iOS |
| PA | web only |

First UI: session `pack` on login splash (currency + “receipts: Soliq / commercial”). Deep POS/fiscal screens follow M2.

---

## 7. Sequence (backend ∥ infra)

```
A1 stamp JWT ──► A2 persist pack ──► T1 register ──► T3–T5 identity
                         │
                         ├── M1–M7 pack is law (do not claim second country before)
                         │
C0–C1 cell-safe TF ──► C2 repo layout ──► C3 EU plan ──► C4 isolation
                         │
                         └── I OIDC after T
```

**Legal single-cell (UZ) launch** still follows [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) R1–R2 (Soliq/GP keys). Global **register** is T+A, not more features.

**Second-country claim** requires M + C4. Not a country picker.

---

## 8. Explicit non-goals

- World marketplace  
- Multi-region Spanner writes  
- Loyalty / saved cards / negotiations  
- Factory planning default on  
- Process-global Auth0 wrapping all JWT  
- Applying EU Terraform before C1 state split  

---

## 9. Next implementation slice (approved order)

1. **A1** — `ClaimsWithMarket` on every `auth.Issue`  
2. **A2** — `Suppliers.MarketCode` / `HomeCell` + login read  
3. **T1** — `POST /v1/platform/tenants/register`  
4. **C1** — backend.hcl + `cell_id` vars (plan only)  
5. **M1–M2** — checkout + fiscal read shipped pack  

Agents used: auth/tenancy explore, terraform/k8s explore, fiscal/PSP explore, per-role blockers explore.
