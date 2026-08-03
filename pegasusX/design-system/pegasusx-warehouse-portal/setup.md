# Setup Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Bare route: `SetupWizardShell` in `app/setup/layout.tsx` (no `WarehouseShell` chrome).
- Steps: Account (register) → Location (`/setup/location`).
- Location form: `PortalField` / `PortalInput`, `setup-card`, `setup-footer` with `portal-btn--primary`.
- Desktop: left progress rail; mobile: `setup-mobile-progress`.
