# pegasusX scripts

Project-local build, guard, and codegen scripts. To be populated alongside features.

Planned:
- `contract_guard.py` — verify producer/consumer event shape parity
- `architecture_guard.py` — enforce package topology
- `design_system_guard.py` — flag emoji/decorative gradients
- `production_safety_guard.py` — verify outbox + cache invalidation coverage on mutations
- `security_guard.py` — scan for body-derived scope, secret leakage

Implemented:
- `smoke_ssmr.sh` — Phase-1 stage-gate harness for the isolated SSMR sandbox. Brings up `infra/docker-compose.ssmr.yml`, waits for Spanner/Redis/Kafka readiness, reruns the idempotent bootstrap, asserts seeded Spanner state and Kafka topic isolation via `apps/backend-go/cmd/ssmr-smokecheck`, checks Redis PING and backend `/v1/health`, then always tears the stack down with `down -v`.
