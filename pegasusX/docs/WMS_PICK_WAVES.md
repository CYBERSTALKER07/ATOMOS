# WMS pick waves + seal gate (§8.7 Wave 1B)

Minimal pick waves from a truck manifest, task confirm with lot depletion, and a hard payload seal gate.

**Flag:** `WMS_PICK_WAVES_ENABLED` (default `false`) — when off, pick APIs return `wms_pick_waves_disabled` and seal is unchanged.

**Depends on:** Wave 1A lots/FEFO ([`WMS_LOTS_FEFO.md`](./WMS_LOTS_FEFO.md), `WMS_LOTS_ENABLED`) for reservation/FEFO allocate at wave create.

**Package:** [`apps/backend-go/stocklots`](../apps/backend-go/stocklots) (`picking.go`, `seal_gate.go`).

**DDL:** [`schema/migrations/20260806_wms_pick_waves.ddl`](../apps/backend-go/schema/migrations/20260806_wms_pick_waves.ddl)

## Tables

| Table | Role |
|-------|------|
| `PickWaves` | PK `WaveId`; links `ManifestId`; status `OPEN`\|`PICKING`\|`READY_TO_SEAL`\|`CANCELLED`; strategy `MANIFEST` |
| `PickTasks` | PK `(WaveId, TaskId)`; lot/location qty; status `PENDING`\|`CONFIRMED`\|`SHORT`\|`SHORT_WAIVED`; `PickSequence` |
| `SupplierTruckManifests.PickWaveId` | Thin link set on wave create; seal gate reads wave status |

## Behavior (flag on)

1. **Create** `POST /v1/warehouse/ops/pick-waves` `{ manifest_id }` — DRAFT/LOADING only; one wave per manifest (409 `pick_wave_exists`). Tasks from `OrderLotReservations` or FEFO-allocate + reserve remaining lines. Ordered by `WarehouseLocations.PickSequence` (not full S-shape).
2. **Confirm** `POST .../tasks/{taskID}/confirm` `{ quantity_picked }` — depletes lot QoH/reserved + V2 roll-up; short → `SHORT` (blocks ready until waived).
3. **Ready** — when all tasks `CONFIRMED` or `SHORT_WAIVED` → wave `READY_TO_SEAL`.
4. **Waive shorts** `POST .../waive-shorts` — warehouse-admin/admin only.
5. **Seal gate** — payload seal paths require wave `READY_TO_SEAL`; missing wave → `409 pick_wave_required`; incomplete → `409 pick_wave_incomplete`. In-memory demo manifests without Spanner rows skip the gate.

## HTTP

| Method | Path |
|--------|------|
| GET/POST | `/v1/warehouse/ops/pick-waves` |
| GET | `/v1/warehouse/ops/pick-waves/{waveID}` |
| POST | `/v1/warehouse/ops/pick-waves/{waveID}/tasks/{taskID}/confirm` |
| POST | `/v1/warehouse/ops/pick-waves/{waveID}/waive-shorts` (admin) |

## Clients

- Warehouse portal: `/pick-waves` (+ link from Manifests)
- Android / iOS: Transfer actions → WMS pick waves (scan product/lot id)
- `@pegasusx/types` + `@pegasusx/api-client` pick-wave methods

## SSMR

`PX_E2E_WMS_PICK_WAVE_OK|_SKIPPED`, `PX_E2E_WMS_SEAL_GATE_OK|_SKIPPED` (skip when flag off / no Spanner wave path).

## Residual (later §8.7)

- Zone S-shape aisle traversal / labor-balanced batching
- LIFO load sequencing
- Cycle counts → **Wave 1C stub** — [`WMS_CYCLE_COUNTS.md`](./WMS_CYCLE_COUNTS.md); residual ABC + apply-on-approve
- Soft-warn seal mode (Wave 1B is hard block only)
- Cold-chain temperature ingestion
