# Warehouse Exception SOP (B05)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose
Provide a repeatable support procedure for warehouse-side operational exceptions in pegasusX.

## Scope
- Warehouse operations reads: `GET /v1/warehouse/ops/dashboard`, `GET /v1/warehouse/ops/orders`, `GET /v1/warehouse/ops/inventory`
- Dispatch planning: `POST /v1/warehouse/ops/dispatch/preview`
- Supply request handling: `GET|POST /v1/warehouse/supply-requests`
- Dispatch lock control: `GET /v1/warehouse/dispatch-locks`, `POST|DELETE /v1/warehouse/dispatch-lock`

## Ownership
1. First response: warehouse support operations.
2. Escalation: backend node-operations team for scope, persistence, or realtime lock inconsistencies.
3. Advisory stakeholders: supplier dispatch lead and factory operations lead.

## Exception triage checklist
1. Capture identifiers: warehouse/home-node context, supplier_id, affected order_id or request_id, UTC timestamp.
2. Validate visibility baseline:
   - `GET /v1/warehouse/ops/orders`
   - `GET /v1/warehouse/ops/inventory`
   - `GET /v1/warehouse/dispatch-locks`
3. Validate intent path:
   - `POST /v1/warehouse/ops/dispatch/preview` for preview integrity.
   - `GET|POST /v1/warehouse/supply-requests` for replenishment request state.
4. If concurrent operator conflict is suspected, verify lock behavior before mutating state.
5. Route unresolved physical exceptions to factory or payload playbooks with full evidence packet.

## Failure handling matrix

| Surface | Typical status/code | Meaning | Support action |
|---|---|---|---|
| Dispatch preview | `400 invalid_json` or validation error | malformed/invalid preview request | correct payload and retry |
| Dispatch preview | `403 forbidden` | warehouse scope mismatch | re-authenticate with correct warehouse context |
| Dispatch preview | `503` class unavailability | temporary backend/load condition | retry with bounded interval, escalate if persistent |
| Supply requests | `400 invalid_json` | malformed request | correct payload and retry |
| Supply requests | `403 forbidden` | node/supplier scope mismatch | validate authenticated scope and retry |
| Dispatch lock acquire | `409`-class contention | another operator or process holds lock | coordinate release owner, do not force conflicting write |
| Dispatch lock release | `404`-class missing lock | lock already released/expired | refresh state and proceed if no conflict |

## Immediate escalation triggers
1. Warehouse cannot read orders/inventory while supplier scope is valid.
2. Dispatch lock remains stuck beyond expected operator activity window.
3. Preview output diverges repeatedly from current order/driver reality after refresh.
4. Supply-request writes succeed but state cannot be read back.

## Evidence package
1. Supplier_id, warehouse/home-node identifier, and affected order/request ids.
2. Endpoint + method sequence used during incident.
3. HTTP status and response body for each failed step.
4. UTC timestamps and operator identity.
5. Whether issue is deterministic or intermittent.
