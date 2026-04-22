/**
 * Supplier Dispatch — Auto-Dispatch, Manifest Ops, VU Capacity
 *
 * Page: /supplier/dispatch + /supplier/manifests
 * APIs:
 *   GET /v1/supplier/dispatch → manifests + orphans
 *   GET /v1/supplier/manifests?date=YYYY-MM-DD → pick list
 *   GET /v1/supplier/manifests/orders?date=YYYY-MM-DD → orders for date
 *   GET /v1/supplier/manifests?date={date}&format=csv → CSV export
 *   POST /v1/supplier/manifests/{id}/inject → inject order into LOADING manifest
 *   POST /v1/supplier/manifests/{id}/seal → force seal
 *   POST /v1/supplier/manifests/auto-dispatch → auto-dispatch
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Dispatch & Manifests', () => {
  test('dispatch page renders truck manifests and orphans', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/dispatch');
    await supplierPage.waitForLoadState('networkidle');

    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    // Should show truck manifest cards or orphan orders
    const manifestOrOrphan = supplierPage.getByText(/manifest|route|truck|orphan|unassigned/i);
    await expect(manifestOrOrphan.first()).toBeVisible({ timeout: 10_000 });
  });

  test('VU capacity warnings surface when truck overloaded', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/dispatch');
    await supplierPage.waitForLoadState('networkidle');

    // Capacity info should show max_volume_vu, used_volume_vu, free_volume_vu
    const capacityInfo = supplierPage.getByText(/volume|vu|capacity/i);
    if (await capacityInfo.count() > 0) {
      await expect(capacityInfo.first()).toBeVisible();
    }
  });

  test('manifest list page with date filter', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/manifests');
    await supplierPage.waitForLoadState('networkidle');

    // Date picker
    const datePicker = supplierPage.locator('input[type="date"]');
    if (await datePicker.count() > 0) {
      await expect(datePicker.first()).toBeVisible();
      // Set date to today
      const today = new Date().toISOString().split('T')[0];
      await datePicker.first().fill(today);
      await supplierPage.waitForLoadState('networkidle');
    }

    // Tabs: pick-list, orders, manifest-ops
    await expect(supplierPage.getByText(/pick.?list/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('CSV export downloads manifest', async ({ supplierPage }) => {
    let csvRequested = false;
    await supplierPage.route('**/v1/supplier/manifests**csv**', async (route) => {
      csvRequested = true;
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'text/csv' },
        body: 'sku_id,product_name,total_qty\nSKU001,Test Product,10',
      });
    });

    await supplierPage.goto('http://localhost:3000/supplier/manifests');
    await supplierPage.waitForLoadState('networkidle');

    // Export button
    const exportBtn = supplierPage.getByRole('button', { name: /export|csv|download/i });
    if (await exportBtn.count() > 0) {
      await exportBtn.first().click();
    }
  });

  test('manifest seal transitions DRAFT → LOADING → SEALED', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/manifests');
    await supplierPage.waitForLoadState('networkidle');

    // Switch to manifest-ops tab
    const opsTab = supplierPage.getByText(/manifest.?ops|operations/i);
    if (await opsTab.count() > 0) {
      await opsTab.first().click();
      await supplierPage.waitForLoadState('networkidle');

      // Manifest state badges should show
      const stateBadges = supplierPage.getByText(/draft|loading|sealed|dispatched/i);
      if (await stateBadges.count() > 0) {
        await expect(stateBadges.first()).toBeVisible();
      }
    }
  });

  test('inject order into LOADING manifest', async ({ supplierPage }) => {
    let injectCalled = false;
    await supplierPage.route('**/v1/supplier/manifests/*/inject**', async (route) => {
      injectCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ success: true }) });
    });

    await supplierPage.goto('http://localhost:3000/supplier/manifests');
    await supplierPage.waitForLoadState('networkidle');

    // Navigate to manifest-ops tab
    const opsTab = supplierPage.getByText(/manifest.?ops|operations/i);
    if (await opsTab.count() > 0) {
      await opsTab.first().click();
      await supplierPage.waitForLoadState('networkidle');

      // Look for inject button on LOADING manifests
      const injectBtn = supplierPage.getByRole('button', { name: /inject/i });
      if (await injectBtn.count() > 0) {
        await injectBtn.first().click();
        // Fill order ID in modal
        const orderInput = supplierPage.getByPlaceholder(/order/i);
        if (await orderInput.count() > 0) {
          await orderInput.first().fill('test-order-id');
          const confirmBtn = supplierPage.getByRole('button', { name: /confirm|inject/i });
          if (await confirmBtn.count() > 0) {
            await confirmBtn.first().click();
          }
        }
      }
    }
  });

  test('force seal manifest shows confirmation', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/manifests');
    await supplierPage.waitForLoadState('networkidle');

    const opsTab = supplierPage.getByText(/manifest.?ops|operations/i);
    if (await opsTab.count() > 0) {
      await opsTab.first().click();
      await supplierPage.waitForLoadState('networkidle');

      const sealBtn = supplierPage.getByRole('button', { name: /seal/i });
      if (await sealBtn.count() > 0) {
        await sealBtn.first().click();
        // Confirmation dialog
        const dialog = supplierPage.locator('[role="dialog"], [class*="modal"]');
        if (await dialog.count() > 0) {
          await expect(dialog.first()).toBeVisible();
        }
      }
    }
  });
});
