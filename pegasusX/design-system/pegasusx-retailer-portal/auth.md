# Auth Page Overrides

- Use split-panel `auth-shell` from `app/auth/layout.tsx`.
- Forms: `PortalField` + `PortalInput` + `portal-btn--primary`.
- Register stepper: `setup-step-*` classes for multi-step identity.
- Auth splash via `sessionStorage` post-mount only (no hydration mismatch).
