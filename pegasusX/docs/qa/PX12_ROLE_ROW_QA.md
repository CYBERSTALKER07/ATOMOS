# PX-12 Role-Row QA Sign-Off

Production v1 capability matrix: [`ROLE_ROW_PARITY_MATRIX.md`](../ROLE_ROW_PARITY_MATRIX.md).  
Manual test steps: [`PX12_MANUAL_QA_RUNBOOK.md`](./PX12_MANUAL_QA_RUNBOOK.md).

---

## Sign-off record

| Field | Value |
|-------|-------|
| QA run id | |
| Tester | Boss |
| Build / commit | |
| Environment | `local-ssmr` / `staging` / `prod` |
| Preflight script | `bash scripts/qa/px12_preflight.sh` → `px12-preflight-ok` |
| SSMR e2e | `make test-ssmr-infra` → `__SSMR_OK__` |

---

## Phase A — Automated gates (required before manual UI)

| Gate | Command | Expected | Auto [ ] | Boss [ ] |
|------|---------|----------|----------|----------|
| Contract full | `make parity-contract-full` | `role-row-contract-full-ok` | | |
| Gap hunter | `make gap-hunter-gate` | `gap-hunter-gate-ok` | | |
| Launch readiness | `make validate-launch-readiness` | `launch-readiness-ok` | | |
| SSMR infra + e2e | `make test-ssmr-infra` | `__SSMR_OK__` + `PX_E2E_*_OK` | | |
| **Bundle** | `make px12-preflight` | `px12-preflight-ok` | | |

SSMR e2e markers (all must print during `ssmr-smokecheck e2e`):

`PX_E2E_ORDER_OK` · `PX_E2E_PAYMENT_OK` · `PX_E2E_WAREHOUSE_OK` · `PX_E2E_FACTORY_OK` · `PX_E2E_DELIVERY_OK` · `PX_E2E_TELEMETRY_OK` · `PX_E2E_PAYLOAD_OK` · `PX_E2E_PAYLOAD_MANIFEST_LIFECYCLE_OK` · `PX_E2E_PAYLOAD_REASSIGN_OK` · `PX_E2E_PAYLOAD_DRIVER_GATE_OK` · `PX_E2E_PAYLOAD_DEVICE_TOKEN_OK` · `PX_E2E_SHOP_CLOSED_OK` · `PX_E2E_NEGOTIATION_OK` · `PX_E2E_CATALOG_OK` · `PX_E2E_DEVICE_TOKEN_OK` · `PX_E2E_DRIVER_EDGES_OK` · `PX_E2E_CLIENT_POLICY_OK`

---

## Phase B1 — Supplier portal layout sign-off (pegasusX UI parity)

Visual/layout checks only — no API or backend contract changes. Compare pegasusX `supplier-portal` against pegasus `admin-portal` supplier surfaces.

| Route | Chrome | Boss [ ] |
|-------|--------|----------|
| `/auth/login` | Split auth panel + `auth-card` | [ ] |
| `/auth/register` | Circle stepper + topology step 2 | [ ] |
| `/setup/billing` | Centered billing gate | [ ] |
| `/dashboard` | `BentoGrid` mosaic + skeletons | [ ] |
| `/orders` | `PageChrome` + filter toolbar + `desk-table` | [ ] |
| `/dispatch` | `PageChrome` + kanban columns | [ ] |
| `/manifests` | `PageChrome` + list toolbar | [ ] |
| `/fleet` | `PageChrome` handoff to org-fleet | [ ] |

Shell invariant: every authenticated route (including `/org-fleet`, `/payments`, `/earnings`, `/ai/recommendations`) renders inside `SupplierShell` with working logout and flat URLs.

---

## Phase B2 — Warehouse portal layout sign-off (pegasusX UI parity)

Visual/layout checks only — no API or backend contract changes. Compare pegasusX `warehouse-portal` against pegasus `warehouse-portal` surfaces.

| Route | Chrome | Boss [ ] |
|-------|--------|----------|
| `/auth/login` | Split auth panel + `auth-card` | [ ] |
| `/` (dashboard) | KPI card grid + `desk-page-header` | [ ] |
| `/orders` | `PageChrome` + filter toolbar + `desk-table` | [ ] |
| `/orders/[id]` | `PageChrome` + warehouse mutation panel | [ ] |
| `/dispatch` | `PageChrome` + two-column preview | [ ] |
| `/manifests` | `PageChrome` + date filter + `desk-table` | [ ] |
| `/dispatch-locks` | `PageChrome` + acquire/release toolbar | [ ] |
| `/supply-requests` | `PageChrome` + state tabs + `desk-table` | [ ] |
| `/transfers` | `PageChrome` + pegasusX transfer panel | [ ] |

Shell invariant: every authenticated route renders inside `WarehouseShell` with `/transfers` in Operations nav and working logout.

---

## Phase B — Role-row manual sign-off

| Phase | Role | Clients | Contract / SSMR anchor | Manual runbook | Boss [ ] | Date |
|-------|------|---------|------------------------|----------------|----------|------|
| PX12-F | DRIVER | Android + iOS | `PX_E2E_DELIVERY_OK`, driver edges | [F1–F8](./PX12_MANUAL_QA_RUNBOOK.md#px12-f--driver) | [ ] | |
| PX12-G | RETAILER | desktop + Android + iOS | `PX_E2E_ORDER_OK`, catalog | [G1–G8](./PX12_MANUAL_QA_RUNBOOK.md#px12-g--retailer) | [ ] | |
| PX12-H | SUPPLIER | portal + native | supplier portal API sweep | [H1–H13](./PX12_MANUAL_QA_RUNBOOK.md#px12-h--supplier) | [ ] | |
| PX12-I | WAREHOUSE | portal + mobile | `PX_E2E_WAREHOUSE_OK` | [I1–I11](./PX12_MANUAL_QA_RUNBOOK.md#px12-i--warehouse) | [ ] | |
| PX12-J | FACTORY | portal + mobile | `PX_E2E_FACTORY_OK` | [J1–J6](./PX12_MANUAL_QA_RUNBOOK.md#px12-j--factory) | [ ] | |
| PX12-K | PAYLOAD | Expo + tablet native | `PX_E2E_PAYLOAD_OK` + lifecycle/reassign/gate/device-token sub-markers | [K1–K7](./PX12_MANUAL_QA_RUNBOOK.md#px12-k--payload) | [ ] | |

### Intentional v1 exclusions (do not block sign-off)

- Supplier native: broadcast / payment-bypass / empathy adoption (portal-only in v1).
- Full Pegasus supplier ~59 routes (P2-01).
- Rust optimizer sidecar (P2-02).

---

## Phase C — Staging (PX12-E)

| Step | Doc | Boss [ ] |
|------|-----|----------|
| GCP cutover | [`CLOUD_CUTOVER_RUNBOOK.md`](../CLOUD_CUTOVER_RUNBOOK.md) | [ ] |
| Cloud smoke | `PUBLIC_BASE_URL=https://api.staging.<domain> bash scripts/cloud_smoke_ssmr.sh` | [ ] |
| Load cert (staging) | `PUBLIC_BASE_URL=... make load-cert-cloud` | [ ] |
| Record run id | [`LOAD_TEST_REPORT.md`](../LOAD_TEST_REPORT.md) | [ ] |

---

## Phase D — Production (PX12-L)

| Step | Doc | Boss [ ] |
|------|-----|----------|
| Release train | [`RELEASE_TRAIN.md`](../RELEASE_TRAIN.md) | [ ] |
| 72h hypercare | [`INCIDENT_RESPONSE_RUNBOOK.md`](../INCIDENT_RESPONSE_RUNBOOK.md) | [ ] |
| Launch readiness owner sign-off | [`LAUNCH_READINESS_RUNBOOK.md`](../LAUNCH_READINESS_RUNBOOK.md) | [ ] |

---

## Quick start (Boss)

```bash
cd pegasusX
make px12-preflight          # automated gates
make ssmr-infra-up           # keep stack up for manual UI
make supplier-portal-dev     # :3000 — separate terminal
# Follow PX12_MANUAL_QA_RUNBOOK.md per role; check boxes above.
```
