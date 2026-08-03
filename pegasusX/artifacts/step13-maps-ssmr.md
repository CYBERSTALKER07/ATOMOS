# Step 13 — Maps API key + geocode (SSMR)

**Date:** 2026-07-27  
**Project:** `pegasus-503013`  
**Namespace:** `pegasusx-ssmr`

## Verdict

| Item | Status |
|------|--------|
| Maps Platform APIs | **PASS** (already enabled) |
| API Keys API | **PASS** — enabled |
| Real API key created | **PASS** — display name `pegasusx-ssmr-maps` |
| Key restricted to Maps services | **PASS** — geocoding + places + maps-backend |
| GSM `pegasusx-ssmr-google-maps-api-key` | **PASS** — version 2 (replaced mock) |
| K8s secret / pod env | **PASS** — no longer `mock-staging-…` |
| Reverse geocode | **PASS** — Tashkent 41.2995,69.2401 |
| Forward geocode | **PASS** — Amir Temur Avenue |
| Place details | **PASS** — by `place_id` |
| Autocomplete | **PASS** — `?input=` (uses Places `types=address`) |

## Key resource

```
projects/1002695564567/locations/global/keys/b896070e-9f9a-4cdd-b795-8b09ae07d2ad
displayName: pegasusx-ssmr-maps
```

API targets:

- `geocoding-backend.googleapis.com`
- `places-backend.googleapis.com`
- `places.googleapis.com`
- `maps-backend.googleapis.com`

Stored in GSM secret `pegasusx-ssmr-google-maps-api-key` and K8s key `google-maps-api-key` on `backend-go-secrets` (env `GOOGLE_MAPS_API_KEY`).

## Smoke (Ingress)

```bash
HOST=api-ssmr.pegasusx.app
IP=136.69.43.141   # or current Ingress ADDRESS

# reverse
curl -fsS --resolve "${HOST}:80:${IP}" \
  "http://${HOST}/v1/platform/geocode/reverse?lat=41.2995&lng=69.2401"

# forward
curl -fsS --resolve "${HOST}:80:${IP}" \
  -H 'Content-Type: application/json' \
  -d '{"address":"Amir Temur Avenue, Tashkent, Uzbekistan"}' \
  "http://${HOST}/v1/platform/geocode/forward"

# autocomplete (param name is input, not q)
curl -fsS --resolve "${HOST}:80:${IP}" \
  --get --data-urlencode 'input=Amir Temur Tashkent' \
  "http://${HOST}/v1/platform/geocode/autocomplete"

# place
curl -fsS --resolve "${HOST}:80:${IP}" \
  --get --data-urlencode 'place_id=PLACE_ID_FROM_REVERSE' \
  "http://${HOST}/v1/platform/geocode/place"
```

Sample reverse body (abridged):

```json
{
  "address": "76XR+R25, Tashkent, Uzbekistan",
  "lat": 41.2995,
  "lng": 69.2401,
  "place_id": "ChIJ_ZtQTQCLrjgRkZRgfZRE6sQ",
  "formatted_address": "76XR+R25, Tashkent, Uzbekistan"
}
```

## App wiring

Backend already injects `GOOGLE_MAPS_API_KEY` from ESO/GSM (`deployment.yaml`).  
Clients use the platform geocode routes above; Redis caches geocode results when available.

## Hardening (optional follow-ups)

1. Add HTTP referrer / IP / Android+iOS app restrictions on the API key in Console.
2. Rotate key if it was ever pasted into chat/logs (Cloud Build/console only recommended).
3. Billing alerts for Maps Platform (Geocoding + Places usage).

## Next

**Step 14** — Global Pay staging webhooks.
