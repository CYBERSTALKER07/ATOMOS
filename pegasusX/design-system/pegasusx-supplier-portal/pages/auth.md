# Auth page overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Keep split `auth-shell` layout (brand panel + form panel).
- Form controls: `PortalField` / `PortalInput` with 44px min height.
- Register wizard stepper matches setup rail step badge styling.
- Buttons: `portal-btn--primary` / `portal-btn--ghost`.
- Brand subtitle may use `--font-garamond` for accent only.
