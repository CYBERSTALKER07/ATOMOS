# Setup page overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Use `SetupWizardShell` split rail — do not wrap in `PageChrome`.
- Sections: `SetupSection` / `setup-card`; fields share tokens with `portal-field`.
- Progress: left rail step list (desktop) + mobile progress bar.
- Footer: `SetupFooter` with skip allowed on business/billing.
