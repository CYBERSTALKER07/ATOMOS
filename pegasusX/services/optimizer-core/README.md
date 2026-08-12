# optimizer-core

Python **OR-Tools** VRP sidecar for supplier/warehouse dispatch.

- HTTP: `POST /v1/optimizer/solve` on port **8082**
- Contract: `packages/optimizer-contract/`
- Called only from Go `apps/backend-go/dispatch/optimizerclient/` via `OPTIMIZER_BASE_URL`

**Runtime status (local vs SSMR vs prod):** see [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md).

Clients never call this service directly. If the sidecar is down, dispatch continues with H3 BinPack (`optimizer_source=fallback_phase1`).

## Certification (P2-2)

```bash
# OR-Tools corpus (cold-chain, capacity, max-stops, multi-depot)
cd server && .venv/bin/pytest -q test_contract_solver.py test_certification_harness.py

# Rust greedy sidecar — must never report OPTIMAL
cd server-rust && cargo test greedy_vrp_never_reports_optimal
```

Fixtures live in `testdata/cert/*.json`. Rust status is `HEURISTIC` (proto enum value 5).
