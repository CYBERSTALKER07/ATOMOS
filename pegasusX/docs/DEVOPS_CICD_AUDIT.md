# DevOps + CI/CD — audit

**Date:** 2026-08-18  
**Trees:** `V.O.I.D/.github/workflows/` (**canonical**) and `pegasusX/.github/workflows/` (**nested copy, GitHub does not load**).

**Kind:** point-in-time. **Not** a go-live certificate. Class A hosting stays GKE + Artifact Registry — **not** Firebase Hosting, **not** Vercel.

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`INFRA_AUDIT.md`](./INFRA_AUDIT.md)

---

## 0. Verdict

```
VERDICT: PARTIAL
EVIDENCE: .github/workflows/pegasusx-ci.yml vs pegasusX/.github/workflows/ci.yml
DOCS vs CODE: nested ci.yml looks like the full program; GitHub never runs it
NEXT: copy nested-only gates into root; fix retailerapp; gate docker-push on Gate-0. Not terraform apply.
```

---

## 1. Two CI trees

GitHub Actions only loads **repository-root** `.github/workflows/`. Nested file admits it:

`pegasusX/.github/workflows/desktop-windows-build.yml:1-3`

| Tree | Executes? |
|------|-----------|
| `V.O.I.D/.github/workflows/pegasusx-ci.yml` + siblings | **Yes** |
| `pegasusX/.github/workflows/ci.yml` | **No** |

Live pegasusX CI entry: `pegasusx-ci.yml` (paths `pegasusX/**`) + `pegasusx-native-mobile-build.yml` + `pegasusx-desktop-build.yml` + `sandbox-infra.yml` + `pegasusx-docker-push.yml` + `pegasusx-deploy-gke.yml`.

### Go version drift

| Source | Version |
|--------|---------|
| `apps/backend-go/go.mod` | **1.25.0** |
| Dockerfiles | **golang:1.25-bookworm** |
| Live `pegasusx-ci.yml` backend jobs | **1.26.0** (`:36`) |
| Nested `ci.yml` | **1.25** |
| ai-worker job | `go-version-file` → 1.25 |

CI toolchain ≠ images.

---

## 2. What actually runs (root)

| Job | Live |
|-----|------|
| Unit + race subset | `reusable-go-unit.yml` — `go test ./... -short`; race on `auth order payment outbox fxrates idempotency cache` only — **not** `./... -race` |
| Parity + gap-hunter + gen-contracts | `make parity-contract(-full)`, `gap-hunter-gate`, `gen-contracts-gate` |
| K8s | `validate-backend-k8s`, `kubectl kustomize` prod |
| Placeholder images | `scripts/ci_fail_placeholder_images.sh` on prod overlay |
| Cell isolation + no unattended TF | `ci_no_unattended_terraform_apply.sh` |
| golangci-lint | action v8, `pegasusX/.golangci.yml` |
| govulncheck | **`continue-on-error: true`** (`pegasusx-ci.yml`) |
| gitleaks | pinned binary, `--source pegasusX` |
| Desktop vitest | retailer-desktop + three portals + desktop-bridge (**not** admin-portal, **not** payload-terminal) |
| Android compile | separate workflow, 6 apps |
| iOS compile | macos-15 matrix — **retailer path typo** |

**Not on GitHub (only nested `ci.yml`):** Spanner emulator `go test`, `money_path_gate.sh`, `phase1_gate.sh`, `ci_enterprise_gates.sh`, `ci_fail_todo_inject.sh`, `ci_no_mock_control_tower.sh`, blocking govulncheck, admin-portal typecheck.

Sandbox-infra compose is smoke, not the nested emulator integration suite.

---

## 3. Broken retailer iOS cell

Live matrix (`pegasusx-native-mobile-build.yml:94-96`):

- scheme `retailerapp`
- project `.../retailerapp.xcodeproj`

On disk: `apps/retailer-app-ios/retailerapp/retailerapp.xcodeproj`. Same typo in `scripts/ci_ios_apps.sh`. That matrix cell **cannot** succeed.

---

## 4. Deploy (ungated)

| Workflow | Fact |
|----------|------|
| `pegasusx-docker-push.yml` | Push SHA **and `:latest`**. **No `needs:` on pegasusx-ci.** |
| `pegasusx-deploy-gke.yml` | Runs after docker-push success. Dispatch default tag **`latest`**. **No** `environment: layer-b-ops`. kubectl apply + healthz. |
| Cloud Build YAMLs | Manual `gcloud builds submit`; no tests. `cloudbuild-go.yaml` hardcodes `pegasus-503013`. |
| Makefile `terraform-apply` | Interactive, not GHA. `cell-plan` never apply. |

Placeholder-image gate **rejects** `:latest` in prod overlays while docker-push **tags** `:latest`.

Legacy `deploy-production.yml` still `terraform apply -auto-approve` on **`pegasus/`** — pegasusX apply-guard does **not** scan it.

---

## 5. Gaps vs `golang-continuous-integration` skill

| Skill | Live |
|-------|------|
| `./... -race` + shuffle + coverprofile | **Missing** (race subset only) |
| `go mod tidy` diff | **Missing** |
| govulncheck blocking | Soft-fail |
| CodeQL / gosec | **Missing** on pegasusX |
| Dependabot / Renovate | **Missing** at repo root |
| GoReleaser | **Missing** |
| Trivy / SBOM / provenance on images | **Missing** |
| Pin `@latest` | Fail — `govulncheck@latest` |
| Playwright pegasusX | **GONE** (no `playwright.config` under pegasusX) |

---

## 6. How it should be modularized

1. **One** CI SoT: root `pegasusx-ci.yml`. Delete or stub nested `ci.yml` so agents do not treat it as live.
2. Pin Go to `go.mod` (1.25) **or** bump module+Docker together — not CI-only 1.26.
3. `needs: [pegasusx-ci]` (or equivalent) on docker-push; deploy `environment: layer-b-ops`.
4. Fix `retailerapp` scheme/path.
5. Port nested Spanner/money/enterprise jobs with monorepo `working-directory: pegasusX/...`.
6. Add admin-portal + payload-terminal to the desktop vitest job.

Do not add Firebase Hosting or Vercel pipelines for Class A.

---

## 7. Next slice (when asked)

Fix `retailerapp` in the live native workflow + `ci_ios_apps.sh`. Then promote nested money-path job into root. Not Layer B keys.
