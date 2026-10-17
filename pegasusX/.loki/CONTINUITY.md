# Continuity

## Active

- Retailer confirm-ai, reject-ai, edit-preorder, confirm-preorder, accept-delivery-proposal, reject-delivery-proposal, and reject-preorder require `Idempotency-Key` before the order-lifecycle availability check. Android, iOS, and desktop send those keys.
- Retailer profile update (`PUT /v1/retailer/profile`) and favorite supplier add/remove (`POST /v1/retailer/suppliers/{supplierID}/add|remove`) enforce strict idempotency and are covered in `mutation_idempotency_test.go`.
- Driver rescue response parity resolved across platforms:
  - iOS `APIClient.respondRescue`, `FleetService`, and `FleetViewModel` now pass `broken_driver_id`.
  - Android `DriverWebSocket`, `ManifestViewModel`, and `HomeScreen` now receive `RESCUE_PROPOSED` and display `RescueAcceptanceDialog`.
  - Warehouse dispatch rescue (`POST /v1/warehouse/ops/dispatch/rescue/propose`) fans out `RESCUE_PROPOSED` to the target driver's WebSocket room (`driver:<rescueDriverID>`) post-commit.
- Strict idempotency enforced with client parity and test suites across all remaining mutation surfaces:
  - `POST /v1/delivery/report-damage`: flipped to `guardIdempotencyStrict`. `ssmr-smokecheck` and `ecosystem-simulator` pass `driver-report-damage:<driverId>:<orderId>`. Covered in `driver_edge_idempotency_test.go`.
  - `POST /v1/delivery/report-condition`: flipped to `guardIdempotencyStrict`. Deterministic helpers added to Android `DriverIdempotencyKeys.kt`, iOS `DriverIdempotency.swift`, and TypeScript `api-core`/`api-client`. Covered in `driver_edge_idempotency_test.go`.
  - `POST /v1/delivery/negotiate` & `POST /v1/supplier/negotiate/resolve`: flipped to `guardIdempotencyStrict` and reordered before Spanner check. `ssmr-smokecheck` updated. Covered in `TestNegotiationMutations_RequireIdempotencyKey`.
  - `POST /v1/supplier/reconciliation/resolve`: flipped to `guardIdempotencyStrict`. Covered in `driver_edge_idempotency_test.go`.
  - `POST /v1/claims/{claimID}/reject`: flipped to `guardIdempotencyStrict`. `claimApproveKey` and `claimRejectKey` helpers added to `api-client`/`api-core`. Covered in `claims/handlers_idempotency_test.go`.
- Antimater web app:
  - Cleaned up empty unrouted directories `app/{about,pricing,privacy,terms}` that produced broken Next.js type definitions.
  - Verified `npx tsc --noEmit` passes with 0 errors.
  - Audited contact form in-memory fallback for unconfigured Supabase deployments.

Verified:
- `go test ./apps/backend-go/... -count=1` -> PASS across all packages (order, driver, retailer, warehouse, claims, cmd, kafka, etc.)
- `npx tsc --noEmit` in `pegasusX/antimater` -> PASS (0 errors)

## Lessons

- Do not use a replayed idempotency key on quantity-increment endpoints such as load-ledger scan. A stable key would freeze the scanned count.
- Flip a handler to strict only after every shipped client for that route sends a key. Update the handler tests in the same change.
- Do not edit Go with bulk regex. Unique function context keeps neighboring soft guards intact.
- When emitting events for peer drivers, fan out via the WebSocket hub (`driver:<id>`) post-commit in addition to transactional outbox publishing.
- Always check auth claims and guard idempotency (`readLimitedBody` + `guardIdempotencyStrict`) BEFORE checking Spanner or service nil-guards, so requests without headers consistently receive 400 Bad Request `idempotency_key_required`.

## Still open

- Load-ledger scan stays incremental (deliberate design choice to allow multi-scan increments).
- Real-time simulator stress tests under heavy concurrency.
