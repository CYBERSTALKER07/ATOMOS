# Task Plan: PegasusX B2B Operating Model Alignment & Implementation Plan

<!-- 
  WHAT: Working memory on disk for executing the PegasusX B2B Operating Model.
  TARGET: /Users/shakhzod/Desktop/PegasusX-B2B-Operating-Model.md
  SCOPE: End-to-end alignment across backend-go, optimizer-core, Spanner, portals, and mobile apps.
-->

## Goal
Map, audit, plan, and verify the complete operational architecture of the **PegasusX B2B Operating Model** across all 6 daily rhythm stages, 8 decision rights, 7 retailer UX principles, and 51 cross-surface edge cases in `/Users/shakhzod/Desktop/pegasusx`, establishing the zero-touch automated distribution engine where >=95% of orders execute with zero supplier human touch.

## Current Phase
All Phases Complete

## Phases

### Phase 1: Requirements & Ground Truth Code Gap Analysis
- [x] Ingest all 6 core sections of `PegasusX-B2B-Operating-Model.md`
- [x] Map the 6 daily rhythm stages (Retailer Cutoff, Supplier Overnight, Warehouse Morning, Doorstep Delivery, Depot Blind Vault, Supplier Exception Desk) to backend Go routes, Spanner DDL, and frontend/mobile apps
- [x] Audit the 8 Decision Rights and their automated escalation rules against `foundry/pre_execution.go` and `foundry/edge_guards.go`
- [x] Audit the 51 specific edge cases across Ordering (10), Credit/Collections (10), Warehouse/Truck (12), Doorstep (9), After Delivery (6), and Upstream (4)
- [x] Document all findings, current code coverage, and gap matrix in `findings.md`
- **Status:** complete

### Phase 2: Autonomous Ordering & Retailer Experience Architecture
- [x] Specify and verify Retailer "One Home Screen" model: tomorrow's delivery, debt owed, and proposed replenishment order
- [x] Specify Owner vs Seller mandate permissions: daily cash/credit cap, price negotiation block, owner-phone push authorization
- [x] Specify Pack Size Snapping logic: enforcing case multiples at cart/proposal stage with zero driver split-case capability
- [x] Specify Split-Tender Checkout & Doorstep Recalculation: cash + credit + card waterfall with oldest-first dunning
- [x] Specify Offline-First QR Manifest Handshake: offline cryptographic QR minted on manifest, zero duplicate submissions
- [x] Update `task_plan.md`, `findings.md`, and `progress.md`
- **Status:** complete

### Phase 3: Overnight Autonomous Cascade & Optimizer Integration
- [x] Detail Cutoff Clock & Auto-Confirm Engine: standing rules, silent expiration, and next-wave rollover
- [x] Specify Silent Stock Reservation & Cut/Substitute Engine: family substitution allow-lists, must-have line cuts vs splits
- [x] Align CVRPTW Solver with dynamic constraints: receiving windows, volume/weight axle capacities, multi-wave spillover
- [x] Specify Warehouse Short-Pick notification pipeline: pre-seal resolution and automatic shop alerts before truck roll
- [x] Update `task_plan.md`, `findings.md`, and `progress.md`
- **Status:** complete

### Phase 4: Doorstep Execution, Dispute Tribunal & Blind Vault Ops
- [x] Specify Shop Closed Protocol: driver photo/location capture, multi-channel alert (SMS/Push/Telegram), credit-leave fallback vs truck return
- [x] Specify Doorstep Count & Split-Tender Partial Rejection: accepted = legal invoice, refused = return manifest, instant tier recalculation
- [x] Specify Damaged/Crushed Case Protocol: photo attachment, auto-credit note under money threshold, exception queue escalation above threshold
- [x] Specify Depot Dual-Blind Cash Bag Ingestion: hidden expected total, blind count ratchet, RRN card reclassification, CIT handover
- [x] Update `task_plan.md`, `findings.md`, and `progress.md`
- **Status:** complete

### Phase 5: Exception Desk, 1C Silent Posting & Governance Limits
- [x] Specify the 15-Minute Supplier Exception Desk: single unified queue strictly sorted by Money-at-Risk
- [x] Codify the 6 Immutable Human Actions: credit limit raise outside policy, write-off above floor, recall release, Soliq skip stamp, broken cash seal, new legal entity approval
- [x] Specify Silent 1C Accounting Integration: automated journal posting, rejected document queue, month-end exception ledger
- [x] Specify Dispute & Concealed Damage Tribunal: POS register sales reconciliation, photo uniqueness hash, claims abuse penalty
- [x] Update `task_plan.md`, `findings.md`, and `progress.md`
- **Status:** complete

### Phase 6: Verification Strategy, Test Matrix & Staging Rollout
- [x] Define comprehensive integration test plan across all 51 edge cases
- [x] Run and document verification gates across Go backend, Python optimizer, and web/mobile surfaces
- [x] Define the 4-Phase Transition Rollout Roadmap (Digital Ordering -> Proposed Orders -> Automated Scripts -> Silent 1C)
- [x] Final review, signoff, and artifact persistence
- **Status:** complete

---

## Errors Encountered & Resolutions
| Error / Gap | Phase | Attempt | Resolution |
| :--- | :--- | :--- | :--- |
| `pytest: command not found` in direct bash invocation | Phase 6 | 1 | Used canonical CI script `scripts/ci_optimizer_core.sh` which executes inside the dedicated virtual environment `.venv/bin/pytest`. |
| Optimizer test execution timeout default | Phase 6 | 1 | Allowed complete run of 21 Python certification tests and 7 Rust unit tests; all 28 passed cleanly in 26.45s. |
