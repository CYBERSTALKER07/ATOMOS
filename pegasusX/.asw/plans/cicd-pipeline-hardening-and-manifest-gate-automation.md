# CI/CD Pipeline Hardening & Manifest Gate Automation Plan

## TL;DR
Automate and harden continuous integration, security vulnerability gates, and Kubernetes manifest validation across the entire PegasusX platform. This implementation introduces an ultra-fast universal Go-based offline Kubernetes manifest validator (`apps/backend-go/cmd/validate-k8s`), integrates automated Python and Rust testing for `services/optimizer-core` in CI (`scripts/ci_optimizer_core.sh`), adds Trivy container and IaC misconfiguration scanning, modernizes brittle line-numbered gates (`scripts/validate_spanner_stale_reads.sh`, `scripts/assert_cell_backend.sh`, `scripts/cell_isolation_proof.sh`, `scripts/validate_production_profile.sh`), and mirrors the full PegasusX CI pipeline to the repository root at `.github/workflows/pegasusx-ci.yml`.

## Objective
Establish an enterprise-grade, fail-fast CI/CD pipeline that guards all 71+ Kubernetes manifests, verifies zero-trust network policies and Pod Security Standards, tests Python and Rust optimization microservices alongside Go backend services, and runs container security vulnerability scans before any artifact is deployed to staging or production.

## Non-goals
- Modifying production Spanner database schemas or business domain tables.
- Introducing required external third-party paid SaaS monitoring or scanning services.
- Altering the VRP solver algorithmic models or order transition state graphs.

---

## TODOs

### Wave 1: Brittle Gate Modernization & Infrastructure Stability
- [x] **Task 1: Production Profile Replicas Alignment**
  - Updated `infra/k8s/backend-go/deployment.yaml` (`replicas: 2`) to satisfy HA, PDB (`minAvailable: 1`), and `scripts/validate_production_profile.sh`.
- [x] **Task 2: Cell Backend & Isolation Proof Dynamic Path Resolution**
  - Updated `scripts/assert_cell_backend.sh` and `scripts/cell_isolation_proof.sh` to resolve terraform paths dynamically across `$TF` and `$TF/legacy/`.
  - Restored `infra/terraform/README.md` documenting cell isolation rules.
- [x] **Task 3: Spanner Stale Read Allowlist Modernization**
  - Upgraded `scripts/validate_spanner_stale_reads.sh` with whole-file allowlisting and `--update` flag.
  - Refreshed `spanner_stale_read_allowlist.txt`.

### Wave 2: Universal Kubernetes Manifest & Overlay Validator
- [x] **Task 4: Universal Go K8s Validator (`cmd/validate-k8s`)**
  - Built `apps/backend-go/cmd/validate-k8s/main.go` using `gopkg.in/yaml.v3`.
  - Parses all 71 manifests and 104 Kubernetes documents.
  - Audits Pod Security Standards (`runAsNonRoot: true`, `allowPrivilegeEscalation: false`, resource requests/limits, health probes).
  - Validates all 8 Kustomize overlays (`dev`, `pilot`, `prod`, `sandbox`, `ssmr`, `staging`, `cells/eu`, `cells/uz`).
  - Audits `prod` overlay against placeholder and `:latest` images.
- [x] **Task 5: Shell Wrapper & Makefile Integration**
  - Created `scripts/validate_all_k8s.sh`.
  - Added `validate-all-k8s` target to `Makefile` and wired into `wire-ready`.

### Wave 3: Microservice CI & Security Gates
- [x] **Task 6: Optimizer-Core Automated Test Suite**
  - Created `scripts/ci_optimizer_core.sh`.
  - Runs Python pytest suite (`test_observability.py`, `test_contract_solver.py`, `test_certification_harness.py` — 10 passed).
  - Runs Rust solver test suite (`services/optimizer-core/server-rust` — 4 passed).
- [x] **Task 7: GitHub Actions CI Hardening & Trivy Integration**
  - Updated `pegasusX/.github/workflows/ci.yml` with `validate-all-k8s`, `optimizer-core`, and `trivy-security-audit`.
  - Fixed `golangci-lint` version to `v1.64.6`.
  - Created repository-root `.github/workflows/pegasusx-ci.yml` triggering on pushes and pull requests affecting `pegasusX/**`.

---

## Final Verification Wave
- [x] `make validate-all-k8s`: 71 manifests, 104 documents, 8 overlays, 0 errors.
- [x] `bash scripts/validate_production_profile.sh`: PASS.
- [x] `bash scripts/assert_cell_backend.sh`: PASS.
- [x] `bash scripts/cell_isolation_proof.sh`: PASS.
- [x] `bash scripts/validate_spanner_stale_reads.sh`: PASS.
- [x] `bash scripts/ci_optimizer_core.sh`: PASS (10 Python tests, 4 Rust tests).
- [x] Workflow YAML syntax validation: PASS.
