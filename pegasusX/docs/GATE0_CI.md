# Gate-0 CI for pegasusX (lint / race / secrets / 12 native apps).
# Live workflows (monorepo root `.github/workflows/`):
# - `pegasusx-ci.yml` — backend build/vet/test/race, golangci-lint, govulncheck, gitleaks, desktop portals
# - `pegasusx-native-mobile-build.yml` — Android ×6 + iOS ×6 compile matrices
#
# Nested `pegasusX/.github/workflows/ci.yml` is a mirror for local/docs; GitHub Actions
# only loads workflows from the repository root `.github/workflows/`.

## Local helpers

```bash
# Android (all 6)
bash pegasusX/scripts/ci_android_apps.sh

# iOS (all 6; requires Xcode + optional xcodegen)
bash pegasusX/scripts/ci_ios_apps.sh

# golangci (needs v2.12+ on Go 1.26 hosts)
cd pegasusX/apps/backend-go
golangci-lint run --config=../../.golangci.yml ./...

# secrets
gitleaks detect --source pegasusX --config pegasusX/.gitleaks.toml
```

## Residual

- ESLint + jsx-a11y hard gate not yet added (desktop typecheck/vitest already run).
- `govulncheck` may soft-fail until vulns are triaged (`continue-on-error` on that step).
- Driver iOS local package path fixed to `../../../packages/mobile-ios-kit`.
