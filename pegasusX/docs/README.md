# pegasusX docs

Operational runbooks and migration notes. Architecture lives in `../context/`.

Specifically:
- `../context/plan.md` — canonical phased execution roadmap (PX0–PX12, COOL-W*, PX-ECS)
- `../context/plan_ecosystem_sync.md` — cross-role realtime/data-flow audit (PX-ECS-1..5)
- `../context/plan_90.md` / `../context/PlanDigitalBrain.md` — planning brain (PX90/PX91)
- `../context/plan_local_closure.md` — close all non-cloud anchors (PX-LC-0..6; no GCP billing)
- `../context/PEGASUSX_CURRENT.md` for the current execution & planning architecture
- `../context/PEGASUS_REFERENCE.md` for the multi-tenant reference architecture (read-only; out of pegasusX scope)

## Canonical system spec

- **`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`** — living architect/product spec: role contracts, replenishment + co-locate flows, comms matrix, verification gates, PX-ECO tracker, infra budget
- **`CLOUD_BUDGET_MODEL.md`** — full-GCP-minimal cost table and Terraform budget wiring

## B02 supplier bootstrap support artifacts

- `SUPPLIER_ONBOARDING_SOP.md`
- `BILLING_RECOVERY_SCRIPT.md`
- `TOPOLOGY_ENTRY_SUPPORT_GUIDE.md`

## B03 retailer commerce support artifacts

- `RETAILER_ONBOARDING_SUPPORT_FLOWS.md`
- `PRICING_AUTHORITY_RULES.md`
- `ZONE_MISS_COMMUNICATION_POLICY.md`

## B04 payment integrity support artifacts

- `PAYMENT_EXCEPTION_SOP.md`
- `FINANCE_SUPPORT_WORKFLOW.md`
- `DISPUTE_CLASSIFICATION_VOCABULARY.md`

## B05 node operations support artifacts

- `WAREHOUSE_EXCEPTION_SOP.md`
- `REASSIGNMENT_SUPPORT_PLAYBOOK.md`
- `TRANSFER_CANCELLATION_RUNBOOK.md`

## B06 driver and live-delivery support artifacts

- `DRIVER_SUPPORT_PLAYBOOK.md`
- `LIVE_TRACKING_EXPECTATIONS.md`
- `DELIVERY_ESCALATION_POLICY.md`
- `MIGRATION_RUNBOOK_MANIFEST_ROUTE_GEOMETRY.md` — Spanner DDL, OSRM config, backfill, fleet live-map API parity

## v1 staging closure (2026-06-29)

- `REAL_WORLD_CASE_MATRIX.md` — role × lifecycle × edge case × guard × SOP
- `SHOP_CLOSED_E2E_SOP.md`
- `PARTIAL_DISPATCH_RECOVERY_SOP.md`
- `BARCODE_GO_LIVE_CHECKLIST.md`
- `RETAILER_RECEIVING_WINDOWS_GUIDE.md`
- `V1_STAGING_CLOSURE_CHECKLIST.md` — LC-01–LC-06 boss checklist

## PX7 launch-readiness support artifacts

- `AI_WORKER_LAUNCH_RUNBOOK.md`
- `LAUNCH_READINESS_RUNBOOK.md`

## pegasusX completion (2026-06-04)

- `ROLE_ROW_PARITY_MATRIX.md` — Pegasus vs pegasusX screen/API matrix
- `LOAD_TEST_SLO.md` — 10k retailer concurrency profile
- `CLOUD_CREDENTIALS_CHECKLIST.md` — Boss handoff for GCP/Firebase/payments
- `CLOUD_CUTOVER_RUNBOOK.md` — staging/production cutover sequence
- `LOAD_TEST_REPORT.md` — load run results template
