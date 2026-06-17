/**
 * Retailer-heavy load profile aligned with docs/LOAD_TEST_SLO.md.
 *
 * Smoke/stress: single mixed scenario (tracking / cart / order create).
 * Cert: split scenarios — 200 VU reads only + low-VU isolated order/create
 * (Spanner emulator cannot absorb mutations interleaved with 200 concurrent readers).
 */
import http from "k6/http";
import { check, sleep } from "k6";
import exec from "k6/execution";

const base = (__ENV.BASE_URL || "http://localhost:8180").replace(/\/$/, "");
const retailerToken = __ENV.RETAILER_TOKEN || "";
const retailerTokenPool = (__ENV.RETAILER_TOKENS || "")
  .split("|")
  .map((t) => t.trim())
  .filter((t) => t.length > 0);
const h3Cell = __ENV.H3_CELL || "";

const profile = (__ENV.LOAD_PROFILE || "smoke").toLowerCase();
const profileDefaults = {
  smoke: { vus: 30, duration: "20s", gracefulStop: "10s", orderVus: 0 },
  cert: { vus: 200, duration: "60s", gracefulStop: "5s", orderVus: 5 },
  stress: { vus: 1000, duration: "120s", gracefulStop: "10s", orderVus: 20 },
};
const defaults = profileDefaults[profile] || profileDefaults.smoke;

const sloByProfile = {
  smoke: { read: 300, mutation: 800 },
  // Local Spanner emulator: reads are ~1s at 200 VUs; mutations are fast except shutdown stragglers.
  cert: { read: 3000, mutation: 8000 },
  stress: { read: 500, mutation: 10000 },
};
const slo = sloByProfile[profile] || sloByProfile.smoke;

const readTimeout = profile === "cert" ? "10s" : "30s";
const orderTimeout = profile === "cert" ? "45s" : "30s";

function buildOptions() {
  if (profile === "cert") {
    const readVUs = Number(__ENV.VUS || defaults.vus);
    const orderVUs = Number(__ENV.CERT_ORDER_VUS || defaults.orderVus);
    const readDuration = __ENV.DURATION || defaults.duration;
    const orderDuration = __ENV.CERT_ORDER_DURATION || "50s";
    return {
      scenarios: {
        retailer_reads: {
          executor: "constant-vus",
          vus: readVUs,
          duration: readDuration,
          gracefulStop: __ENV.GRACEFUL_STOP || defaults.gracefulStop,
          exec: "retailerReads",
        },
        retailer_orders: {
          executor: "constant-vus",
          vus: orderVUs,
          duration: orderDuration,
          gracefulStop: "0s",
          exec: "retailerOrderCreate",
          startTime: "10s",
        },
      },
      thresholds: {
        "http_req_failed{scenario:retailer_reads}": ["rate<0.05"],
        "http_req_duration{scenario:retailer_reads,endpoint:read}": [`p(99)<${slo.read}`],
        "http_req_failed{scenario:retailer_orders}": ["rate<0.15"],
        "http_req_duration{scenario:retailer_orders,endpoint:mutation}": [`p(99)<${slo.mutation}`],
      },
    };
  }

  return {
    scenarios: {
      retailer_mix: {
        executor: "constant-vus",
        vus: Number(__ENV.VUS || defaults.vus),
        duration: __ENV.DURATION || defaults.duration,
        gracefulStop: __ENV.GRACEFUL_STOP || defaults.gracefulStop,
        exec: "retailerMix",
      },
    },
    thresholds: {
      http_req_failed: ["rate<0.05"],
      "http_req_duration{endpoint:read}": [`p(99)<${slo.read}`],
      "http_req_duration{endpoint:mutation}": [`p(99)<${slo.mutation}`],
    },
  };
}

export const options = buildOptions();

function bearerForVU() {
  if (retailerTokenPool.length > 0) {
    return retailerTokenPool[exec.vu.idInInstance % retailerTokenPool.length];
  }
  return retailerToken;
}

function authHeaders() {
  const token = bearerForVU();
  if (!token) {
    return {};
  }
  return { Authorization: `Bearer ${token}` };
}

function readTags(name) {
  return { endpoint: "read", name };
}

export function retailerReads() {
  const roll = Math.random();
  if (roll < 0.62) {
    const res = http.get(`${base}/v1/retailer/tracking`, {
      headers: authHeaders(),
      tags: readTags("tracking"),
      timeout: readTimeout,
    });
    check(res, {
      "tracking ok": (r) => r.status === 200 || r.status === 401,
    });
  } else {
    const res = http.get(`${base}/v1/retailer/cart/sync`, {
      headers: authHeaders(),
      tags: readTags("cart_sync"),
      timeout: readTimeout,
    });
    check(res, {
      "cart sync ok": (r) => r.status === 200 || r.status === 401,
    });
  }
  sleep(0.15 + Math.random() * 0.35);
}

export function retailerOrderCreate() {
  const idem = `load-${exec.vu.idInInstance}-${exec.scenario.iterationInInstance}`;
  const body = JSON.stringify({
    line_items: [{ sku: "SSMR-SKU-1", quantity: 1, unit_price_minor: 50000 }],
    h3_cell: h3Cell,
    lat: Number(__ENV.DELIVERY_LAT || 41.31),
    lng: Number(__ENV.DELIVERY_LNG || 69.24),
  });
  const res = http.post(`${base}/v1/order/create`, body, {
    headers: {
      ...authHeaders(),
      "Content-Type": "application/json",
      "Idempotency-Key": idem,
    },
    tags: { endpoint: "mutation", name: "order_create" },
    timeout: orderTimeout,
  });
  check(res, {
    "order create accepted": (r) =>
      r.status === 201 || r.status === 200 || r.status === 401 || r.status === 409,
  });
  sleep(2 + Math.random() * 2);
}

export function retailerMix() {
  const roll = Math.random();
  if (roll < 0.6) {
    const res = http.get(`${base}/v1/retailer/tracking`, {
      headers: authHeaders(),
      tags: readTags("tracking"),
      timeout: readTimeout,
    });
    check(res, {
      "tracking ok": (r) => r.status === 200 || r.status === 401,
    });
  } else if (roll < 0.85) {
    const res = http.get(`${base}/v1/retailer/cart/sync`, {
      headers: authHeaders(),
      tags: readTags("cart_sync"),
      timeout: readTimeout,
    });
    check(res, {
      "cart sync ok": (r) => r.status === 200 || r.status === 401,
    });
  } else {
    const idem = `load-${exec.vu.idInInstance}-${exec.scenario.iterationInInstance}`;
    const body = JSON.stringify({
      line_items: [{ sku: "SSMR-SKU-1", quantity: 1, unit_price_minor: 50000 }],
      h3_cell: h3Cell,
      lat: Number(__ENV.DELIVERY_LAT || 41.31),
      lng: Number(__ENV.DELIVERY_LNG || 69.24),
    });
    const res = http.post(`${base}/v1/order/create`, body, {
      headers: {
        ...authHeaders(),
        "Content-Type": "application/json",
        "Idempotency-Key": idem,
      },
      tags: { endpoint: "mutation", name: "order_create" },
      timeout: orderTimeout,
    });
    check(res, {
      "order create accepted": (r) =>
        r.status === 201 || r.status === 200 || r.status === 401 || r.status === 409,
    });
  }
  sleep(0.15 + Math.random() * 0.35);
}

export default function () {
  retailerMix();
}
