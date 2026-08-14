# PegasusX — Global-scale enterprise program

**Date:** 2026-08-14  
**SoT tree:** `pegasusX/`  
**Inventory:** [`FEATURES_BY_APP_ROLE.md`](./FEATURES_BY_APP_ROLE.md) + live `*routes` / shells  
**Honesty:** [`ROLE_FEATURES_DOCS_VS_CODE.md`](./ROLE_FEATURES_DOCS_VS_CODE.md)  
**Tenancy:** [`MULTI_TENANCY_GATE5_PHASE1.md`](./MULTI_TENANCY_GATE5_PHASE1.md)  
**Ops residuals:** [`session-2026-08-13/RESIDUAL_REGISTER.md`](./session-2026-08-13/RESIDUAL_REGISTER.md)

**North star:** A company anywhere can **register into a home cell**, receive a **market pack** checkout actually uses, invite roles, and run the Class A loop (order → stock → truck → cash/credit → fiscal → payout) without seed IDs and without a UZ-only fork.

**Enterprise backend + infra (modules, cells, IAM):** [`GLOBAL_SCALE_BACKEND_INFRA.md`](./GLOBAL_SCALE_BACKEND_INFRA.md)

**Do not:** redesign Spanner→outbox→Kafka; merge factory/payload tables; flip factory planning / auto-order place globally; invent a second tenant key.

---

## 0. What “global enterprise” means here

| Layer | Today (code) | Global bar |
|-------|--------------|------------|
| Tenant | JWT `SupplierId` + `PreferTenant`; seed fail-closed in ssmr/prod | Self-serve **register** + home **cell** + market pack |
| Market | `countrycfg` UZ seed; `checkout_reads_this: false` | Versioned **MarketPack** that checkout, fiscal, SMS, maps **read** |
| Identity | Per-role password/Firebase login; PA MFA; no OIDC | OIDC/SAML for buyer org; SCIM later; staff JWT unchanged |
| Money | GP/cash/credit; MY_SOLIQ contract; commercial receipts default | Adapter per pack (Soliq / PEPPOL / none); PSP per pack |
| Cloud | One GKE + Spanner project (`infra/terraform`, one prod overlay) | **Regional cells** (same chart, different project/region) |
| Clients | 3 platforms per role (except driver/PA) | Same routes; pack-driven copy, currency, maps, OTP |

---

## 1. Role feature inventory (docs + code)

Tags: **KEEP** (works globally with config) · **ADAPT** (same logic, pack parameters) · **RESHAPE** (wrong grain for multi-market) · **LOCAL** (market adapter) · **DEFER** (not required for register+run)

### 1.1 Shared auth / platform

| Feature | Code | Global action |
|---------|------|----------------|
| JWT HS256 + `jti` revoke | `auth/jwt.go` | **ADAPT** — add `market_code`, `home_cell`; do not change HS256 until cell-local secrets |
| TenantContext / PreferTenant | `auth/tenant.go` | **KEEP** — still one `SupplierId`; add pack on context |
| Seed fallback | `SeedFallbackAllowed` | **KEEP** fail-closed prod |
| Refresh / logout | `platformroutes` | **KEEP** |
| Retailer multi-org select | `PendingOrgSelect` | **KEEP** — model for multi-cell later |
| PA login + TOTP | `platformadmin` | **KEEP** break-glass |
| OIDC / SAML | residual | **RESHAPE** new module after register |
| Firebase phone | env | **LOCAL** — SHA-1 / APNs per cell |
| `GET /v1/auth/session` | **GS-A1 (this slice)** | Returns identity + resolved MarketPack |

### 1.2 Retailer (desktop + Android + iOS)

| Feature | Action | Notes |
|---------|--------|--------|
| Register / login / memberships / locations | **ADAPT** | Register takes `market_code`; profile stores tax ID type from pack |
| Catalog / cart / quote | **ADAPT** | Currency + decimal from pack (already minor units) |
| Cash / card checkout | **LOCAL** | Card = pack PSP; cash stays; B2B 410 stays |
| Unified create | **KEEP** | Pay after offload is product law |
| Saved cards / loyalty | **DEFER** | 410; do not globalize theatre |
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
| Factory planning P5 | **KEEP** flag-off | Per-tenant soak; stub dispatch still in-memory (P5-H later) |
| MEIO cost_aware_v2 | **KEEP** | Heuristic; currency minor from pack |
| Credit programs / claims | **ADAPT** | Legal copy + windows from pack |
| EDI / 1C / ASN | **LOCAL** | 1C CIS; PEPPOL EU; SAP only if sold |
| Payout policy + bank-file | **LOCAL** | Rail per pack; `no_live_rail` until keys |
| CRM / entity resolution | **KEEP** | API-only extras |
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
| Default dispatch stub | **ADAPT later (P5-H)** | Read Spanner transfers before global flip |
| SLA board (request due) | **ADAPT** | Default hours from pack (not 48h UZ-only) |
| Transfer SLA / pull matrix | **KEEP** flags | Per-tenant |
| Staff bcrypt/invite | **KEEP** | |

### 1.6 Payload

| Feature | Action | Notes |
|---------|--------|--------|
| Seal / inject / seal-all / load ledger | **KEEP** | |
| Capacity 410 | **KEEP** | VU on manifest |
| Labels / GS1 | **LOCAL** | FNC1 already; legal label language from pack |

### 1.7 Driver

| Feature | Action | Notes |
|---------|--------|--------|
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

```
GS-A  Auth + session market pack     ← start here (this slice)
GS-T  Self-serve tenant register
GS-M  Checkout/fiscal/maps READ pack
GS-C  Regional cell (terraform/k8s)
GS-I  Enterprise identity (OIDC)
GS-R  Role UI parity (pack-aware copy)
GS-P  Partner/legal packs per sale
```

| Phase | Goal | Backend | Frontend | Cloud | Exit |
|-------|------|---------|----------|-------|------|
| **GS-A** | Session knows market | JWT + `/v1/auth/session` + pack catalog | Any login can show pack code | `DEFAULT_MARKET_CODE=UZ` | Tests; checkout still **does not** charge by pack |
| **GS-T** | Company registers | `POST /v1/platform/tenants/register` | Marketing + supplier setup wizard | None | New supplier row; no seed |
| **GS-M** | Pack is live law | Checkout, proximity, fiscal resolve pack | Currency/locale from session | Secrets per adapter | `checkout_reads_this: true` for UZ |
| **GS-C** | Second cell possible | `home_cell` on tenant | Cell-local API URL | Duplicate tfvars/overlay | EU cell empty adapters OK |
| **GS-I** | Buyer SSO | OIDC attach to supplier org | Portal “Login with IdP” | Secrets | Optional per tenant |
| **GS-R** | Role apps | No new domains | Currency, units, fiscal labels on 3 platforms | — | Parity matrix + pack |
| **GS-P** | Sold integrations | EDI/PEPPOL/1C adapters | Partner settings | Certs | Per contract |

**Dependency:** GS-A → GS-T → GS-M before any second-country **claim**. GS-C can scaffold in parallel. GS-I after GS-T. GS-R continuous. GS-P never blocks register.

---

## 5. GS-A (implement now) — Auth session + market pack

**In scope**

- `MarketPack` catalog in code (UZ **shipped**, others `planned`)  
- JWT claims `market_code`, `home_cell` (additive; empty = default UZ)  
- `GET /v1/auth/session` — authenticated any role  
- `GET /v1/platform/market-packs` — public list (no secrets)  
- `GET /v1/platform/market-packs/{code}` — 404 if unknown  
- Honesty: `checkout_reads_this: false` until GS-M  

**Out of scope**

- Checkout reading pack  
- OIDC  
- Multi-region Spanner  
- New login UI (session JSON is enough for clients to bind later)

---

## 6. Platform parity rule (GS-R)

Every pack-visible field must appear on **all clients that role actually uses**:

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

- World marketplace / discovery  
- One global Spanner  
- Loyalty, saved cards, negotiations  
- Live Soliq/GP keys in this program (ops residual)  
- Factory planning on by default  
- Linguistic-complete i18n  

---

## 8. Proof (GS-A)

```bash
go test ./auth/ ./platformroutes/ -count=1
# JWT round-trip market_code/home_cell
# UZ pack listed; XX → 404
# session 401 without token
```
