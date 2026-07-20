# D4 Memorystore Redis — 2026-07-20

## Instance

| Field | Value |
|-------|-------|
| Project | `void-494000` |
| Name | `pegasusx-staging-redis` |
| Region | `asia-south1` |
| Tier | STANDARD_HA |
| Version | REDIS_7_0 |
| Memory | 1 GB |
| Host | `10.42.205.148` |
| Port | `6378` |
| AUTH | enabled |
| TLS | SERVER_AUTHENTICATION |
| VPC | `pegasusx-staging-vpc` |
| State | READY |

## Secrets (GSM)

- `pegasusx-staging-redis-auth` — AUTH password  
- `pegasusx-staging-redis-addr` — `10.42.205.148:6378`  

```bash
# Read auth (operators only)
gcloud secrets versions access latest \
  --secret=pegasusx-staging-redis-auth --project=void-494000
```

## Prove (GKE Job in VPC)

Ephemeral Job `redis-d4-prove-job` (redis:7-alpine):

```
==> PING
PONG
==> SET pegasusx:d4:prove:…
OK
==> GET …
ok
D4_REDIS_PROVE_OK
```

Job status: Complete 1/1. Prove secret deleted after run.

## App wiring (D9)

Backend needs:

```
REDIS_ADDR=10.42.205.148:6378
REDIS_PASSWORD=<from GSM pegasusx-staging-redis-auth>
REDIS_TLS_ENABLED=true
```

Local overlay still uses `redis.pegasusx.svc.cluster.local:6379` — staging overlay must override to Memorystore.

## Console

Memorystore → Redis → `pegasusx-staging-redis`  
https://console.cloud.google.com/memorystore/redis/instances?project=void-494000

## Cost

STANDARD_HA 1 GB always-on — part of monthly envelope. Delete when idle for weeks if not needed.
