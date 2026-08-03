# Gap Closure — Phase 1 Staging Foundation

## Preconditions

- Branch `main` includes gap-closure routes (`cashreconroutes`, `creditnoteroutes`), workers, and supplier-portal pages.
- Flags off on staging: `CASH_RECONCILIATION_REQUIRED=false`, `CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false`, `CREDIT_NOTE_AUTO_FROM_CLAIM=false`, `CREDIT_SCORE_ENFORCEMENT_ENABLED=false`.

## Verify build

```bash
cd apps/backend-go && go build . && go test ./cashrecon/... ./creditnote/... ./credit/... ./analytics/... ./replenishment/... ./notifications/... -count=1
```

## Spanner migrations (staging)

Apply if not already live:

- `schema/migrations/20260803_phase_a_reconciliation.ddl`
- `schema/migrations/20260804_phase_c_credit_risk.ddl`
- `schema/migrations/20260804_phase_c_reorder.ddl`
- `schema/migrations/20260804_phase_d_analytics.ddl`
- `schema/migrations/20260804_phase_d_notifications.ddl`

```bash
make phase0-migrate   # or your staging Spanner DDL job
```

## Redeploy (staging only)

1. `backend-go`
2. `supplier-portal`

Sync Secret Manager / env from `.env.example` for the four flags above.

## Smoke (flags off)

| Surface | Check |
|---------|--------|
| `/compliance` | Dashboard loads |
| `/credit/collections` | List with score columns |
| `/exceptions` | Merged finance exceptions |
| `/finance/credit-notes` | List + create |
| `/treasury/cash-reconciliations` | List |
| `/analytics/route-performance` | Page loads (may be empty) |
| `/settings/notification-preferences` | GET/PATCH |

No new Kafka topics required — outbox relay on `pegasusx-main` is sufficient.
