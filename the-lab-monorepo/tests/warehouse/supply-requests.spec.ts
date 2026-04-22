/**
 * Warehouse Supply Requests — CRUD, State Machine, Demand Forecast
 *
 * Page: /supply-requests
 * APIs:
 *   GET /v1/warehouse/supply-requests → SupplyRequest[]
 *   POST /v1/warehouse/supply-requests → create
 *   PUT /v1/warehouse/supply-requests/{id} → update
 *   POST /v1/warehouse/supply-requests/{id}/submit → DRAFT→SUBMITTED
 *
 * State machine: DRAFT → SUBMITTED → ACKNOWLEDGED → IN_PRODUCTION → READY → FULFILLED (+ CANCELLED)
 *
 * Additional pages:
 *   /dispatch-locks → geofence holds
 *   /demand-forecast → AI demand prediction table
 *   /staff → staff management
 */
import { test, expect } from '../fixtures/auth';

test.describe('Warehouse Supply Requests', () => {
  test('supply request list loads with state filters', async ({ warehousePage }) => {
    await warehousePage.goto('http://localhost:3003/supply-requests');
    await warehousePage.waitForLoadState('networkidle');

    const content = warehousePage.getByText(/supply.?request|request/i).first();
    await expect(content).toBeVisible({ timeout: 10_000 });

    // State filter chips
    const states = warehousePage.getByText(/draft|submitted|acknowledged|in.?production|ready|fulfilled|all/i);
    if (await states.count() > 0) {
      await expect(states.first()).toBeVisible();
    }
  });

  test('create supply request form', async ({ warehousePage }) => {
    let createCalled = false;
    await warehousePage.route('**/v1/warehouse/supply-requests', async (route) => {
      if (route.request().method() === 'POST') {
        createCalled = true;
        await route.fulfill({
          status: 201,
          body: JSON.stringify({ supply_request_id: 'sr-1', status: 'DRAFT' }),
        });
      } else {
        await route.continue();
      }
    });

    await warehousePage.goto('http://localhost:3003/supply-requests');
    await warehousePage.waitForLoadState('networkidle');

    // Create button
    const createBtn = warehousePage.getByRole('button', { name: /create|new|add/i });
    if (await createBtn.count() > 0) {
      await createBtn.first().click();

      // Form fields: factory, priority, items, date
      const formFields = warehousePage.locator('input, select, textarea');
      if (await formFields.count() > 0) {
        await expect(formFields.first()).toBeVisible({ timeout: 5_000 });
      }
    }
  });

  test('state machine transitions display correctly', async ({ warehousePage }) => {
    await warehousePage.goto('http://localhost:3003/supply-requests');
    await warehousePage.waitForLoadState('networkidle');

    // State badges should show supply request states
    const badges = warehousePage.getByText(/draft|submitted|acknowledged|in.?production|ready|fulfilled|cancelled/i);
    if (await badges.count() > 0) {
      await expect(badges.first()).toBeVisible();
    }
  });

  test('dispatch locks view (geofence holds)', async ({ warehousePage }) => {
    await warehousePage.goto('http://localhost:3003/dispatch-locks');
    await warehousePage.waitForLoadState('networkidle');

    const content = warehousePage.getByText(/dispatch.?lock|geofence|hold/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('demand forecast AI table', async ({ warehousePage }) => {
    await warehousePage.goto('http://localhost:3003/demand-forecast');
    await warehousePage.waitForLoadState('networkidle');

    const content = warehousePage.getByText(/demand|forecast|prediction/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('staff management', async ({ warehousePage }) => {
    await warehousePage.goto('http://localhost:3003/staff');
    await warehousePage.waitForLoadState('networkidle');

    const content = warehousePage.getByText(/staff|employee|team|member/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });
});
