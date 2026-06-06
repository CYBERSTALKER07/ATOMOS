/**
 * Lightweight retailer-heavy profile for local SSMR/staging.
 * Usage: k6 run -e BASE_URL=http://localhost:8180 -e VUS=50 scripts/load/k6_retailer_smoke.js
 */
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: Number(__ENV.VUS || 50),
  duration: __ENV.DURATION || "30s",
  thresholds: {
    http_req_duration: ["p(99)<800"],
  },
};

const base = (__ENV.BASE_URL || "http://localhost:8180").replace(/\/$/, "");

export default function () {
  const health = http.get(`${base}/v1/health`);
  check(health, { "health ok": (r) => r.status === 200 });
  sleep(0.2);
  const tracking = http.get(`${base}/v1/retailer/tracking`, {
    headers: { Authorization: `Bearer ${__ENV.RETAILER_TOKEN || ""}` },
  });
  check(tracking, { "tracking reachable": (r) => r.status === 200 || r.status === 401 });
  sleep(0.5);
}
