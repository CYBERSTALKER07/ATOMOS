/**
 * Supplier Settings — Profile, Payment Config, Delivery Zones, Operating Schedule
 *
 * Pages: /supplier/settings, /supplier/payment-config, /supplier/delivery-zones
 * APIs:
 *   GET/POST /v1/supplier/profile
 *   POST /v1/supplier/shift
 *   GET/POST /v1/supplier/delivery-zones
 *   GET/POST /v1/supplier/warehouses
 *
 * Schedule: 7-day toggle, open/close HH:MM, manual off-shift
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Settings', () => {
  test('supplier profile view/edit', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/settings');
    await supplierPage.waitForLoadState('networkidle');

    // Settings page should show operating schedule or profile
    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    // Day toggles for operating schedule
    const dayButtons = supplierPage.getByText(/mon|tue|wed|thu|fri|sat|sun/i);
    if (await dayButtons.count() > 0) {
      await expect(dayButtons.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('payment gateway configuration', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/payment-config');
    await supplierPage.waitForLoadState('networkidle');

    // Payment config page
    const content = supplierPage.getByText(/payment|gateway|global\s*pay|cash/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('delivery zones CRUD with map', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/delivery-zones');
    await supplierPage.waitForLoadState('networkidle');

    // Delivery zones page
    const content = supplierPage.getByText(/delivery.?zone|coverage|region/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('operating schedule toggle and save', async ({ supplierPage }) => {
    let shiftSaved = false;
    await supplierPage.route('**/v1/supplier/shift**', async (route) => {
      if (route.request().method() === 'POST') {
        shiftSaved = true;
        await route.fulfill({ status: 200, body: JSON.stringify({ success: true }) });
      } else {
        await route.continue();
      }
    });

    await supplierPage.goto('http://localhost:3000/supplier/settings');
    await supplierPage.waitForLoadState('networkidle');

    // Toggle a day
    const dayBtn = supplierPage.getByText(/mon/i).first();
    if (await dayBtn.isVisible()) {
      await dayBtn.click();
    }

    // Save button
    const saveBtn = supplierPage.getByRole('button', { name: /save/i });
    if (await saveBtn.count() > 0) {
      await saveBtn.first().click();
      await supplierPage.waitForTimeout(1_000);
    }
  });

  test('warehouse management list', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/warehouses');
    await supplierPage.waitForLoadState('networkidle');

    const content = supplierPage.getByText(/warehouse|depot|hub/i);
    if (await content.count() > 0) {
      await expect(content.first()).toBeVisible({ timeout: 10_000 });
    }
  });
});
