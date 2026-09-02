# Comprehensive Codebase Audit: Track 1 — Core Infrastructure, Auth, Admin & Middleware
**Codebase**: `pegasusX/apps/backend-go`  
**Date**: 2026-08-30  
**Scope**: Entrypoints (`main.go`, `runtime_workers.go`), Bootstrap & Config (`bootstrap/*`), Authentication & Middleware (`auth/*`, `mfa/*`, `staffinvite/*`, `orgoidc/*`), Platform Admin & Feature Flags (`platformadmin/*`, `featureflags/*`, `platform/*`), Routes (`platformroutes/*`, `infraroutes/*`, `telemetryroutes/*`), Shared Utilities & Telemetry (`spannerutils/*`, `pkg/*`, `telemetry/*`), and CLI Commands (`cmd/*`).

---

## 1. Executive Summary

Track 1 represents the security and infrastructure backbone of the PegasusX ecosystem. It enforces cell isolation, multi-tenant boundaries, role-based access control (RBAC), multi-factor authentication (MFA/TOTP), corporate OIDC identity bridging, distributed rate limiting, and outbox/runtime operations.

Our line-by-line audit identified **18 verified defects and vulnerabilities** across the codebase:
- **3 Critical Severity**: Cell isolation bypass on missing claims, MFA enrollment overwrite/step-up bypass, and no-op mutation protection.
- **7 High Severity**: Unauthenticated OIDC privilege escalation to `RoleAdmin`, missing OIDC JWKS caching & key matching, DoS via bcrypt generation on unauthenticated login, Prometheus metric cardinality explosion via 404 URL paths, in-memory rate limiter memory leaks, broken token refresh for expired tokens, and missing TOTP replay protection.
- **5 Medium Severity**: Worker startup notification consumer race, unexpected warehouse entity mutation during user registration, fragile Spanner error checking, re-usable staff invite tokens, and dual-control asymmetry on tenant suspension.
- **3 Low / Performance**: Unindexed Spanner full table scans in SLO telemetry polling, redundant Spanner transaction retries, and context-propagated read-only transaction session leaks.

Below is the exhaustive, evidence-backed breakdown of each finding with exact file:line references, architectural context, blast radius analysis, and concrete remediation code.

---

## 2. Exhaustive Findings Table

| ID | File & Line | Severity | Category | Flaw Summary | Blast Radius |
|---|---|---|---|---|---|
| **VULN-01** | `auth/cell_isolation.go:32-35` | **CRITICAL** | Security / Tenancy | Omitted `home_cell` claim bypasses regional cell isolation | Regional tenancy breach / foreign cell access |
| **VULN-02** | `mfa/service.go:77-98`, `mfa/handlers.go:161-164` | **CRITICAL** | Security / Auth | MFA `BeginEnroll` overwrites active TOTP without step-up | Account takeover & MFA bypass on admin accounts |
| **VULN-03** | `auth/route_guard.go:14-22` | **CRITICAL** | Security / Middleware | `ProtectMutations` creates group without attaching auth/guard | Unprotected mutation routes relying on guard |
| **VULN-04** | `orgoidc/service.go:113-119` | **HIGH** | Security / Auth | OIDC SSO exchange unconditionally grants `RoleAdmin` | Any corporate IdP user gains full supplier admin |
| **VULN-05** | `orgoidc/jwks.go:27-57`, `service.go:101-108` | **HIGH** | Security / Reliability | No JWKS cache & takes first RSA key ignoring JWT `kid` | DoS via latency + SSO broken during key rotation |
| **VULN-06** | `platformadmin/login.go:50-65`, `145-166` | **HIGH** | Security / DoS | Generates bcrypt hash dynamically on invalid login | Unauthenticated CPU exhaustion DoS attack |
| **VULN-07** | `telemetry/http_metrics.go:63-70` | **HIGH** | Stability / Memory | Fallback to raw `r.URL.Path` on 404/unmatched routes | Prometheus cardinality explosion & memory crash |
| **VULN-08** | `bootstrap/reliability_middleware.go:461-503` | **HIGH** | Stability / Memory | Unbounded `map[string]rateBucket` without eviction or TTL | Unbounded memory leak under diverse IP traffic |
| **VULN-09** | `auth/refresh.go:28-32`, `jwt.go:174-176` | **HIGH** | Security / Logic | Refresh rejects expired tokens & fails to revoke prior jti | 401 client recovery fails; active token duplication |
| **VULN-10** | `mfa/service.go:129-148`, `totp.go:52-76` | **HIGH** | Security / MFA | TOTP verification lacks replay prevention cache | Intercepted OTP code reusable for 90 seconds |
| **BUG-11** | `main.go:117`, `runtime_workers.go:240-254` | **MEDIUM** | Concurrency / Logic | Notification consumer started on boot is never stopped | Double consumption & duplicate push notifications |
| **BUG-12** | `warehouse/auth_register.go:142-146` | **MEDIUM** | Data Integrity | Staff registration overwrites `Warehouses` metadata | Inadvertent overwrite of warehouse entity data |
| **BUG-13** | `mfa/spanner.go:24-27` | **MEDIUM** | Error Handling | Direct `err == spanner.ErrRowNotFound` check | Status error wrapping causes 500 on missing row |
| **BUG-14** | `staffinvite/invite.go:122-163` | **MEDIUM** | Security / Auth | Stateless HMAC staff invites lack single-use consumption | Leaked invite tokens allow unlimited registrations |
| **BUG-15** | `platformadmin/service.go:204-228` | **MEDIUM** | Governance | Single-admin ability to suspend or offboard tenants | Rogue admin can unilaterally halt supplier commerce |
| **PERF-16** | `telemetry/slo_metrics.go:142-152` | **LOW / PERF** | Performance / DB | `outboxLagP99` orders by computed ms without index | Periodic full Spanner table scan every 60s |
| **PERF-17** | `spannerutils/retry.go:26-50` | **LOW / PERF** | Concurrency / DB | Outer retry loop wraps Spanner SDK's internal retries | Retry amplification & lock contention storms |
| **LEAK-18** | `spannerutils/retry.go:65-75` | **LOW / LEAK** | Resource Mgmt | Unmanaged `*spanner.ReadOnlyTransaction` in context | Leaks Spanner client sessions if caller neglects Close |

---

## 3. In-Depth Vulnerability & Defect Analysis

### VULN-01: Regional Cell Isolation Bypass via Omitted `home_cell` Claim
- **Exact Location**: `pegasusX/apps/backend-go/auth/cell_isolation.go:32-35`
- **Code Snippet**:
  ```go
  func rejectForeignCell(c Claims) error {
      current := CurrentCellID()
      if current == "" {
          return nil
      }
      got := strings.ToLower(strings.TrimSpace(c.HomeCell))
      if got == "" {
          return nil // <--- VULNERABILITY: Empty claim accepted on regional cell!
      }
      if got != current {
          return ErrForeignCellAccess
      }
      return nil
  }
  ```
- **Flaw Explanation**:
  The system is designed with a "local-first" cell architecture where regional instances (e.g. `cell-eu`, `cell-us`, `cell-kz`) must reject traffic intended for other cells. If an incoming token omits the `home_cell` claim (or sets it to empty), `got == ""` evaluates to `true`, and `rejectForeignCell` returns `nil` (ACCEPTED). An attacker possessing a token without `home_cell` can access any regional cell instance worldwide.
- **Blast Radius**:
  Completely compromises global tenant isolation and regional data sovereignty. Foreign tenant requests execute against local cell Spanner databases.
- **Remediation**:
  ```go
  func rejectForeignCell(c Claims) error {
      current := CurrentCellID()
      if current == "" {
          return nil
      }
      got := strings.ToLower(strings.TrimSpace(c.HomeCell))
      if got == "" || got != current {
          return ErrForeignCellAccess
      }
      return nil
  }
  ```

---

### VULN-02: MFA Enrollment Overwrite & Step-Up Bypass on Enrolled Admin Accounts
- **Exact Location**: `pegasusX/apps/backend-go/mfa/service.go:77-98`, `mfa/handlers.go:161-164`
- **Code Snippet**:
  ```go
  // In mfa/handlers.go:
  if strings.Contains(r.URL.Path, "/platform-admin/mfa/") {
      next.ServeHTTP(w, r)
      return
  }

  // In mfa/service.go:
  func (s *Service) BeginEnroll(ctx context.Context, subject string) (secret, otpauthURL string, err error) {
      secret, err = GenerateSecret()
      ...
      if err := s.repo.Upsert(ctx, Record{
          Subject:   subject,
          Secret:    secret,
          Enabled:   false, // <--- Overwrites existing enabled TOTP without check!
          CreatedAt: now,
      }); err != nil {
          return "", "", err
      }
      return secret, OTPAuthURL(s.issuer, subject, secret), nil
  }
  ```
- **Flaw Explanation**:
  All routes under `/platform-admin/mfa/` are exempt from step-up validation. If an attacker gains an unstepped temporary session token for an admin subject, they can call `POST /v1/platform-admin/mfa/enroll`. `BeginEnroll` does not verify whether the subject already has an active MFA secret and unconditionally overwrites the database record with a new secret, setting `Enabled: false`. The attacker then calls `POST /v1/platform-admin/mfa/confirm` with an OTP code generated from their new secret, gaining full `mfa_verified: true` step-up access and locking out the legitimate administrator.
- **Blast Radius**:
  Complete account takeover and MFA security control bypass across all `PLATFORM_ADMIN` identities.
- **Remediation**:
  In `BeginEnroll`, inspect the current record. If `rec.Enabled == true`, require an existing MFA step-up or valid recovery token before allowing re-enrollment.

---

### VULN-03: No-Op Mutation Protection in `ProtectMutations`
- **Exact Location**: `pegasusX/apps/backend-go/auth/route_guard.go:14-22`
- **Code Snippet**:
  ```go
  func ProtectMutations(r chi.Router, cfg MutationGuardConfig, mount func(chi.Router)) {
      r.Group(func(gr chi.Router) {
          // No middleware attached to gr!
          mount(gr)
      })
  }
  ```
- **Flaw Explanation**:
  `ProtectMutations` was designed to attach mutation guards (such as idempotency checks, CSRF validation, or authentication verification) across route groups. However, the function only executes `mount(gr)` without injecting any middleware into `gr`.
- **Blast Radius**:
  Any developer routing mutation endpoints via `ProtectMutations` under the expectation of automated protection leaves the endpoints completely unguarded.
- **Remediation**:
  Inject the intended mutation protection and authentication middleware into `gr` before calling `mount(gr)`.

---

### VULN-04: Unconditional `RoleAdmin` Escalation on OIDC ID Token Exchange
- **Exact Location**: `pegasusX/apps/backend-go/orgoidc/service.go:113-119`
- **Code Snippet**:
  ```go
  claims := auth.Claims{
      Subject:      sub,
      Role:         auth.RoleAdmin, // <--- Any valid IdP user gets RoleAdmin!
      SupplierID:   c.SupplierID,
      IsRegistered: true,
      IsConfigured: true,
  }
  ```
- **Flaw Explanation**:
  When a user logs in via corporate SSO / OIDC (`/v1/auth/oidc/exchange`), the service validates the RSA signature of the ID token. Upon successful signature validation, it unconditionally grants `Role: auth.RoleAdmin`. There is no check of the user's groups, claims, email prefix, or admin allowlist.
- **Blast Radius**:
  Any employee in a supplier organization with an active corporate Google Workspace / Okta / Azure AD account is automatically granted full admin rights to modify catalog, bank accounts, warehouse configurations, and credit lines.
- **Remediation**:
  Incorporate group/role mapping from the ID token claims (`groups`, `roles`, or an admin email allowlist configured in `orgoidc.Config`).

---

### VULN-05: Missing JWKS Caching and Missing `kid` Matching in OIDC ID Token Verification
- **Exact Location**: `pegasusX/apps/backend-go/orgoidc/jwks.go:27-57`, `orgoidc/service.go:101-108`
- **Code Snippet**:
  ```go
  func FetchJWKS(ctx context.Context, issuer string) (*rsa.PublicKey, error) {
      ...
      client := &http.Client{Timeout: 8 * time.Second}
      resp, err := client.Do(req)
      ...
      for _, k := range doc.Keys {
          if !strings.EqualFold(k.Kty, "RSA") { continue }
          if k.Use != "" && !strings.EqualFold(k.Use, "sig") { continue }
          return rsaPublicFromJWK(k.N, k.E) // <--- Returns first key, ignores token kid!
      }
      return nil, fmt.Errorf("oidc jwks: no rsa key")
  }
  ```
- **Flaw Explanation**:
  1. No caching: Every `/exchange` request spawns an uncached HTTP GET request to `{issuer}/.well-known/jwks.json` with an 8-second timeout.
  2. No `kid` matching: `FetchJWKS` iterates through the keys array and returns the very first RSA signing key found, completely ignoring the `kid` header in the incoming JWT. When IdPs rotate keys (e.g. Google, Microsoft, Okta have 2-3 keys published simultaneously), signature verification fails for any token signed by non-primary keys.
- **Blast Radius**:
  SSO login failures during key rotation windows and latency/SSRF DoS vulnerabilities.
- **Remediation**:
  Cache JWKS keys with TTL (e.g. 1 hour) and look up public keys by matching `token.Header["kid"]` against `jwk.Kid`.

---

### VULN-06: Denial of Service via Dynamic Bcrypt Hashing on Unauthenticated Platform Admin Login
- **Exact Location**: `pegasusX/apps/backend-go/platformadmin/login.go:50-65`, `145-166`
- **Code Snippet**:
  ```go
  func envBootstrapAdmin(subject, email string) (sub, hash string, ok bool) {
      envSub := strings.TrimSpace(os.Getenv("PLATFORM_ADMIN_SUBJECT"))
      envPass := os.Getenv("PLATFORM_ADMIN_PASSWORD")
      if envSub == "" || envPass == "" { return "", "", false }
      ...
      h, err := bcrypt.GenerateFromPassword([]byte(envPass), bcrypt.DefaultCost)
      if err != nil { return "", "", false }
      return want, string(h), true
  }
  ```
- **Flaw Explanation**:
  On every login attempt where the subject is not found in Spanner (or before DB migration), `envBootstrapAdmin` dynamically computes `bcrypt.GenerateFromPassword` with `bcrypt.DefaultCost` (cost 10 = ~100ms CPU time). An unauthenticated attacker sending 50 concurrent requests/sec to `POST /v1/auth/platform-admin/login` will pin all CPU cores at 100%, causing denial of service.
- **Blast Radius**:
  Severe CPU starvation affecting the entire backend API and all concurrent user requests.
- **Remediation**:
  Pre-hash `PLATFORM_ADMIN_PASSWORD` at startup (`sync.Once` or package initialization) and store the resulting hash in memory.

---

### VULN-07: Prometheus High-Cardinality Metric Explosion in `HTTPMetricsMiddleware`
- **Exact Location**: `pegasusX/apps/backend-go/telemetry/http_metrics.go:63-70`
- **Code Snippet**:
  ```go
  route := chi.RouteContext(r.Context()).RoutePattern()
  if route == "" {
      route = r.URL.Path // <--- EXPLOSION: Random 404 paths become metric labels!
  }
  statusClass := strconv.Itoa(rec.status/100) + "xx"
  httpRequestsTotal.WithLabelValues(route, statusClass).Inc()
  httpRequestDuration.WithLabelValues(route).Observe(time.Since(start).Seconds())
  ```
- **Flaw Explanation**:
  If a request does not match any chi route pattern (e.g. 404 Not Found), `RoutePattern()` returns empty string `""`. The middleware falls back to using `r.URL.Path` as the Prometheus label value `route`. If an automated scanner or attacker sends requests with random UUIDs or random paths, each distinct path creates new Prometheus time series in memory.
- **Blast Radius**:
  Unbounded memory growth, Prometheus scraping failure, metrics collector crash, and container OOM kill.
- **Remediation**:
  ```go
  if route == "" {
      route = "unmatched"
  }
  ```

---

### VULN-08: In-Memory Fixed-Window Rate Limiter Unbounded Memory Leak
- **Exact Location**: `pegasusX/apps/backend-go/bootstrap/reliability_middleware.go:461-503`
- **Code Snippet**:
  ```go
  type fixedWindowRateLimiter struct {
      mu      sync.Mutex
      buckets map[string]rateBucket // <--- No eviction, no TTL, unbounded growth!
      limit   int
      window  time.Duration
  }
  ```
- **Flaw Explanation**:
  The default in-memory rate limiter stores rate buckets in a standard Go `map[string]rateBucket`. Keys formatted as `actor:<class>` and `ip:<address>` are inserted on every request and never purged, even after the rate-limiting window has expired.
- **Blast Radius**:
  In dev/staging or during Redis outages where memory rate limiting is active, memory leaks continuously with unique client IP addresses.
- **Remediation**:
  Add an active LRU cache with a maximum capacity (e.g. 50,000 keys) or a periodic background cleanup goroutine that deletes expired window entries.

---

### VULN-09: Token Refresh Rejects Expired Tokens and Duplicates Active Tokens
- **Exact Location**: `pegasusX/apps/backend-go/auth/refresh.go:28-32`, `auth/jwt.go:174-176`
- **Code Snippet**:
  ```go
  claims, err := Parse(raw, secret) // <--- Fails with ErrInvalidToken when expired!
  if err != nil {
      writeErr(w, http.StatusUnauthorized, "invalid_token")
      return
  }
  ```
- **Flaw Explanation**:
  1. `HandleTokenRefresh` parses the Bearer token using `Parse()`, which strictly checks `claims.ExpiresAt < now` and returns `ErrInvalidToken (expired)`. Standard mobile client token refresh patterns (calling `/refresh` upon encountering 401 token expiration) are rejected.
  2. When an active token is refreshed, the existing token's `jti` is not revoked. This allows both the old and new tokens to be used simultaneously, creating multiple valid parallel token branches.
- **Blast Radius**:
  Mobile and web clients cannot silently refresh expired tokens without prompting the user to log in again. In addition, exfiltrated tokens can be refreshed indefinitely without invalidating the original token.
- **Remediation**:
  Implement dedicated refresh tokens (stored securely with `token_use=refresh` and longer TTL), and revoke the previous token ID upon issuing a new pair.

---

### VULN-10: Missing TOTP Replay Prevention Window Vulnerability
- **Exact Location**: `pegasusX/apps/backend-go/mfa/service.go:129-148`, `mfa/totp.go:52-76`
- **Code Snippet**:
  ```go
  func (s *Service) Verify(ctx context.Context, subject, code string) (token string, err error) {
      ...
      ok, err := VerifyCode(rec.Secret, code, s.now())
      if err != nil || !ok {
          return "", ErrInvalidCode
      }
      // <--- Missing check: Has this code/time-step already been used?
      return auth.Issue(claims, ...)
  }
  ```
- **Flaw Explanation**:
  `VerifyCode` calculates valid TOTP codes across a 3-step window (`[-1, 0, +1]`, total 90 seconds). However, the service does not store recently consumed OTP steps. An attacker intercepting an OTP code (via network eavesdropping, shoulder surfing, or compromised transit) can reuse the identical code multiple times within 90 seconds to perform multiple step-ups or mint additional tokens.
- **Blast Radius**:
  Replay vulnerability compromising the non-repudiation and single-use guarantee of multi-factor authentication.
- **Remediation**:
  Store `mfa:used:<subject>:<time_step>` in Redis/memory with a 90-second TTL upon successful verification, rejecting any code corresponding to an already used step.

---

### BUG-11: Background Notification Consumer Leak on Startup Race
- **Exact Location**: `pegasusX/apps/backend-go/main.go:117`, `runtime_workers.go:240-254`
- **Flaw Explanation**:
  `startNotificationConsumerIfNoWorker()` checks Redis worker liveness once during API server initialization. If the API tier boots before the dedicated worker tier publishes its initial heartbeat, the API server launches an in-process Kafka notification consumer in a background goroutine. This goroutine is never stopped when the worker tier starts up.
- **Blast Radius**:
  Both the API pod and worker pods consume the notification topic simultaneously, resulting in duplicate push notifications and SMS messages delivered to drivers and retailers.
- **Remediation**:
  Wrap consumer execution with a periodic heartbeat check so the API pod shuts down its consumer if worker pods become active.

---

### BUG-12: Overwriting Warehouse Metadata on Staff Registration
- **Exact Location**: `pegasusX/apps/backend-go/warehouse/auth_register.go:142-146`
- **Flaw Explanation**:
  When a staff member registers under `HandleWarehouseRegister`, the code appends an `InsertOrUpdateMap("Warehouses", ...)` mutation updating `CountryCode` and `UpdatedAt` on the warehouse entity. Staff registration should only insert into `WarehouseUsers` and never mutate the core `Warehouses` infrastructure entity.
- **Blast Radius**:
  Undesired schema updates and audit trail pollution on warehouse records during employee onboarding.
- **Remediation**:
  Remove the `Warehouses` mutation from `HandleWarehouseRegister`.

---

### BUG-13: Fragile Error Handling in Spanner MFA Repository
- **Exact Location**: `pegasusX/apps/backend-go/mfa/spanner.go:24-27`
- **Flaw Explanation**:
  `Get` compares `if err == spanner.ErrRowNotFound` rather than checking `spanner.ErrCode(err) == codes.NotFound` or `errors.Is`. If the Spanner client wraps the error or returns a gRPC status error, `Get` returns an unexpected error instead of `ok = false`.
- **Blast Radius**:
  Admin users without an MFA record encounter 500 internal server errors instead of a clean `status: unconfigured` response.
- **Remediation**:
  Use `if spanner.ErrCode(err) == codes.NotFound { return Record{}, false, nil }`.

---

### BUG-14: Stateless Staff Invites Lack One-Time-Use Consumption
- **Exact Location**: `pegasusX/apps/backend-go/staffinvite/invite.go:122-163`
- **Flaw Explanation**:
  HMAC staff invites are completely stateless. They contain an expiry timestamp and HMAC signature, but have no nonces or database state. A single invite link sent to an employee can be reused indefinitely until the 7-day expiration window elapses.
- **Blast Radius**:
  Unauthorized multiple registrations if an invite link is shared or forwarded.
- **Remediation**:
  Track invite token hashes in Spanner/Redis with a `Status: CONSUMED` flag upon registration.

---

### BUG-15: Dual-Control Asymmetry in Platform Tenant Lifecycle
- **Exact Location**: `pegasusX/apps/backend-go/platformadmin/service.go:204-228`
- **Flaw Explanation**:
  While tenant approval requires two distinct platform administrators (`RequestedBy != Approver`), destructive transitions to `SUSPENDED` and `OFFBOARDED` can be executed unilaterally by a single administrator.
- **Blast Radius**:
  A single rogue or compromised platform administrator can instantly suspend or offboard active suppliers, halting commerce operations across an entire market without peer review.
- **Remediation**:
  Enforce dual-control or multi-party approval on `SUSPENDED` and `OFFBOARDED` status transitions.

---

### PERF-16: Unindexed Full Scan in Telemetry SLO Collector Polling Loop
- **Exact Location**: `pegasusX/apps/backend-go/telemetry/slo_metrics.go:142-152`
- **Flaw Explanation**:
  `outboxLagP99` runs `SELECT TIMESTAMP_DIFF(PublishedAt, CreatedAt, MILLISECOND) FROM OutboxEvents ... ORDER BY lag_ms DESC LIMIT 100` every 60 seconds without an index supporting computed `lag_ms`, forcing full scan of all events in the last hour.
- **Blast Radius**:
  High Spanner CPU utilization under production event volumes.
- **Remediation**:
  Compute lag in-memory from a bounded sample of recent published events or maintain a sliding latency histogram in the outbox relay.

---

### PERF-17: Nested Retries Duplicate Spanner SDK Internal Retry Loop
- **Exact Location**: `pegasusX/apps/backend-go/spannerutils/retry.go:26-50`
- **Flaw Explanation**:
  Spanner Go SDK's `client.ReadWriteTransaction` already retries aborted transactions. Wrapping it in an outer retry loop without exponential backoff jitter can cause retry amplification.
- **Blast Radius**:
  In high concurrency lock contention, transaction retry storms can degrade Spanner performance.
- **Remediation**:
  Rely on SDK's built-in transaction retry or add jitter to outer retry backoff.

---

### LEAK-18: Spanner ReadOnlyTransaction Session Leak Risk in Context Propagation
- **Exact Location**: `pegasusX/apps/backend-go/spannerutils/retry.go:65-75`
- **Flaw Explanation**:
  Passing `*spanner.ReadOnlyTransaction` through `context.Context` without enforcing explicit `defer txn.Close()` lifecycle management risks leaking Spanner client sessions if caller does not close.
- **Blast Radius**:
  Depletion of Spanner client session pool under error/cancellation conditions.
- **Remediation**:
  Provide scoped helper functions with guaranteed `defer txn.Close()`.

---

## 4. Deep Architectural & Edge-Case Open Questions

1. **Cross-Cell Data Consistency vs. Distributed Outbox Lag**:
   - In a global multi-cell deployment (e.g. `cell-uz`, `cell-eu`, `cell-us`), if a supplier registered in `cell-uz` updates their product catalog or pricing while an EU retailer is browsing, how is catalog state propagated across cells when Spanner instances are per-cell regional instances? Does the outbox relay support cross-cell Kafka topic mirroring, or does cross-market commerce require synchronous RPC with fallback?

2. **Zero-Downtime JWT Secret Rotation**:
   - Currently, JWT verification in `auth/jwt.go` relies on a single static `JWT_SECRET` string. When rotating secrets in production, all existing issued JWT tokens immediately fail verification with `ErrInvalidToken`, abruptly logging out all active drivers, retailers, and warehouse operators. Should the JWT verifier support a list/keyring of valid secrets (`JWT_SECRETS_PREVIOUS`) or a JWKS endpoint with `kid` headers?

3. **Distributed Token Revocation in High-Volume Redis Clusters**:
   - The token revocation mechanism in `auth/revoke_redis.go` stores individual `jti` revocation entries in Redis with TTL. In a multi-region deployment with independent Redis clusters per cell, how are revocation events (e.g. forced admin logout, password reset, or compromised credential) synchronized across cell Redis instances to prevent token reuse in another cell?

4. **Multi-Tenant Spanner Session Pool Contention Under Uneven Tenant Load**:
   - Spanner client sessions are shared across all tenants within a single backend instance. If Tenant A initiates a heavy batch import or complex reporting query, how does the system prevent session pool starvation for latency-critical operations (such as driver dispatch or checkout) belonging to Tenant B? Should priority semaphores be partitioned per tenant tier?

5. **Idempotency Key Collision Across Multi-Instance Retries**:
   - `IdempotencyMiddleware` stores idempotency keys in Redis. If a network partition occurs between the backend and Redis, does the middleware fail-open or fail-closed? Under fail-open, concurrent retries of order creation or payment capture can lead to duplicate payments in the external PSP.

---

## 5. Verification Commands & Regression Testing

To independently verify the findings and ensure zero regression:

```bash
# 1. Run all unit and integration tests across Track 1 packages
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go
go test -v -race ./auth/... ./bootstrap/... ./mfa/... ./platformadmin/... ./featureflags/... ./orgoidc/... ./staffinvite/... ./platform/... ./spannerutils/... ./pkg/... ./telemetry/...

# 2. Verify Schema Drift and Migration Parity (Offline)
go run ./cmd/schema-drift -offline

# 3. Test Contract Generation and Linting
make gen-contracts-gate

# 4. Multi-Tenant Seed & Role Matrix Verification (Requires Spanner Emulator or live instance)
SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/verify-multitenant
```
