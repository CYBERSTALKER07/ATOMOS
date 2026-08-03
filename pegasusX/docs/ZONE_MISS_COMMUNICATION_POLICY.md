# Zone-Miss Communication Policy (B03)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose
Standardize communication and escalation when retailer order attempts fail because the retailer location is outside active service coverage.

## Definition
Zone miss means the system cannot assign a valid supplier warehouse/service perimeter for the retailer coordinates in the active topology.

## Client-facing response standard
1. Use clear, non-blaming language: location is currently outside serviceable delivery area.
2. Tell the retailer what to do next: verify location details, retry, and contact support if unchanged.
3. Do not expose internal routing internals or raw infrastructure details.

## Support triage playbook
1. Capture retailer id, supplier id, and attempted coordinates.
2. Confirm payload integrity (`lat`, `lng`, and `h3_cell` where applicable).
3. Confirm supplier topology has at least one active, on-shift warehouse.
4. Re-run order attempt after topology correction or coordinate correction.
5. Escalate to backend platform if failures persist after topology and payload validation.

## Error-to-message mapping

| Internal category | Support-facing message | Action |
|---|---|---|
| `zone_miss` / serviceability miss | "Your location is currently outside active delivery coverage." | Verify location and coverage, then retry |
| Missing active warehouse topology | "Delivery is temporarily unavailable in your area." | Supplier ops must restore active warehouse coverage |
| Invalid coordinate payload | "We could not validate your delivery location." | Correct coordinate payload and retry |

## Escalation packet
1. Endpoint and method used for order attempt.
2. Request payload fields related to location.
3. Status code and response body.
4. Supplier warehouse activity snapshot at attempt time.
5. UTC timestamp and environment.

## Operational note
This policy is additive and support-facing. It does not change backend error contracts; it standardizes communication quality and incident handoff while B03 serviceability and pricing work continues.
