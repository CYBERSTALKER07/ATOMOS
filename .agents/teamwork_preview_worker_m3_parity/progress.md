# Progress - teamwork_preview_worker_m3_parity

Last visited: 2026-09-25T17:58:15+05:00

## Current Status
- Initialized briefing and progress tracking.
- Investigating requirements, survey analysis, and existing codebase.

## Plan
1. Read ORIGINAL_REQUEST.md, survey_domain_parity.md, DUAL_SYSTEM_ARCHITECTURE_AND_PARITY.md, PEGASUSX_USER_FLOWS.md, and PROJECT.md.
2. Inspect target files:
   - pegasus.x/backend/internal/models/claims.go
   - pegasus.x/backend/internal/api/ (router setup, handlers, DB access, outbox DLQ tables)
   - pegasus.x/apps/field-sales-mobile/src/screens/ProxyOrderScreen.tsx
   - packages/types/ (order status definitions, canonicalizeOrderStatus)
3. Implement RoleFieldSales in claims.go.
4. Align ProxyOrderScreen.tsx payload.
5. Implement POST /v1/cash/payment-legs handler and route.
6. Implement GET & POST /v1/admin/ops/dead-letters (and /replay) handlers and routes.
7. Implement / refine canonicalizeOrderStatus in packages/types.
8. Add Go unit tests for cash payment legs, outbox DLQ inspection/replay, and role validation.
9. Run `go vet ./...`, `go test -v -count=1 ./internal/...`, and `pnpm check-types`.
10. Finalize handoff.md and send completion message to parent.
