# Dispatch Order — Reviewer M4-1 (Roles 5 & 6 Driver Doorstep & Retailer B2B Wholesale)

**Timestamp**: 2026-09-23T06:21:00Z  
**Assigned Subagent**: `reviewer_m4_1` (`teamwork_preview_reviewer`)  
**Working Directory**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_1`  
**Target Monorepo**: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`  
**Orchestrator Conversation ID**: `9c492746-e261-4f02-867a-381f30f56aae`  

---

## Mandatory Reference Documents
- **Original User Request**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md`
- **Approved Specification**: `/Users/shakhzod/.gemini/antigravity-cli/brain/07a686c0-bf89-4aac-8ca4-a8586dc28332/prompt_draft.md`
- **Worker M4 Handoff**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/worker_m4_gen2/handoff.md`
- **Master Plan**: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_10/plan.md`

---

## Review Scope & Objectives
Conduct a rigorous code review and adversarial challenge of Milestone 4:
1. **Role 5 (Driver Doorstep Handshake)**:
   - Dynamic 6-digit numeric OTP and QR token generation & validation in `doorstep_handshake_tokens` table (`internal/doorstep`).
   - Haversine proximity validation ($\le 100\text{m}$) and fallback photo bypass for urban canyon GPS drift.
   - Itemized offload with damaged carton rejection reason codes (`TRANSIT_CRUSH`, `PACKAGE_PUNCTURE`, `EXPIRED_LOT`, `RETAILER_REFUSAL`).
   - Camera lockout: strict rejection of gallery file uploads (`GALLERY`, `GALLERY_UPLOAD`, `DEVICE_STORAGE`), enforcing `CAMERA_DIRECT`.
   - Real-time bilateral tiyin price recalculation and credit note generation.
2. **Role 6 (Retailer Pure B2B Wholesale Scope)**:
   - Verify complete quarantine of in-store grocery POS, cashier shifts, drawer counting, and shelf counting in `internal/retailer`.
   - Verify active B2B wholesale procurement endpoints: tracking, handshake display, doorstep review, payment selection, and Soliq fiscal receipt display.
3. **Execution & Boundary Verification**:
   - Run tests: `go test -count=1 -v -race ./internal/doorstep/... ./internal/epod/... ./internal/retailer/...`
   - Check zero Spanner/Kafka references in target packages.
   - Verify zero mock data or dummy stubs.
4. **Handoff & Verdict**:
   - Write `/Users/shakhzod/Desktop/V.O.I.D/.agents/reviewer_m4_1/handoff.md` with explicit verdict: `APPROVE` or `REQUEST_CHANGES`.
   - Send notification to orchestrator via `send_message`.
