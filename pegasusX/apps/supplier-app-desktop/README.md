# Supplier desktop (pegasusX)

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


**Canonical desktop surface:** [`../supplier-portal`](../supplier-portal) (Next.js + Tauri 2), same product role as legacy `admin-portal`.

```bash
cd pegasusX
pnpm install
cd apps/supplier-portal
pnpm run tauri:dev      # desktop
pnpm run tauri:build    # release
```

| Item | Value |
|------|--------|
| API | `http://localhost:8180` (local) |
| Native mobile | `supplier-app-ios`, `supplier-app-android` |

There is no separate `supplier-app-desktop` app tree; this folder is a **discoverability anchor**. `pnpm dev` / `tauri:dev` print the redirect and exit non-zero so CI does not treat an empty app as build success.
