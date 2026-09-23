# SCOPE: Independent Victory Audit of pegasus.x

## Target Subsystems
- Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x`
- Backend Directory: `/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/backend`
- Reference Project Orchestrator Report: `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_11/handoff.md`

## Audit Verification Criteria
1. **Strict Two-System Architectural Boundary**:
   - Strictly PostgreSQL 16 (`pgx/v5`) + Redis 7 Streams.
   - ZERO references to Spanner (`cloud.google.com/go/spanner`, Spanner DDL, mutations) or Kafka (`sarama`, `kafka-go`, `confluent-kafka-go`) in `pegasus.x/backend` (check all Go files, `go.mod`, `go.sum`).

2. **Zero Mock Data Policy & Fail-Closed Constructors**:
   - Zero in-memory repository fallbacks or dummy seeds in non-test Go production packages across all packages.
   - All production service and repository constructors (`matching`, `fscm`, `copa`, `ewm`, `consignment`, `rebate`, `payout`, `wmsops`, etc.) require a valid `*db.Pool` and fail closed (`panic`) if nil.
   - `cmd/server/main.go` fails closed (`log.Fatalf`) if `db.Connect` fails.

3. **Strict 64-Bit Integer Minor Unit Arithmetic**:
   - All financial amounts, prices, fees, margins, and taxes must be calculated and stored strictly in 64-bit integer tiyins (`int64`). Zero floats for currency.
   - Statutory Uzbekistan 12% Soliq VAT integer round-half-up math: `(price * 12 + 50) / 100`.

4. **Cross-Role Real-Time Monotonic Pipeline Parity**:
   - Mutating lifecycle endpoints pair entity state mutation and outbox event write in the exact same `pgx.Tx` transaction closure.
   - `internal/ws/hub.go` sequence numbering is strictly monotonic under lock (`h.recentMu.Lock()`), and slow client pruning is safe under write lock.
   - Desktop applications listen for real-time invalidation.

5. **Independent Test Execution & Scale Benchmarks**:
   - `go build -v ./cmd/server` and `go build -v ./cmd/smokecheck` exit with code 0.
   - `go vet ./...` exits with code 0 and 0 diagnostics.
   - `go test -race ./...` passes cleanly with 0 failures and 0 race conditions across all packages.
   - Scale benchmarks: 1,000-order H3 clustering (<100ms, 0 abandoned), 100-order CVRP dispatch (0 abandoned), 3L-CVRP axle statics (11.5T single axle, 20% steer ratio), driver breakdown rescue hot-swap.
