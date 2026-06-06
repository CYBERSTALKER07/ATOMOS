# Topology Entry Support Guide (B02)

## Purpose
Guide support and operations teams through safe entry and troubleshooting of supplier topology data.

## Authority
- Read: `GET /api/supplier/topology`
- Write: `PUT /api/supplier/topology`
- Backend owner: supplier service (`HandleTopology`)

## Required payload structure
`PUT /api/supplier/topology` requires JSON:

```json
{
  "warehouses": [
    {
      "warehouse_id": "optional-id",
      "name": "Warehouse Tashkent Central",
      "lat": 41.3111,
      "lng": 69.2797,
      "coverage_radius_km": 10,
      "is_active": true,
      "is_on_shift": true
    }
  ],
  "factories": [
    {
      "factory_id": "optional-id",
      "name": "Factory North",
      "lat": 41.35,
      "lng": 69.32,
      "is_active": true
    }
  ]
}
```

## Validation rules
1. `warehouses` must contain at least one item.
2. Warehouse name is required and must be non-empty after trim.
3. Warehouse latitude must be between `-90` and `90`.
4. Warehouse longitude must be between `-180` and `180`.
5. Factory name is required when a factory item is provided.
6. Factory latitude/longitude must use the same geographic bounds.
7. If `coverage_radius_km` is omitted or non-positive, backend defaults to `10.0` km.
8. If `is_active` or `is_on_shift` are omitted, backend defaults to `true`.

## Common errors and fixes

| Error value | Cause | Fix |
|---|---|---|
| `invalid_json` | Body cannot be parsed | Correct JSON format and retry |
| `warehouses_required` | Empty or missing `warehouses` | Add at least one warehouse |
| `warehouses[i].name_required` | Warehouse name blank | Provide non-empty name |
| `warehouses[i].lat_out_of_range` | Latitude invalid | Use value in `[-90, 90]` |
| `warehouses[i].lng_out_of_range` | Longitude invalid | Use value in `[-180, 180]` |
| `factories[i].name_required` | Factory name blank | Provide non-empty name |
| `factories[i].lat_out_of_range` | Latitude invalid | Use value in `[-90, 90]` |
| `factories[i].lng_out_of_range` | Longitude invalid | Use value in `[-180, 180]` |
| `persist_supplier_topology_failed` | Backend persistence failed | Escalate with request/response evidence |

## Post-write verification
1. Run `GET /api/supplier/topology` and verify warehouse/factory entries match submitted values.
2. Run `GET /api/supplier/profile` to confirm supplier context is still readable.
3. Confirm supplier portal pages depending on topology can load without redirect loops.

## Escalation criteria
1. Repeated `500` on topology read/write.
2. Topology write returns success but read immediately omits persisted rows.
3. Support can reproduce deterministic validation failure with payload that satisfies documented constraints.
