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
kubectl kustomize infra/k8s/overlays/cells/uz --load-restrictor LoadRestrictionsNone
kubectl kustomize infra/k8s/overlays/cells/eu --load-restrictor LoadRestrictionsNone
```

| Overlay | Namespace | Notes |
|---------|-----------|-------|
| `prod/` | `pegasusx` | HA API/worker + ai-worker; digest-pinned images; ManagedCertificate TLS; **ExternalSecret included**; optimizer-core `replicas: 0` until real AR image |
| `staging/` | `pegasusx-staging` | Dual-write Kafka topics; remaps optimizer image; live only if image published + deployed |
| `ssmr/` | `pegasusx-ssmr` | Cloud cutover; **optimizer-core included, `replicas: 1`** (heuristic fallback if pod/image unhealthy); prod keeps `replicas: 0` until AR publish |
| `dev/` | `pegasusx-dev` | Single replica, debug logging |
| `cells/uz/` | `pegasusx` | **GS-C1** named UZ cell (merges `prod/`). `HOME_CELL=cell-uz`, `DEFAULT_MARKET_CODE=UZ`, fiscal as today. Env overlays are not cells. Do not apply from the catalog. |
| `cells/eu/` | `pegasusx` | **GS-C3** named EU cell (merges `base/`, not prod). `HOME_CELL=cell-eu`, pack EU/planned, commercial fiscal, empty adapters OK, Spanner `pegasusx-cell-eu`. Do not apply from the catalog. |

**Optimizer + routing runtime SoT:** [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md)

See also: [WS_INGRESS_AFFINITY.md](../../docs/WS_INGRESS_AFFINITY.md)

Ingress manifests live in `infra/k8s/ingress/`:
- `backendconfig.yaml` — REST (120s) and WS (3600s + cookie affinity) BackendConfigs
- `ingress.yaml` — GCE Ingress for `api.pegasusx.app` with HTTPS redirect

Apply BackendConfig before Ingress so the Service annotation resolves.
