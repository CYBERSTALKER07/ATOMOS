# Worker M2 Working Directory
Assigned: Milestone 2 (Geography, Maps, and Security)
- Enforce H3 resolution 7 in matching writers (`MatchingResolution = 7`, `MatchingH3Cell`), use distinct named field/helper for resolution 9 in settlement/perimeter (`SettlementH3Cell`, `H3CellRes9`).
- Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) to geocode routes in `platformroutes/routes.go` and add country-bias support (`components=country:uz` / `countrycodes=uz` and namespaced cache keys) in `geolocation/`.
- Switch factory fleet list (`GET /v1/factory/fleet` and `HandleFleetVehicles`) in `factory/ios_compat.go` / `factory/service.go` from in-memory mock data to Spanner `Vehicles` joined with `FactoryTruckManifests` and `Drivers`.
- Run tests in `./proximity/...`, `./geolocation/...`, `./order/...`, and `./factory/...` to verify.
