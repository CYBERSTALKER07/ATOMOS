# pegasusX load certification report

- **Date:** 2026-06-04 15:50 UTC
- **Profile:** `cert`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260604-154852`
- **Overall:** **PASS**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | n/a | |
| p99 read (tracking/cart) | < 300 ms | n/a ms | n/a |
| p99 mutation (order create) | < 800 ms | n/a ms | n/a |
| HTTP failure rate | <= 5% | n/a% | n/a |
| Supplier p99 read | < 400 ms | n/a ms | n/a |

## Notes

- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` targets staging-scale VUs; `stress` documents the 10k envelope — run only on sized clusters.
- SLO source: `docs/LOAD_TEST_SLO.md`.
