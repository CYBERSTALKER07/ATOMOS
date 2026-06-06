# pegasusX scripts

Project-local build, guard, and code generation scripts. To be populated alongside features.

Planned:

- `contract_guard.py` — verify producer/consumer event shape parity
- `architecture_guard.py` — enforce package topology
- `design_system_guard.py` — flag emoji/decorative gradients
- `production_safety_guard.py` — verify outbox + cache invalidation coverage on mutations
- `security_guard.py` — scan for body-derived scope, secret leakage

Implemented:

- `smoke_ssmr.sh` — Phase-1 stage-gate harness for the isolated SSMR sandbox. Brings up `infra/docker-compose.ssmr.yml`, waits for Spanner, cache, and Kafka readiness, reruns the idempotent bootstrap, asserts seeded Spanner state and Kafka topic isolation via `apps/backend-go/cmd/ssmr-smokecheck`, checks cache PING and backend `/v1/health`, then always tears the stack down with `down -v`.
- `validate_ai_worker_k8s.sh` — PX7-A3 local release-gate for `infra/k8s/ai-worker/{configmap,deployment,service}.yaml`. Parses the manifests with Ruby's built-in YAML loader and asserts the worker image placeholder, config map wiring, probe paths, and service port contract.
- `validate_launch_readiness.py` — PX0-A5/PX7-A3 aggregate launch-readiness gate. Verifies the support run book bundle, SSMR and ai-worker validation entry points, Terraform observability evidence, Kubernetes packaging, and synchronized plan/context inventory before launch approval.
