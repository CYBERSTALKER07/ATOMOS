# Kustomize overlays

Render production manifests:

```bash
kubectl kustomize infra/k8s/base --load-restrictor LoadRestrictionsNone
# or
kubectl kustomize infra/k8s/overlays/prod --load-restrictor LoadRestrictionsNone
```

| Overlay | Namespace | Notes |
|---------|-----------|-------|
| `prod/` | `pegasusx` | Full HA: 3+ API replicas, worker split, PDB/HPA |
| `staging/` | `pegasusx-staging` | Dual-write Kafka topics enabled for consumer migration |
| `dev/` | `pegasusx-dev` | Single replica, debug logging |

See also: [WS_INGRESS_AFFINITY.md](../../docs/WS_INGRESS_AFFINITY.md)

Ingress manifests live in `infra/k8s/ingress/`:
- `backendconfig.yaml` — REST (120s) and WS (3600s + cookie affinity) BackendConfigs
- `ingress.yaml` — GCE Ingress for `api.pegasusx.app` with HTTPS redirect

Apply BackendConfig before Ingress so the Service annotation resolves.
