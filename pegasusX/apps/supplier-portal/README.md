# supplier-portal

Next.js 15 + React 19 supplier portal (web + Tauri 2 desktop shell). Pegasus equivalent: `admin-portal`. Renamed for clarity — the user is a SUPPLIER; "admin" was only a legacy JWT name.

Bootstrap onboarding wizard (per `../../context/ui-design.md`):
1. Account
2. Topology (factories + warehouses)
3. Business
4. Categories
5. Billing gate (`/setup/billing`)
6. Dashboard

Material 3 via Tailwind v4 + hand-rolled M3 CSS tokens. No `@material/web` Lit components. No emoji icons.
