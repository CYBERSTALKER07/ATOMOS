# pegasusX load certification report

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- **Date:** 2026-06-17 18:06 UTC
- **Profile:** `smoke`
- **Base URL:** `http://localhost:8180`
- **Artifacts:** `/Users/shakhzod/Desktop/V.O.I.D/pegasusX/artifacts/load/20260617-180549`
- **Overall:** **FAIL**
- **k6 thresholds:** **FAIL**

| Metric | Target | Observed | Pass |
|--------|--------|----------|------|
| Retailer VUs (max) | profile-defined | 30 | |
| p99 read (tracking/cart) | < 300 ms | n/a ms | n/a |
| p99 mutation (order create) | < 800 ms | n/a ms | n/a |
| HTTP failure rate | <= 5% | 8.108108108108109% | FAIL |
| Supplier p99 read | < 400 ms | n/a ms | n/a |

## Notes

- **k6 threshold breaches:**
  - `http_req_failed rate<0.05`
- Profile `smoke` is the local/CI gate (`make load-cert`).
- Profile `cert` uses relaxed mutation/supplier SLO on local SSMR (emulator); staging uses production targets in `docs/LOAD_TEST_SLO.md`.
- Cert/stress bootstrap `LOAD_RETAILER_POOL_SIZE` distinct retailer JWTs so per-actor rate limits reflect real concurrency.
- SLO source: `docs/LOAD_TEST_SLO.md`.
