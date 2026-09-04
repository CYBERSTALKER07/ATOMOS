# Credit risk scoring v1 (G3.B)

**Package:** `apps/backend-go/credit` (`scoring.go`)  
**Flag:** `CREDIT_SCORING_ENABLED` (default **true** when unset)

## Score (0–100, higher = healthier)

| Component | Weight | Inputs |
|-----------|--------|--------|
| Utilization | 35% | `(balance + reserved) / limit` → inverse |
| Delinquency | 25% | `DelinquencyCount` (−15 per count, floor 0) |
| DPD | 25% | Max days past due on open `ArInvoices` (soft-fail if no AR) |
| Pay velocity | 15% | Cleared `OrderCreditReservations` last 90d vs expected |

Status floors: BLACKLISTED → 0 / BLOCK; FROZEN caps score ≤ 30.

## API

| Method | Path | Notes |
|--------|------|-------|
| GET | `/v1/supplier/credit-profiles` | Desk rows include `risk_score`, `risk_tier`, `scoring_enabled` |
| GET | `/v1/supplier/credit-scores?retailer_ids=` | Full score objects + factors_json |

## Auto-hold (optional)

```
CREDIT_SCORE_AUTO_HOLD_ENABLED=true
CREDIT_SCORE_AUTO_HOLD_MAX=25
```

Desk marks `needs_attention` when score ≤ max. Automatic HoldRelationship on score is **not** force-on — use dual-control dunning CREDIT_HOLD path for production freezes unless product enables a separate worker.

## Honesty

G1.C removed empty score pretence. G3.B fills real algorithm-backed scores only when enabled; never invent random numbers.
