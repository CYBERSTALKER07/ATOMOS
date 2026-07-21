# Wave B deploy status (2026-07-21)

## Done
- Image pushed: `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-staging-images/backend-go:staging-glibc-202607211531`
- Dockerfile fixed: runtime `debian:bookworm-slim` (CGO/glibc; Alpine caused `exec ... no such file`)
- Org policy `compute.vmExternalIpAccess`: denyAll → allowAll at org 667485536839
- WI binding: `pegasusx-staging/backend-go` → `staging-backend@pegasus-503013`
- Deployments + secrets + configmap applied in `pegasusx-staging` (API + worker; worker scaled 0 temporarily)
- Node IAM: container.defaultNodeServiceAccount etc. on compute SA

## Blocker
- **SSD_TOTAL_GB quota asia-south1: 200/250 used** (2×100GB Autopilot node disks)
- Cannot create 3rd node for user workload; system pods fill 2 nodes (~94% CPU)
- Consumer override max is 250 — need **quota increase request** to ≥500 GB SSD (and preferably IN_USE_ADDRESSES ≥8)

## Console action required
1. Quotas: increase `SSD_TOTAL_GB` (Persistent Disk SSD) for `asia-south1` to **500+**
   https://console.cloud.google.com/iam-admin/quotas?project=pegasus-503013
2. After approval: `kubectl scale deploy/backend-go deploy/backend-go-worker -n pegasusx-staging --replicas=1`

## Image tag
staging-glibc-202607211531
