# Supplier desktop (pegasusX)

**Canonical desktop surface:** `pegasusX/apps/supplier-portal` (Next.js + Tauri 2), same product role as legacy `admin-portal`.

```bash
cd pegasusX
pnpm install
cd apps/supplier-portal
pnpm run tauri:dev      # desktop
pnpm run tauri:build    # release
```

API: `http://localhost:8180`. Native mobile row clients: `supplier-app-ios`, `supplier-app-android`.

There is no separate `supplier-app-desktop` codebase; this folder is a discoverability anchor only.
