# Original User Request

## Initial Request — 2026-08-29T06:40:48Z

# Teamwork Project Prompt — Draft

> Status: Launched
> Goal: Craft prompt → get user approval → delegate to teamwork_preview
> Requested team: Full multi-agent team

Implement the architectural fixes and system constraints finalized in `pegasusx_master_architecture_design.md` and `pegasusx_decision_log.md` across the PegasusX codebase.

Working directory: /Users/shakhzod/Desktop/V.O.I.D/pegasusX
Integrity mode: development

## Requirements

### R1. Kafka DLQ UI Transparency
Update the Kafka consumer logic (`apps/backend-go/kafka/consumer.go`) so that messages routed to the DLQ update their corresponding Spanner job status to `FAILED`. Update the Supplier UI to explicitly display these failures. Note: A schema migration to add SupplierId to OutboxDeadLetters was already created.

### R2. SSE Migration
Replace the existing Supplier WebSocket implementation with lightweight Server-Sent Events (SSE) in the Go backend and the Supplier frontend (web/native).

### R3. Frontend Geospatial Validation
Integrate logic (e.g., Turf.js) into the Mapbox Draw UI to auto-simplify drawn delivery polygons to <50 vertices and prevent self-intersecting shapes from being submitted to the backend.

### R4. FCM Stale Token Race Condition Fix
Update the `purgeStaleToken` logic in `apps/backend-go/notifications/fcm.go` (and the `DeviceTokens` schema if necessary) to explicitly match tokens to the specific device session, preventing the blind deletion of newly issued tokens during rapid re-installs.

## Acceptance Criteria

### Kafka DLQ Transparency
- [ ] A programmatic test or script proves that causing a consumer panic 4 times successfully writes a `FAILED` status to the relevant Spanner table.

### SSE Migration
- [ ] A `curl` command against the new SSE endpoint successfully keeps the connection open and receives pushed events.

### Geospatial Validation
- [ ] The frontend codebase contains explicit validation logic that intercepts polygon submission and throws a client-side error (or auto-simplifies) if vertices > 50 or if lines intersect.

### FCM Token Fix
- [ ] A Go unit/integration test verifies that executing `purgeStaleToken` on an expired session does *not* delete a newer token registration for the same user.
