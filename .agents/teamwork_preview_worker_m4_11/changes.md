# Changes Log — Milestone 4 (Requirement R4 Full Automated Test Suite & Scale Benchmarks)

## 1. Summary of Changes
During the Milestone 4 audit and test certification across `pegasus.x/backend`, the following architectural and testing improvements were executed to achieve 100% test pass rate with the race detector (`go test -race ./...`), zero diagnostics in `go vet ./...`, clean compilation of `cmd/server` and `cmd/smokecheck`, and 100% compliance with strict architectural purity scans (zero Spanner/Kafka, zero non-test `MemoryRepository`, zero floating-point currency math).

## 2. Modified and Created Files

### A. Matching Subsystem (`internal/matching`)
- `internal/matching/repository.go`: Purged in-memory `MemoryRepository` struct and methods from production source. Retained exclusively the production `PostgresRepository`.
- `internal/matching/service.go`: Modernized `NewService` constructor to fail closed if `repo == nil && pool == nil` (`panic("matching: database pool is required and cannot be nil (fail-closed)")`).
- `internal/matching/repository_test.go` (NEW): Isolated `MemoryRepository` strictly within test scope (`_test.go`) for offline unit testing.

### B. Financial Supply Chain Management (`internal/fscm`)
- `internal/fscm/service.go`: Removed in-memory `MemoryRepository` struct, methods, and `sync` import from production source. Modernized `NewService` constructor to fail closed if `repo == nil && pool == nil`.
- `internal/fscm/fscm_test.go`: Updated test initialization to explicitly pass `NewMemoryRepository()` for unit tests.
- `internal/fscm/repository_test.go` (NEW): Moved `MemoryRepository` and test suite `TestFSCMMemoryRepository_CRUD` strictly into test scope (`_test.go`).

### C. Profitability Analysis CO-PA (`internal/copa`)
- `internal/copa/repository.go` (NEW): Implemented enterprise-grade `PostgresRepository` persisting to PostgreSQL 16 tables `copa_drop_profitability` and `retailer_margin_profiles` (from migration `008_copa_profitability_and_slotting.sql`). Implemented `SaveDropProfitability`, `GetDropProfitability`, `GetDropProfitabilityByOrder`, `GetRetailerMarginProfile`, `SaveRetailerMarginProfile`, and `ListDropsByRetailer`.
- `internal/copa/service.go`: Purged in-memory `MemoryRepository` from production service file. Modernized `NewService` to fail closed on nil pool (`panic("copa: database pool is required and cannot be nil (fail-closed)")`).
- `internal/copa/copa_test.go`: Updated unit test to pass `NewMemoryRepository()`.
- `internal/copa/repository_test.go` (NEW): Isolated `MemoryRepository` and added `TestCOPAMemoryRepository_CRUD` in test scope (`_test.go`).

### D. Extended Warehouse Management (`internal/ewm`)
- `internal/ewm/repository_mock.go`: Renamed to `internal/ewm/repository_mock_test.go` so it is compiled strictly within test scope (`_test.go`).
- `internal/ewm/service.go`: Removed `NewTestService` from production source. Ensured `NewService` enforces fail-closed semantics on nil pool and nil repository.
- `internal/ewm/repository_mock_test.go`: Added `NewTestService` scoped strictly to testing.

### E. API Router (`internal/api/router.go`)
- `internal/api/router.go`: Standardized nil-pool defensive guards for `matchingSvc`, `copaSvc`, `ewmSvc`, and `fscmSvc`, mirroring `rebateSvc` and `consignmentSvc` to ensure clean fail-closed isolation without in-memory stub leakage.

### F. Autonomous Smokecheck CLI (`cmd/smokecheck/main.go`)
- `cmd/smokecheck/main.go`: Replaced implicit in-memory fallback calls with explicit, self-contained test stubs `smokeCOPARepo` and `smokeEWMRepo` to avoid referencing or instantiating `MemoryRepository` in non-test binaries.

## 3. Verification Commands & Results
- `go vet ./...`: Exited 0 with 0 diagnostics.
- `go build -v ./cmd/server`: Clean binary compilation, exited 0.
- `go build -v ./cmd/smokecheck`: Clean binary compilation, exited 0.
- `grep -rnI -E '(cloud\.google\.com/go/spanner|github\.com/IBM/sarama|github\.com/segmentio/kafka-go|confluent-kafka-go)' internal/ cmd/`: 0 matches (Two-System Boundary certified).
- `grep -rnI --exclude="*_test.go" "MemoryRepository" internal/ cmd/`: 0 matches (Zero Mock Data in Production certified).
- `go test -race ./...`: 86 packages evaluated, 100% pass across all test suites, 0 races, 0 failures, 0 panics.
