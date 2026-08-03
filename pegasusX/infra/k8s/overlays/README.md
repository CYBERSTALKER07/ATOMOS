# Kustomize overlays

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Render production manifests:

```bash
kubectl kustomize infra/k8s/base --load-restrictor LoadRestrictionsNone
# or
kubectl kustomize infra/k8s/overlays/prod --load-restrictor LoadRestrictionsNone
kubectl kustomize infra/k8s/overlays/staging --load-restrictor LoadRestrictionsNone
```

| Overlay | Namespace | Notes |
|---------|-----------|-------|
| `prod/` | `pegasusx` | Full HA: 3+ API replicas, worker split, ai-worker + optimizer-core, PDB/HPA, PodMonitoring |
| `staging/` | `pegasusx-staging` | Dual-write Kafka topics; `OPTIMIZER_BASE_URL=http://optimizer-core:8082` |
| `dev/` | `pegasusx-dev` | Single replica, debug logging |

See also: [WS_INGRESS_AFFINITY.md](../../docs/WS_INGRESS_AFFINITY.md)

Ingress manifests live in `infra/k8s/ingress/`:
- `backendconfig.yaml` — REST (120s) and WS (3600s + cookie affinity) BackendConfigs
- `ingress.yaml` — GCE Ingress for `api.pegasusx.app` with HTTPS redirect

Apply BackendConfig before Ingress so the Service annotation resolves.
