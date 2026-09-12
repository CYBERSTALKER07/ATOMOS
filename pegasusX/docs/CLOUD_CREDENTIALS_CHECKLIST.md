# PegasusX Cloud Credentials & External Secrets Checklist

**Document Version:** 2.0.0  
**Target Environments:** SSMR Staging (`pegasusx-ssmr`), Production (`pegasusx-prod`)  
**Secret Store:** Google Cloud Secret Manager (GSM) + Kubernetes External Secrets Operator (ESO) + HashiCorp Vault  
**Last Updated:** 2026-08-20  

---

## 1. Architectural Distinction: Layer A vs Layer B

To ensure complete clarity regarding system readiness and deployment requirements:

- **Layer A (Code & Logic Parity — 100% Complete)**:
  * All Go backend route handlers, Spanner transactional outbox workflows, DDL tables, WebSocket routers, and cross-role client applications are fully implemented and verified with genuine logic.
  * Local test execution, mock-free client test suites, and SSMR smokecheck tests pass without dummy responses.
- **Layer B (Live Cloud Secrets & Infrastructure Operations)**:
  * External cloud credentials managed in GCP Secret Manager, Vault, or mounted secret volumes required for live communication with external third-party SaaS and banking rails (e.g., GlobalPay live merchant credentials, Soliq OFD PKCS#12 signing certs, PlayMobile SMS gateway, live APNs tokens).

---

## 2. Required GSM Secrets (`backend-go-secrets`)

The Kubernetes ExternalSecret `backend-go-secrets` is **atomic** (all keys declared under `spec.data` are required for successful synchronization). In the SSMR staging environment, secrets are prefixed with `pegasusx-ssmr-*`.

| K8s `secretKey` | GSM Secret Resource ID | Purpose & Consumer | Status (Real vs Stub) |
|---|---|---|---|
| `jwt-secret` | `…-jwt-secret` | Primary HS256 JWT signing secret for human sessions across all 6 roles + Platform Admin. | **Real** (Mandatory before boot) |
| `internal-api-key` | `…-internal-api-key` | Inter-service API authentication (e.g., OR-Tools Optimizer sidecar, background cron triggers). | **Real** (Mandatory) |
| `google-maps-api-key` | `…-google-maps-api-key` | Server-side Google Maps Platform API key (Geocoding, Places, Routes API `routes.googleapis.com`). | **Real** (Mandatory for routing) |
| `global-pay-webhook-secret` | `…-global-pay-webhook-secret` | HMAC signature validation for inbound GlobalPay payment webhooks (`/v1/webhooks/globalpay`). | **Real** (Required for card webhooks) |
| `global-pay-service-id` | `…-global-pay-service-id` | Merchant Service ID issued by GlobalPay acquiring bank. | **Real** (Required for live card payments) |
| `global-pay-username` | `…-global-pay-username` | Merchant API username for GlobalPay checkout sessions. | **Real** (Required for live card payments) |
| `global-pay-password` | `…-global-pay-password` | Merchant API password for GlobalPay checkout authentication. | **Real** (Required for live card payments) |
| `redis-password` | `…-redis-auth` | Redis Memorystore AUTH token for caching, hot driver GPS telemetry, and rate-limiting. | **Real** (Mandatory for Redis cluster) |
| `adyen-webhook-secret` | `…-adyen-webhook-secret` | HMAC signature verification secret for Adyen webhooks. | Stub placeholder OK until Adyen rail activated |
| `stripe-webhook-secret` | `…-stripe-webhook-secret` | HMAC signature verification secret for Stripe webhooks. | Stub placeholder OK until Stripe rail activated |
| `payme-webhook-secret` | `…-payme-webhook-secret` | Webhook verification secret for Payme rail. | Stub placeholder OK (Route disabled in launch) |
| `click-webhook-secret` | `…-click-webhook-secret` | Webhook verification secret for Click rail. | Stub placeholder OK (Route disabled in launch) |

---

## 3. Fiscal Receipt & SMS Gateway Secrets (Layer B Operative Shells)

Fiscalization and SMS notification credentials must be provisioned in Secret Manager prior to enabling live fiscal mode (`FISCAL_PROVIDER=MY_SOLIQ` or `FISCAL_PROVIDER=SOLIQ`):

| K8s `secretKey` | GSM Secret Resource ID | Environment Variable | Purpose & Format |
|---|---|---|---|
| `fiscal-my-soliq-base-url` | `…-fiscal-my-soliq-base-url` | `FISCAL_MY_SOLIQ_BASE_URL` | Base URL for State Tax Committee (MySoliq / Soliq OFD) fiscal API. |
| `fiscal-my-soliq-api-key` | `…-fiscal-my-soliq-api-key` | `FISCAL_MY_SOLIQ_API_KEY` | API token for MySoliq fiscal integration. |
| `fiscal-my-soliq-tin` | `…-fiscal-my-soliq-tin` | `FISCAL_MY_SOLIQ_TIN` | Taxpayer Identification Number (INN/TIN) of the operating legal entity. |
| `fiscal-my-soliq-signer` | `…-fiscal-my-soliq-signer` | `FISCAL_MY_SOLIQ_SIGNER` | Digital signature provider type (`pkcs12` or `e-imzo`). |
| `fiscal-my-soliq-pkcs12-password` | `…-fiscal-my-soliq-pkcs12-password` | `FISCAL_MY_SOLIQ_PKCS12_PASSWORD` | Passphrase for encrypting/decrypting the E-IMZO PKCS#12 certificate. |
| `playmobile-login` | `…-playmobile-login` | `PLAYMOBILE_LOGIN` | PlayMobile SMS gateway account username for OTP and dunning dispatch. |
| `playmobile-password` | `…-playmobile-password` | `PLAYMOBILE_PASSWORD` | PlayMobile SMS gateway account password. |

> **Security Mandate**: The binary PKCS#12 digital signature certificate (`.pfx` / `.p12`) must **NEVER** be stored as a string in Secret Manager or committed to git. It is mounted directly into the container pod as a secure Kubernetes secret volume at `FISCAL_MY_SOLIQ_PKCS12_FILE`.

---

## 4. Mobile Client Push & Authentication Credentials

Mobile apps (Driver, Retailer, Warehouse, Factory, Payload, Supplier) require client configuration files and push notification keys:

```
+-------------------------------------------------------------------------------------------------------+
|                                    Mobile Credentials Hierarchy                                       |
+-------------------+-----------------------------------+-----------------------------------------------+
| Client Platform   | Credential Artifact               | Provisioning & Placement                      |
+-------------------+-----------------------------------+-----------------------------------------------+
| Android (6 Apps)  | `google-services.json`            | Co-located in each `app/` folder; real debug  |
|                   |                                   | SHA-1 fingerprint registered in Firebase.     |
+-------------------+-----------------------------------+-----------------------------------------------+
| iOS (6 Apps)      | `GoogleService-Info.plist`        | Co-located in Xcode project folders; bundle   |
|                   |                                   | ID registered in Firebase console.            |
+-------------------+-----------------------------------+-----------------------------------------------+
| Backend (FCM)     | Firebase Admin SDK ServiceAccount | Managed via GCP Workload Identity or GSM      |
|                   | Credentials (`FIREBASE_AUTH_JSON`)| secret volume for server-side push dispatch.  |
+-------------------+-----------------------------------+-----------------------------------------------+
| Apple APNs        | Apple Push Notification Key (.p8) | Key ID + Team ID + AuthKey registered in GSM  |
|                   |                                   | for direct APNs token-based push.             |
+-------------------+-----------------------------------+-----------------------------------------------+
```

---

## 5. Google Maps Platform Key Isolation Architecture

To prevent cross-environment abuse and enforce strict billing limits, API keys are segregated by client type and restricted by domain and package signature:

| Key Role | Secret Identifier | Restricted Consumers | Applied API Restrictions & Security Rules |
|---|---|---|---|
| **Server Backend** | `…-google-maps-api-key` | `backend-go` pods | Restricted by GKE NAT egress static IP. Enabled APIs: Geocoding API, Places API, Routes API (`routes.googleapis.com`). |
| **Android SDK** | `…-maps-android-api-key` | Android mobile apps | Restricted by Android Package Names (`com.pegasusx.*`) + SHA-1 / SHA-256 signing certificate fingerprints. |
| **iOS SDK** | `…-maps-ios-api-key` | iOS mobile apps | Restricted by Apple Bundle Identifier (`com.pegasusx.*`). (Apple MapKit requires no Google key). |

---

## 6. B2B Partner API & AS2 Integration Secrets

B2B Partner keys and AS2 transport certificates are managed dynamically:
- **Partner Machine Keys**: Issued via authenticated `POST /v1/admin/partner-keys` and stored hashed in Spanner (`PartnerApiKeys`).
- **Partner OAuth Tokens**: Signed using `PARTNER_JWT_SECRET` (or securely derived from `JWT_SECRET`).
- **AS2 Digital Certificates**: Partner AS2 private keys and X.509 certificates are referenced via `SecretRef` pointers to GSM keys (`PARTNER_AS2_SECRET_<REF>`) and never stored in plain text.

---

## 7. Operational Readiness & Bootstrap Sequence

1. **Infrastructure Provisioning**: Run Terraform in `infra/terraform/` to provision GKE clusters, Cloud Spanner instances, Redis Memorystore, and empty GSM secret shells.
2. **Secret Synchronization**: Execute `scripts/phase0_sync_gsm_secrets.sh` to populate GSM secrets. If operating in sandbox mode without active PSP contracts, set `ENSURE_ES_STUBS=1` to write safe placeholder tokens for inactive rails.
3. **Secret Verification**: Verify ExternalSecrets syncing in Kubernetes:
   ```bash
   kubectl get externalsecrets -n pegasusx-ssmr
   kubectl get secrets backend-go-secrets -n pegasusx-ssmr
   ```
4. **Backend Boot**: Deploy `backend-go` with `REQUIRE_INFRA_ADAPTERS=true` and confirm that all mandatory secrets are successfully loaded during bootstrap.
