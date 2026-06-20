# Auth Page Overrides

- Split `auth-shell` layout with inline SVG `BrandLogo` (no `next/image` logos).
- Forms use `PortalField`, `PortalInput`, `PortalSelect`, `FormAlert`.
- Actions use `portal-btn portal-btn--primary`.
- Register stepper uses `setup-step-*` classes from `setup-onboarding.css`.
- Auth splash uses `sessionStorage` key `auth-splash-shown` (post-mount only).
