# Layer B readiness + sandbox (ops sequence)

**Status:** ops sequence, **not** the product destination. Destination remains
[`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) +
[`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) +
[`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md). Goal file:
[`.agents/memory/GOAL.md`](../../.agents/memory/GOAL.md).

**Phased code plan (how we get READY):**
[`LAYER_B_ECOSYSTEM_READINESS_PLAN.md`](./LAYER_B_ECOSYSTEM_READINESS_PLAN.md)
(LB-0…LB-G). This file is **ops only** after that gate.

**Honesty:** this document does **not** certify Layer B. Code wins. Do not
terraform apply, inject Stripe/Soliq keys, or flip `checkout_reads_this` from
this file.

## Glossary

| Term | Meaning |
|------|---------|
| **sandbox** | Isolated integration environment (compose + optional GKE overlay). Not a sold country. Not a second tenant key. Canonical `PEGASUSX_ENV=sandbox`. |
| **cell** | One GCP project + region + stack (`cell-uz` live; EU/US/KZ planned). |
| **pack** | `MarketPack` (UZ shipped; others planned). Catalog in code is product law, not hardcoded theatre. |
| **ssmr** | Deprecated alias for sandbox. `PEGASUSX_ENV=ssmr` still parses as sandbox. Live GCS prefix `pegasusx/ssmr` stays until an ops ticket. |

## What Layer B is (and is not)

| Layer A (code) | Layer B (ops — later, gated) |
|----------------|------------------------------|
| Fail-closed packs, tenants, PSP honesty, sandbox proofs | GSM secrets, Soliq EDS, GP merchant, Maps, SMS |
| CI: unit, Spanner emulator, compose sandbox, kustomize validate | `terraform apply`, second live cell, `kubectl` prod |
| `PEGASUSX_ENV=sandbox` | Flip `checkout_reads_this` only when sandbox fiscal runtime **is** pack `MY_SOLIQ` |

Gate for any Layer B PR: spine still REAL on the live path; remaining work is
secrets/env/DNS/IAM only.

## Explicit non-goals

- EU `terraform apply` / `pegasusx-cell-eu`
- Stripe / Adyen / PAYME / CLICK live keys in GitHub
- Flipping `checkout_reads_this`
- U-motion, SAML/SCIM
- Rewriting live GCS prefix `pegasusx/ssmr` or UZ `project_id = pegasus-503013`

## Local sandbox

```
make sandbox-infra-up          # alias: make ssmr-infra-up
make test-sandbox-infra        # alias: make test-ssmr-infra
```

Compose: `infra/docker-compose.sandbox.yml`. Env: `.env.sandbox.example`.
One-release aliases keep `docker-compose.ssmr.yml` and `.env.ssmr.example`.

## After LB-G (not this train)

Only when [`LAYER_B_ECOSYSTEM_READINESS_PLAN.md`](./LAYER_B_ECOSYSTEM_READINESS_PLAN.md)
prints **READY FOR LAYER B**. That is **LB-B** (B1–B6): maps to R1–R2 in
[`PROD_READINESS_SEQUENCE.md`](./PROD_READINESS_SEQUENCE.md) — Soliq EDS, GP
merchant, SMS/push, GSM JWT, then consider the flag. Observability / managed
Kafka / optimizer replicas are B5. `make cell-plan CELL=eu` stays **plan only**
until a sold EU cell (B6 forbidden until then).
