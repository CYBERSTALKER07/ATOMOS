# Google Maps Geolocation & Places Integration Plan

## Goal
Integrate Google Maps & Places backend proxy with Redis caching into `pegasus.x` so suppliers and warehouses can perform live address autocomplete, place details resolution, draggable pin geocoding, and location sharing restricted to Uzbekistan.

## Tasks
- [x] Task 1: Add `GoogleMapsAPIKey` to `backend/internal/config/config.go` and `.env.example` → Verify: `go build ./...` compiles with config struct field.
- [x] Task 2: Create `backend/internal/geolocation/models.go` with `ResolvedLocation`, `AutocompletePrediction`, and DTOs → Verify: Structs serialize cleanly to/from JSON.
- [x] Task 3: Implement `backend/internal/geolocation/service.go` with Google Places Autocomplete, Place Details, Geocoding, and Reverse Geocoding (`country:uz` filter, OpenStreetMap fallback, Redis 24h/7d cache) → Verify: Unit tests pass with mock HTTP server.
- [x] Task 4: Implement `backend/internal/geolocation/handlers.go` with Chi endpoints `/v1/platform/geocode/autocomplete`, `/place`, `/reverse`, `/forward` enforcing authenticated claims → Verify: Handlers return expected JSON structure and reject unauthenticated requests.
- [x] Task 5: Register and mount routes in `backend/internal/api/router.go` under `s.geolocationSvc` → Verify: `curl` to endpoints returns 200 on local test server.
- [x] Task 6: Add step in `backend/cmd/smokecheck/main.go` verifying geocoding lifecycle (autocomplete -> place -> reverse) → Verify: `make smokecheck` passes all living loop checks.

## Done When
- [x] All 4 geolocation endpoints are live and responding in `pegasus.x`.
- [x] Redis caching prevents redundant external calls for identical queries.
- [x] Uzbekistan bounds (`country:uz`) are strictly enforced.
- [x] `go test ./...` and `make smokecheck` execute cleanly with zero errors.

## Notes
- Desktop apps (Tauri v2 / Next.js) will consume `/v1/platform/geocode/*` instead of holding Google API keys client-side.
- Reverse geocoding provides street names for the "Share My Location" (GPS) feature.
