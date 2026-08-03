# Step 8 — amd64 images → Artifact Registry (SSMR) — DONE

**Project:** pegasus-503013  
**Repo:** `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images`  
**Tag:** `ssmr-4a0796fd`  
**Platform:** `linux/amd64`  
**Date:** 2026-07-26

## Images

| Image | Tag | Digest | Cloud Build ID |
|-------|-----|--------|----------------|
| `…/backend-go` | `ssmr-4a0796fd` | `sha256:8f279eba68c1a80cd26b6ec9c76c4729b4d5d2ba7a420ee5bd03ebbefb2b6857` | `bf39194f-5bc6-4046-8dc6-cf03d11307c2` |
| `…/ai-worker` | `ssmr-4a0796fd` | `sha256:550d639781fdf2bef6703f1b38fd9e218146fc2d1c67297ba05a0a7a51b991a9` | `ebf94e28-8b6f-44e8-bdef-efeffe7973f8` |

Full refs:

```
asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-4a0796fd
asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/ai-worker:ssmr-4a0796fd
```

## Config

- `cloudbuild.backend.yaml` — default `_REPO=pegasusx-ssmr-images`
- `cloudbuild.ai-worker.yaml` — added for Step 8

## Notes

- Backend runtime is **debian:bookworm-slim** (glibc / CGO for h3-go).
- Source upload was large (~1.4 GiB); add a tighter `.gcloudignore` later (exclude `node_modules`, app build artifacts, `visuals`, etc.).

## Next (Step 9)

Apply K8s Deployments on `pegasusx-ssmr-gke` / `pegasusx-ssmr` using:

- Image: `…/backend-go:ssmr-4a0796fd`
- Secrets from ESO: `backend-go-secrets` (already SecretSynced)
