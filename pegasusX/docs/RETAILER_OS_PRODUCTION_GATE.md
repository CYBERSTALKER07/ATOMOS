# Retail OS production gate (Phase 7)

**Policy:** Production may run **CORE-only** retailers early. Declaring multi-scale / enterprise retail ready requires packs **0–6 code complete** and this checklist green.

## Automated

- [ ] `cd apps/backend-go && go test ./retailer/ ./retailerroutes/ -count=1`
- [ ] Capability pack enable/disable tests green (`capability_packs_test`)
- [ ] No retailer client demo supplier id:

```bash
# expect no matches in retailer apps
! grep -R "sup-demo-1" apps/retailer-app-desktop apps/retailer-app-android apps/retailer-app-ios --include='*.tsx' --include='*.ts' --include='*.kt' --include='*.swift' 2>/dev/null
```

## Product surfaces (code complete)

| Pack | Desktop | Android | iOS | Backend |
|------|---------|---------|-----|---------|
| CORE | Wired | Wired | Wired | Wired |
| TEAM | Wired | Wired | Wired | Wired |
| LOCATIONS | Wired | Wired | Wired | Wired |
| STORE_STOCK | Wired | Wired | Wired | Wired |
| POS | Wired | Wired | Wired | Wired |
| SHIFTS | Wired | Wired | Wired | Wired |
| SECTIONS | Wired | Wired | Wired | Wired |
| REPORTS_PRO | Wired | Wired | Wired | Wired |
| CUSTOMER_ASSIST | Wired | Wired | Wired | Wired |
| CT pulse (honest) | Wired | Wired | Wired | Wired |

## Ops / integrity

- [ ] Money remains **int64 minor units** (no float prices in APIs)
- [ ] Spanner DDL P0–P6 applied on target env (or documented pending)
- [ ] Control Tower shows **empty or live** — never mock BarMarks / `sup-demo-*`
- [ ] CORE regression path: auth → catalog → cart → checkout (existing smoke if available)

## Docs present

- [ ] `docs/RETAILER_OS_E2E_MATRIX.md`
- [ ] `docs/REAL_WORLD_CASE_MATRIX.md` filled
- [ ] `docs/ECOSYSTEM_FEATURES_BY_ROLE.md` Part 4 Retail OS sections
- [ ] `context/parity-ledger.md` Retail OS divergences
- [ ] Pack docs `RETAILER_*.md` for stock/pos/shifts/sections/reports/assist

## Close-out (2026-08-02)

- [x] Family → Team: `POST /v1/retailer/family-members/migrate-to-team`
- [x] Auto-order execution (draft): `POST …/auto-order/run` + `GET …/runs`
- [ ] SSMR image includes close-out routes (deploy pending)
- See `docs/RETAILER_OS_CLOSEOUT.md`

## Explicit non-blockers (document only)

- Auto-order full `order.create` place mode (draft is v1)
- Reports inventory without full COGS valuation
- Offline POS sales (online-required v1)
- Planogram vision (aisle/shelf tags only)
