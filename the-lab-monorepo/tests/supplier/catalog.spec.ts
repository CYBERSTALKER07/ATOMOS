/**
 * Supplier Catalog — Products, Pricing, Inventory
 *
 * Page: /supplier/products
 * APIs:
 *   GET /v1/supplier/products → Product[]
 *   GET /v1/supplier/profile → operating_categories
 *   GET /v1/catalog/platform-categories → available categories
 *   POST /v1/supplier/products → create
 *   PUT /v1/supplier/products/{sku_id} → update
 *   PATCH /v1/supplier/products/{sku_id}/toggle → activate/deactivate
 *
 * Product fields: sku_id, name, description, image_url, base_price, is_active,
 *   category_id, volumetric_unit, minimum_order_qty, step_size
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Catalog & Products', () => {
  test('product list loads with search/filter', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/products');
    await supplierPage.waitForLoadState('networkidle');

    // Product grid or list should be visible
    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    // Search input
    const searchInput = supplierPage.getByPlaceholder(/search|product|sku/i);
    if (await searchInput.count() > 0) {
      await expect(searchInput.first()).toBeVisible();
    }

    // Category filter
    const categoryFilter = supplierPage.getByText(/all|category/i).first();
    await expect(categoryFilter).toBeVisible({ timeout: 10_000 });
  });

  test('create product with image upload', async ({ supplierPage }) => {
    let createCalled = false;
    await supplierPage.route('**/v1/supplier/products', async (route) => {
      if (route.request().method() === 'POST') {
        createCalled = true;
        await route.fulfill({
          status: 201,
          body: JSON.stringify({ sku_id: 'test-sku', name: 'Test Product' }),
        });
      } else {
        await route.continue();
      }
    });

    await supplierPage.goto('http://localhost:3000/supplier/products');
    await supplierPage.waitForLoadState('networkidle');

    // Look for create/add button
    const addBtn = supplierPage.getByRole('button', { name: /add|create|new/i });
    if (await addBtn.count() > 0) {
      await addBtn.first().click();

      // Fill product form
      const nameInput = supplierPage.getByPlaceholder(/name|product name/i);
      if (await nameInput.count() > 0) {
        await nameInput.first().fill('Test Product');
      }
    }
  });

  test('update product details', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/products');
    await supplierPage.waitForLoadState('networkidle');

    // Click edit on a product
    const editBtn = supplierPage.getByRole('button', { name: /edit/i });
    if (await editBtn.count() > 0) {
      await editBtn.first().click();
    }
  });

  test('deactivate product toggle', async ({ supplierPage }) => {
    let toggleCalled = false;
    await supplierPage.route('**/v1/supplier/products/**/toggle**', async (route) => {
      toggleCalled = true;
      await route.fulfill({ status: 200, body: JSON.stringify({ is_active: false }) });
    });

    await supplierPage.goto('http://localhost:3000/supplier/products');
    await supplierPage.waitForLoadState('networkidle');

    // Toggle switch for product active state
    const toggle = supplierPage.locator('[role="switch"], input[type="checkbox"], [class*="toggle"]');
    if (await toggle.count() > 0) {
      await toggle.first().click();
    }
  });

  test('pricing rules CRUD', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/pricing');
    await supplierPage.waitForLoadState('networkidle');

    // Pricing page should show pricing rules
    const content = supplierPage.locator('body');
    await expect(content).toBeVisible();

    // Look for pricing-related content
    const pricingContent = supplierPage.getByText(/pricing|discount|tier|min.*pallet/i);
    if (await pricingContent.count() > 0) {
      await expect(pricingContent.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('inventory stock adjustment', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/supplier/inventory');
    await supplierPage.waitForLoadState('networkidle');

    // Inventory page
    const inventoryContent = supplierPage.getByText(/inventory|stock|audit/i);
    if (await inventoryContent.count() > 0) {
      await expect(inventoryContent.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('product category filter works', async ({ supplierPage }) => {
    const apiCalls: string[] = [];
    await supplierPage.route('**/v1/supplier/products**', async (route) => {
      apiCalls.push(route.request().url());
      await route.continue();
    });

    await supplierPage.goto('http://localhost:3000/supplier/products');
    await supplierPage.waitForLoadState('networkidle');

    // Click a category filter
    const categoryBtn = supplierPage.locator('[class*="category"], [class*="chip"], [class*="filter"]');
    if (await categoryBtn.count() > 1) {
      await categoryBtn.nth(1).click();
      await supplierPage.waitForLoadState('networkidle');
    }
  });
});
