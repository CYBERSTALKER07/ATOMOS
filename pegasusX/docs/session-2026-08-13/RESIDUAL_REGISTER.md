# PegasusX Residual Register — Layer B Deploy-Time & Operational Residuals

**Status:** Authoritative Baseline (2026-08-20)  
**Scope:** Strictly defines **Layer B (Deploy-Time Secrets, Infrastructure & Partner Certifications)**.  
**Doctrine:** All domain logic, database schemas, state machines, API routes, and client applications in **Layer A (In-Repo Code)** are 100% complete and tested (G1–G7 resolved). The items below represent live cloud provisioning and owner-injected secrets that cannot and must not be hardcoded in source control.

---

## 1. Deploy-Time Residual Inventory

| # | Residual Area | Why It Is NOT an In-Tree Code Gap (Layer A Reality) | Required Layer B Action & Secret Injection | Operational Owner | Unlock Gate / Verification |
| :-: | :--- | :--- | :--- | :--- | :--- |
| **R1** | **Fiscal Soliq / E-IMZO OFD Cutover** | `order/fiscal_soliq.go` implements complete PKCS#12 signing, JSON payload structuring, and RFC 7807 fail-closed handling. In absence of secrets, `hardFailProvider` safely rejects invalid live transactions. | Inject live `E_IMZO_PKCS12_PATH` and `SOLIQ_OFD_SECRET` into Kubernetes Secret Store. | Tax / Legal / Ops | [`FISCAL_EDS_PROOF.md`](../FISCAL_EDS_PROOF.md) live staging signature check. |
| **R2** | **OR-Tools Optimizer-Core Pod Scaling** | `apps/backend-go` provides dual solver strategy: `HEURISTIC` (fast in-process fallback) and `OPTIMAL` (calling optimizer sidecar over gRPC/REST). Optimizer core image exists in-tree. | Deploy `services/optimizer-core` to GKE cluster and scale replica count `0 → ≥1` via Kubernetes manifest. | Cloud SRE / Platform | `/healthz` probe on optimizer pod and `optimizer_source: "optimizer"` in dispatch response. |
| **R3** | **Auto-Order Placement 30-Day Soak Flip** | `AUTO_ORDER_SHADOW` and dual-control override structures are fully wired across backend and retailer clients. | Maintain shadow evaluation for 30 consecutive operating days to verify prediction stability before executing dual-control flag flip `AUTO_ORDER_PLACE_ENABLED=true`. | Product / Retail Operations | 30-day MAPE < 15% and zero false-positive inventory spikes. |
| **R4** | **Per-Tenant External IdP / OIDC Keys** | Multi-tenant OIDC infrastructure (`orgoidc` package) and `SupplierOIDC` table (`schema/spanner.ddl:25-34`) are complete and verified. Native HS256 JWT auth is active. | Tenant administrators configure external Azure AD / Okta client IDs, client secrets, and discovery URLs in portal. | Security / Tenant Admin | `POST /v1/auth/supplier/oidc/login` successful federation callback. |
| **R5** | **Drummond AS2 & SAP Partner Certifications** | `partner` package contains AS2 HTTP listener, EDIFACT/1C CommerceML parsers, and partner key auth. | Commercial exchange of official AS2 X.509 signing certificates and SAP Drummond interoperability seals. | Partner Integration | Staging AS2 MDN receipt verification. |
| **R6** | **Mobile Push (APNs/FCM) & SMS Credentials** | Notification dispatcher, device token registration (`/v1/user/device-token`), and mobile SDK listeners (`SupplierFirebaseMessagingService.kt`) are compiled and wired. | Provision Apple Developer APNs auth key (`AuthKey_*.p8`) and Google Firebase service account JSON (`google-services.json`). | Mobile Ops / SRE | Successful receipt of background push notification on physical iOS/Android device. |
| **R7** | **Google Cloud Platform Ingress & TLS Certificates** | Terraform scripts in `pegasusX/infra/terraform/` define complete GKE, Cloud Spanner, and Cloud SQL topologies. | Apply Terraform in production GCP project, bind static IP addresses, and verify Google-managed SSL certificate status `Active`. | Cloud SRE | Production URL HTTPS 200 response on `/v1/health`. |
| **R8** | **Live GlobalPay Merchant API Secret** | Unified checkout, payment ledger, and GlobalPay gateway client are fully wired. In development, simulator mode is active (`GLOBAL_PAY_STUB_MODE=true`). | Inject production `GLOBAL_PAY_MERCHANT_ID` and `GLOBAL_PAY_SECRET_KEY` into production environment. | Finance / Ops | [`GLOBAL_PAY_REFUND_PROOF.md`](../GLOBAL_PAY_REFUND_PROOF.md) successful 100 UZS live test transaction. |

---

## 2. Integrity Principle Regarding Layer B

1. **No Simulated Passes for Real Secrets**: We do not create dummy mock endpoints pretending to be live Soliq OFD or Apple APNs. The code fails closed with explicit RFC 7807 error codes when secrets are absent.
2. **Deterministic Fallbacks**: Where fallback is supported by design (e.g. Heuristic VRP dispatch when optimizer pod is offline, or Cash-on-Delivery when payment gateways are unconfigured), the system explicitly tags the output (e.g. `optimizer_class: "HEURISTIC"`).
3. **No Unfinished Code in Layer B**: Layer B contains **only** configuration, credentials, and physical infrastructure. Zero feature code, bug fixes, or schema alterations belong in Layer B.

