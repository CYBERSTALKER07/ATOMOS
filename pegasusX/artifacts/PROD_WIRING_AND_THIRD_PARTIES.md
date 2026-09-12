# Production wiring status + third-party inventory

> **HISTORICAL / FROZEN (2026-07-28 snapshot, re-bannered 2026-08-12).**  
> Prefer [`../context/current_status.md`](../context/current_status.md) · [`../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) · [`../docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md).  
> TLS/DNS/ManagedCert and optimizer SSMR rows in body may be outdated (SSMR ManagedCert Active; optimizer SSMR replicas 1; prod 0).


**Date:** 2026-07-28  
**Stack:** `pegasus-503013` / `pegasusx-ssmr-gke` / ns `pegasusx-ssmr`  
**Scope:** What is already wired on GCP via CLI, what still needs you, and which **non-GCP / non–Global Pay** third parties the product uses.

> **Supersession (2026-08-05):** Maps + optimizer runtime truth lives in [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md). Treat §C below as historical; Google Routes is now the primary geometry path when the Maps key has Routes API enabled.

---

## 1. What is already on the server (GCP + in-cluster)

| Layer | Component | Status |
|-------|-----------|--------|
| **Compute** | GKE Standard `pegasusx-ssmr-gke` (3 nodes) | Live |
| **API** | `backend-go` 1/1 | Live (`ssmr-s15-pegasus-receipts`) |
| **Worker** | `backend-go-worker` 1/1 | Live |
| **AI** | `ai-worker` 1/1 | Live |
| **DB** | Spanner `pegasusx-ssmr-spanner` / `pegasusx-ssmr-db` | Live |
| **Cache** | Memorystore Redis TLS+AUTH | Live |
| **Events** | Strimzi Kafka in-cluster | Live |
| **Secrets** | GSM + ESO SecretSynced | Live |
| **Ingress** | GCE LB `136.69.43.141` host `api-ssmr.pegasusx.app` | Live |
| **TLS** | Self-signed interim; ManagedCert **Provisioning** until DNS | Partial |
| **Maps** | Real API key | Live |
| **Firebase Auth/FCM** | Project + verifier + FCM ADC | Live (apps still need configs) |
| **Receipts** | `FISCAL_PROVIDER=PEGASUS` (no Soliq) | Live |
| **GP webhooks** | Endpoint + CAPTURED smoke | Live (SUCCESS needs real merchant password) |
| **Assets/OTA** | GCS `pegasus-503013-ssmr-assets` + `UPDATES_BASE_URL` | Wired via CLI |
| **HPA / PDB** | min1–max2 / maxUnavailable 1 | Wired via CLI (small cluster) |

**Not set to `PEGASUSX_ENV=production` yet** — that profile fails closed until real Global Pay merchant password + non-placeholder webhooks + trusted HTTPS. Current tenant env remains `ssmr` (staging-shaped).

---

## 2. What I can (and did) wire via CLI

Done without waiting on third parties:

- GCS bucket + CORS for assets/updates  
- Strong random secrets for `internal-api-key`, Payme/Click webhook placeholders (replace when you use those PSPs)  
- ConfigMap: `GCS_BUCKET_NAME`, `UPDATES_BASE_URL`, fiscal/Kafka/Firebase/Maps already set  
- HPA + PDB sized for this cluster  

Cannot finish without **you** or external vendors:

| Blocker | Why |
|---------|-----|
| DNS A for `api-ssmr.pegasusx.app` → `136.69.43.141` | Managed Google TLS + trusted HTTPS for apps/webhooks |
| Global Pay staging/prod merchant password (and confirmed service_id/user) | SUCCESS settlement verify + real checkout |
| Global Pay receipt API (if separate from checkout) | Optional second receipt layer |
| Soliq/OFD sandbox | Tax fiscal receipts only when you want OFD |
| Real Payme / Click / Adyen / Stripe accounts | Only if you enable those rails |
| Apple notarize / Windows codesign | Desktop distribution (secrets already in GSM structure) |
| Domain registrar access | DNS |
| Datadog / other APM keys | Optional observability agent |

---

## 3. Third-party services **beyond Google Cloud and Global Pay**

These are the external products the monorepo is designed to talk to. **Bold = needed for a full marketplace launch in Uzbekistan** as currently modeled; others are optional rails or toolchains.

### A. Must-have / high-value for product (not GCP, not Global Pay)

| Service | Purpose | Who provides | Status today |
|---------|---------|--------------|--------------|
| **Firebase Auth (phone OTP)** | Retailer/driver phone login | Google Firebase (same GCP project) | Backend on; **mobile/web Firebase config files** still needed |
| **Firebase Cloud Messaging** | Push notifications | Google Firebase | Backend FCM online; **device tokens from apps** |
| **Soliq / My.soliq / OFD** (or licensed OFD operator) | **Tax fiscal receipts** (legal OFD) | Uzbekistan tax / OFD vendor | **Deferred** — using Pegasus platform receipts |
| **Apple Developer** | iOS apps, push APNs, notarize, App Store | Apple | Codesign secrets scaffolded in GSM |
| **Google Play Console** | Android apps, FCM linked to apps | Google | App-side |
| **Domain + DNS** | Public hostname for API | Registrar / Cloudflare / etc. | **You** — A record missing |

### B. Optional payment rails (code present; not required if Global Pay only)

| Service | Purpose | Needed if… |
|---------|---------|------------|
| **Payme** | UZ card/wallet | You offer Payme as checkout method |
| **Click** | UZ wallet | You offer Click |
| **Stripe** | International cards | Multi-country / card fallback |
| **Adyen** | International PSP | Enterprise multi-acquirer |
| **Airwallex** | FX / direct execution | Flag-gated experimental path |

If **Global Pay is the only PSP**, you can leave others as webhook secrets only (already non-dev randoms for some).

### C. Maps / routing / dispatch solver

| Service | Purpose | Notes |
|---------|---------|-------|
| **Google Maps Platform** | Geocode, Places, **Routes** (primary polyline) | Server key in GSM; `ROUTING_PROVIDER=auto` |
| **OSRM** (self-hosted) | Optional geometry fallback | `ROUTING_OSRM_URL` + PVC extract; often empty on SSMR |
| **optimizer-core** (OR-Tools) | Dispatch VRP | **Code-wired, cloud undeployed** (SSMR omit / prod replicas 0) |

See [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md). OSRM/optimizer are **not** third-party SaaS when self-hosted.

### D. Observability / ops (optional SaaS)

| Service | Purpose | Notes |
|---------|---------|-------|
| **Datadog** (or Grafana Cloud / New Relic) | APM, logs, metrics | Tracer already emits; agent not in-cluster → errors are noisy but non-blocking |
| **Sentry** | Error tracking | Only if you add SDK later |

### E. Comms / marketing (if you add later — not core backend today)

| Service | Purpose |
|---------|---------|
| **SMS gateway** (beyond Firebase phone) | Transactional SMS if not using Firebase only |
| **Email** (Resend / SES / SendGrid) | Receipts, onboarding email |
| **Telegram bot** | Ops alerts (optional patterns exist in some monorepos) |

### F. Desktop / packaging (retailer desktop etc.)

| Service | Purpose |
|---------|---------|
| **Apple notarization** | macOS desktop |
| **Windows code signing** | Windows desktop |
| **Tauri updater** | Auto-update (pubkey/private key already in GSM structure) |

---

## 4. Recommended “launch minimum” third parties (excluding GCP + Global Pay)

For a **real client pilot** with your current architecture:

1. **Domain + DNS** (so API HTTPS works)  
2. **Firebase** app configs in iOS/Android/web (OTP + FCM)  
3. **Apple + Google Play** developer accounts (ship apps)  
4. **Soliq/OFD** — only when legal tax receipts are mandatory for those clients  
5. **Payme/Click** — only if Global Pay does not cover those wallets  

Everything else (Stripe, Adyen, Datadog, OSRM SaaS, email) is optional until product asks for it.

---

## 5. Production flip checklist (when Global Pay sends credentials)

1. Put real `GLOBAL_PAY_*` into GSM (service_id, username, **password**, webhook secret if they dictate).  
2. DNS A → `136.69.43.141`; wait ManagedCertificate **Active**.  
3. Enable HTTPS redirect on FrontendConfig.  
4. Register webhook: `https://api-ssmr.pegasusx.app/v1/webhooks/global-pay`.  
5. Optionally enable GP receipt env vars when receipt API exists.  
6. Set `PEGASUSX_ENV=production` only after secrets pass `ValidateProductionProfile`.  
7. Scale cluster / HPA before raising minReplicas to 3.  
8. Destroy old `void-494000` when billing is single-project.

---

## 6. App pointing (Step 16) — still on you

Backends are ready at:

```text
http://api-ssmr.pegasusx.app  (with /etc/hosts or DNS → 136.69.43.141)
https://api-ssmr.pegasusx.app (self-signed until managed cert)
```

Point retailer/driver/warehouse apps and portals at that base URL + ship Firebase `google-services` / `GoogleService-Info.plist`.
