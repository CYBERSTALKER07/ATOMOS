# Logistics Exception Implementation

## Goal
Implement a robust logistics exception management system including Spanner schema for Claims, Kafka topics for telemetry/exceptions, and Go backend logic for post-delivery claims and visual proofs.

## Tasks
- [x] Task 1: Apply Spanner schemas for `Claims` and `ClaimEvidences` tables → Verify: Run `gcloud spanner databases ddl describe` to see the tables.
- [x] Task 2: Provision new Kafka topics `logistics.exceptions.v1` and `logistics.telemetry.v1` via Strimzi → Verify: `kubectl get kafkatopics -n pegasus-staging` shows the topics as Ready.
- [x] Task 3: Create `apps/backend-go/claims` package with repository and structs → Verify: `go test ./claims/...` builds successfully.
- [x] Task 4: Implement Retailer Claim API (`POST /v1/orders/{id}/claims`) → Verify: `curl` endpoint returns 201 Created and inserts into Spanner.
- [x] Task 5: Modify `order/amend.go` to support `DAMAGED` reasons and post-delivery time-gates → Verify: Go unit tests in `amend_test.go` pass for `StatusCompleted`.
- [x] Task 6: Refactor `order/driver_edges.go:HandleMissingItems` to `HandleExceptionReport` (add photo proof support) → Verify: Uploading exception with valid image URL emits `REVERSE_LOGISTICS_REQUIRED` event.

## Done When
- [ ] Retailers can file damage claims post-delivery.
- [ ] Drivers can upload visual proof of OS&D during delivery handshakes.
- [ ] All exception events flow through the new Kafka topics.
