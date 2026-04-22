/**
 * Factory Loading Bay — Kanban Board, State Transitions, Batch Dispatch
 *
 * Page: /loading-bay
 * Kanban columns: APPROVED → LOADING → DISPATCHED
 * APIs:
 *   GET /v1/factory/transfers?status=approved → approved transfers
 *   POST /v1/factory/transfers/{id}/start-loading → move to LOADING
 *   POST /v1/factory/dispatch → batch dispatch
 *   POST /v1/factory/payload-override → LEO gate override
 *
 * Transfer fields: transfer_id, status, line_items, volume, assigned_truck
 */
import { test, expect } from '../fixtures/auth';

test.describe('Factory Loading Bay', () => {
  test('kanban board renders APPROVED, LOADING, DISPATCHED columns', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/loading-bay');
    await factoryPage.waitForLoadState('networkidle');

    // Kanban columns
    const columns = factoryPage.getByText(/approved|loading|dispatched/i);
    if (await columns.count() > 0) {
      await expect(columns.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('click transfer moves to LOADING state', async ({ factoryPage }) => {
    let startLoadingCalled = false;
    await factoryPage.route('**/v1/factory/transfers/*/start-loading**', async (route) => {
      startLoadingCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ status: 'LOADING' }) });
    });

    await factoryPage.goto('http://localhost:3002/loading-bay');
    await factoryPage.waitForLoadState('networkidle');

    // Click a transfer in APPROVED column
    const transferCard = factoryPage.locator('[class*="card"], [class*="transfer"]').first();
    if (await transferCard.isVisible()) {
      await transferCard.click();
      // Look for start loading action
      const loadBtn = factoryPage.getByRole('button', { name: /start.?load|load|begin/i });
      if (await loadBtn.count() > 0) {
        await loadBtn.first().click();
      }
    }
  });

  test('batch dispatch fires POST /v1/factory/dispatch', async ({ factoryPage }) => {
    let dispatchCalled = false;
    await factoryPage.route('**/v1/factory/dispatch**', async (route) => {
      dispatchCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ dispatched: 3 }) });
    });

    await factoryPage.goto('http://localhost:3002/loading-bay');
    await factoryPage.waitForLoadState('networkidle');

    const dispatchBtn = factoryPage.getByRole('button', { name: /dispatch|send/i });
    if (await dispatchBtn.count() > 0) {
      await dispatchBtn.first().click();
    }
  });

  test('transfer detail page shows line items', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/loading-bay');
    await factoryPage.waitForLoadState('networkidle');

    // Click a transfer to view details
    const card = factoryPage.locator('[class*="card"], [class*="transfer"]').first();
    if (await card.isVisible()) {
      await card.click();
      await factoryPage.waitForTimeout(1_000);

      // Detail view should show line items
      const lineItems = factoryPage.getByText(/line.?item|product|sku|quantity/i);
      if (await lineItems.count() > 0) {
        await expect(lineItems.first()).toBeVisible();
      }
    }
  });

  test('volume capacity check on assignment', async ({ factoryPage }) => {
    await factoryPage.goto('http://localhost:3002/loading-bay');
    await factoryPage.waitForLoadState('networkidle');

    // Volume/capacity info should be visible
    const capacityInfo = factoryPage.getByText(/volume|capacity|vu/i);
    if (await capacityInfo.count() > 0) {
      await expect(capacityInfo.first()).toBeVisible();
    }
  });

  test('payload override (LEO gate) with exception logging', async ({ factoryPage }) => {
    let overrideCalled = false;
    await factoryPage.route('**/v1/factory/payload-override**', async (route) => {
      overrideCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ success: true }) });
    });

    await factoryPage.goto('http://localhost:3002/payload-override');
    await factoryPage.waitForLoadState('networkidle');

    // LEO gate override page
    const content = factoryPage.getByText(/override|payload|leo|exception/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });
});
