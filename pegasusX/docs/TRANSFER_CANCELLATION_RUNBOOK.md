# Transfer Cancellation Runbook (B05)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose
Provide a consistent procedure for factory transfer cancellation so operational risk is contained and auditability is preserved.

## Scope
- Manifest read and lifecycle context:
  - `GET /v1/factory/manifests`
  - `GET /v1/factory/manifests/{manifestID}`
- Cancellation actions:
  - `POST /v1/factory/manifests/cancel-transfer`
  - `POST /v1/factory/manifests/cancel`
- Related exception context:
  - `GET /v1/factory/manifest-exceptions`
  - `POST /v1/payload/manifest-exception`

## Ownership
1. First response: factory operations support.
2. Co-owners: payload and warehouse ops when transfer cancellation impacts active loading/offloading.
3. Engineering escalation: backend node-operations team for cancellation-state inconsistency.

## Cancellation decision checklist
1. Confirm cancellation reason category:
   - inventory unavailability
   - vehicle/driver unavailability
   - safety/compliance hold
   - route or destination invalidation
2. Verify current manifest lifecycle state using manifest read endpoints.
3. Determine action path:
   - `cancel-transfer` for transfer-specific unwind.
   - `cancel` for full manifest cancellation.
4. Execute cancellation with stable request payload and record timestamp.
5. Verify post-cancel state through manifest reads and exception queue.

## Failure handling matrix

| Surface | Typical status/code | Meaning | Support action |
|---|---|---|---|
| Cancel transfer | `400 invalid_json` or validation error | malformed cancellation request | correct payload and retry |
| Cancel transfer | `409`-class conflict | lifecycle state changed before cancel commit | refresh manifest state and retry once |
| Cancel transfer | `422`-class invalid state | cancellation not allowed in current phase | choose correct action or escalate to rebalance path |
| Cancel manifest | `403 forbidden` | scope mismatch for factory/home node | re-authenticate with correct role scope |
| Cancel manifest | `5xx` | persistence/eventing path failed | escalate immediately with evidence packet |

## Escalation triggers
1. Cancellation success response does not persist on subsequent manifest reads.
2. Repeated cancellation attempts alternate between conflict and success without stable terminal state.
3. Downstream payload/warehouse surfaces continue executing a canceled transfer.
4. Exception queue cannot be reconciled with cancellation outcome.

## Evidence package
1. Manifest_id, supplier_id, related route/vehicle/driver ids (if present).
2. Cancellation endpoint used and payload snippet.
3. HTTP status/body and UTC timestamps for each attempt.
4. Linked exception ids and downstream impact notes.
5. Operator identity and role context.
