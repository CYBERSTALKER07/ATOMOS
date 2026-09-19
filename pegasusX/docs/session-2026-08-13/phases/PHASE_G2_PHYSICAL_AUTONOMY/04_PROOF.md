# G2 Proof

## Backend tests (2026-08-13)

```text
go test ./stocklots/ ./payload/ ./warehouse/ ./events/ ./featureflags/ ./factory/ -count=1
# all ok
```

## Evidence map

| Gap | Evidence |
|-----|----------|
| G2-D1 Option B | `docs/MANIFEST_DUAL_PLANE.md`; `events.ManifestEvent.ManifestDomain`; factory/payload/depart emit domain |
| G2-A1 seal-class flags | `stocklots/flag.go` Effective*; `featureflags` dual-control WMS_* + load/labor; bootstrap `SetFlagEvaluator` |
| G2-A2 outbox density | Putaway/pick confirm/cycle approve already emit `emitWMSEvent` (handlers.go) — residual documented not silent-fail |
| G2-B1 load ledger | DDL `20260813_g2_manifest_load_ledger.ddl`; `stocklots/load_ledger.go`; payload handlers + routes; seal via `assertPhysicalSealGates` |
| G2-C1 cold baseline | `stocklots/cold_baseline.go` in physical seal gates |
| G2-C2 labor | `stocklots/labor_gate.go` + warehouse `ExecuteDispatch` |
| G2-E1 place | Existing soak/dual-control path retained; no prod place flip without artifact (honesty) |

## Client residual

- Payload Android/iOS/terminal: local checklist still used for UX; backend gate is authoritative when `PAYLOAD_LOAD_LEDGER_ENABLED`. Full client API rewire residual (anti-drift next pass).
- Honesty comment on Android `HomeViewModel.checkedItems`.

## Markers / ops

- Migrations: apply load ledger DDL before flag-on  
- Ops: `docs/WMS_GATE4_OPS.md` seal-class section  
