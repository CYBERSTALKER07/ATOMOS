# Surface audits — agent index (2026-08-18)

**Kind:** point-in-time audits with `file:line`. **Not** a go-live certificate. **Not** permission to terraform apply, flip `checkout_reads_this`, paste PSP/Soliq/Maps keys, or swap MapLibre for Google Maps.

**Honesty:** Current source is SoT. If an audit and code disagree, **code wins**. Re-open cited paths before using a row as status. Matrix **"Wired"** does not override these files.

**Destination (goal, not status):** [`GLOBAL_SCALE_PROGRAM.md`](./GLOBAL_SCALE_PROGRAM.md) + [`GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](./GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) + [`GLOBAL_SCALE_CLIENT_UI.md`](./GLOBAL_SCALE_CLIENT_UI.md).

## How other agents should use this

1. Load `.agents/memory/GOAL.md` then this index.
2. Open the **one** surface doc that matches the task. Do not treat the whole folder as “everything is PARTIAL so do nothing.”
3. Re-trace the live path (route → persist → outbox/consumer/client) before implementing.
4. Next slice is **code** unless the audit’s remaining work is explicitly secrets/DNS/IAM **and** the product spine on that path is REAL.

## Suite (backend · infra · UI)

| Surface | Verdict | Doc |
|---------|---------|-----|
| Maps / geography / display | **PARTIAL** | [`MAPS_AUDIT.md`](./MAPS_AUDIT.md) |
| Kafka + outbox + consumers | **PARTIAL** | [`KAFKA_AUDIT.md`](./KAFKA_AUDIT.md) |
| Redis (cache / hot loc / perimeter) | **PARTIAL** | [`REDIS_AUDIT.md`](./REDIS_AUDIT.md) |
| Infra (TF / GKE / K8s / cells) | **NOT READY FOR LAYER B** | [`INFRA_AUDIT.md`](./INFRA_AUDIT.md) |
| DevOps + CI/CD | **PARTIAL** | [`DEVOPS_CICD_AUDIT.md`](./DEVOPS_CICD_AUDIT.md) |
| Firebase + App Hosting | **PARTIAL** (Hosting **GONE**) | [`FIREBASE_AUDIT.md`](./FIREBASE_AUDIT.md) |
| Backend programming / system design | **PARTIAL** | [`BACKEND_SYSTEM_DESIGN_AUDIT.md`](./BACKEND_SYSTEM_DESIGN_AUDIT.md) |
| UI (web / desktop / native) | **PARTIAL** | [`UI_SURFACE_AUDIT.md`](./UI_SURFACE_AUDIT.md) |

Related living specs (not audits): [`DATA_FLOW_AS_IMPLEMENTED.md`](./DATA_FLOW_AS_IMPLEMENTED.md), [`OPTIMIZER_AND_ROUTING_RUNTIME.md`](./OPTIMIZER_AND_ROUTING_RUNTIME.md), [`MANIFEST_DUAL_PLANE.md`](./MANIFEST_DUAL_PLANE.md), [`E2_PER_SUPPLIER_PERIMETER_DESIGN.md`](./E2_PER_SUPPLIER_PERIMETER_DESIGN.md).

## Shared product laws (do not violate in any slice)

1. Tenant key is `SupplierId`. Pack / cell / country are attributes.
2. Class A SoT is **Spanner + same-txn outbox**. Kafka is the bus. Redis is not the ledger.
3. Dual manifest planes: factory trucks ≠ supplier trucks.
4. Integer minor money. Fiscal hard-gate. Pay-at-delivery.
5. Unkeyed ≠ success (`501` / `no_live_keys`). Never a fake 200.
6. Factory planning / auto-order **place** stay flag-off.
7. Display maps: MapLibre / MapKit. `MapsAdapter: GOOGLE_ROUTES` is routing.

## Skills used for this suite (already installed)

Kafka event-contracts + kafka-development · Redis Inc suite (`redis-core`, connections, clustering, security, observability) · Spanner discipline · GKE/terraform/kubernetes · golang-* (layout, CI, testing, security) · firebase-basics / firebase-auth-basics · devops · nextjs / vercel-react-best-practices · honest-code-gate · gap-hunter.

**Not installed this session:** `google-firebase/skills@firebase-hosting-basics` (clone auth failed). Hosting verdict still **GONE** from `firebase.json` (no `"hosting"` key) — do not add App Hosting.

## Cross-cutting ranked next (when asked to implement)

1. Control-tower MapLibre + pack camera ([`MAPS_AUDIT.md`](./MAPS_AUDIT.md)).
2. Copy nested-only CI jobs into **root** `.github/workflows/pegasusx-ci.yml`; fix `reatilerapp` → `retailerapp` ([`DEVOPS_CICD_AUDIT.md`](./DEVOPS_CICD_AUDIT.md)).
3. Do not treat Redis perimeter / payment-bypass as SoT ([`REDIS_AUDIT.md`](./REDIS_AUDIT.md)).
4. Keep outbox-in-txn; do not flip `KAFKA_TOPIC_CONSUME_DOMAIN` without dual-write + topics ([`KAFKA_AUDIT.md`](./KAFKA_AUDIT.md)).
5. Firebase stays OTP + FCM sidecar; JWT is HTTP session ([`FIREBASE_AUDIT.md`](./FIREBASE_AUDIT.md)).

Not Layer B. Not terraform apply.
