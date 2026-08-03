# Auth Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Use split-panel `auth-shell` from `app/auth/layout.tsx`.
- Forms: `PortalField` + `PortalInput` + `portal-btn--primary`.
- Register stepper: `setup-step-*` classes for multi-step identity.
- Auth splash via `sessionStorage` post-mount only (no hydration mismatch).
