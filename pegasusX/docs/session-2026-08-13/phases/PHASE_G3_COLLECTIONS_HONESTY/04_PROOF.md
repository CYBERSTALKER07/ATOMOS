# G3 Proof — Collections, credit, client honesty

## Tests

```text
go test ./credit/ ./ar/ ./retailer/ ./supplier/ -count=1
# all ok
```

## Evidence

| ID | Evidence |
|----|----------|
| G3-A1 | `ar/dunning_channels.go` transports; bootstrap MultiChannelNotify; `HandleDunningStatus` |
| G3-B1 | `credit/scoring.go` + `scoring_test.go`; desk `HandleListSupplierProfiles` risk_score; `GET /v1/supplier/credit-scores` |
| G3-C1 | `retailer-app-desktop/.../settings/page.tsx` — settlement theatre removed; notif mute local-only copy |
| G3-C2 | `retailer/core_handlers.go` attachLiveLocations LAST_KNOWN; tests updated |
| G3-C3 | `POST /v1/retailer/pos/scan` (`HandlePOSScan`) |
| G3-D1 | `supplier/portal_handlers.go` earnings 503 with code; portal ledger_fallback banner |

## Scoring formula (g3_v1)

util 35% + delinquency 25% + DPD 25% + pay velocity 15% → score 0–100; tier LOW/MEDIUM/HIGH/BLOCK.

Flags: `CREDIT_SCORING_ENABLED` (default on), optional `CREDIT_SCORE_AUTO_HOLD_*`.
