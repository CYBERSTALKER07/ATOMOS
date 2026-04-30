# Batch 02D - Cross-Role Alternative Embodiments

## Embodiment Matrix

| Feature Family | Base Embodiment | Alternative A | Alternative B | Alternative C | Data Integrity Guard |
|---|---|---|---|---|---|
| Factory-to-Warehouse Replenishment | Demand-driven restock request accepted by factory and materialized into payload manifest | Time-window batch replenishment with consolidated manifests | Emergency override replenishment with supervisor lock | Inter-hub pooling with temporary node handoff | Home-node scope checks + immutable event lineage |
| Cross-Role Dispatch Governance | Freeze lock protocol pauses AI during human override | Role-tiered lock durations by urgency class | Multi-party approval for high-impact reassignment | SLA breach auto-lock with explicit human release | Lock state persisted and evented with TTL safety |
| Financial and Treasury Mutation Path | Idempotent mutation + RW transaction + outbox + cache invalidate | Delayed treasury settlement window with queued reconciliation | Multi-gateway split settlement by region | Offline provisional settlement with later reconciliation merge | Double-entry ledger parity and replay suppression |
| Forecast and Look-Ahead Intelligence | Predictive demand surfaced as advisory recommendations | Confidence-banded recommendations with auto-hide below threshold | Supplier-specific model ensembles | Event-driven reforecast on major disruption signals | Accepted recommendation required before operational mutation |
| Multi-Client Operational Parity | Shared backend contracts consumed by web and native clients | Staged rollout behind per-client feature flags | Desktop-first exposure with mobile handoff markers | Mobile-first field rollout with additive wire fields | Backward-compatible DTOs and stable event discriminators |

## Continuity Notes

- All embodiments maintain role-scoped authorization and immutable audit traces.
- Alternatives change control policy or interaction mode, not transactional correctness rules.
- Cross-role operations remain resilient to retries and partial transport failures.
