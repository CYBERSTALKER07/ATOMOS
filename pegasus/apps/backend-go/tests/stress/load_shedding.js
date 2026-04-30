/**
 * Kill-Net Stress Test — F.R.I.D.A.Y. Priority Shedding Validation
 *
 * Verifies that the BackpressureEngine + PrioritySheddingMiddleware correctly
 * protects P0 (checkout) traffic while shedding P2 (telemetry/analytics)
 * under extreme load.
 *
 * Key corrections from naive blueprint:
 *   - Priority is PATH-CLASSIFIED, not header-based (X-Priority header is ignored)
 *   - Checkout routes: /v1/checkout/b2b and /v1/checkout/unified
 *   - P2 routes: /v1/analytics/, /v1/supplier/dashboard, /v1/sync/batch
 *   - All routes require Bearer auth — uses debug/mint-token in dev mode
 *   - Backend returns 503 (not 429) for load shedding with X-Backpressure-Interval
 *
 * Run:
 *   k6 run tests/stress/load_shedding.js
 *   (or via Docker: docker run --rm -i --network=host grafana/k6 run - <tests/stress/load_shedding.js)
 *
 * Requires: Backend running on localhost:8080 with Redis connected.
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ── Custom Metrics ──────────────────────────────────────────────────────────

const p0Latency  = new Trend('p0_checkout_latency', true);
const p2Latency  = new Trend('p2_analytics_latency', true);
const p0Success  = new Rate('p0_success_rate');
const p2Shed     = new Rate('p2_shed_rate');
const backpressureSignals = new Counter('backpressure_signals');

// ── Options ─────────────────────────────────────────────────────────────────

export const options = {
    stages: [
        { duration: '30s', target: 200 },   // Warm-up
        { duration: '1m',  target: 1000 },  // Ramp to stress
        { duration: '2m',  target: 2000 },  // Peak stress — shedding should kick in
        { duration: '30s', target: 500 },   // Gradual cool-down
        { duration: '30s', target: 0 },     // Drain
    ],
    thresholds: {
        // P0 checkout MUST stay fast — 99th percentile under 500ms
        'p0_checkout_latency': ['p(99)<500'],
        // P0 checkout MUST succeed — 99%+ success rate
        'p0_success_rate': ['rate>0.99'],
        // P2 analytics SHOULD be shed under peak load — we expect >10% rejection
        'p2_shed_rate': ['rate>0.1'],
    },
};

// ── Setup: Mint Dev Tokens ──────────────────────────────────────────────────

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
    // Mint a supplier/admin token for checkout (P0)
    const adminRes = http.get(`${BASE}/debug/mint-token?role=ADMIN`);
    if (adminRes.status !== 200) {
        console.error(`Failed to mint ADMIN token: ${adminRes.status} — is the backend running in dev mode?`);
    }
    const adminToken = adminRes.body.trim();

    // Mint a supplier token for analytics (P2)
    const supplierRes = http.get(`${BASE}/debug/mint-token?role=SUPPLIER`);
    const supplierToken = supplierRes.status === 200 ? supplierRes.body.trim() : adminToken;

    console.log(`[SETUP] Tokens minted. Admin: ${adminToken.substring(0, 20)}...`);
    return { adminToken, supplierToken };
}

// ── Main VU Function ────────────────────────────────────────────────────────

export default function (data) {
    const { adminToken, supplierToken } = data;

    // ── P2 Noise: Analytics dashboard (90% of traffic) ──
    // ClassifyRequest("/v1/analytics/") → P2_TELEMETRY — shed first at 50ms Redis latency
    group('P2 Analytics Flood', () => {
        const res = http.get(`${BASE}/v1/analytics/dashboard`, {
            headers: {
                'Authorization': `Bearer ${supplierToken}`,
                'Content-Type': 'application/json',
            },
            tags: { priority: 'P2' },
        });

        p2Latency.add(res.timings.duration);

        // Track shedding: 503 = load shed, 429 = rate limited
        const wasShed = res.status === 503 || res.status === 429;
        p2Shed.add(wasShed ? 1 : 0);

        // Track backpressure headers
        const bpHeader = res.headers['X-Backpressure-Interval'];
        if (bpHeader && parseInt(bpHeader) > 0) {
            backpressureSignals.add(1);
        }
    });

    // ── P2 Noise: Supplier dashboard (additional P2 traffic) ──
    group('P2 Dashboard Flood', () => {
        http.get(`${BASE}/v1/supplier/dashboard`, {
            headers: {
                'Authorization': `Bearer ${supplierToken}`,
                'Content-Type': 'application/json',
            },
            tags: { priority: 'P2' },
        });
    });

    // ── P0 Signal: Checkout (10% of traffic) ──
    // ClassifyRequest("/v1/checkout/") → P0_CRITICAL — never shed
    if (__ITER % 10 === 0) {
        group('P0 Checkout', () => {
            const payload = JSON.stringify({
                retailer_id: 'STRESS-TEST-RETAILER',
                global_paynt_gateway: 'CASH',
                latitude: 41.311081,
                longitude: 69.279737,
                items: [{
                    product_id: 'STRESS-TEST-PRODUCT',
                    quantity: 1,
                    unit_price: 100,
                }],
            });

            const res = http.post(`${BASE}/v1/checkout/b2b`, payload, {
                headers: {
                    'Authorization': `Bearer ${adminToken}`,
                    'Content-Type': 'application/json',
                },
                tags: { priority: 'P0' },
            });

            p0Latency.add(res.timings.duration);

            // P0 should NEVER be 503-shed. 4xx from business logic is OK
            // (invalid retailer etc), but 503 means shedding leaked to P0.
            const notShed = res.status !== 503;
            p0Success.add(notShed ? 1 : 0);

            check(res, {
                'P0 not load-shed (not 503)': (r) => r.status !== 503,
                'P0 not rate-limited (not 429)': (r) => r.status !== 429,
            });

            // Log any backpressure interval returned (for informational purposes)
            const bpHeader = res.headers['X-Backpressure-Interval'];
            if (bpHeader) {
                backpressureSignals.add(1);
            }
        });
    }

    // ── P1 Baseline: Order list (occasional) ──
    // ClassifyRequest("/v1/orders/") → P1_OPERATIONAL — shed at 150ms Redis
    if (__ITER % 20 === 0) {
        group('P1 Order List', () => {
            http.get(`${BASE}/v1/orders?limit=5`, {
                headers: {
                    'Authorization': `Bearer ${adminToken}`,
                    'Content-Type': 'application/json',
                },
                tags: { priority: 'P1' },
            });
        });
    }

    sleep(0.05); // ~20 requests/sec per VU at peak
}

// ── Teardown ────────────────────────────────────────────────────────────────

export function teardown(data) {
    console.log('='.repeat(60));
    console.log('F.R.I.D.A.Y. Kill-Net Stress Test Complete');
    console.log('='.repeat(60));
    console.log('Key metrics to verify:');
    console.log('  p0_success_rate   — Must be >99% (P0 never shed)');
    console.log('  p2_shed_rate      — Should be >10% at peak (shedding working)');
    console.log('  p0_checkout_latency p(99) — Must be <500ms');
    console.log('  backpressure_signals — Should be >0 (backend signaling clients)');
    console.log('='.repeat(60));
}
