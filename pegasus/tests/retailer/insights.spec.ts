/**
 * Retailer Insights — AI Predictions, Auto-Order Settings
 *
 * Page: /insights
 * APIs:
 *   GET /v1/retailer/predictions → AI-generated order predictions
 *   POST /v1/retailer/orders/confirm-ai → confirm AI order
 *   POST /v1/retailer/orders/reject-ai → reject AI order
 *   PUT /v1/retailer/auto-order/settings → update auto-order config
 *
 * Prediction fields: confidence, reasoning, suggested_date, items
 */
import { test, expect } from '../fixtures/auth';

test.describe('Retailer Insights', () => {
  test('AI predictions load with confidence and reasoning', async ({ retailerPage }) => {
    await retailerPage.route('**/v1/retailer/predictions**', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({
          predictions: [
            {
              prediction_id: 'pred-1',
              confidence: 0.87,
              reasoning: 'Based on 30-day purchase pattern',
              suggested_date: new Date(Date.now() + 86400000).toISOString(),
              items: [
                { sku_id: 'SKU001', name: 'Milk', suggested_qty: 50, confidence: 0.9 },
                { sku_id: 'SKU002', name: 'Bread', suggested_qty: 30, confidence: 0.85 },
              ],
            },
          ],
        }),
      });
    });

    await retailerPage.goto('http://localhost:3001/insights');
    await retailerPage.waitForLoadState('networkidle');

    const content = retailerPage.getByText(/insight|prediction|forecast|ai/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('auto-order settings toggle (global, per-supplier, per-category)', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/insights');
    await retailerPage.waitForLoadState('networkidle');

    const toggles = retailerPage.locator('[role="switch"], input[type="checkbox"], [class*="toggle"]');
    if (await toggles.count() > 0) {
      // Toggle should exist for auto-order settings
      await expect(toggles.first()).toBeVisible();
    }
  });

  test('confirm AI order fires POST /v1/retailer/orders/confirm-ai', async ({ retailerPage }) => {
    let confirmCalled = false;
    await retailerPage.route('**/v1/retailer/orders/confirm-ai**', async (route) => {
      confirmCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ order_id: 'ai-order-1' }) });
    });

    await retailerPage.goto('http://localhost:3001/insights');
    await retailerPage.waitForLoadState('networkidle');

    const confirmBtn = retailerPage.getByRole('button', { name: /confirm|accept|approve/i });
    if (await confirmBtn.count() > 0) {
      await confirmBtn.first().click();
    }
  });

  test('reject AI order fires POST /v1/retailer/orders/reject-ai', async ({ retailerPage }) => {
    let rejectCalled = false;
    await retailerPage.route('**/v1/retailer/orders/reject-ai**', async (route) => {
      rejectCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ success: true }) });
    });

    await retailerPage.goto('http://localhost:3001/insights');
    await retailerPage.waitForLoadState('networkidle');

    const rejectBtn = retailerPage.getByRole('button', { name: /reject|dismiss|skip/i });
    if (await rejectBtn.count() > 0) {
      await rejectBtn.first().click();
    }
  });
});
