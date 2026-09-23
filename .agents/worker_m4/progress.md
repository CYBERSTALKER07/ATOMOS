# Progress Tracking - Worker M4 (Roles 5 & 6)

Last visited: 2026-09-22T22:16:45Z

## Status
Initializing investigation and baseline testing.

## Task Breakdown
- [ ] 1. Read context files: ORIGINAL_REQUEST.md, prompt_draft.md, PROJECT.md, survey_report.md
- [ ] 2. Investigate existing code in target packages:
  - backend/internal/doorstep/
  - backend/internal/epod/
  - backend/internal/retailer/
  - backend/internal/fleet/
  - backend/internal/api/handlers_fleet_driver.go
  - backend/internal/api/handlers_retailer.go
  - Check database migrations (especially 074 or related doorstep/epod migrations)
- [ ] 3. Create concrete implementation plan
- [ ] 4. Purge mock stubs and unify driver delivery endpoints on PostgreSQL 16
- [ ] 5. Wire dynamic doorstep OTP/QR handshake tokens (100m geofence, dynamic token generator & verification, emergency fallback)
- [ ] 6. Implement itemized offload screen with damaged carton rejection (reason codes, camera lockout, bilateral tiyin recalculation)
- [ ] 7. Implement dual-tender doorstep settlement (cash collection to cash drawer, corporate card/softPOS, Soliq OFD fiscal QR receipt, digital ePoD)
- [ ] 8. Harden retailer package to pure B2B wholesale procurement scope (quarantine consumer grocery POS/cashier shifts)
- [ ] 9. Write and run tests with -race verification
- [ ] 10. Write handoff report and notify orchestrator
