---
name: role-row-clients
description: Role-row client parity across Android, iOS, portal/desktop/terminal for each role.
---

# Role-row clients

## Matrix (required surfaces)

| Role | Clients |
|------|---------|
| Supplier | portal (web+Tauri), Android, iOS |
| Retailer | desktop (Next+Tauri), Android, iOS |
| Driver | Android, iOS |
| Warehouse | portal (web+Tauri), Android, iOS |
| Factory | portal (web+Tauri), Android, iOS |
| Payload | terminal (Expo primary), Android, iOS |
| Platform admin | admin-portal (web, thin) |

## Parity rules

1. Shared contracts first: `packages/types`, `packages/api-client`, then clients.
2. Feature not done until all row clients use the same contract (or deferred in gap register).
3. Desktop stack: **Next + Tauri 2** — do not recommend Electron migration without hard blocker.
4. Payload natives are historically thin — terminal is SoT until natives catch up.
5. Known holes: retailer AR/HQ mobile; supplier planning web; factory↔payload loop.

## Nav evidence

- Shells: `*Shell.tsx`
- Android: `*Section.kt`, `*Navigation.kt`, `DriverRoutes`
- iOS: `ContentView` / `Screens/`
- Features inventory: `docs/FEATURES_BY_APP_ROLE.md`
