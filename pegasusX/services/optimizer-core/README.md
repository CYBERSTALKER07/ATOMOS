# optimizer-core

Python **OR-Tools** VRP sidecar for supplier/warehouse dispatch.

- HTTP: `POST /v1/optimizer/solve` on port **8082**
- Contract: `packages/optimizer-contract/`
- Called only from Go `apps/backend-go/dispatch/optimizerclient/` via `OPTIMIZER_BASE_URL`

**Runtime status (local vs SSMR vs prod):** see [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md).

Clients never call this service directly. If the sidecar is down, dispatch continues with H3 BinPack (`optimizer_source=fallback_phase1`).
