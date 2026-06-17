# Launch Readiness Run Book

This run book is the release-owner checklist for the currently implemented pegasusX engine. It covers the support, evidence, rollback, and launch-support gates for the core supplier, retailer, driver, warehouse, factory, payload, payment, telemetry, and ai-worker seams. It does not mark deferred AI recommendation product surfaces as complete.

## Ownership

1. Release owner: approves the launch evidence bundle and owns rollback calls.
2. Backend owner: verifies Spanner, Kafka, cache, web hook, web socket, and SSMR evidence.
3. Client owner: verifies supplier portal, retailer row, driver row, and node-operator surfaces.
4. Support owner: confirms the SOP set is current and staffed for launch support.
5. Finance owner: confirms payment, ledger, dispute, and reconciliation support workflows.

## Required Evidence

1. SSMR proof: `make test-ssmr-infra` exits cleanly and records the Spanner, cache, backend health, spatial, and Kafka round-trip markers.
2. Kubernetes proof: `make validate-ai-worker-k8s` exits cleanly for the ai-worker config map, deployment, probes, and service contract.
3. Launch evidence proof: `make validate-launch-readiness` exits with `launch-readiness-ok`.
4. Load cert (PX9-F, staging-sized): `make load-cert-ssmr` prints `__LOAD_CERT_OK__` and updates `docs/LOAD_TEST_REPORT.md` (requires [k6](https://grafana.com/docs/k6/latest/) for full SLO thresholds).
5. Observability proof: Terraform contains ai-worker up, readiness, Kafka lag alert policies, optional uptime checks, and a launch dashboard.
6. Support proof: the indexed docs in `docs/README.md` exist for supplier onboarding, retailer commerce, payment exceptions, node operations, delivery execution, and ai-worker launch.

## Preflight Sequence

0. Production env contract: `PEGASUSX_ENV=production`, `REQUIRE_INFRA_ADAPTERS=true`, non-`dev-*` webhook secrets, and portal demo seed flags unset (`FACTORY_PORTAL_SEED`, `PAYLOAD_PORTAL_SEED`, `WAREHOUSE_PORTAL_SEED`).
1. Run `make test-ssmr-infra` from the `pegasusX` root.
2. Run `make validate-ai-worker-k8s` from the `pegasusX` root.
3. Run `make validate-launch-readiness` from the `pegasusX` root.
4. Optional before staging cutover: `make load-cert-ssmr` (smoke profile) or `LOAD_PROFILE=cert make load-cert` on a warmed cluster.
5. Review `docs/PAYMENT_EXCEPTION_SOP.md`, `docs/FINANCE_SUPPORT_WORKFLOW.md`, and `docs/DISPUTE_CLASSIFICATION_VOCABULARY.md` with finance support.
6. Review `docs/DRIVER_SUPPORT_PLAYBOOK.md`, `docs/LIVE_TRACKING_EXPECTATIONS.md`, and `docs/DELIVERY_ESCALATION_POLICY.md` with delivery support.
7. Review `docs/AI_WORKER_LAUNCH_RUNBOOK.md` with the worker operator.

## Launch Decision

Launch can proceed only when every required evidence item is present. If any gate fails, keep the environment in pre-launch state and assign a named owner before retrying.

## Rollback

1. Stop new rollout waves immediately.
2. Preserve the current logs, SSMR output, and failed gate output before changing state.
3. Roll back the affected deployment or configuration to the previous known-good artifact.
4. Re-run `make validate-launch-readiness` after rollback to prove the support evidence bundle is still coherent.
5. Record the failed gate, owner, and recovery action in the release notes.

## Launch Support

1. Monitor ai-worker up, readiness, and Kafka lag for the first launch window.
2. Monitor payment exception and dispute queues using the finance support workflow.
3. Monitor live tracking and delivery escalation reports using the driver support playbook.
4. Keep release, backend, client, finance, and support owners available until all launch-window alerts remain stable.
