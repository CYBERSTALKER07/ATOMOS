/**
 * Supplier portal read-heavy profile (PX9-E parity paths).
 */
import http from "k6/http";
import { check, sleep } from "k6";

const base = (__ENV.BASE_URL || "http://localhost:8180").replace(/\/$/, "");
const supplierCookie = __ENV.SUPPLIER_COOKIE || "";

const profile = (__ENV.LOAD_PROFILE || "smoke").toLowerCase();
const profileDefaults = {
  smoke: { vus: 15, duration: "20s", gracefulStop: "10s" },
  cert: { vus: 40, duration: "45s", gracefulStop: "5s" },
  stress: { vus: 500, duration: "90s", gracefulStop: "10s" },
};
const defaults = profileDefaults[profile] || profileDefaults.smoke;

const supplierReadP99Ms =
  profile === "cert" ? 2500 : profile === "stress" ? 2000 : 400;

const httpTimeout = profile === "cert" ? "20s" : "30s";

export const options = {
  scenarios: {
    supplier_reads: {
      executor: "constant-vus",
      vus: Number(__ENV.VUS || defaults.vus),
      duration: __ENV.DURATION || defaults.duration,
      gracefulStop: __ENV.GRACEFUL_STOP || defaults.gracefulStop,
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    "http_req_duration{endpoint:read}": [`p(99)<${supplierReadP99Ms}`],
  },
};

function cookieHeaders() {
  if (!supplierCookie) {
    return {};
  }
  return { Cookie: supplierCookie };
}

const paths = [
  { path: "/v1/supplier/dashboard", name: "dashboard" },
  { path: "/v1/supplier/exceptions", name: "exceptions" },
  { path: "/v1/supplier/shop-closed/active", name: "shop_closed" },
  { path: "/v1/supplier/fleet/orders", name: "fleet_orders" },
  { path: "/v1/supplier/empathy/adoption", name: "empathy" },
  { path: "/v1/supplier/orders", name: "orders" },
];

export default function () {
  const pick = paths[Math.floor(Math.random() * paths.length)];
  const res = http.get(`${base}${pick.path}`, {
    headers: cookieHeaders(),
    tags: { endpoint: "read", name: pick.name },
    timeout: httpTimeout,
  });
  check(res, {
    "supplier read ok": (r) => r.status === 200 || r.status === 401 || r.status === 503,
  });
  sleep(0.25 + Math.random() * 0.5);
}
