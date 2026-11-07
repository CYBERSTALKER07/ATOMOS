# PegasusX Security Policy

PegasusX is committed to maintaining the highest standards of security, privacy, and integrity across all software, infrastructure, and financial transaction pipelines.

---

## Supported Versions

Only the latest active release and release candidates receive security patches:

| Version / Branch | Supported          |
| ---------------- | ------------------ |
| `main`           | :white_check_mark: |
| `staging`        | :white_check_mark: |
| Older releases   | :x:                |

---

## Reporting a Vulnerability

If you discover a security vulnerability within PegasusX, **do NOT open a public GitHub issue**. Instead, follow the responsible disclosure process below:

1. **Email Submission**: Send an encrypted or detailed report to `security@pegasusx.app`.
2. **Include Key Details**:
   - Component / service affected (`apps/backend-go`, `infra/k8s`, client apps, etc.).
   - Vulnerability class (IDOR, authentication bypass, tenant leakage, SSRF, memory exhaustion, etc.).
   - Reproducible proof-of-concept steps or payload.
   - Potential impact assessment.
3. **Response Timelines**:
   - **Initial Acknowledgement**: Within 24 hours.
   - **Triage & Reproduction**: Within 48 hours.
   - **Remediation & Patch Deployment**: Within 7 calendar days for critical issues.

---

## Mandatory Security Invariants

All platform components must satisfy:
- **No Hardcoded Secrets**: Secrets and cryptographic keys must be provisioned via Secret Manager / ExternalSecrets.
- **Fail-Closed Design**: Payment gateways, Soliq fiscal reporting, and partner transports must fail closed when credentials or invariants are missing.
- **Constant-Time Verification**: All HMAC and token signatures must use constant-time comparison (`subtle.ConstantTimeCompare`).
- **Request Size Ceilings**: All HTTP endpoints must enforce bounded request sizes via `http.MaxBytesReader`.
