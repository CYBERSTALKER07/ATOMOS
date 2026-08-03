# Setup Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Bare route: `SetupWizardShell` in `app/setup/layout.tsx` (no `FactoryShell` chrome).
- Steps: Account (register) → Factory (`/setup/factory`).
- Factory form: `PortalField` / `PortalInput`, `LocationPicker` in `PortalSection`, `portal-btn--primary` footer.
- Desktop: left progress rail; mobile: `setup-mobile-progress`.
