# pegasusX RETAILER Role — Phased Execution Ledger

**Scope:** pegasusX only · **Parent plan:** `VEGETABLE_PLAN.md` §2.5  
**Last updated:** 2026-06-14

## Status model

`TODO` → `IN_PROGRESS` → `WIRED` → `E2E_SSMR_GREEN` → `PROD_CANDIDATE`

---

## Phase RT-1 — Receiving window parity (cross-client)

| ID | Feature | Backend | Desktop | Android | iOS | Status |
|----|---------|---------|---------|---------|-----|--------|
| RT1-01 | Profile GET/PUT receiving windows | `PUT /v1/retailer/profile` + `proximity.ValidateReceivingWindow` | `/settings` edit form | `AccountProfileViewModel` | `AccountProfileView` | **E2E_SSMR_GREEN** (`PX_E2E_RETAILER_RECEIVING_WINDOW_OK`) |
| RT1-02 | Registration capture | `POST /v1/auth/retailer/register` | `/auth/register` wizard | `AuthScreen` | registration flow | **WIRED** |
| RT1-03 | Shared contracts | `RetailerProfileResponse` / `RetailerProfileUpdateRequest` | `lib/types.ts` + `lib/receiving-window.ts` | `Models.kt` | `AppModels.swift` | **WIRED** |

**Exit:** Retailer can set receiving windows at registration and edit later on every client surface; dispatch SLA reads snapshotted windows on new orders.

---

## Verification

```bash
cd pegasusX/apps/backend-go && go test ./retailer/...
cd pegasusX && make test-ssmr-infra   # PX_E2E_RETAILER_RECEIVING_WINDOW_OK
```

---

## Next execution batch

1. ~~Receiving window desktop edit + validation + SSMR~~ — **E2E_SSMR_GREEN**
2. **Cross-role next** — Boss-picked role row per `VEGETABLE_PLAN.md` §3
