# Phase 6 — SSMR lifecycle vertical (enterprise order spine)

**Goal:** Prove end-to-end **create → dispatch → seal → complete** without waiting on the full ecosystem e2e matrix.

## Spine

```text
Retailer create order
    → Warehouse fleet + dispatch execute (MANIFEST draft/load)
    → Payload seal (+ driver gate open)
    → Driver arrive → confirm cash → collect cash → COMPLETED
```

## Commands

```bash
cd pegasusX

# Focused vertical (Docker SSMR stack)
make test-ssmr-lifecycle
# or
pnpm run test:ssmr:lifecycle
# or
bash scripts/smoke_ssmr_lifecycle.sh

# Full ecosystem (still required before launch)
make test-ssmr-infra
```

Requires **Docker** (Spanner emulator, Redis, Kafka, backend-go).

## Markers (gate)

See `contracts/ssmr_lifecycle_vertical_markers.json`.

| Marker | Meaning |
|--------|---------|
| `PX_E2E_LIFECYCLE_CREATE_OK` | Order create + retailer tracking |
| `PX_E2E_LIFECYCLE_DISPATCH_OK` | Warehouse dispatch execute with manifest |
| `PX_E2E_LIFECYCLE_SEAL_OK` | Payload seal + driver gate |
| `PX_E2E_LIFECYCLE_COMPLETE_OK` | Arrive + cash collect |
| `PX_E2E_LIFECYCLE_VERTICAL_OK` | Umbrella |

Also emits compatible umbrellas: `PX_E2E_ORDER_OK`, `PX_E2E_DELIVERY_OK`, `PX_E2E_PAYLOAD_OK`.

## Implementation

| Piece | Path |
|-------|------|
| E2E driver | `apps/backend-go/cmd/ssmr-smokecheck/e2e_lifecycle_vertical.go` |
| CLI mode | `go run ./cmd/ssmr-smokecheck lifecycle-vertical` |
| Smoke script | `scripts/smoke_ssmr_lifecycle.sh` |
| Marker contract | `contracts/ssmr_lifecycle_vertical_markers.json` |

## Relationship to full SSMR

| | Lifecycle vertical | Full `test-ssmr-infra` |
|--|--------------------|------------------------|
| Scope | Order spine only | Config + replenish + preorder + WS matrix + … |
| Time | ~ minutes (stack-dependent) | Longer full e2e |
| Gate | lifecycle markers | `contracts/ssmr_ecosystem_markers.json` |
| Use | Daily spine regression | Pre-launch green |

## Exit criteria

1. `make test-ssmr-lifecycle` prints `__SSMR_LIFECYCLE_VERTICAL_OK__`  
2. Marker gate passes  
3. No silent multi-pod gaps on seal / dispatch (covered by prior gap-hunter dispatcher fixes)

## Proof (local)

| Date | Result |
|------|--------|
| 2026-07-20 | **GREEN** — `__SSMR_LIFECYCLE_VERTICAL_OK__` + `ssmr-ecosystem-marker-gate-ok` after QR handoff fix in complete path |

### Notes from first failure

- Post-seal + depart leaves order **IN_TRANSIT** — do not force reverse transition to LOADED.
- `confirm-cash` requires **AWAITING_PAYMENT** (after arrive → QR scan), not bare ARRIVED.
