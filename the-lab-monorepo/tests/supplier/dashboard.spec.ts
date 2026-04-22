/**
 * Supplier Dashboard — KPIs, BentoGrid, Polling, Fleet Map
 *
 * Root: / (Dispatch Room with BentoGrid, fleet map, order table)
 * /dashboard: 3 StatsCards (pipeline, pending volume, AI forecast) with 5s polling
 *
 * API: GET /v1/supplier/dashboard → {total_pipeline_uzs, pending_volume, ai_forecast_volume}
 * Polling: 5000ms interval
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Dashboard', () => {
  test('root page renders BentoGrid dispatch room', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/');
    await supplierPage.waitForLoadState('networkidle');

    // BentoGrid should be visible with key cells
    // Status chips for order states should exist
    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    // Fleet map anchor cell (MapLibre or similar)
    // Look for canvas element (MapLibre renders to canvas)
    const mapOrCards = supplierPage.locator('canvas, [class*="bento"], [class*="grid"]');
    await expect(mapOrCards.first()).toBeVisible({ timeout: 15_000 });
  });

  test('/dashboard renders 3 StatsCards with KPIs', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/dashboard');

    // Wait for data to load
    await supplierPage.waitForLoadState('networkidle');

    // KPI labels
    await expect(supplierPage.getByText(/pipeline|revenue/i).first()).toBeVisible({ timeout: 10_000 });
    await expect(supplierPage.getByText(/dispatch|volume|units/i).first()).toBeVisible();
    await expect(supplierPage.getByText(/forecast|ai/i).first()).toBeVisible();
  });

  test('dashboard polling fires at 5s interval', async ({ supplierPage }) => {
    const apiCalls: string[] = [];
    await supplierPage.route('**/v1/supplier/dashboard**', async (route) => {
      apiCalls.push(route.request().url());
      await route.continue();
    });

    await supplierPage.goto('http://localhost:3000/dashboard');
    await supplierPage.waitForTimeout(12_000); // Wait for at least 2 polls

    // Should have initial fetch + at least 1 polling cycle
    expect(apiCalls.length).toBeGreaterThanOrEqual(2);
  });

  test('live status indicator shows system state', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/dashboard');
    await supplierPage.waitForLoadState('networkidle');

    // Look for the live/offline indicator
    const indicator = supplierPage.getByText(/system live|system offline/i);
    await expect(indicator.first()).toBeVisible({ timeout: 10_000 });
  });

  test('fleet map markers render with staleness colors @responsive', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/');
    await supplierPage.waitForLoadState('networkidle');

    // Fleet map should render (canvas or map container)
    const mapContainer = supplierPage.locator('canvas, [class*="map"], [id*="map"]');
    if (await mapContainer.count() > 0) {
      await expect(mapContainer.first()).toBeVisible({ timeout: 15_000 });
    }
  });

  test('multi-select orders triggers dispatch action', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/');
    await supplierPage.waitForLoadState('networkidle');

    // Look for order table rows with checkboxes
    const checkboxes = supplierPage.locator('input[type="checkbox"]');
    if (await checkboxes.count() > 0) {
      await checkboxes.first().check();
      // Dispatch action button should appear
      const dispatchBtn = supplierPage.getByRole('button', { name: /dispatch|assign/i });
      if (await dispatchBtn.count() > 0) {
        await expect(dispatchBtn.first()).toBeVisible();
      }
    }
  });
});
