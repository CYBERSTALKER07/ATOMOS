# pegasusX load certification report

- **Date:** 2026-06-04 18:14 UTC
- **Profile:** `cert`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260604-181132`
- **Overall:** **FAIL**
- **k6 thresholds:** **FAIL**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | 205 | |
| p99 read (tracking/cart) | < 300 ms | n/a ms | n/a |
| p99 mutation (order create) | < 30000 ms | n/a ms | n/a |
| HTTP failure rate | <= 5% | 0.038724667613269655% | PASS |
| Supplier p99 read | < 2500 ms | n/a ms | n/a |

## Notes

- **k6 threshold breaches:**
  - `http_req_duration{scenario:retailer_orders,endpoint:mutation} p(99)<30000`
  - `http_req_duration{scenario:retailer_reads,endpoint:read} p(99)<300`
- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` uses relaxed mutation/supplier SLO on local SSMR (emulator); staging uses production targets in `docs/LOAD_TEST_SLO.md`.
- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.
- SLO source: `docs/LOAD_TEST_SLO.md`.
