# Track 1 Handoff Report: Core Infrastructure, Auth, Admin & Middleware

**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track1_core_auth`  
**Target Codebase**: `pegasusX/apps/backend-go`  
**Timestamp**: 2026-08-30T05:25:00Z  
**Type**: Hard Handoff (Investigation Complete)

---

## 1. Observation

Direct line-by-line observations made across the Track 1 codebase:

1. **`auth/cell_isolation.go:32-35`**:
   ```go
   got := strings.ToLower(strings.TrimSpace(c.HomeCell))
   if got == "" {
       return nil
   }
   ```
   *Finding*: A token lacking `home_cell` returns `nil` (ACCEPTED) on a regional cell node, allowing foreign requests to bypass cell filtering.

2. **`mfa/service.go:77-98` & `mfa/handlers.go:161-164`**:
   ```go
   // mfa/handlers.go:161-164:
   if strings.Contains(r.URL.Path, "/platform-admin/mfa/") {
       next.ServeHTTP(w, r)
       return
   }
   // mfa/service.go:88-95:
   s.repo.Upsert(ctx, Record{Subject: subject, Secret: secret, Enabled: false, CreatedAt: now})
   ```
   *Finding*: `BeginEnroll` unconditionally overwrites an existing enabled TOTP record with a new unconfirmed secret without requiring step-up or password verification.

3. **`auth/route_guard.go:14-22`**:
   ```go
   func ProtectMutations(r chi.Router, cfg MutationGuardConfig, mount func(chi.Router)) {
       r.Group(func(gr chi.Router) {
           mount(gr)
       })
   }
   ```
   *Finding*: No authentication or protection middleware is mounted onto `gr`.

4. **`orgoidc/service.go:113-119`**:
   ```go
   claims := auth.Claims{
       Subject:      sub,
       Role:         auth.RoleAdmin,
       SupplierID:   c.SupplierID,
       IsRegistered: true,
       IsConfigured: true,
   }
   ```
   *Finding*: Any employee in the corporate IdP logging in via OIDC is unconditionally minted a JWT with `Role: RoleAdmin`.

5. **`orgoidc/jwks.go:27-57` & `orgoidc/service.go:101-108`**:
   *Finding*: `FetchJWKS` does not cache public keys and returns the first RSA key in JWKS, ignoring the JWT's `kid` header.

6. **`platformadmin/login.go:50-65`, `145-166`**:
   *Finding*: `envBootstrapAdmin` calls `bcrypt.GenerateFromPassword([]byte(envPass), bcrypt.DefaultCost)` dynamically on every unauthenticated failed lookup.

7. **`telemetry/http_metrics.go:63-70`**:
   *Finding*: When `RoutePattern()` is empty (e.g. 404 / unmatched path), `route = r.URL.Path`, creating an unbounded number of Prometheus metric label values.

8. **`bootstrap/reliability_middleware.go:461-503`**:
   *Finding*: In-memory rate limiter `buckets` map has no eviction or TTL cleanup.

9. **`auth/refresh.go:28-32` & `auth/jwt.go:174-176`**:
   *Finding*: `HandleTokenRefresh` parses tokens with `Parse()` which fails when `ExpiresAt < now`, blocking 401 client recovery.

10. **`mfa/service.go:129-148` & `mfa/totp.go:52-76`**:
    *Finding*: TOTP verification allows a 90s skew window but never records used timestamps, allowing replay attacks.

---

## 2. Logic Chain

1. **Regional Cell Tenancy**:
   - `auth/cell_isolation.go` is the designated gatekeeper for multi-cell isolation.
   - Because `got == ""` returns `nil`, tokens without `home_cell` are accepted unconditionally on all regional cells.
   - *Inference*: Tenant queries and mutations are routed to whichever cell receives the HTTP request, breaking cell isolation guarantees.

2. **MFA Account Takeover**:
   - `/platform-admin/mfa/` endpoints are exempt from step-up enforcement.
   - An unstepped session can call `POST /v1/platform-admin/mfa/enroll` and overwrite an existing admin's TOTP secret with a new secret, setting `Enabled: false`.
   - The attacker confirms the new secret via `POST /v1/platform-admin/mfa/confirm`, minting a fully stepped-up token with `mfa_verified: true`.
   - *Inference*: MFA protection on platform administrator accounts is completely bypassable if an unstepped token is intercepted.

3. **High-Cardinality Metric Denial of Service**:
   - `telemetry.HTTPMetricsMiddleware` intercepts all HTTP requests in `main.go:129`.
   - On unmatched routes (e.g., 404s with random paths), `route` defaults to `r.URL.Path`.
   - *Inference*: Attackers sending randomized URLs will induce unbounded memory growth in the Prometheus registry, leading to OOM crash of the backend.

4. **Bcrypt CPU Exhaustion**:
   - `POST /v1/auth/platform-admin/login` allows public unauthenticated requests.
   - Every lookup failure with environment credentials triggers synchronous `bcrypt.GenerateFromPassword`.
   - *Inference*: 50 req/sec will saturate all CPU cores, causing total service unresponsiveness.

---

## 3. Caveats

- **Scope Boundary**: Track 1 strictly covers Core Infrastructure, Auth, Admin, and Middleware. Application domain services (WMS, Factory planning, Driver dispatch, Retailer catalog/ordering, and Payment settlement) are audited in their respective tracks.
- **Assumptions**: We assume Cloud Spanner emulator is used for local integration tests while multi-region Cloud Spanner is targeted in production.
- **Alternative Interpretations**: `envBootstrapAdmin` bcrypt dynamic hashing was likely introduced as a quick dev convenience, but in any deployment with `PLATFORM_ADMIN_PASSWORD` set in environment variables, it creates a severe denial-of-service vulnerability.

---

## 4. Conclusion

The PegasusX Track 1 implementation provides robust foundational architecture with clean domain separation, fail-closed market pack contracts, and structured multi-tenant scoping. However, the identified **18 vulnerabilities and defects** (particularly the cell isolation bypass, MFA re-enrollment overwrite, OIDC admin privilege escalation, and Prometheus cardinality explosion) represent immediate security and stability risks that must be remediated before production certification.

All findings, evidence, and remediation code are fully documented in:
`/Users/shakhzod/Desktop/V.O.I.D/.agents/audit_track1_core_auth/findings.md`

---

## 5. Verification Method

To independently verify all findings and test remediations:

```bash
# 1. Run all Track 1 package tests
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go
go test -v -race ./auth/... ./bootstrap/... ./mfa/... ./platformadmin/... ./featureflags/... ./orgoidc/... ./staffinvite/... ./platform/... ./spannerutils/... ./pkg/... ./telemetry/...

# 2. Check Schema Drift and Parity
go run ./cmd/schema-drift -offline

# 3. Verify Multitenant and Role Matrix
SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/verify-multitenant
```
