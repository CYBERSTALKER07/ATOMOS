# ADR-004: Event triple-lock

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** Six role rows + web + native clients must agree on event names and payloads for WS refresh and Kafka fan-out.

## Decision

Every business event is defined in three places, generated or checked in CI:

1. **Go source of truth:** `apps/backend-go/events/events.go` (`EventType` constants + payload structs)
2. **JSON Schema:** `contracts/events.schema.json` (via `make gen-contracts`)
3. **TypeScript types:** `packages/types` and `@pegasusx/ws-refresh-contract` for portals

Native apps consume generated stubs under `Generated/` per app. Changing an event requires updating all three layers in one change set; `make gen-contracts-gate` fails PRs on drift.

## Consequences

- Outbox emits use typed `events.OrderEvent`, etc. — no ad-hoc string literals in hot paths.
- WS clients subscribe to envelope `type` fields matching schema `enum`.
- Kafka topic split (future) dual-writes from relay; event names remain stable.

## References

- `events/events.go`, `contracts/events.schema.json`
- `.github/workflows/ci.yml` — `gen-contracts-gate`, `parity-contract-full`
