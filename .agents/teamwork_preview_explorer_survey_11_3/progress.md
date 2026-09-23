# Progress — Requirement R3 Real-Time Monotonic Pipeline Audit

Last visited: 2026-09-23T10:52:45Z

## Status
- [x] Initial setup: DISPATCH.md, BRIEFING.md, progress.md initialized
- [x] 1. Two-System Boundary Audit (verified 0 Spanner, 0 Kafka in pegasus.x/backend; clean go.mod)
- [x] 2. Outbox Relay & Redis 7 Streams Audit (`backend/internal/outbox/relay.go`: FOR UPDATE SKIP LOCKED verified, XADD verified, partition key verified, discovered case mismatch between relay lowercase channels and ws hub uppercase channels)
- [x] 3. WebSocket Hub & Monotonic Envelope Audit (`backend/internal/ws/hub.go`: RealtimeEnvelope with seq, dual event_type/type, recentEvents ring buffer verified; discovered raw Broadcast bypass across 6+ packages, single-role replay route /v1/supplier/sync)
- [x] 4. Atomic Outbox Pairing Audit across all 7 Roles:
  - Supplier: UpdateOnboardingStatus is atomic; Product/Topology/Pricing mutations lack outbox entirely.
  - Warehouse Admin: waves/lots/docks are atomic; warehouse registration/onboarding/blind-receiving exhibit separated transaction anti-pattern; wmsops has zero outbox.
  - Payloader & Picker: pickwave and dock repositories are atomic in pgx.Tx; payloader-tablet lacks WebSocket listener.
  - Dispatcher: CommitDispatch and ExecuteBreakdownRescue are atomic in pgx.Tx.
  - Driver: EPOD stops, DVIR inspection, UMP doorstep adjustments are atomic in pgx.Tx.
  - Retailer: retailer/service.go has 39 mutating operations calling broadcastEvent directly with zero outbox events.
  - Finance & Auditor: Cash recon, credit notes, soliq receipts, payment handover/webhooks are atomic in pgx.Tx; payout/service.go has zero outbox events; rebate/consignment exhibit separated transaction anti-pattern.
  - Ignored errors: _ = outbox.Emit found in ump, inventory, order, claims.
- [x] 5. Desktop / Client Real-Time Invalidation Audit:
  - retailer-desktop: ws.tsx parses dual event_type/type, triggers session reconciliation on reconnectEpoch; lacks seq gap detection.
  - warehouse-desktop: fleet-ws-events.ts has parseWsEventType string bug (returns raw JSON string instead of parsed type); sync-invalidate has 0 listeners.
  - supplier-desktop: use-supplier-ws-refresh.ts attempts EventSource SSE to /v1/supplier/events which does not exist in backend.
  - payloader-tablet & telegram-miniapp: lack WebSocket connections entirely.
- [x] 6. Comprehensive Analysis and Handoff generation (completed analysis.md and handoff.md)
