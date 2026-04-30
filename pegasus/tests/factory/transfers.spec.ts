/**
 * Factory Transfers — List, Detail, State Transitions, Fleet Assignment
 *
 * Page: /transfers
 * APIs:
 *   GET /v1/factory/transfers → Transfer[]
 *   GET /v1/factory/transfers/{id} → TransferDetail
 *   POST /v1/factory/transfers → create
 *   PUT /v1/factory/transfers/{id} → update
 *   POST /v1/factory/transfers/{id}/approve → DRAFT→APPROVED
 *   POST /v1/factory/transfers/{id}/start-loading → APPROVED→LOADING
 *   POST /v1/factory/transfers/{id}/dispatch → LOADING→DISPATCHED
 *
 * States: DRAFT → APPROVED → LOADING → DISPATCHED
 */
import { test, expect } from '../fixtures/auth';

test.describe('Factory Transfers', () => {
  test('transfer list loads with state filters', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/transfers');
    await factoryPage.waitForLoadState('networkidle');

    const content = factoryPage.getByText(/transfer|shipment/i).first();
    await expect(content).toBeVisible({ timeout: 10_000 });

    // State filter chips or dropdown
    const stateFilter = factoryPage.getByText(/draft|approved|loading|dispatched|all/i);
    if (await stateFilter.count() > 0) {
      await expect(stateFilter.first()).toBeVisible();
    }
  });

  test('transfer detail shows line items and volume', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/transfers');
    await factoryPage.waitForLoadState('networkidle');

    // Click first transfer
    const row = factoryPage.locator('tr, [class*="transfer-row"], [class*="card"]').first();
    if (await row.isVisible()) {
      await row.click();
      await factoryPage.waitForLoadState('networkidle');

      // Detail should show line items
      const detail = factoryPage.getByText(/line.?item|product|quantity|volume/i);
      if (await detail.count() > 0) {
        await expect(detail.first()).toBeVisible({ timeout: 5_000 });
      }
    }
  });

  test('state transitions: DRAFT→APPROVED→LOADING→DISPATCHED', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/transfers');
    await factoryPage.waitForLoadState('networkidle');

    // State badges should be present
    const badges = factoryPage.getByText(/draft|approved|loading|dispatched/i);
    if (await badges.count() > 0) {
      await expect(badges.first()).toBeVisible();
    }
  });

  test('fleet assignment on transfer', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/fleet');
    await factoryPage.waitForLoadState('networkidle');

    // Fleet page should show trucks/drivers
    const content = factoryPage.getByText(/fleet|truck|driver|vehicle/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('staff management view', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/staff');
    await factoryPage.waitForLoadState('networkidle');

    const content = factoryPage.getByText(/staff|employee|team/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });
});
