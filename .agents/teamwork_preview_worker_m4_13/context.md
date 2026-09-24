# Worker M4 Context: Comprehensive Verification & Zero-Regression Assurance

Working Directory: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_m4_13`
Target Workspace Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
Authoritative Requirements: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
Project Plan: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/PROJECT.md`
Gate Status: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_13/GATE_STATUS.md`

Mission & Tasks (Task 9 of modularization-plan.md & Requirement R4):
Execute the complete, end-to-end full-stack verification matrix across backend, frontend, and infrastructure:

1. **Backend Verification**:
   - Run `go vet ./...` in `backend/` -> confirm 0 diagnostics.
   - Run `go test -v -race ./...` (or comprehensive package test suites with `-race`) in `backend/` -> confirm 100% pass, 0 failures, 0 race conditions.
   - Verify `backend/internal/api/router.go` line count (confirm <950 lines).
   - Verify Sovereign Core constraints:
     - `grep -rnI "spanner" backend/` (excluding comments/docs/git if any, verify 0 imports)
     - `grep -rnI "kafka" backend/` (verify 0 imports)
     - `grep -rnI "float" backend/internal/retailer/` or pricing/currency (confirm int64 tiyins)

2. **Frontend Monorepo Verification**:
   - Run `pnpm --filter @pegasusx/types build` -> confirm 0 errors.
   - Run `pnpm --filter @pegasusx/pulse-ui build` -> confirm 0 errors.
   - Run `pnpm --filter @pegasusx/ui-kit build` -> confirm 0 errors.
   - Run `pnpm --filter @pegasusx/warehouse-desktop build` -> confirm clean compilation.
   - Run `pnpm test` -> confirm 100% tests pass.
   - Run `grep -rn "firebase" apps/warehouse-desktop/package.json` -> confirm 0 matches.

3. **Infrastructure Verification**:
   - Run `docker compose config`
   - Run `docker compose -f docker-compose.base.yml -f docker-compose.dev.yml config`
   - Run `docker compose -f docker-compose.base.yml -f docker-compose.prod.yml config`
   - Run `docker compose -f docker-compose.prod.yml config`
   - Confirm all exit 0 with 0 warnings.

4. **Compile Comprehensive Verification Artifact**:
   - Document all command outputs, exit codes, test counts, execution times, and line counts in `handoff.md` in your working directory.
   - Notify parent via `send_message`.
