# WebSocket Ingress & Session Affinity

Operational runbook for production WebSocket traffic behind GCP HTTP(S) Load Balancer or Gateway API.

## Problem

WebSocket connections are long-lived. Naive round-robin across API pods breaks unless every pod shares hub state. PegasusX uses **Redis Pub/Sub relay** (`ws/` package) so any pod can deliver to any room — but connection count and CPU still scale per pod.

## Recommended production setup

### Option A — Redis relay only (current default)

Works when:

- All API replicas run WS hubs + Redis relay subscriber
- Connection count per region stays under ~5k per replica at HPA max

Ingress config:

- Path `/v1/ws/*` → `backend-go` Service port 80
- Backend timeout **3600s** (WS idle keepalive)
- Enable HTTP/2 to origin if supported; WS upgrades use HTTP/1.1

No sticky sessions required if Redis relay is healthy.

### Option B — Session affinity (optional)

Use when:

- Redis relay latency is measurable cross-AZ
- Debugging requires pinning a client to one pod

GKE Ingress annotation example:

```yaml
metadata:
  annotations:
    cloud.google.com/backend-config: '{"default": "pegasusx-ws-backendconfig"}'
```

BackendConfig:

```yaml
apiVersion: cloud.google.com/v1
kind: BackendConfig
metadata:
  name: pegasusx-ws-backendconfig
spec:
  sessionAffinity:
    affinityType: "GENERATED_COOKIE"
    affinityCookieTtlSec: 3600
```

Apply affinity **only** to WS path via URL map / separate backend service if REST traffic should stay stateless.

### Option C — Dedicated WS Deployment (future)

Split when connection count > ~5k per region:

- `backend-go-ws` Deployment: WS hubs only, no HTTP REST
- `backend-go` Deployment: REST only
- Shared Redis relay + same JWT validation

## Origin allowlist

`WS_ALLOWED_ORIGINS` in `infra/k8s/backend-go/configmap.yaml` must list every portal origin (prod + staging). Do not widen to `*` in production.

## Health & deploy

- `/ready` on API pods should fail when Redis is required and unreachable (strict mode).
- Rolling deploy: clients reconnect with backoff; mobile apps use `reconnectEpoch` / silent refresh (see realtime bounce fix).
- `ws/limits.go` sheds connections under pressure before OOM.

## Timeouts

| Path | LB timeout | Notes |
|------|------------|-------|
| REST `/v1/*` | 60–120s | Smart dispatch execute may need 120s |
| WS `/v1/ws/*` | 3600s | Match ping interval in hub |
| Webhooks `/v1/webhooks/*` | 30s | Fast ACK; processing async |

## Monitoring

- Per-pod: active WS connections (expose metric or log sample)
- Redis: relay channel lag, pub/sub errors
- Alert on relay disconnect storm after deploy

## References

- `apps/backend-go/ws/`
- `infra/k8s/backend-go/configmap.yaml` — `WS_ALLOWED_ORIGINS`
- ADR-003 — API vs worker split (WS stays on API pods today)
