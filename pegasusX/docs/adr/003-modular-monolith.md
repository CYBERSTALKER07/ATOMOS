# ADR-003: Modular monolith with selective process splits

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Status:** Accepted  
**Date:** 2026-06-22  
**Context:** ~100k LOC backend-go, single Spanner database, strong transactional boundaries. Microservices would add network partitions without relieving data coupling.

## Decision

Keep **one deployable binary** (`backend-go`) with domain packages (`order`, `warehouse`, `payment`, …) and thin `*routes` HTTP mounts. Split **processes** only where failure domains or scale differ:

1. **API pods** (`PEGASUSX_RUN_MODE=api`) — HTTP + WebSocket
2. **Worker pods** (`PEGASUSX_RUN_MODE=worker`) — outbox relay, Kafka consumers, dispatch warmer, preorder sweeper
3. **ai-worker** — VRP optimizer HTTP only (decouple from `order/service.go` over time)

Local/docker default: `PEGASUSX_RUN_MODE=all`.

Do **not** split into per-domain microservices until measured pain (QPS, blast radius, team ownership) justifies Spanner transaction boundaries breaking.

## Consequences

- `bootstrap/` is the composition root; in-memory scaffolds live in `bootstrap/memory/`.
- God files are split by concern (`order/state_machine.go`, `order/preorder_service.go`, …) not by deployment.
- K8s: separate `backend-go` and `backend-go-worker` Deployments share image + ConfigMap.

## References

- `bootstrap/run_mode.go`, `infra/k8s/backend-go-worker/`
- `docs/BACKEND_ECOSYSTEM_READINESS.md`
