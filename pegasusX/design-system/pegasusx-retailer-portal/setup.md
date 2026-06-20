# Setup Page Overrides

- Bare route under `SetupWizardShell` (Account → Tax → Address).
- Steps: `/setup/tax`, `/setup/address`; `/setup` redirects to tax.
- `PortalField` / `PortalInput` / `FormAlert` / `portal-btn`.
- Pass tax ID via `sessionStorage` between tax and address steps.
