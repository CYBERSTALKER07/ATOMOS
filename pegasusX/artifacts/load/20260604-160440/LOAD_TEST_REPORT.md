# pegasusX load certification report

- **Date:** 2026-06-04 16:09 UTC
- **Profile:** `cert`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260604-160440`
- **Overall:** **FAIL**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | 200 | |
| p99 read (tracking/cart) | < 300 ms | n/a ms | n/a |
| p99 mutation (order create) | < 800 ms | n/a ms | n/a |
| HTTP failure rate | <= 5% | 66.40067198656027% | FAIL |
| Supplier p99 read | < 700 ms | n/a ms | n/a |

## Notes

- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` targets staging-scale VUs; `stress` documents the 10k envelope — run only on sized clusters.
- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.
- SLO source: `docs/LOAD_TEST_SLO.md`.
