# pegasusX load certification report

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- **Date:** 2026-06-04 17:48 UTC
- **Profile:** `cert`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260604-174408`
- **Overall:** **FAIL**
- **k6 thresholds:** **FAIL**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | 200 | |
| p99 read (tracking/cart) | < 300 ms | n/a ms | n/a |
| p99 mutation (order create) | < 5000 ms | n/a ms | n/a |
| HTTP failure rate | <= 5% | 0.003908845717859516% | PASS |
| Supplier p99 read | < 1500 ms | n/a ms | n/a |

## Notes

- **k6 threshold breaches:**
  - `http_req_duration{endpoint:mutation} p(99)<800`
  - `http_req_duration{endpoint:read} p(99)<700`
- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` uses relaxed mutation/supplier SLO on local SSMR (emulator); staging uses production targets in `docs/LOAD_TEST_SLO.md`.
- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.
- SLO source: `docs/LOAD_TEST_SLO.md`.
