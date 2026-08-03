# Setup Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Bare route under `SetupWizardShell` (Account → Tax → Address).
- Steps: `/setup/tax`, `/setup/address`; `/setup` redirects to tax.
- `PortalField` / `PortalInput` / `FormAlert` / `portal-btn`.
- Pass tax ID via `sessionStorage` between tax and address steps.
