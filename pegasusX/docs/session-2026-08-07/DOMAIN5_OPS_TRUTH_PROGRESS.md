# Domain 5 — Tenancy & Ops Truth (P1) · Progress

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


Date: 2026-08-11 · Roadmap ref: `/Users/shakhzod/.cursor/plans/Ecosystem Capability Roadmap-7cbf327a.plan.md` (Domain 5).

## 5.1 Admin console UI + dual-control money flags

**Backend — dual-control feature flags.**
- Migration `schema/migrations/20260811_feature_flag_dual_control.ddl` adds
  `Status`, `ApprovedBy`, `ApprovedAt` to `FeatureFlagOverrides`; the original
  `20260811_platform_admin_flags.ddl` and `schema/spanner.ddl` were brought in
  line so fresh emulator setups get the columns.
- `featureflags/service.go` — money-affecting overrides are stored `PENDING`
  (with a required `Reason`) and only honored by `Evaluate` once `ACTIVE`.
  `ApproveOverride` activates a pending override **only** when the approver is a
  different `PLATFORM_ADMIN` than the setter (self-approval rejected).
- `featureflags/spanner.go` persists/reads the three new columns.
- `featureflags/handlers.go` — `HandleSetOverride` now returns the override
  `status`; new `POST /v1/platform-admin/flags/{flagKey}/approve` route.
- `featureflags/service_test.go` — `TestMoneyFlagDualControl` covers PENDING →
  self-approve rejected → cross-admin approve → ACTIVE.

**Frontend — real console replacing the redirect stub.**
- `apps/admin-portal` is a real Next.js 15 / React 19 / TS / Tailwind app
  (removed the old `redirect.mjs` stub; added to `pnpm-workspace.yaml`).
- `lib/api.ts` typed client for the `PLATFORM_ADMIN` surface; `lib/session.tsx`
  in-memory bearer-token session (break-glass governance identity, never
  persisted).
- `components/TenantsPanel.tsx` — list/filter `PlatformTenants`, run lifecycle
  transitions with optional KYB notes.
- `components/FlagsPanel.tsx` — evaluate + set overrides; money flags surface the
  PENDING/dual-control flow with a separate "Approve (2nd admin)" action.
- `components/AuditPanel.tsx` — reads `PlatformAdminAudit`.
- `pnpm build` passes (type-check + lint clean).

## 5.2 SLO burn alerts wired to live data

- The Terraform policies in `infra/terraform/observability.tf` (outbox lag p99
  > 30s, fiscal success < 99%, capture success < 99%) filter on
  `void_outbox_lag_seconds` / `void_fiscal_success_ratio` /
  `void_capture_success_ratio`, but **nothing emitted those metrics**.
- New `telemetry/slo_metrics.go` — `SLOCollector` polls Spanner every 60s and
  exposes the three gauges into the default Prometheus registry (scraped at
  `/metrics`, already mounted by `infraroutes`). No-traffic windows report
  ratio = 1.0 (no false breach).
- `main.go` starts the collector when `app.Spanner != nil`.
- `telemetry/slo_metrics_test.go` — verifies all three metrics are emitted and
  the no-traffic ratio is 1.0, against the Spanner emulator. Passing.

## Gate / verification

- `go build ./...` (backend) — clean.
- `go vet ./telemetry/` — clean.
- `go test ./featureflags/ ./telemetry/` — pass.
- `pnpm build` in `apps/admin-portal` — pass.

## Remaining in Domain 5

- None blocking; SLO alert *routing* (notification channels) is configured via
  `var.alert_notification_channels` at `terraform apply`, not code.
