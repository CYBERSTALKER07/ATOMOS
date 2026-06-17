# pegasusX load certification report

## Latest gate attempt (2026-06-17)

- **Profile:** `smoke` via `make load-cert-ssmr`
- **Result:** **BLOCKED** — Docker daemon unavailable on runner (`redis:7-alpine` pull failed).
- **Pre-req:** Start Docker Desktop, then re-run `make load-cert-ssmr` (smoke) and `LOAD_PROFILE=cert make load-cert-ssmr` (cert).
- **Backend pre-check:** `go test ./apps/backend-go/...` green after E1 portal-seed gates and idempotency concurrency test.

---

- **Date:** 2026-06-04 18:21 UTC
- **Profile:** `cert`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260604-181832`
- **Overall:** **PASS**
- **k6 thresholds:** **PASS**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | 205 | |
| p99 read (tracking/cart) | < 3000 ms | 142 ms | PASS |
| p99 mutation (order create) | < 8000 ms | 481 ms | PASS |
| HTTP failure rate | <= 5% | 0.02% | PASS |
| Supplier p99 read | < 2500 ms | 198 ms | PASS |

## Notes

- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` uses relaxed mutation/supplier SLO on local SSMR (emulator); staging uses production targets in `docs/LOAD_TEST_SLO.md`.
- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.
- SLO source: `docs/LOAD_TEST_SLO.md`.
