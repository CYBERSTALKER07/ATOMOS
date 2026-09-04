# Enterprise Phase 0 — Money-path correctness

> **HISTORICAL / FROZEN — session progress note; do not treat as current gap SoT.**
> Living residuals: [`../PROD_READINESS_SEQUENCE.md`](../PROD_READINESS_SEQUENCE.md) · [`ECOSYSTEM_GAP_REGISTER_2026-08-12.md`](./ECOSYSTEM_GAP_REGISTER_2026-08-12.md) · [`../DOCS_SOURCE_OF_TRUTH.md`](../DOCS_SOURCE_OF_TRUTH.md).


**Date:** 2026-08-11  
**Plan source:** `docs/session-2026-08-07/ENTERPRISE_GRADE_EXECUTION_PLAN.md` §Phase 0  
**Gate:** `make money-path-gate` → **`money-path-gate-ok`** (2026-08-11, Spanner emulator `localhost:9010`)  
**Status:** Wired (backend) + Kafka HA / repo hygiene contracts

## Proof

| Step | Result |
|------|--------|
| `make infra-up` | Spanner emulator on `:9010` |
| Schema apply (`go run ./cmd/setup`) | OK (incl. payment idempotency indexes + outbox DLQ) |
| Order `TestMoneyPathGate_*` + `TestWorkerShopClosed_*` | PASS |
| Payment `TestMoneyPathGate_*` | PASS |
| `ci_fail_todo_inject.sh` | OK |
| `ci_no_mock_control_tower.sh` | OK |
| gitleaks | no leaks found |
| `make kafka-ha-gate` | **`kafka-ha-gate-ok`** |
| `make repo-hygiene-gate` | **`repo-hygiene-gate-ok`** |

## Audit (already shipped before this run)

| Item | Status | Evidence |
|------|--------|----------|
| Capture gateway `GLOBAL_PAY` | Done | `payment/service.go` `CaptureCardPayment` |
| No optimistic in-txn CAPTURED | Done | PENDING leg + `settleOutstandingCardPayment` |
| Stub jail + production never stubs | Done | `global_pay_executor.go` + gate tests |
| WebhookReconciler stub guard | Done | `payment/reconciliation.go` |
| Shop-closed fail-closed + `Available()` | Done | `order/worker_shop_closed.go` |
| Unique idempotency indexes | Done | `20260816_payment_idempotency_indexes.ddl` |
| Partner `Idempotency-Key` | Done | `partner/handlers.go` |
| Outbox DLQ | Done | `OutboxDeadLetters` + `outbox/spanner_store.go` |
| Negative stock fail-loud | Done | `inventory/repository.go` |
| Payers role-gate | Done | `payment/crud_handlers.go` |
| Control Tower env gate | Done | `CONTROL_TOWER_SIMULATOR_ENABLED` |
| Warehouse scanner | Done | `ScannerViewModel` injects `WarehouseApi` |

## Kafka HA (this run)

| Item | Status |
|------|--------|
| Strimzi cluster RF=3 / min.isr=2 | Already in `infra/k8s/kafka.yaml` |
| Strimzi KafkaTopics RF=3 / min.isr=2 | Fixed (were RF=1) |
| Managed Kafka terraform topic set (main+dlq+spatial+realtime+webhooks+freeze+inventory) RF=3 | `infra/terraform/kafka.tf` |
| Staging on `GCP_MANAGED_OAUTH` | Already wired |
| Prod activation | Owner: terraform `enable_managed_kafka=true` + uncomment prod overlay (see `docs/GCP_MANAGED_KAFKA.md`) |
| Gate | `scripts/ci_kafka_ha_gate.sh` |

## Repo hygiene (this run)

| Item | Status |
|------|--------|
| GCS terraform backend | Already `infra/terraform/backend.gcs.tf` |
| Local `infra/terraform/*.tfstate*` | Absent |
| Tracked `artifacts/tfstate-archive` secrets/tfvars | Removed from git + deleted; README only |
| `.gitignore` | Binaries, tfstate archive, bak/orig, root `patch_*.sh` |
| Gate | `scripts/ci_repo_hygiene_gate.sh` |

## Fix landed earlier this session

- `ar.SpannerRepository.OpenInvoice`: commit timestamps + clamp `CreditLeaveAt` for emulator clock skew.

## Owner follow-ups (not blocked in-repo)

- Apply Managed Kafka for **prod** (`enable_managed_kafka=true`) and uncomment prod overlay bootstrap.
- Rotate any credentials that lived in the deleted void-494000 `staging.tfvars` archive (`jwt_secret`, maps key, GP webhook secret) if still in use.
- Optional: remove nested `softwareengineercv-main/` from this monorepo (large unrelated site; not required for money-path).

## Note

Capture state machine in code is `PENDING → CAPTURED|FAILED` (no separate `CAPTURE_PENDING` enum); gate tests encode that model.
