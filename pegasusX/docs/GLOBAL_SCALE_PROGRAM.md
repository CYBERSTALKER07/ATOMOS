# PegasusX — Global-scale enterprise program

**Final goal (2026-08-16):** this file + [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) are the destination. Agent load: `.agents/memory/GOAL.md`. Not status.

**Date:** 2026-08-16  
**SoT tree:** `pegasusX/`  
**Inventory:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) + live `*routes` / shells  
**Honesty:** [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md)  
**Tenancy:** [`MULTI_TENANCY_GATE5_PHASE1.md`](./MULTI_TENANCY_GATE5_PHASE1.md)  
**Ops residuals:** [`session-2026-08-13/RESIDUAL_REGISTER.md`](./session-2026-08-13/RESIDUAL_REGISTER.md)

**North star:** A **global multi-supplier** ecosystem: companies anywhere **register** (new `SupplierId`, not seed), land in a **home cell**, receive a **market pack** checkout actually uses, invite roles, and run Class A (order → stock → truck → cash/credit → fiscal → payout). Retailers trade with **many** suppliers; mixed carts split per supplier. Same code, cloned cells — not a UZ-only fork.

**Enterprise backend + infra (modules, cells, IAM):** [`GLOBAL_SCALE_BACKEND_INFRA.md`](./GLOBAL_SCALE_BACKEND_INFRA.md)  
**Every backend feature classified (KEEP/ADAPT/RESHAPE/LOCAL/DEFER):** [`GLOBAL_SCALE_BACKEND_FEATURES.md`](./GLOBAL_SCALE_BACKEND_FEATURES.md)  
**Phased modules × exits (Part I):** [`GLOBAL_SCALE_BACKEND_FEATURES.md`](./GLOBAL_SCALE_BACKEND_FEATURES.md#part-i--phased-modular-plan) — implement phases, not 250 inventory rows. GS-A0–A2 + T1–T5 + **M1–M7** + **C1–C5** + **GS-I** + **GS-R bind** + **GS-P** shipped (C plan-only). No next phase from this catalog. Leftover only when asked (flag, apply, live PEPPOL/PSP, SAML/SCIM).  
**Local-first warehouse / pack PSP (GS-L + GS-K, W1–W26):** [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) — does not replace A/T/M/C. Next slices **L0 + K1**.

**Do not:** redesign Spanner→outbox→Kafka; merge factory/payload tables; flip factory planning / auto-order place globally; invent a second tenant key.

---

## 0. What “global enterprise” means here

| Layer | Today (code) | Global bar |
|-------|--------------|------------|
| Tenant | JWT `SupplierId` + `PreferTenant`; Gate 5 isolation live; seed fail-closed in ssmr/prod | Many companies register; each isolated by `SupplierId` + home **cell** + market pack |
| Multi-supplier | Schema + request isolation; retailer attach + ParentOrders (backend) | Retailer shops **multiple** suppliers; child orders stay supplier-scoped |
| Market | MarketPack advertised; M1–M7 read pack (checkout, fiscal, radius, TZ, currency, payout_rail, tax country); `checkout_reads_this: false` | Versioned **MarketPack** that checkout, fiscal, SMS, maps **read** |
| Identity | Per-role password/Firebase login; PA MFA; **GS-I** per-supplier OIDC attach + exchange (process-global Auth0 wrap removed) | SAML/SCIM later; staff JWT unchanged |
| Money | GP/cash/credit; MY_SOLIQ contract; commercial receipts default | Adapter per pack (Soliq / PEPPOL / none); PSP per pack |
| Cloud | One GKE + Spanner project (`infra/terraform`, one prod overlay) | **Regional cells** (same chart, different project/region) |
| Clients | 3 platforms per role (except driver/PA) | Same routes; pack-driven copy, currency, maps, OTP |

---

## 1. Role feature inventory (docs + code)

Tags: **KEEP** (works globally with config) · **ADAPT** (same logic, pack parameters) · **RESHAPE** (wrong grain for multi-market) · **LOCAL** (market adapter) · **DEFER** (not required for register+run)

### 1.1 Shared auth / platform

| Feature | Code | Global action |
|---------|------|----------------|
| JWT HS256 + `jti` revoke | `auth/jwt.go` | **ADAPT** — A1 stamps `market_code`, `home_cell` on every `Issue`; do not change HS256 until cell-local secrets |
| TenantContext / PreferTenant | `auth/tenant.go` | **KEEP** — still one `SupplierId`; add pack on context |
| Seed fallback | `SeedFallbackAllowed` | **KEEP** fail-closed prod |
| Refresh / logout | `platformroutes` | **KEEP** |
| Retailer multi-org select | `PendingOrgSelect` | **KEEP** — model for multi-cell later |
| PA login + TOTP | `platformadmin` | **KEEP** break-glass |
| OIDC / SAML | `orgoidc` + portal Login with IdP | **RESHAPE done** for OIDC (optional per tenant). SAML/SCIM later. Do not remount AUTH0_DOMAIN wrap |
| Firebase phone | env | **LOCAL** — SHA-1 / APNs per cell |
| `GET /v1/auth/session` | **GS-A0 done** | Returns identity + resolved MarketPack. JWT stamp is A1 (done) |

### 1.2 Retailer (desktop + Android + iOS)

| Feature | Action | Notes |
|---------|--------|--------|
| Register / login / memberships / locations | **ADAPT** | T4: register requires `supplier_id` or invite (no silent seed). Demo `"1234"` only in ssmr. Profile tax ID type from pack |
| Catalog / cart / quote | **ADAPT** | Currency + decimal from pack (already minor units) |
| Cash / card checkout | **LOCAL** | Card = pack PSP; cash stays; B2B 410 stays |
| Unified create | **KEEP** | Pay after offload is product law |
| Saved cards / loyalty | **ADAPT** | Cards stay 410. Loyalty is live `{enrolled:false}` / earn — do not fake Bronze or invent burn |
| Tracking / dock / shop-closed | **ADAPT** | Breach radius, grace minutes from pack (today UZ hardcoded 150m/10m) |
| POS / stock / shifts / HQ / assist | **KEEP** | Locale + tax on POS receipt via pack |
| Credit / AR | **ADAPT** | Terms days, currency, dunning channels (SMS/WhatsApp/email) per pack |
| Auto-order place | **KEEP** flag-off | Never global default |
| Fiscal receipt | **LOCAL** | Soliq vs PEPPOL vs commercial |
| i18n | **ADAPT** | Keys exist (draft); pack sets default locale |

### 1.3 Supplier (`ADMIN` + portal/Tauri/Android/iOS)

| Feature | Action | Notes |
|---------|--------|--------|
| Register / business setup / billing setup | **RESHAPE** | Today UZ tax/billing; become pack onboarding wizard |
| Dispatch H3 + score + optional OR-Tools | **ADAPT** | Weights stay; maps/OSRM URL per cell; Rule of 25 stay |
| Factory planning P5 | **KEEP** flag-off | Per-tenant soak. Factory dispatch is **live Spanner** (warehouse solver → `FactoryTruckManifests`); not in-memory stub |
| MEIO cost_aware_v2 | **KEEP** | Heuristic; currency minor from pack |
| Credit programs / claims | **ADAPT** | Legal copy + windows from pack |
| EDI / 1C / ASN | **LOCAL** | 1C CIS; PEPPOL EU; SAP only if sold |
| Payout policy + bank-file | **LOCAL** | Rail per pack; `no_live_rail` until keys |
| CRM / entity resolution | **KEEP** | CRM `lines[]` + entity-resolution UI (portal + Android + iOS) |
| Country overrides | **RESHAPE** → MarketPack | `checkout_reads_this` must become true |

### 1.4 Warehouse

| Feature | Action | Notes |
|---------|--------|--------|
| Dispatch execute / freeze / WMS | **KEEP** | Class A already |
| Labor / cold / pick waves | **ADAPT** | Labor law hours, °C/°F from pack |
| Demand forecast | **KEEP** | Honest empty; no scaffold in prod |
| Treasury / financials | **ADAPT** | Currency; fee schedule per cell |
| QC on supply requests | **KEEP** | Process, not geography |

### 1.5 Factory

| Feature | Action | Notes |
|---------|--------|--------|
| Loading-bay seal / payload dual plane | **KEEP** | Do not merge tables |
| Default dispatch | **KEEP** | Live Spanner warehouse solver → `FactoryTruckManifests` only; nil-Spanner tests still `pick_n_created_v1` |
| SLA board (request due) | **ADAPT** | Default hours from pack (not 48h UZ-only) |
| Transfer SLA / pull matrix | **KEEP** flags | Per-tenant |
| Staff bcrypt/invite | **KEEP** | T5: public register is invite or ADMIN+node; bcrypt on write |

### 1.6 Payload

| Feature | Action | Notes |
|---------|--------|--------|
| Seal / inject / seal-all / load ledger | **KEEP** | |
| Capacity 410 | **KEEP** | VU on manifest |
| Labels / GS1 | **LOCAL** | FNC1 already; legal label language from pack |

### 1.7 Driver

| Feature | Action | Notes |
|---------|--------|--------|
| Login | **KEEP** | T5: `Drivers` row + bcrypt PinHash; no `+998` default |
| Doorstep cash/credit/QR/fiscal | **ADAPT** | Currency, fiscal adapter, cash-custody hours from pack |
| Offline queue | **KEEP** | |
| Maps / geofence | **ADAPT** | Maps key + breach radius from cell/pack |
| Negotiate 410 | **KEEP** | |

### 1.8 Platform admin

| Feature | Action | Notes |
|---------|--------|--------|
| Tenants / flags / dual-control | **ADAPT** | Approve **market pack assign** + cell |
| Outbox / dead-letters | **KEEP** per cell | |
| Billing invoices | **ADAPT** | Platform fee currency = cell |

---

## 2. Technical reshape (logic / math / cloud)

| Area | Today | Global change |
|------|--------|----------------|
| Currency | Implicit UZS, 2 decimals | Pack `currency_code` + `decimal_places`; still **integer minor** only |
| Time | `Asia/Tashkent` in labor/SQL | Pack timezone; never store local without offset |
| Distance | Haversine km, H3 | Pack `grid=H3` keep; `distance_unit` display only |
| Shop-closed / proximity | Hardcoded meters/minutes | Pack numbers; same state machine |
| Safety stock / MEIO | Unitless qty + minor | No formula change; money fields use pack currency |
| Optimizer | OR-Tools or H3 heuristic | Per-cell sidecar URL; honesty labels stay |
| Maps | Google Routes → OSRM | Cell secret + optional regional OSRM extract |
| Spanner | One instance | **One instance per cell** (same DDL). No multi-region writes in v1 |
| Terraform | `infra/terraform` single project | `cells/{uz,eu,us}/` tfvars; same modules |
| K8s | one overlay prod/ssmr | Overlay per cell; `HOME_CELL`, `DEFAULT_MARKET_CODE` |
| Kafka | one cluster | Per cell; no cross-cell topics |
| FCM/APNs/SMS | residual secrets | Per cell providers |

**Math/edge that must stay:**

- Money: integer minor, no float  
- Credit leave + AR same txn  
- Fiscal fail-closed if provider required and keys missing  
- Seed never in prod  
- Dual manifest planes  
- Factory planning default off  

---

## 3. Non-technical (business + regulation + local services)

| Concern | UZ (exists) | Other markets (adapter) |
|---------|-------------|-------------------------|
| Tax / OFD | MY_SOLIQ + EDS | PEPPOL / fiscalization / “commercial receipt only” |
| PSP | Global Pay + cash | Stripe/Adyen/local; cash + credit always |
| KYC | Informal setup | Pack required fields (VAT, EIN, INN) |
| Privacy | — | GDPR DPA + residency = home cell |
| Labor | Labor capacity table | Max hours / rest from pack (display + soft gate) |
| Comms | PlayMobile / Twilio residual | Pack SMS/email/WhatsApp |
| Contracts | — | DPA + SLA + status page before self-serve EU/US |
| Language | en/ru/uz draft | Pack `locales[]`; no fake complete i18n |

**One new country = one MarketPack + 1–3 adapters (fiscal, PSP, SMS).** Not a fork.

---

## 4. Phased modular program (GS)

Module exits and BF lists live in [`GLOBAL_SCALE_BACKEND_FEATURES.md` Part I](./GLOBAL_SCALE_BACKEND_FEATURES.md#part-i--phased-modular-plan). This section is the product summary.

```
GS-0  Honesty stamp (docs)           ← stamped 2026-08-15
GS-A  Auth + session market pack     ← A0–A2 done
GS-T  Self-serve tenant register     ← T1–T5 done
GS-M  Checkout/fiscal/maps READ pack
GS-C  Regional cell (terraform/k8s)
GS-I  Enterprise identity (OIDC)      ← done (SAML/SCIM later)
GS-R  Role UI parity (pack-aware copy) ← bind done
GS-P  Partner/legal packs per sale  ← done (never blocks register)
```

| Phase | Goal | Backend | Frontend | Cloud | Exit |
|-------|------|---------|----------|-------|------|
| **GS-A** | Session knows market | JWT + `/v1/auth/session` + pack catalog | Any login can show pack code | `DEFAULT_MARKET_CODE=UZ` | Tests; M1 checkout reads currency/PSP; flag still false |
| **GS-T** | Company registers | T1–T5 **done** | Marketing + supplier setup wizard | None | New supplier row; no seed |
| **GS-M** | Pack is live law | M1–M7 done. Leftover: flag | Currency/locale from session | Secrets per adapter | `checkout_reads_this: true` only when SSMR fiscal matches pack MY_SOLIQ |
| **GS-C** | Second cell possible | `home_cell` on tenant | Cell-local API URL | **C1–C5 done** (session `api_url` + global DNS/AR plan). EU/global not applied | EU cell empty adapters OK |
| **GS-I** | Buyer SSO | **Done** — `SupplierOIDC` + `/v1/auth/oidc/{discovery,exchange}` + `/v1/supplier/oidc`. AUTH0 wrap removed | Portal “Login with IdP” + Integrations attach | Secrets stay out of Spanner | Optional per tenant. SAML/SCIM later |
| **GS-R** | Role apps | **Bind done** — session pack chip; no new domains | Currency + receipts on portal+Android+iOS; native pin | — | Deep UZS leftover continuous |
| **GS-P** | Sold integrations | **Done** — dialect catalog + fail-closed 1C/PEPPOL/X12/SAP/AS2 gates. Register unblocked | Partner settings | Certs | Per contract; PEPPOL execute still unimplemented |

**Dependency:** GS-A → GS-T → GS-M before any second-country **claim**. GS-C can scaffold in parallel. GS-I after GS-T. GS-R continuous. GS-P never blocks register.

---

## 5. GS-A — Auth session + market pack (A0–A2 done)

**In scope**

- `MarketPack` catalog in code (UZ **shipped**, others `planned`)  
- JWT claims `market_code`, `home_cell` — **A1 stamps on every `Issue`** (claim → profile → env)  
- **A2** persists nullable `Suppliers.MarketCode` / `HomeCell`; session `source: claim|profile|env`; empty ≠ chosen UZ  
- `GET /v1/auth/session` — authenticated any role  
- `GET /v1/platform/market-packs` — public list (no secrets)  
- `GET /v1/platform/market-packs/{code}` — 404 if unknown  
- Honesty: `checkout_reads_this: false` until SSMR fiscal matches pack MY_SOLIQ  
- **M1:** checkout quote/unified/preview/card/cash read shipped currency + PSP  
- **M2:** fiscal collect/retry/worker/receipts fail-close on shipped `fiscal_adapter`  
- **M3:** one `breach_radius_meters` from shipped pack (UZ 150; 500 deleted)  
- **M4:** shop-closed grace, labor TZ/hours, weather scope, factory SLA hours, calendar TZ from shipped pack  
- **M5:** empty order/payment currency from shipped pack (not hardcoded `UZS`)  
- **M6:** pack `payout_rail` (UZ bank-file); unknown+live → `no_live_rail`  
- **M7:** tax stamp from pack country (`countryFromCurrency` deleted)  
- **C1:** cell-safe same TF root (plan only; `backend-ssmr.hcl` vs `backend-cell.example.hcl`)
- **C2:** `cells/{uz,eu}/` + `make cell-plan CELL=eu` (plan only; no live GCS write)
- **C3:** project factory `pegasusx-cell-eu`; empty adapters OK; same DDL; new JWT; no UZ restore (plan only)
- **C4:** isolation proof written (`make cell-isolation-proof`). UZ JWT 401 on EU (`CELL_JWT_ENFORCE` / production). IAM/Kafka/GSM structural. Live gcloud deny waits for apply.
- **C5:** session `api_url` / `ws_url` from `home_cell`; `GET /v1/platform/cells`; global DNS/AR plan (`make global-plan`); Next.js/Tauri portals pin JWT cell (localhost stays local).
- **GS-R:** native pin closed; session pack splash (currency + receipts) on role clients. Deep UZS leftover continuous.

**Out of scope**

- Cell terraform apply / live EU or global project (ops)  
- Flipping `checkout_reads_this`  
- SAML / SCIM (after GS-I OIDC)  
- Multi-region Spanner  
- Deep POS UZS labels / maps SDK swap (GS-R leftover, continuous)

---

## 6. Platform parity rule (GS-R bind done)

Every pack-visible field must appear on **all clients that role actually uses**. Bind shipped: session chip + native pin. Deep screens may still hardcode UZS until they call `packCurrency`.

| Role | Surfaces |
|------|----------|
| Supplier | portal + Android + iOS |
| Retailer | desktop + Android + iOS |
| Warehouse | portal + Android + iOS |
| Factory | portal + Android + iOS |
| Payload | terminal + Android + iOS |
| Driver | Android + iOS |
| Platform admin | web only |

No “web-only currency” for retailer/supplier.

---

## 7. Explicit non-goals

- Open-world **discovery / ads** marketplace (browse-the-planet) as a v1 claim — multi-supplier **register + attach + parent-order split** is in-goal  
- One global Spanner  
- Saved cards, negotiations, loyalty **burn** (earn/tier are live KEEP)  
- Live Soliq/GP keys in this program (ops residual)  
- Factory planning on by default  
- Linguistic-complete i18n  
- A second tenant key (do not replace `SupplierId`)  

---

## 8. Proof (GS-A + M1 + M2)

```bash
go test ./auth/ ./order/ ./payment/ ./promotion/ ./bootstrap/ -count=1
# JWT round-trip market_code/home_cell
# UZ pack listed; XX → 404
# session 401 without token
# M1: planned pack 404; EUR on UZ 422; STRIPE 422; PEGASUS → GLOBAL_PAY
# M2: EU fiscal 404; PEPPOL unimplemented; FAKE forbidden in prod; poller UZ-only
```
