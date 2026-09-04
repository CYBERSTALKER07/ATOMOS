# Manual Critical-Path Walkthroughs (Staging)

Evidence: screenshot + API ids in run log. Complete before enabling `CASH_RECONCILIATION_REQUIRED`.

## 4a — Driver cash reconciliation

1. Complete a cash delivery on staging (or use seeded shift with expected cash).
2. Driver Android/iOS: return-to-warehouse → enter declared cash (intentional mismatch).
3. Supplier web: `/treasury/cash-reconciliations` → accept mismatch (or `/exceptions` resolve).
4. Driver: `return-complete` succeeds.
5. Record: `reconciliation_id`, driver id, timestamps.

**Automated counterpart:** `PX_E2E_CASH_RECON_OK`, `PX_E2E_EXCEPTION_RESOLVE_OK` in `ssmr-smokecheck`.

## 4b — Warehouse reverse logistics

1. Supplier: issue credit note (draft → issue) for a completed order.
2. Warehouse portal: `/returns` → reverse task → receive with SKU qty map.
3. Verify: task `RECEIVED`, inventory increased, compliance `openReverseLogisticsTasks` decreases.

**Automated counterpart:** `PX_E2E_CREDIT_NOTE_OK`, `PX_E2E_REVERSE_LOGISTICS_OK`.

## 4c — Supplier finance desk loop (web)

1. `/credit/collections` — AR / dunning list loads (**no score columns** — credit risk scoring removed Phase A).
2. `/compliance` — deep links to cash recon and credit notes.
3. `/exceptions` — resolve cash discrepancy, issue credit note draft, unfreeze credit (one per kind).
4. `/settings/notification-preferences` — toggle event; confirm inbox formatting.

**Automated counterpart:** notification prefs + exception resolve markers in gap_closure E2E.

## Sign-off template

| Path | Operator | Date | Pass | Notes |
|------|----------|------|------|-------|
| 4a Driver cash | | | | |
| 4b Warehouse reverse | | | | |
| 4c Finance desk | | | | |
