# Sustainability & Risk (Technical Notes)

## Carbon

```
co2_kg ≈ distance_km * vehicle_factor * (1 / max(load_factor, ε))
```

Store as integer grams if needed for reporting. Optimizer multi-objective weights: cost, service, carbon.

## Risk score

```
risk = f(credit_tier, claim_velocity, shop_closed_rate, driver_score, external_signals)
```

Actions: freeze credit, force cash-only, require photo, raise claim scrutiny.

## Collaboration / scorecards

Supplier scorecard metrics: on-time, fill rate, claim rate, fiscal compliance, capacity reliability → planning priority weight.
