# Reassignment Support Playbook (B05)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose
Standardize payload/factory reassignment handling so exception recovery stays auditable and role-consistent.

## Scope
- Payload recommendation and apply:
  - `POST /v1/payloader/recommend-reassign`
  - `POST /v1/payloader/reassign-order`
- Factory rebalance operations:
  - `POST /v1/factory/manifests/rebalance`
- Exception reporting and review:
  - `POST /v1/payload/manifest-exception`
  - `GET /v1/payloader/manifest-exceptions`
  - `GET /v1/factory/manifest-exceptions`

## Ownership
1. First response: payload operations support.
2. Escalation: factory operations support for rebalance/cancel coordination.
3. Backend escalation: node-operations engineering when reassignment persistence or fanout appears inconsistent.

## Reassignment flow checklist
1. Confirm incident context: manifest_id, order_id, source route, target route (if known), UTC timestamp.
2. Request recommendation with `POST /v1/payloader/recommend-reassign`.
3. Validate recommendation viability against current manifest/exception state.
4. Apply reassignment with `POST /v1/payloader/reassign-order`.
5. Confirm outcome through exception feeds and manifest reads.
6. If cross-manifest balancing is still required, execute `POST /v1/factory/manifests/rebalance` with coordinated ownership.

## Failure handling matrix

| Surface | Typical status/code | Meaning | Support action |
|---|---|---|---|
| Recommend reassign | `400 invalid_json` or validation error | malformed recommendation request | correct payload and retry |
| Recommend reassign | `403 forbidden` | payload scope mismatch | retry under correct payload/factory scope |
| Reassign apply | `409`-class conflict | order/manifest state changed during operation | refresh current manifest state and retry once |
| Reassign apply | `422`-class validation | target/source pairing is not allowed | adjust target and retry |
| Factory rebalance | `400` or `422` | rebalance request invalid for current lifecycle state | inspect manifest status and rerun with valid phase |
| Exception read/write | `5xx` | exception pipeline unavailable | capture evidence and escalate immediately |

## Escalation triggers
1. Reassignment apply reports success but source/target state does not converge on subsequent reads.
2. Same order oscillates between manifests/routes after repeated reassignment attempts.
3. Exception queues accumulate unresolved duplicates for a single order/manifest pair.
4. Factory rebalance and payload reassignment produce contradictory ownership outcomes.

## Evidence package
1. Order_id, source manifest_id, target manifest_id (if present), route metadata.
2. Sequence of endpoints invoked with request/response payload snippets.
3. UTC timestamps and operator ids.
4. Any exception ids generated during the incident.
5. Scope context (payload/factory role and home node).
