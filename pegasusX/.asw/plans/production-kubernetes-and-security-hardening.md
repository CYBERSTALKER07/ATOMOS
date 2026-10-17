# Production Kubernetes (GKE) and Zero-Trust Security Hardening Plan

## TL;DR
Translate the newly hardened container runtimes, non-root users, and health gates into production-grade Google Kubernetes Engine (GKE) manifests and zero-trust security controls. This plan hardens `infra/k8s/` deployments (`backend-go`, `ai-worker`, `optimizer-core`) with deterministic Pod and Container `securityContext` boundaries, startup probes, pod disruption budgets, horizontal pod autoscalers, and zero-trust `NetworkPolicy` rules isolating the data tier. Additionally, it enforces cryptographic delivery token validation (`LegacyOrderIDFallback=false` for production), GCP Workload Identity bindings, and runs full end-to-end race and smoke checks.

## Objective
Establish a hardened, compliant, and auditable production GKE topology in `infra/k8s/` with zero-trust network policies, non-root execution (UID 10001), high-availability scheduling (anti-affinity & PDBs), and cryptographic proof-of-delivery handoffs with 100% automated test passing.

## Non-goals
- Modifying Cloud Spanner DDL or altering core order lifecycle states.
- Re-architecting Kafka partition schemes or topic retention topologies.
- Deploying live infrastructure or running `kubectl apply` against production clusters without staging gates.
- Breaking local docker-compose developer simulations (`LegacyOrderIDFallback` remains enabled for local dev).

## Discovery
- `infra/k8s/backend-go/deployment.yaml`: Currently runs without a `securityContext` (defaults to root), lacks startup probes, and has no `podAntiAffinity` rules for multi-zone distribution.
- `infra/k8s/ai-worker/deployment.yaml`: Lacks `securityContext`, has no `PodDisruptionBudget`, and has no `HorizontalPodAutoscaler`.
- `infra/k8s/optimizer-core/deployment.yaml`: Lacks `securityContext` and startup probes.
- `infra/k8s/`: Zero `NetworkPolicy` manifests exist; all pods in namespace `pegasusx` currently have unrestricted flat network communication.
- `infra/k8s/serviceaccount.yaml`: Has Workload Identity annotation `iam.gke.io/gcp-service-account: staging-backend@pegasus-503013.iam.gserviceaccount.com`.
- `packages/handoff/engine.go`: Supports `LegacyOrderIDFallback` via environment variable `HANDOFF_LEGACY_ORDER_ID_FALLBACK`. In production overlays, this fallback must be disabled (`false`) to enforce SHA-256 hashed cryptographic delivery tokens.
- `infra/k8s/ingress/ingress.yaml`: Ingress exposes `/v1/ws`, `/v1`, `/partner`, `/healthz`, and `/ready` with BackendConfig timeouts.

## Decisions
- **Decision 1: Zero-Trust Network Isolation**: Implement default-deny ingress `NetworkPolicy` for namespace `pegasusx`, explicitly whitelisting only ingress-to-backend-go, backend-to-optimizer, and backend/worker egress to Spanner, Redis, Kafka, and DNS.
- **Decision 2: Strict Non-Root Pod Security Standards**: Enforce `runAsNonRoot: true`, `runAsUser: 10001`, `runAsGroup: 10001`, `readOnlyRootFilesystem: true`, and `capabilities.drop: [ALL]` across all core container specs matching our Dockerfile user specifications.
- **Decision 3: Fail-Closed Cryptographic Handshakes**: Enforce `HANDOFF_LEGACY_ORDER_ID_FALLBACK="false"` in production and staging K8s ConfigMaps to guarantee delivery tokens cannot be spoofed by guessing order UUIDs.
- **Decision 4: High-Availability Guarantees**: Provide `PodDisruptionBudget` (maxUnavailable: 1) and `HorizontalPodAutoscaler` (min 2, max 10, targetCPU 75%) for all user-facing services.

---

## TODOs

### Wave 1: Production Pod Security Contexts & High Availability

- [x] **Task 1: Harden Deployment Pod Security Contexts and Probes**
  - **What to do**: Update `infra/k8s/backend-go/deployment.yaml`, `infra/k8s/ai-worker/deployment.yaml`, and `infra/k8s/optimizer-core/deployment.yaml` with:
    - Pod `securityContext`: `runAsNonRoot: true`, `runAsUser: 10001`, `runAsGroup: 10001`, `fsGroup: 10001`, `seccompProfile.type: RuntimeDefault`.
    - Container `securityContext`: `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: [ALL]`.
    - Add `emptyDir: {}` volume mount for `/tmp` and cache directories requiring write permissions.
    - Add `startupProbe` to each deployment to prevent cold-start liveness flapping during Spanner/Kafka initialization.
    - Add `podAntiAffinity` (preferredDuringScheduling) across `topology.kubernetes.io/zone` and `kubernetes.io/hostname`.
  - **What not to do**: Do not run containers as root or with `privileged: true`.
  - **Files or directories**:
    - `infra/k8s/backend-go/deployment.yaml`
    - `infra/k8s/ai-worker/deployment.yaml`
    - `infra/k8s/optimizer-core/deployment.yaml`
  - **References**: `apps/backend-go/Dockerfile`, `apps/ai-worker/Dockerfile`, `services/optimizer-core/Dockerfile`
  - **RED**: Manifests lack `securityContext` and startup probes.
  - **GREEN**: All deployments contain hardened securityContext, startup probes, and non-root UID 10001.
  - **Real-surface QA**:
    ```text
    Scenario: Verify K8s deployment security context configuration
    Channel: tmux
    Steps:
      1. grep -A 10 "securityContext:" infra/k8s/backend-go/deployment.yaml
      2. grep -A 10 "startupProbe:" infra/k8s/backend-go/deployment.yaml
    Expected: "runAsNonRoot: true", "runAsUser: 10001", "readOnlyRootFilesystem: true", and "startupProbe" present.
    Evidence: grep output displaying security policies.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Manifests formatted and verified.
  - **Acceptance criteria**: 100% compliance with Kubernetes Restricted Pod Security Standard.
  - **Commit**: YES
    - Message: `feat(k8s): enforce restricted securityContext, startup probes, and zone anti-affinity`
    - Files: `infra/k8s/**/deployment.yaml`
    - Reason: Enterprise production container isolation.

- [x] **Task 2: Define High-Availability Pod Disruption Budgets & Autoscaling**
  - **What to do**: Create and update `PodDisruptionBudget` and `HorizontalPodAutoscaler` manifests for `ai-worker` and `optimizer-core` matching `backend-go`:
    - `infra/k8s/ai-worker/pdb.yaml` (maxUnavailable: 1).
    - `infra/k8s/ai-worker/hpa.yaml` (minReplicas: 2, maxReplicas: 8, target CPU: 80%).
    - `infra/k8s/optimizer-core/pdb.yaml` (maxUnavailable: 1).
    - `infra/k8s/optimizer-core/hpa.yaml` (minReplicas: 2, maxReplicas: 6, target CPU: 75%).
    - Verify `infra/k8s/backend-go/pdb.yaml` and `infra/k8s/backend-go/hpa.yaml`.
  - **What not to do**: Do not allow `maxUnavailable: 0` when `minReplicas: 1` as it blocks cluster node draining.
  - **Files or directories**:
    - `infra/k8s/ai-worker/pdb.yaml`
    - `infra/k8s/ai-worker/hpa.yaml`
    - `infra/k8s/optimizer-core/pdb.yaml`
    - `infra/k8s/optimizer-core/hpa.yaml`
  - **References**: `infra/k8s/backend-go/hpa.yaml`, `infra/k8s/backend-go/pdb.yaml`
  - **RED**: Missing PDB and HPA resources for background workers.
  - **GREEN**: All microservices possess valid PDBs and HPAs.
  - **Real-surface QA**:
    ```text
    Scenario: Verify PDB and HPA manifests
    Channel: tmux
    Steps:
      1. ls infra/k8s/ai-worker/pdb.yaml infra/k8s/optimizer-core/pdb.yaml
    Expected: Manifests present and syntactically valid.
    Evidence: Directory listing.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Valid YAML manifests created.
  - **Acceptance criteria**: High-availability budgets and autoscaling configured across all services.
  - **Commit**: YES
    - Message: `feat(k8s): add pod disruption budgets and horizontal pod autoscalers for workers`
    - Files: `infra/k8s/ai-worker/*`, `infra/k8s/optimizer-core/*`
    - Reason: Production high-availability resilience.

---

### Wave 2: Zero-Trust Kubernetes Network Policies

- [x] **Task 3: Implement Zero-Trust NetworkPolicies**
  - **What to do**: Implement defense-in-depth Kubernetes `NetworkPolicy` manifests under `infra/k8s/network-policies/`:
    - `default-deny-ingress.yaml`: Deny all ingress traffic by default in namespace `pegasusx`.
    - `allow-ingress-to-backend.yaml`: Allow ingress controller (GKE GLBC / Gateway API) to access `backend-go` and `backend-go-ws` on port 8080.
    - `allow-backend-to-optimizer.yaml`: Allow `backend-go` to connect to `optimizer-core` on port 8082; deny all external traffic to optimizer.
    - `allow-backend-and-worker-egress.yaml`: Allow egress to Cloud Spanner (port 443 / 9010), Redis (port 6379), Kafka (port 9092 / 29092), and CoreDNS (port 53 UDP/TCP).
  - **What not to do**: Do not leave egress open to arbitrary external IPs without restrictions.
  - **Files or directories**:
    - `infra/k8s/network-policies/default-deny.yaml`
    - `infra/k8s/network-policies/backend-go-policy.yaml`
    - `infra/k8s/network-policies/optimizer-core-policy.yaml`
    - `infra/k8s/network-policies/ai-worker-policy.yaml`
  - **References**: Kubernetes NetworkPolicy v1 documentation.
  - **RED**: No NetworkPolicies active in `infra/k8s/`.
  - **GREEN**: Complete zero-trust isolation policies defined and verified.
  - **Real-surface QA**:
    ```text
    Scenario: Validate NetworkPolicy YAML syntax
    Channel: tmux
    Steps:
      1. ls -la infra/k8s/network-policies/*.yaml
    Expected: 4 valid NetworkPolicy files.
    Evidence: File listing and YAML syntax validation.
    Cleanup: None.
    ```
  - **Cleanup receipt**: NetworkPolicy files formatted.
  - **Acceptance criteria**: Microservice traffic restricted strictly to least-privilege paths.
  - **Commit**: YES
    - Message: `feat(k8s): implement zero-trust network policies for backend, workers, and data tier`
    - Files: `infra/k8s/network-policies/*.yaml`
    - Reason: Pod-level zero-trust network segmentation.

---

### Wave 3: Cryptographic Handshakes & Security Boundary Enforcement

- [x] **Task 4: Enforce Cryptographic Proof-of-Delivery Tokens in Staging/Prod**
  - **What to do**: In `infra/k8s/backend-go/configmap.yaml` and overlays, set `HANDOFF_LEGACY_ORDER_ID_FALLBACK: "false"` so that production and staging order completion strictly verifies the SHA-256 hashed delivery token rather than allowing plain order UUIDs.
  - **What not to do**: Do not disable `LegacyOrderIDFallback` in local dev compose (`docker-compose.yml`) to preserve rapid developer simulation flows.
  - **Files or directories**:
    - `infra/k8s/backend-go/configmap.yaml`
    - `packages/handoff/engine.go`
    - `packages/handoff/engine_test.go`
  - **References**: `packages/handoff/engine.go`
  - **RED**: `HANDOFF_LEGACY_ORDER_ID_FALLBACK` is either missing or set to `"true"` in `infra/k8s/backend-go/configmap.yaml`.
  - **GREEN**: Explicitly configured as `"false"` in Kubernetes configmaps; verified by unit test `TestEngine_EnforceCryptographicTokenWhenFallbackDisabled`.
  - **Real-surface QA**:
    ```text
    Scenario: Unit test cryptographic delivery token enforcement
    Channel: tmux
    Steps:
      1. cd packages/handoff && go test -v -run TestEngine
    Expected: "PASS", token verification passes with valid minted token and fails with plain order_id.
    Evidence: Test output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Unit tests pass cleanly.
  - **Acceptance criteria**: Cryptographic proof-of-delivery is fail-closed in cloud environments.
  - **Commit**: YES
    - Message: `feat(security): enforce cryptographic proof-of-delivery tokens in k8s config`
    - Files: `infra/k8s/backend-go/configmap.yaml`, `packages/handoff/engine_test.go`
    - Reason: Tamper-proof delivery verification.

- [x] **Task 5: Tenant Registration & Reliability Rate-Limit Audit**
  - **What to do**: Run test audits on `tenantreg` and `reliability_middleware` to ensure multi-tenant boundary checks and admission rate limiters cannot be bypassed by spoofed HTTP headers.
  - **What not to do**: Do not weaken rate-limiting thresholds or remove tenant isolation guards.
  - **Files or directories**:
    - `apps/backend-go/tenantreg/service_test.go`
    - `apps/backend-go/tenantreg/handlers_test.go`
    - `apps/backend-go/bootstrap/reliability_middleware_test.go`
  - **References**: `apps/backend-go/bootstrap/reliability_middleware.go`
  - **RED**: Any failure in tenant boundary or rate limit evasion tests.
  - **GREEN**: 100% pass across tenant and reliability middleware test suites.
  - **Real-surface QA**:
    ```text
    Scenario: Verify tenant boundary and rate limiting tests
    Channel: tmux
    Steps:
      1. cd apps/backend-go && go test -race ./tenantreg/... ./bootstrap -run "TestReliability"
    Expected: "PASS", 0 data races, rate limits properly enforced.
    Evidence: Test output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Tests pass cleanly.
  - **Acceptance criteria**: Tenant boundary isolation and rate limiting confirmed.
  - **Commit**: NO


---

## Parallel Execution Waves

```text
Wave 1 (Security Contexts & High Availability):
  - Task 1: Harden Deployment Pod Security Contexts and Probes (backend-go, ai-worker, optimizer-core)
  - Task 2: Define High-Availability Pod Disruption Budgets & Autoscaling (ai-worker, optimizer-core)

Wave 2 (Zero-Trust Network Policies):
  - Task 3: Implement Zero-Trust NetworkPolicies (default-deny, ingress, egress)

Wave 3 (Cryptographic Handshakes & Security Boundaries):
  - Task 4: Enforce Cryptographic Proof-of-Delivery Tokens in Staging/Prod
  - Task 5: Tenant Registration & Reliability Rate-Limit Audit

Critical Path:
Task 1 & 2 -> Task 3 -> Task 4 & 5 -> Final Verification Wave
```

---

## Dependency Matrix

| Task | Depends on | Blocks | Can parallelize with |
|---|---|---|---|
| **Task 1** (Security Contexts & Probes) | none | Task 3 | Task 2 |
| **Task 2** (PDBs & HPAs) | none | Task 3 | Task 1 |
| **Task 3** (Network Policies) | Task 1, 2 | Final Wave | Task 4 |
| **Task 4** (Crypto Delivery Tokens) | none | Final Wave | Task 1, 2, 3, 5 |
| **Task 5** (Tenant & Rate Limit Audit) | none | Final Wave | Task 4 |

---

## Final Verification Wave

- [x] **1. K8s Manifest Lint & Validation**: Verify all manifests in `infra/k8s/` parse with zero schema errors.
- [x] **2. Full Automated Backend Test Suite**:
  - `cd apps/backend-go && go test -race ./...` (0 failures, 0 races).
  - `cd packages/handoff && go test -v ./...` (0 failures).
- [x] **3. Security Posture Audit**:
  - All deployments verify `runAsNonRoot: true` and non-root UID 10001.
  - All deployments have active `readinessProbe`, `livenessProbe`, and `startupProbe`.
  - NetworkPolicies enforce zero-trust segmentation for namespace `pegasusx`.
  - `HANDOFF_LEGACY_ORDER_ID_FALLBACK` confirmed `"false"` in cloud configmap.
- [x] **4. Git Working Tree Review**:
  - `git status` verifies clean atomic commits with zero uncommitted artifacts.

---

Next: `start-work production-kubernetes-and-security-hardening`
