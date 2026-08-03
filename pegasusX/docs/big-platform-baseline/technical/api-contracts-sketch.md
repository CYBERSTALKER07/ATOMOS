# API Contracts Sketch

Extend `packages/types` + `packages/api-client` + `*routes` in the same PR as handlers.

## New / extended surfaces (indicative)

| Method | Path | Role | Purpose |
|--------|------|------|---------|
| GET | `/v1/planning/demand-signals` | supplier, warehouse | Causal demand |
| POST | `/v1/planning/scenarios` | supplier admin | Clone scenario |
| POST | `/v1/planning/scenarios/{id}/run` | supplier admin | Re-optimize |
| POST | `/v1/planning/scenarios/{id}/publish` | supplier admin | Baseline |
| GET/POST | `/v1/planning/meio/recommendations` | supplier | MEIO outputs |
| POST | `/v1/delivery/partial-offload` | driver | Partial accept |
| POST | `/v1/delivery/settlement-unlock/check` | driver | Proximity unlock |
| GET | `/v1/compliance/fiscal-open` | admin | Audit |
| GET | `/v1/compliance/force-completes` | admin | Audit |
| POST | `/v1/cart-sessions` | retailer | Multi-supplier parent |
| POST | `/v1/cart-sessions/{id}/checkout` | retailer | Split orders |
| GET/POST | `/v1/playbooks/*` | admin | Disruption playbooks |
| POST | `/v1/warehouse/tasks/*` | warehouse | WMS tasks |
| GET | `/v1/tms/eta/{orderId}` | multi | Predictive ETA |

## Contract rules

- All money: `*_minor` int64 + `currency`  
- All lists: supplier/retailer scoped; IDOR tests  
- Events: add to `events.schema.json` via gen-contracts  
