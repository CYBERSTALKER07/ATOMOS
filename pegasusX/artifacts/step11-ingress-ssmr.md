# Step 11 — Ingress + DNS + TLS (SSMR)

**Date:** 2026-07-27  
**Cluster:** `pegasusx-ssmr-gke` / project `pegasus-503013`  
**Namespace:** `pegasusx-ssmr`

## Verdict

| Item | Status |
|------|--------|
| Global static IP | **PASS** — `pegasusx-ssmr-api-ip` = **136.69.43.141** |
| GCE Ingress | **PASS** — `pegasusx-ssmr-api` ADDRESS assigned |
| BackendConfig + NEG health | **PASS** — HC path `/healthz:8080`, backend **HEALTHY** |
| HTTP external smoke | **PASS** — `/healthz`, `/ready`, client-policy **200** |
| HTTPS (self-signed interim) | **PASS** — `curl -k` / `--resolve` → **200** |
| ManagedCertificate (Google) | **PENDING DNS** — status `Provisioning` for `api-ssmr.pegasusx.app` |
| Public DNS A record | **USER ACTION** — domain not configured in this project |

## Endpoints

| | |
|--|--|
| Host | `api-ssmr.pegasusx.app` |
| Static IP | `136.69.43.141` |
| `PUBLIC_BASE_URL` (ConfigMap) | `https://api-ssmr.pegasusx.app` |

### Smoke (until real DNS)

```bash
IP=136.69.43.141
HOST=api-ssmr.pegasusx.app

# HTTP
curl -fsS --resolve "${HOST}:80:${IP}" "http://${HOST}/healthz"
curl -fsS --resolve "${HOST}:80:${IP}" "http://${HOST}/ready"

# HTTPS (self-signed — use -k)
curl -fskS --resolve "${HOST}:443:${IP}" "https://${HOST}/healthz"
curl -fskS --resolve "${HOST}:443:${IP}" \
  "https://${HOST}/v1/platform/client-policy?role=DRIVER&platform=ios&version=1.0.0"
```

Optional `/etc/hosts`:

```
136.69.43.141  api-ssmr.pegasusx.app
```

## What was applied

Manifests under `infra/k8s/overlays/ssmr/`:

| File | Purpose |
|------|---------|
| `backendconfig.yaml` | REST 120s + WS 3600s; HC `/healthz` |
| `service-ws.yaml` | `backend-go-ws` for `/v1/ws` |
| `frontendconfig.yaml` | HTTPS redirect **off** until managed cert Active |
| `managed-certificate.yaml` | Google-managed cert for host |
| `ingress.yaml` | GCE Ingress + static IP + TLS secret |

Also:

- Secret `pegasusx-ssmr-api-tls` (self-signed, 90 days) for immediate HTTPS
- Service annotations on `backend-go`: BackendConfig + NEG

## DNS cutover (you do this once domain is owned)

At your DNS host for `pegasusx.app` (or Cloud DNS zone):

```
api-ssmr.pegasusx.app.  300  IN  A  136.69.43.141
```

Then wait until:

```bash
kubectl -n pegasusx-ssmr get mcrt pegasusx-ssmr-api-cert
# DomainStatus=Active, CertificateStatus=Active
```

Then enable redirect:

```bash
kubectl -n pegasusx-ssmr patch frontendconfig pegasusx-ssmr-frontend --type merge \
  -p '{"spec":{"redirectToHttps":{"enabled":true,"responseCodeName":"MOVED_PERMANENTLY_DEFAULT"}}}'
```

## Notes / gotchas fixed during bring-up

1. **Root `/` returns 404** — BackendConfig health path must be `/healthz` (applied).
2. **ManagedCertificate without DNS** delayed full LB provisioning; detached briefly, used self-signed secret TLS, re-attached managed cert for future activation.
3. **Rolling restart CPU** — prefer pod delete over surge restart on this 3-node cluster (IN_USE regional addresses 3/4).
4. Host header required — Ingress rules match `api-ssmr.pegasusx.app` only (use `--resolve` or `/etc/hosts`).

## Next

**Step 12** — Firebase phone OTP + FCM (real project credentials).  
**Or** complete DNS for Google-managed cert + HTTPS redirect.
