# OR-Tools Optimizer Core (Pattern A Sidecar)

This directory scaffolds stateless OR-Tools gRPC sidecars for V.O.I.D. optimization workloads.

Pattern selected: gRPC sidecar (Pattern A)

Current rollout mode:
- Python sidecar remains available as the canary baseline.
- Rust sidecar is the target compute engine for hybrid Go orchestration + Rust solver execution.

Why this fits the current repo:
- `pegasus/apps/backend-go` already uses asynchronous Kafka worker patterns and transactional outbox writes.
- `pegasus/apps/backend-go/internal/rpc/optimizergrpc` and `pegasus/apps/ai-worker/grpc_server.go` already establish gRPC transport conventions.
- Heavy OR-Tools CPU search must not block checkout RW transactions in Spanner.

## Directory Structure

```text
pegasus/services/optimizer-core/
  proto/
    optimizer_core.proto
  server/
    __init__.py
    main.py
    service.py
    mapping.py
    scaling.py
    vrp_adapter.py
    cpsat_adapter.py
  server-rust/
    Cargo.toml
    build.rs
    src/
      main.rs
      service.rs
      mapping.rs
      scaling.rs
      solver/
        mod.rs
        vrp.rs
        cpsat.rs
  scripts/
    gen_proto.sh
  adapters/
    go/
      go.mod
      Makefile
      cmd/optimizer-worker/main.go
      internal/
        config/config.go
        model/types.go
        scaling/scaling.go
        mapping/bimap.go
        optimizergrpc/client.go
        adapter/translate.go
        adapter/worker.go
```

## Solver Coverage

1. VRP routing model (`CalculateRoute`)
- Input: H3-derived distance matrix, capacities, time windows, drop-off UUID nodes.
- Output: ordered node UUID routes aligned to manifest flow.

2. CP-SAT constraints model (`ResolveConstraint`)
- Input: factory slot capacities, manifest requirements, retailer-priority scores.
- Output: boolean assignment matrix (`manifest_id` -> `factory_node_id`).

## Data Transformation Rules (Implemented)

1. Float-to-int scaling
- Utility factor: `10000`.
- `12.4 km -> 124000`, `1245.50 currency -> 12455000`.

2. Deterministic UUID-index mapping
- Bidirectional map held in-memory for each solve.
- UUIDs map to `0..N` for OR-Tools internals and map back on response.

## Execution Guardrails (Implemented)

1. Time limits
- VRP default: `2000 ms` (CTP/fleet fast budget).
- CP-SAT default: `30000 ms` (MEIO/factory scheduling budget).
- On timeout, adapters return best feasible solution when available.

2. Outbox-only persistence
- Go worker adapter reads optimization jobs from Kafka.
- Writes optimization result event rows to Spanner `OutboxEvents` only.
- Never mutates `Inventory` or `Manifest` tables directly.

## Build Dependencies

Python sidecar:
- `ortools`
- `grpcio`
- `grpcio-tools`
- `protobuf`

Rust sidecar:
- `tonic`
- `prost`
- `tokio`
- `tracing`

Go worker adapter module:
- `cloud.google.com/go/spanner`
- `github.com/segmentio/kafka-go`
- `google.golang.org/grpc`
- `github.com/google/uuid`

## Generation and Run

Generate Python gRPC stubs:

```bash
cd pegasus/services/optimizer-core
./scripts/gen_proto.sh
```

Run sidecar:

```bash
cd pegasus/services/optimizer-core
python -m server.main
```

Run Rust sidecar scaffold:

```bash
cd pegasus/services/optimizer-core/server-rust
cargo run
```

Run Go worker adapter scaffold:

```bash
cd pegasus/services/optimizer-core/adapters/go
go run ./cmd/optimizer-worker
```

Dual-run canary bootstrap:
- Keep Python sidecar active for output parity checks.
- Run Rust sidecar with the same protobuf contract and compare deterministic outputs before traffic cutover.

## Notes

- Existing V.O.I.D. transactional outbox table is `OutboxEvents` (this is the `Transactional_Outbox` concept in current schema).
- This scaffold is intentionally additive and isolated from checkout transaction paths.
