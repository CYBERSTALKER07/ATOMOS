# Frontend Apps — Codebase Status

> **Re-aligned 2026-08-12.** Inventory SoT: [`docs/FEATURES_BY_APP_ROLE.md`](../docs/FEATURES_BY_APP_ROLE.md).  
> There is **no** `supplier-app-desktop` — supplier desktop is `supplier-portal` (Next.js + Tauri 2).  
> Ops residuals: [`docs/PROD_READINESS_SEQUENCE.md`](../docs/PROD_READINESS_SEQUENCE.md).

| App | Role | Stack notes |
|-----|------|-------------|
| `admin-portal` | PLATFORM_ADMIN | **Live** Next console: Tenants / Flags (dual-control) / Audit / Match / Partner + MFA + WS |
| `supplier-portal` | SUPPLIER | Next + Tauri 2; planning `/planning`; finance/ops |
| `supplier-app-android` | SUPPLIER | Kotlin Compose |
| `supplier-app-ios` | SUPPLIER | SwiftUI |
| `retailer-app-desktop` | RETAILER | Next + Tauri 2 + SQL plugin |
| `retailer-app-android` | RETAILER | HQ / Credit-AR / CT |
| `retailer-app-ios` | RETAILER | HQ / Credit / CT |
| `warehouse-portal` | WAREHOUSE | Next + Tauri 2; WMS bins/pick-waves/cycle/cold/labor |
| `warehouse-app-android` | WAREHOUSE | Transfer Actions + cold/labor |
| `warehouse-app-ios` | WAREHOUSE | Transfer Actions + cold/labor |
| `factory-portal` | FACTORY | Next + Tauri 2 |
| `factory-app-android` | FACTORY | Compose |
| `factory-app-ios` | FACTORY | SwiftUI |
| `payload-terminal` | PAYLOAD | Expo SoT seal board + factory loading-bay |
| `payload-app-android` | PAYLOAD | Native peer |
| `payload-app-ios` | PAYLOAD | Native peer |
| `driver-app-android` | DRIVER | Offline queue + delivery |
| `driver-app-ios` | DRIVER | Offline queue + delivery |
| `marketing-site` | Marketing | Not ops |

**Removed paths:** `supplier-app-desktop` (never ship; use supplier-portal Tauri).
