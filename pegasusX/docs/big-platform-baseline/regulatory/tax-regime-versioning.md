# Versioned Tax & Regime Engine

## Goal

Tax rates, simplified regimes, VAT thresholds are **versioned objects**. Every COMPLETED order line stores the regime version so history survives mid-flight law changes.

## Model sketch

```
TaxRegimeVersion {
  id, effective_from, effective_to,
  vat_rates[], simplified_rules, currency
}
OrderLineFiscalSnapshot {
  order_id, line_sku, regime_version_id,
  taxable_minor, vat_minor, total_minor
}
```

## Rules

- Planning what-if may use **future** regimes.  
- Execution completion stamps **regime at COMPLETED time**.  
- Reports group by regime version for Soliq export.

## Edge cases

| Case | Rule |
|------|------|
| Regime overlap | Prefer highest effective_from ≤ completed_at |
| Missing regime | Block COMPLETED in production fiscal mode |
