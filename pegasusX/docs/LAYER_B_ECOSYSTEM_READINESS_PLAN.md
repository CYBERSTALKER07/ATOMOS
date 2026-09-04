# Layer B ecosystem readiness — phased modular plan

**Status:** living plan, **not** a go-live certificate. Code wins.  
**Date:** 2026-08-16  
**Tree:** `pegasusX/`  
**Destination (unchanged):** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) + [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) + [`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md). Goal file: [`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md).  
**Ops sequence (later, gated):** [`LAYER_B_SANDBOX_READINESS.md`](./LAYER_B_SANDBOX_READINESS.md).  
**Honesty:** [`SUBSTANCE_GATE.md`](./SUBSTANCE_GATE.md) + skill `honest-code-gate`.

This file sequences **code + CI + secret-name shells** so the remaining work is credentials / env / DNS / IAM. It does **not** terraform apply, inject Stripe/Soliq values, or flip `checkout_reads_this`.

---

## 0. Verdict (code 2026-08-16)

```
VERDICT: NOT READY FOR LAYER B
DOCS vs CODE: drift — INFRA/FEATURES still say Stripe/Adyen are theatre
  redirects; checkout-init is catalogHonestyExecutor
BLOCKERS: ranked in §2
NEXT: execute LB-1 → LB-G one module per PR; then ops may open LB-B
```

**READY FOR LAYER B** (honest-code-gate): product spine REAL on the live path; data flow durable; unit + CI-equivalent passed **after** the latest edits; remaining work is secrets / env / DNS / IAM — **not** new business logic.

Closing GOAL leftovers (flag, cells apply, live PSP) **is** Layer B. Those stay in **LB-B**. Do not execute them from this file.

---

## 1. What Layer B is (and is not)

| Layer A — this plan (LB-0…LB-G) | Layer B — ops after READY (LB-B) |
|---------------------------------|----------------------------------|
| Fail-closed packs, tenants, PSP honesty | GSM **values**: Soliq EDS, GP merchant, Maps, SMS, JWT |
| Secret **names** declared (ESO + TF shells) | Secret **values** in GSM |
| CI: unit, kustomize, apply-guard, sandbox smoke | `terraform apply`, `kubectl` prod, second live cell |
| `PEGASUSX_ENV=sandbox` | Flip `checkout_reads_this` only when sandbox fiscal runtime **is** pack `MY_SOLIQ` |

### Explicit non-goals (every module)

- Flip `checkout_reads_this` (UZ catalog stays false until LB-B B2 proof).
- `terraform apply` of `cells/eu`, `cells/uz`, global, or live prefix `pegasusx/ssmr`.
- Stripe / Adyen / PAYME / CLICK live keys or real checkout-init executors (UZ pack is GLOBAL_PAY + CASH + unkeyed Payme/Click).
- Rewrite live GCS prefix `pegasusx/ssmr` or UZ `project_id = pegasus-503013`.
- Factory planning / auto-order **place** on; SAML/SCIM; PEPPOL execute; MapLibre → Google Maps; live payout rail; second tenant key; U-motion.

---

## 2. Evidence opened this session (not docs)

| Fact | Path |
|------|------|
| UZ `CheckoutReadsThis: false` | `apps/backend-go/auth/market_pack.go:138-139` |
| Session advertises the flag | `auth/session.go:56`; mount `GET /v1/auth/session` — `platformroutes/routes.go:45` |
| Checkout **does** read pack currency/PSP | `auth/checkout_pack.go:19-23` |
| Fiscal pack is law; PEGASUS on MY_SOLIQ pack allowed outside production | `auth/fiscal_pack.go:81-83`; `order/fiscal.go:167-170` |
| Sandbox/SSMR fiscal default `PEGASUS` | `.env.sandbox.example:118`; `.env.ssmr.example:118` |
| Prod overlay already `MY_SOLIQ` | `infra/k8s/overlays/prod/kustomization.yaml:50` |
| `RequireTenant` cfg default — **LB-1 done** | `bootstrap/bootstrap.go:357-358` is `auth.TenantContextEnforced()`. Mount still `main.go:137`. Production + unset flag → `RequireTenant(true)`. Staging/local stay off |
| Seed fail-closed under sandbox/production | `auth/tenant.go:55-71` |
| PSP checkout-init honesty | `payment/execution.go:149-152` — STRIPE/ADYEN/PAYME/CLICK are `catalogHonestyExecutor` (`:363-387`). Tests: `payment/honesty_test.go` |
| Unkeyed GLOBAL_PAY → `501 no_live_keys` | `payment/global_pay_executor.go:64-73`; HTTP map `payment/service.go:1417-1418` |
| Live payout rail always unimplemented | `auth/payout_pack.go:45-48`. UZ rail is `bank-file` (`market_pack.go:137`) — file rail is the product; live rail is **not** a UZ Layer B prerequisite |
| Only `cell-uz` live | `auth/cell_directory.go:40-44`; session note `auth/session.go:66-68` |
| `make cell-plan` never applies | `scripts/cell_plan.sh:1-5`; `Makefile:344-345` |
| Live GCS prefix `pegasusx/ssmr` | `infra/terraform/backend-ssmr.hcl:3` |
| Redis `BASIC` | `infra/terraform/main.tf:96-98` |
| Prod Kafka comments; optimizer replicas 0 | `infra/k8s/overlays/prod/kustomization.yaml:61-84` |
| ESO has no Soliq/SMS shells | `infra/k8s/overlays/sandbox/backend-go-externalsecret.yaml` (JWT, GP, Maps, Redis, Kafka). Terraform: **zero** soliq/playmobile resources |
| Tests freeze the flag | `auth/market_pack_test.go:16-17` |
| AI-worker empty currency — **LB-5 done** | `synthesis/currency.go` uses `auth.CoalesceCurrency`; empty/planned skips insert (`engine.go:356-362`). Seed default from pack — `config.go:31` |
| Graph routing index only | `context/architecture-graph.json` `generatedAt: null` — misses MarketPack / cells / `checkout_reads_this` |
| Tenant register mounted | `platformroutes/routes.go:47-48`; persist + same-txn outbox `tenantreg/service.go:190-221` |
| Order create + outbox | `orderroutes/routes.go:40`; `order/service.go` same-txn `EventOrderCreated` |
| PSP webhooks mounted (ingress ≠ live checkout-init) | `webhookroutes/routes.go:23-27` |

**Already REAL (do not rebuild):** GS-A0–A2, T1–T5, M1–M7 readers, C1–C5 plan-only, L0–L4, K1–K3, GS-I OIDC, GS-R bind, GS-U0–U9, GS-P dialect gates, sandbox env class (`auth/env.go:40-42`), compose + overlay + apply-guard (`scripts/ci_no_unattended_terraform_apply.sh`), CI `sandbox-infra.yml` + `pegasusX/.github/workflows/ci.yml` cell-isolation job.

**DOC DRIFT — LB-2 done:** living GS docs match `catalogHonestyExecutor`. Frozen `session-2026-08-07/` audit left as historical.

---

## 3. Module graph

```
LB-0  Spine inventory (verify)
  ├── LB-1  Tenant RequireTenant production default          [P0 code]
  ├── LB-2  Doc drift: honesty executors                     [docs]
  ├── LB-3  Secret shells (Soliq + SMS names, no values)     [infra yaml/tf]
  ├── LB-4  Sandbox fiscal path stays fail-closed            [code+test]
  ├── LB-5  Continuous leftover: ai-worker UZS invent        [code]
  ├── LB-6  Sandbox CI / compose proof this session          [verify]
  └── LB-7  Cell / apply guards stay plan-only               [verify]
              └── LB-G  Readiness gate → READY or NOT READY
                            └── LB-B  Ops only (gated; not this program)
```

LB-1…LB-7 are independent after LB-0 except LB-G needs all. **One module per PR.** A slice is not done until its proof command ran after the edit.

---

## 4. Phases

### LB-0 — Spine inventory

| | |
|--|--|
| **Owner** | eng |
| **Kind** | verify (no behavior change) |
| **Depends** | — |
| **Exit** | this inventory; every later PR re-traces its own live path |

Before coding any later module, re-open:

1. **Route mounts** — `POST /v1/order/create` (`orderroutes`), `POST /v1/checkout/unified` (`paymentroutes`), webhooks (`webhookroutes`), session / register / cells / packs (`platformroutes`), fiscal retry (`orderroutes`).
2. **Auth** — claims / `ResolveSupplierID`, not body `supplier_id`.
3. **Mutation** — Spanner + same-txn `outbox.EmitJSON`.
4. **Consumers** — outbox relay (`runtime_workers.go`); Kafka required when `REQUIRE_INFRA_ADAPTERS`.
5. **Role-row** — GS-R already bound; do not add UI in this program.
6. **Tests** named in the module.

---

### LB-1 — Tenant enforcement split (P0)

| | |
|--|--|
| **Owner** | eng |
| **Kind** | code |
| **Depends** | LB-0 |
| **Why it blocks READY** | remaining work is not “secrets only” while production `RequireTenant` can be off |

| Today | Fix |
|-------|-----|
| `cfg.TenantContextEnforced = envBool(..., auth.IsSandbox())` | **Done** — `auth.TenantContextEnforced()` at `bootstrap/bootstrap.go:358` |
| `main.go:137` `RequireTenant(cfg.TenantContextEnforced)` | Unchanged call site; cfg becomes correct |
| `ValidateProductionProfile` already requires `auth.TenantContextEnforced()` | Add a test that **cfg** is true when `PEGASUSX_ENV=production` and the env var is unset |

**Blast radius:** staging stays off (`PEGASUSX_ENV=staging` is not an enforced class — `auth/env.go:55-57`). Local/dev stays off. Sandbox/ssmr already on.

**Proof:** `go test ./auth/ ./bootstrap/ -count=1` including `TestConfigTenantContextEnforced_ProductionDefault`.

**Do not:** set `TENANT_CONTEXT_ENFORCED=true` in GitHub as a substitute.

---

### LB-2 — Honesty doc drift

| | |
|--|--|
| **Owner** | eng |
| **Kind** | docs |
| **Depends** | LB-0 |

**Done 2026-08-16.** Living GS docs (INFRA §0, FEATURES §17 + BF-269, LOCAL §2.5 + K2) match `catalogHonestyExecutor`. Stripe/Adyen stay **DEFER** for live charge. CASH/INTERNAL/CREDIT stay manual `staticProviderExecutor`.

**Proof:** grep `theatre redirect` in living GS docs is gone; `staticProviderExecutor` only describes CASH/INTERNAL/CREDIT or historical frozen files.

---

### LB-3 — Secret shells (names, not values)

| | |
|--|--|
| **Owner** | eng |
| **Kind** | infra yaml / tf / checklist |
| **Depends** | LB-0 |
| **Why** | even if ops puts Soliq in GSM, ESO does not pull fiscal/SMS keys |

**In scope:**

- GSM ids + ExternalSecret `secretKey`s for `FISCAL_MY_SOLIQ_BASE_URL`, `API_KEY`, `TIN`, `SIGNER`, PKCS#12 file/password, and PlayMobile / SMS keys used by `ar/dunning_channels.go`.
- Same names in [`CLOUD_CREDENTIALS_CHECKLIST.md`](./CLOUD_CREDENTIALS_CHECKLIST.md).
- Optional Terraform `google_secret_manager_secret` shells (no versions with real material).
- Keep live prefix `pegasusx-ssmr-*` (do not rename GSM this slice).

**Out of scope:** writing secret values; flipping `FISCAL_PROVIDER`; PKCS#12 in git.

**Done 2026-08-16 (names only, no apply).** TF shells `infra/terraform/fiscal_sms_secrets.tf`. ESO live `spec.data` unchanged (atomic). Names commented on sandbox/ssmr ExternalSecrets. Deployment env refs `optional: true`. Checklist + phase0 hooks. PKCS#12 bytes stay a volume.

**Proof:** `kubectl kustomize infra/k8s/overlays/sandbox` and `overlays/prod` still render; `scripts/ci_no_unattended_terraform_apply.sh` still green; no plaintext keys.

---

### LB-4 — Sandbox fiscal path (fail-closed; default unchanged)

| | |
|--|--|
| **Owner** | eng |
| **Kind** | code + test |
| **Depends** | LB-3 (names exist; values still empty) |
| **Why the flag stays false** | pack `FiscalAdapter=MY_SOLIQ` vs cell `FISCAL_PROVIDER=PEGASUS` (`AssertFiscalRuntime` allows that outside production) |

**In scope:**

- Keep sandbox/SSMR default `FISCAL_PROVIDER=PEGASUS`.
- Prove opt-in `FISCAL_PROVIDER=MY_SOLIQ` without complete creds **hard-fails** create/retry (`fiscal/signer_env.go`, [`SOLIQ_SANDBOX_READINESS.md`](./SOLIQ_SANDBOX_READINESS.md)) with a package test that names `PEGASUSX_ENV=sandbox`.
- Re-read `fiscal/signer_env.go` vs Soliq doc on `dev-hmac` in sandbox/ssmr; align comment and code in the same PR.
- Smokecheck already skips live Soliq when empty — keep that.

**Done 2026-08-16 (fail-closed proof; default unchanged).** Sandbox/SSMR default stays `FISCAL_PROVIDER=PEGASUS`. `SignerFromEnv` already forbids `dev-hmac` in sandbox/ssmr/production (`fiscal/signer_env.go:62-64`). Added `TestSignerFromEnv_SandboxMYSoliqMissingCreds`.

**Out of scope:** setting sandbox default to MY_SOLIQ; live OFD; flipping the catalog flag.

**Proof:** `go test ./fiscal/ ./order/ ./bootstrap/ ./cmd/ssmr-smokecheck/ -count=1` passed 2026-08-16. CI contract stays `PX_E2E_SOLIQ_CONTRACT_OK`.

---

### LB-5 — Continuous leftover (ai-worker UZS)

| | |
|--|--|
| **Owner** | eng |
| **Kind** | code |
| **Depends** | LB-0 |

**Done 2026-08-17.** Empty AI_PREORDER currency uses `auth.CoalesceCurrency` (`synthesis/currency.go`). Planned/unknown pack skips the insert (no `UZS` invent). Seed default is `seedCurrencyFromPack()` (`config.go:31`).

**Proof:** `go test ./apps/ai-worker/synthesis/ ./apps/ai-worker/ -count=1` passed 2026-08-17. Flag still false.

**Defer:** deep POS UZS screens, MapLibre leftover, graph `generatedAt` (not Layer B).

---

### LB-6 — Sandbox proof

| | |
|--|--|
| **Owner** | eng |
| **Kind** | verify |
| **Depends** | LB-4 |
| **Status 2026-08-17** | **PARTIAL** — compose infra + most e2e markers green. Not READY. |

**Ran 2026-08-17 (continue):**

1. Docker daemon up. First Hub pull of `python:3.12-slim-bookworm` timed out; retry pulled OK.
2. `make test-sandbox-infra` progressed after two smoke-script fixes:
   - Do not `up -d backend-setup` then `run --rm backend-setup` (schema/seed race).
   - `assertSeedSupplier` looks up `seed.DefaultSupplierID` — `EnsureDemoScopeLinks` rewrites Name to `SSMR Smoke Supplier` (`auth/seed_scope.go:170-172`).
3. Passed in-compose: `spanner`, `spatial`, `kafka`.
4. Topology persist 500: `ReplaceTopology` wrote `OutboxEvents` without `SupplierId` (`schema/spanner.ddl` NOT NULL). Supplier writers now use `outbox.EventRowMap` via `portalOutboxMutation`. Re-run printed `PX_E2E_TOPOLOGY_EDIT_OK`.
5. Next e2e fail: retailer register `400 trading_partner_required`. `envOr("SSMR_SMOKE_SUPPLIER_ID")` is empty in sandbox when the env key is unset (`sandboxIdentityKey`). `smokeSupplierID()` now falls back to `seed.DefaultSupplierID`. Env examples list the seed id.
6. Flag still false. No terraform apply.

CI: `V.O.I.D/.github/workflows/sandbox-infra.yml` still runs this target. Re-run after retailer attach fix.

---

### LB-7 — Cell / apply guards (plan-only)

| | |
|--|--|
| **Owner** | eng |
| **Kind** | verify |
| **Depends** | LB-0 |

**Done 2026-08-17 (verify, no apply).** Guards still hold. Do not apply. Confirm and keep:

| Guard | Path |
|-------|------|
| `make cell-plan` never apply | `scripts/cell_plan.sh:2-5` |
| Apply-guard | `scripts/ci_no_unattended_terraform_apply.sh` |
| Only `cell-uz` live | `auth/cell_directory.go:40-44` |
| EU tfvars empty JWT + observability off | `infra/terraform/cells/eu/cell.tfvars` |
| UZ tfvars observability off, `jwt_secret=""` | `infra/terraform/cells/uz/cell.tfvars:31-32` |

Redis HA, managed Kafka activation, optimizer replicas ≥ 1, `enable_observability_resources` are **R2 / LB-B**, not Layer A. Do not flip those tfvars here.

---

### LB-G — Readiness gate

Re-read every file from LB-1…LB-5. Re-trace LB-0. Run proofs from LB-1 / LB-4 / LB-5 / LB-6.

```
VERDICT: READY FOR LAYER B
  iff LB-1 cfg/production RequireTenant aligned
  and LB-2 living docs match catalogHonestyExecutor
  and LB-3 ESO/TF declare Soliq+SMS names
  and LB-4 MY_SOLIQ-without-keys fail-closed; default still PEGASUS
  and LB-5 no ai-worker empty-UZS invent
  and LB-6 proofs ran this session (or explicit unverified)
  and checkout_reads_this still false
  and no terraform apply / no Stripe keys
else NOT READY + ranked leftover
```

Only then may ops open **LB-B**.

---

## 5. LB-B — Layer B ops (gated; do not execute from this file)

Maps to [`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) R1–R2 and [`LAYER_B_SANDBOX_READINESS.md`](./LAYER_B_SANDBOX_READINESS.md). Owner: ops / legal / finance. Eng does not put keys in the repo.

| Order | Item | Exit |
|-------|------|------|
| B1 | GSM values for JWT, GP merchant, Maps, Redis AUTH | Prod/sandbox boot without `dev-*` secrets |
| B2 | Soliq sandbox EDS ([`FISCAL_EDS_PROOF.md`](./FISCAL_EDS_PROOF.md)) + `FISCAL_PROVIDER=MY_SOLIQ` on a **sandbox** canary | `PX_E2E_SOLIQ_SANDBOX_LIVE_OK` |
| B3 | SMS / FCM / APNs (R1.3–R1.4) | Dunning / OTP not silent |
| B4 | Flip UZ `CheckoutReadsThis` to **true** **only after B2** | Session flag true; update `auth/market_pack_test.go` |
| B5 | Optional: observability tfvars, managed Kafka brokers, optimizer image + replicas ≥ 1 | R2; not a second country |
| B6 | EU `terraform apply` | **Forbidden** until a sold EU cell + C3/C4 live deny |

---

## 6. How a later PR uses this file

1. Pick **one** ID (LB-1 … LB-5, then LB-6/7, then LB-G).
2. Re-open the evidence paths in §2 for that module.
3. Impact-trace: route, schema, contracts, events, role-row, infra, tests.
4. Edit. Re-read every edited file. Run the module’s proof command.
5. Do not start LB-B. Do not flip the flag. Do not apply terraform.

---

## 7. What “done” is not

- Not “we listed 50 countries.”
- Not cloud-ready, not Stripe-live, not Soliq-live.
- Not a second tenant key, not multi-region Spanner, not cross-border checkout.
- Not flipping `checkout_reads_this` from this file.
- Not applying `cells/eu`.

This file existing is **not** READY FOR LAYER B. Status needs `file:line` after LB-G.
