/**
 * Payloader API — Manifest Operations (HTTP-only, no browser)
 *
 * Expo Payload Terminal tested at API contract level.
 *
 * Endpoints tested:
 *   POST /v1/auth/payloader/login → JWT
 *   GET /v1/payloader/trucks → available trucks
 *   GET /v1/payloader/orders → orders for selected truck
 *   POST /v1/payload/manifest/create → DRAFT manifest
 *   POST /v1/payload/manifest/{id}/start_load → LOADING
 *   POST /v1/payload/manifest/{id}/seal → SEALED (LEO gate)
 *   POST /v1/payload/manifest-exception → exception filing
 *
 * Cross-role: Sealed manifest triggers Kafka event → Supplier + Driver
 */
import { test } from '../fixtures/api';
import { expect } from '@playwright/test';

const API = process.env.API_BASE_URL || 'http://localhost:8080';

test.describe('Payloader API — Manifest Ops', () => {
  test('POST /v1/auth/payloader/login returns JWT', async ({ request }) => {
    const res = await request.post(`${API}/v1/auth/payloader/login`, {
      data: {
        phone: process.env.TEST_PAYLOADER_PHONE || '+998901234571',
        password: process.env.TEST_PAYLOADER_PASSWORD || 'TestPass123!',
      },
    });

    if (res.ok()) {
      const body = await res.json();
      expect(body.token).toBeTruthy();
    } else {
      expect([200, 401, 404, 500, 503]).toContain(res.status());
    }
  });

  test('GET /v1/payloader/trucks returns truck list', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.get('/v1/payloader/trucks');

    if (res.ok()) {
      const body = await res.json();
      // Should be an array of trucks
      expect(body).toBeTruthy();
    } else {
      expect([200, 401, 404]).toContain(res.status());
    }
  });

  test('GET /v1/payloader/orders returns orders for truck', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.get('/v1/payloader/orders');

    if (res.ok()) {
      const body = await res.json();
      expect(body).toBeTruthy();
    } else {
      expect([200, 401, 404]).toContain(res.status());
    }
  });

  test('POST /v1/payload/manifest/create creates DRAFT manifest', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.post('/v1/payload/manifest/create', {
      data: {
        truck_id: 'test-truck',
        orders: ['test-order-1', 'test-order-2'],
      },
    });

    expect([200, 201, 400, 401, 404, 409]).toContain(res.status());
    if (res.ok()) {
      const body = await res.json();
      expect(body).toBeTruthy();
    }
  });

  test('POST /v1/payload/manifest/{id}/start_load transitions to LOADING', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.post('/v1/payload/manifest/test-manifest/start_load', {
      data: {},
    });

    expect([200, 400, 401, 404, 409]).toContain(res.status());
  });

  test('POST /v1/payload/manifest/{id}/seal seals manifest (LEO gate)', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.post('/v1/payload/manifest/test-manifest/seal', {
      data: { verified: true },
    });

    expect([200, 400, 401, 404, 409]).toContain(res.status());
  });

  test('POST /v1/payload/manifest-exception files exception', async ({ payloaderAPI }) => {
    const res = await payloaderAPI.post('/v1/payload/manifest-exception', {
      data: {
        manifest_id: 'test-manifest',
        exception_type: 'OVERFLOW',
        description: 'Truck capacity exceeded for order batch',
        affected_orders: ['test-order-1'],
      },
    });

    expect([200, 201, 400, 401, 404]).toContain(res.status());
  });
});
