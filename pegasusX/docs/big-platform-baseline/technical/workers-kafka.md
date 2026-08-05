# Workers & Kafka Design

## Processes

| Process | Run mode | Duties |
|---------|----------|--------|
| backend-go API | `api` | HTTP/WS; writes Spanner+outbox; **no** outbox relay |
| backend-go-worker | `worker` | Outbox relay, order mutator, warehouse mutator, notifications, reconcilers |
| ai-worker | separate | Import, freeze registry, demand jobs, optional synthesis |
| optimizer-core | separate Deployment (port 8082) | OR-Tools VRP for supplier/WH dispatch; **not** in SSMR overlay; prod `replicas: 0` until AR image |

## Consumer groups (existing + planned)

| Group | Handles |
|-------|---------|
| void-order-mutator | FISCAL_RECEIPT_REQUESTED, payment cleared/failed, … |
| void-notification-dispatcher | Fanout inbox/FCM |
| warehouse mutator | Supply/transfer side effects |
| ai-import / freeze | Inventory import, freeze locks |
| (new) planning-demand | DEMAND_SIGNAL materialization |
| (new) playbook-runner | Disruption actions |

## Topics

Keep SSMR topics; add:

- `*.planning.demand`  
- `*.planning.scenarios`  
- `*.compliance.audit` (optional)  
- logistics exceptions already present  

## Idempotency

- Dedup store on (group, event_id)  
- Fiscal attempt_id single success  
- Never apply fiscal twice to COMPLETED  

## Known gap (from live smoke)

Orders can stick in FISCALIZING if fiscal event path lags — investigate outbox publish of `FISCAL_RECEIPT_REQUESTED` + consumer health before Phase 1 Soliq.
