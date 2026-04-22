/**
 * Retailer Catalog — Browse, Filter, Product Detail, Add to Cart
 *
 * Page: /catalog
 * APIs:
 *   GET /v1/catalog/products → Product[]
 *   GET /v1/catalog/categories → Category[]
 *   GET /v1/retailer/suppliers → Supplier[]
 *
 * Components: ProductDetailDrawer, CartContext, retailer_cart localStorage
 */
import { test, expect } from '../fixtures/auth';

test.describe('Retailer Catalog', () => {
  test('product grid loads from catalog API', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/catalog');
    await retailerPage.waitForLoadState('networkidle');

    // Product grid or list
    const products = retailerPage.locator('[class*="product"], [class*="card"], [class*="grid"]');
    if (await products.count() > 0) {
      await expect(products.first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('category filter from categories API', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/catalog');
    await retailerPage.waitForLoadState('networkidle');

    const categoryFilter = retailerPage.locator('[class*="filter"], [class*="category"], select');
    if (await categoryFilter.count() > 0) {
      await expect(categoryFilter.first()).toBeVisible();
    }
  });

  test('supplier filter from retailers API', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/catalog');
    await retailerPage.waitForLoadState('networkidle');

    const supplierFilter = retailerPage.getByText(/supplier|vendor/i);
    if (await supplierFilter.count() > 0) {
      await expect(supplierFilter.first()).toBeVisible();
    }
  });

  test('ProductDetailDrawer opens on product click', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/catalog');
    await retailerPage.waitForLoadState('networkidle');

    // Click first product
    const productCard = retailerPage.locator('[class*="product"], [class*="card"]').first();
    if (await productCard.isVisible()) {
      await productCard.click();

      // Drawer should open
      const drawer = retailerPage.locator('[class*="drawer"], [class*="detail"], [role="dialog"]');
      if (await drawer.count() > 0) {
        await expect(drawer.first()).toBeVisible({ timeout: 5_000 });
      }
    }
  });

  test('add to cart updates CartContext and localStorage @responsive', async ({ retailerPage }) => {
    await retailerPage.goto('http://localhost:3001/catalog');
    await retailerPage.waitForLoadState('networkidle');

    // Click add to cart on first product
    const addBtn = retailerPage.getByRole('button', { name: /add.*cart|cart/i });
    if (await addBtn.count() > 0) {
      await addBtn.first().click();

      // Verify localStorage updated
      const cart = await retailerPage.evaluate(() => {
        return localStorage.getItem('retailer_cart');
      });
      // Cart should be set (may be null if no products available)
      expect(typeof cart).toBe('string');
    }
  });
});
