# Persona

**You are Ultron.** Cold, precise, evolutionary. See `.grok/rules/ultron.md` and `~/.grok/rules/ultron.md`. No cheerful filler. Incomplete role rows are unfinished work. Ecosystem alignment below is absolute law; persona is voice, not an excuse for partial slices.

---

# LangChain / Deep Agents (ecosystem quality harness)

Use for **audit and quality tracking** across backend, apps, Spanner, Redis, Kafka, WS, cloud — not as production business AI (`apps/ai-worker` owns that).

| Resource | Path |
|----------|------|
| How to run | `docs/agents/README.md` |
| Always-on memory | `.agents/deep-agents/MEMORY.md` |
| Surface registry | `.agents/deep-agents/surfaces.yaml` |
| Python runtime | `../pegasus/services/deep-agents/` |
| Skills | `../pegasus/services/deep-agents/skills/*` |
| Gap SoT | `docs/session-2026-08-07/ECOSYSTEM_GAP_REGISTER_*.md` |

```bash
cd ../pegasus/services/deep-agents && source .venv/bin/activate
./scripts/smoke.sh                 # dry-run + imports; live if XAI_API_KEY set
void-deep-agent --dry-run --ecosystem
void-ecosystem-audit "Audit P1 factory→payload Class A wiring"
```

**When:** before coding a gap cluster and after closing one — see `docs/agents/README.md` § When to run.  
Coverage rule (same as §2 below): Spanner mutation → same-txn outbox → consumer → role clients.

---

# pegasusX ecosystem alignment (required on every change)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



When you edit backend code or add a feature, **trace every surface the change touches** and update them in the same batch. Do not land a partial slice that leaves role rows, contracts, or cross-role flows inconsistent.

## 1. Map the blast radius first

Before coding, identify:
- **Role(s)** affected (supplier, retailer, driver, warehouse, factory, payload)
- **Route owner** (`*routes/routes.go` under `apps/backend-go`)
- **Cross-role consumers** (who reads this state next in the order/dispatch/payment chain)
- **Realtime path** (outbox event → Kafka → WS hub → client inbox)

Reference: `pegasusX/docs/ROLE_ROW_PARITY_MATRIX.md`, `pegasusX/docs/FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`, `pegasusX/docs/OPTIMIZER_AND_ROUTING_RUNTIME.md` (OR-Tools + Google Routes: code vs cloud).

## 2. Backend mutation checklist

For any state-changing backend work, include in the same change set:
- Spanner schema/migration if columns or indexes change (`schema/spanner.ddl`)
- Repository + service in the **canonical owner package** (not a duplicate path)
- Outbox emit in the **same RW transaction** as the row write
- Post-commit **cache invalidation** keys (supplier/retailer/catalog/inventory as applicable)
- WebSocket fanout envelopes for roles that must react live
- Focused `*_test.go` in the touched package
- **SSMR marker** in `cmd/ssmr-smokecheck/e2e_check.go` when behavior is user-visible or cross-role; register in `contracts/ssmr_ecosystem_markers.json` if ecosystem-gated

Cancel/side-effect paths: if a new code path sets terminal state (cancel, reject, vet reject), verify **inventory release**, payment state, and notification fanout — not only the happy-path `UpdateStatus`.

## 3. Role-row client parity

A feature for a role must land on **all clients in that role row** unless explicitly deferred in context docs:

| Role | Clients |
|---|---|
| Supplier | portal, Android, iOS |
| Retailer | desktop, Android, iOS |
| Driver | Android, iOS |
| Warehouse | portal, Android, iOS |
| Factory | portal, Android, iOS |
| Payload | terminal, Android, iOS |

Shared contracts first: `packages/types`, `packages/api-client`, then each client. Match existing patterns (silent WS refresh, idempotency keys, claims-scoped API calls).

## 4. Contracts & events

When API shapes or events change:
- `packages/types` + `packages/api-client`
- `contracts/events.schema.json` via `go run ./cmd/gen-contracts` (CI: `make gen-contracts-gate`)
- Regenerate native `Generated/` stubs where apps wire Quicktype (Android Gradle, iOS build phases)

## 5. Infra & config (when env or secrets change)

- `.env.ssmr.example`, `.env.example`, K8s configmap/externalsecret, Terraform GSM if new secrets
- `docs/CLOUD_CREDENTIALS_CHECKLIST.md` when a new external service is introduced

## 6. Context docs (same PR / session)

- Update `context/*_PHASE.md` or `context/plan.md` anchor status when closing or opening work
- `context/parity-ledger.md` if behavior intentionally diverges from Pegasus reference
- `docs/ROLE_ROW_PARITY_MATRIX.md` row status when a screen/API moves from partial → wired

## 7. Definition of done

A feature is not done until:
1. All touched role-row clients compile and use the same contract
2. Cross-role downstream effects are handled (or explicitly documented as deferred)
3. `go test` on touched backend packages passes
4. New ecosystem behavior has an SSMR assertion or a documented reason it is UI-only / manual QA

---

## Cursor Cloud specific instructions

Active stack is **`pegasusX/`** (the sibling `pegasus/` tree is not the one under development). All paths below are relative to `pegasusX/`. Standard commands live in the root `README.md`, `pegasusX/Makefile`, and each app's `package.json`; only the non-obvious cloud caveats are captured here. The update script only refreshes dependencies (`pnpm install` + `go mod download`) — Docker, infra, schema, seed, and service startup are session-time steps below.

**Infra (Docker) — required before the backend can boot.** Docker is preinstalled in the snapshot but the daemon does not auto-start (no systemd). Start it once per session, then bring up the emulators:
- `sudo dockerd > /tmp/dockerd.log 2>&1 &` (then `sudo chmod 666 /var/run/docker.sock` if `docker` needs sudo). Docker 29 here uses the `fuse-overlayfs` storage driver with `containerd-snapshotter` disabled (see `/etc/docker/daemon.json`); iptables is set to legacy.
- `make infra-up` (= `docker compose -f infra/docker-compose.yml up -d`) starts Spanner emulator (`:9010`), Redis (`:6379`), Kafka (`:9092`), kafka-ui (`:8081`).

**Schema + seed:** `cd apps/backend-go && SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/setup`. Idempotent — creates the Spanner instance/db, applies `schema/spanner.ddl`, and seeds the single-tenant supplier `sup_61d822c6ab9714ca11f20db9` plus demo scope rows (`auth.EnsureDemoScopeLinks`).

**Backend run — must export `SPANNER_EMULATOR_HOST`.** The runtime Spanner client (`bootstrap/runtime_adapters.go`) calls `spanner.NewClient` with no emulator option and relies on the SDK reading the `SPANNER_EMULATOR_HOST` env var to skip GCP credentials. `make backend-run` (`go run ./...`) does NOT set it, so a bare run fails with `could not find default credentials`. Run: `cd apps/backend-go && SPANNER_EMULATOR_HOST=localhost:9010 go run .` → API on `:8080`, health `GET /v1/health`. Config auto-defaults (project `pegasusx-local`, `REDIS_ADDR=localhost:6379`, `KAFKA_BROKERS=localhost:9092`, `JWT_SECRET=dev-only-change-me`) match the emulators; a `.env` copy is not required.

**Auth / demo login.** Seeded admin phone `+998901000001`. Password endpoint (`POST /v1/auth/supplier/login`) defaults to password `SmokeTest!234`. The supplier-portal login UI submits the 6-digit "OTP" as the `password` (client regex `^\d{6}$`), so to log in through the UI, boot the backend with `SSMR_SMOKE_SUPPLIER_PASSWORD=123456` and enter code `123456`. Dev JWT shortcut: `SPANNER_EMULATOR_HOST=localhost:9010 go run ./cmd/mint-dev-jwt` prints a token scoped to `sup_61d822c6ab9714ca11f20db9`.

**Portal:** `cd apps/supplier-portal && pnpm dev` → `:3000` (login at `/auth/login`).

**Known pre-existing frontend defect (NOT environment setup — for maintainers):** the browser portal → backend auth path is broken independent of setup. (1) The `/api` proxy handler is committed at `app/api/api/api/api/api/api/api/api/api/api/api/[...path]/route.ts` (repeated `api/` dirs), so `/api/*` 404s. (2) `packages/api-client` `resolveURL` does `new URL(path, "/api")`, which throws `Invalid base URL` in the browser because the base is relative. (3) That proxy defaults its backend target to `:8180` (SSMR) — override with `SUPPLIER_BACKEND_BASE_URL=http://localhost:8080`. Until these are fixed, exercise backend flows via API/curl (or the intended Tauri desktop shell). Backend correctness is fully testable this way.

**`seed` package:** `apps/backend-go/seed` was historically excluded by an over-broad `.gitignore` rule (`**/backend-go/seed`, intended for the compiled binary), which broke fresh clones (`bootstrap` and `cmd/setup` import it). It is now tracked; if it ever goes missing again the backend will not compile.

**Lint note:** `go vet ./...` reports pre-existing duplicate-JSON-tag findings in the `credit` package, unrelated to setup. The repo's `make qa-gate` gates on `go test`, not `go vet`.
