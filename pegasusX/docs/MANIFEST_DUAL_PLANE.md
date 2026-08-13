# Manifest dual plane (G2.D Option B)

**Decision (2026-08-13):** Keep **two durable tables** with explicit domains. Do **not** merge into a single SoT in G2.

| Domain | Table | Service | Meaning |
|--------|-------|---------|---------|
| **FACTORY** | `FactoryTruckManifests` | `factory/*` | Factory → warehouse **transfer** loading bay |
| **SUPPLIER** | `SupplierTruckManifests` | `payload/*`, dispatch | Supplier **delivery** truck (payload seal / driver depart) |

## Events

Shared type names (`MANIFEST_SEALED`, …) remain for Kafka routing, but every new emit sets:

```json
"manifest_domain": "FACTORY" | "SUPPLIER"
```

Field: `events.ManifestEvent.ManifestDomain` (`events.ManifestDomainFactory` / `ManifestDomainSupplier`).

Consumers must not assume one table when handling `MANIFEST_*` — branch on `manifest_domain` (or `factory_id` vs delivery `warehouse_id`/`driver_id` heuristics for legacy events without the field).

## Clients

Payload terminal / Android / iOS intentionally dual-call:

- Factory loading-bay APIs for transfer manifests  
- Payloader seal/inject for delivery trucks  

UI copy should say **Factory transfer** vs **Delivery truck** — never claim a single SoT.

## Explicit non-goals (G2)

- Full dual-write / single-table merge (Option A)  
- Shared Spanner transaction across factory seal + payload seal  

Option A remains a future program item if product requires it.
