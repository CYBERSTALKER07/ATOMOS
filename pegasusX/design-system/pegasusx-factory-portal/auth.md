# Auth Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Split `auth-shell` layout with inline SVG `BrandLogo` (no `next/image` logos).
- Forms use `PortalField`, `PortalInput`, `PortalSelect`, `FormAlert`.
- Actions use `portal-btn portal-btn--primary`.
- Register stepper uses `setup-step-*` classes from `setup-onboarding.css`.
- Auth splash uses `sessionStorage` key `auth-splash-shown` (post-mount only).
