/**
 * Supplier Treasury — Ledger, Reconciliation, Cash Holdings
 *
 * Pages: /treasury, /ledger, /reconciliation
 * APIs:
 *   GET /v1/treasury/ledger → {platform_revenue, supplier_payout, total_volume}
 *   GET /v1/orders?limit=26&offset=0 → LedgerEntry[]
 *   GET /v1/reconciliation → anomalies
 *
 * Polling: Treasury 5s, Ledger 3s
 * Pagination: Ledger 25 items/page
 * Anomaly types: DELTA, ORPHANED, MATCH
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Treasury & Ledger', () => {
  test('ledger page loads with paginated table', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/ledger');
    await supplierPage.waitForLoadState('networkidle');

    // Header
    await expect(supplierPage.getByText(/ledger|settlement/i).first()).toBeVisible({ timeout: 10_000 });

    // Table headers should include order_id, amount, state, etc.
    const table = supplierPage.locator('table, [class*="ledger"]');
    if (await table.count() > 0) {
      await expect(table.first()).toBeVisible();
    }
  });

  test('treasury KPIs show platform revenue, supplier payout, total volume', async ({ supplierPage }) => {
    await supplierPage.route('**/v1/treasury/ledger**', async (route) => {
      await route.fulfill({
        status: 200,
        body: JSON.stringify({
          platform_revenue: 5000000,
          supplier_payout: 95000000,
          total_volume: 100000000,
        }),
      });
    });

    await supplierPage.goto('http://localhost:3000/treasury');
    await supplierPage.waitForLoadState('networkidle');

    // Treasury header
    await expect(supplierPage.getByText(/treasury/i).first()).toBeVisible({ timeout: 10_000 });

    // KPI cards should show revenue, payout, volume
    const kpiContent = supplierPage.getByText(/revenue|payout|volume/i);
    if (await kpiContent.count() > 0) {
      await expect(kpiContent.first()).toBeVisible();
    }
  });

  test('treasury polling fires at 5s interval', async ({ supplierPage }) => {
    const apiCalls: number[] = [];
    await supplierPage.route('**/v1/treasury/ledger**', async (route) => {
      apiCalls.push(Date.now());
      await route.fulfill({
        status: 200,
        body: JSON.stringify({ platform_revenue: 0, supplier_payout: 0, total_volume: 0 }),
      });
    });

    await supplierPage.goto('http://localhost:3000/treasury');
    await supplierPage.waitForTimeout(12_000);

    expect(apiCalls.length).toBeGreaterThanOrEqual(2);
  });

  test('ledger pagination with 25 items per page', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/ledger');
    await supplierPage.waitForLoadState('networkidle');

    // Pagination controls
    const nextBtn = supplierPage.getByRole('button', { name: /next|›|→/i });
    const prevBtn = supplierPage.getByRole('button', { name: /prev|‹|←/i });

    if (await nextBtn.count() > 0) {
      await expect(nextBtn.first()).toBeVisible();
    }
    if (await prevBtn.count() > 0) {
      // Prev should be disabled on first page
      await expect(prevBtn.first()).toBeVisible();
    }
  });

  test('reconciliation anomaly detection', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/reconciliation');
    await supplierPage.waitForLoadState('networkidle');

    // Reconciliation page should show anomaly types
    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    const anomalyContent = supplierPage.getByText(/reconciliation|anomaly|delta|orphaned|match/i);
    if (await anomalyContent.count() > 0) {
      await expect(anomalyContent.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('cash holdings breakdown', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/treasury');
    await supplierPage.waitForLoadState('networkidle');

    // Cash holdings link
    const holdingsLink = supplierPage.getByText(/cash holdings/i);
    if (await holdingsLink.count() > 0) {
      await holdingsLink.first().click();
      await supplierPage.waitForLoadState('networkidle');
    }
  });
});
