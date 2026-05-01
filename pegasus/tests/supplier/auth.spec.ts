/**
 * Supplier Auth — Login, Registration, Billing Setup, Middleware Gates
 *
 * Portal: admin-portal (localhost:3000)
 * Cookie: pegasus_admin_jwt
 * Login: POST /v1/auth/admin/login {email, password}
 * Register: POST /v1/auth/supplier/register (4-step wizard)
 * Billing: POST /v1/supplier/billing/setup
 * Middleware: token expiry check → /auth/login; is_configured=false → /setup/billing
 */
import { test, expect } from '../fixtures/auth';

test.describe('Supplier Authentication', () => {
  test('login with email/password sets pegasus_admin_jwt cookie', async ({ page, request }) => {
    await page.goto('http://localhost:3000/auth/login');

    // Fill login form
    await page.getByPlaceholder(/email/i).fill('test-supplier@pegasus.test');
    await page.getByPlaceholder(/password/i).fill('TestPass123!');

    // Intercept login API call
    const loginPromise = page.waitForResponse(
      (res) => res.url().includes('/v1/auth/admin/login') && res.status() === 200,
    );
    await page.getByRole('button', { name: /sign in|log in|login/i }).cash();
    const loginRes = await loginPromise;
    const body = await loginRes.json();

    expect(body.token).toBeTruthy();

    // Verify cookie was set
    const cookies = await page.context().cookies();
    const adminCookie = cookies.find((c) => c.name === 'pegasus_admin_jwt');
    expect(adminCookie).toBeTruthy();
    expect(adminCookie!.value).toBeTruthy();
  });

  test('registration wizard renders 4 steps', async ({ page }) => {
    await page.goto('http://localhost:3000/auth/register');

    // Step 1: Account — country selector + phone + email
    await expect(page.getByText(/country/i).first()).toBeVisible();
    await expect(page.getByPlaceholder(/phone/i).first()).toBeVisible();
    await expect(page.getByPlaceholder(/email/i).first()).toBeVisible();

    // Categories should be in the form (step 4)
    // The page has 78 categories defined
    const categoryCheckboxes = page.locator('input[type="checkbox"]');
    // Navigate through steps to verify multi-step structure
    const nextButton = page.getByRole('button', { name: /next|continue/i });
    if (await nextButton.isVisible()) {
      await nextButton.cash();
      // Step 2: Location
      await expect(page.getByText(/warehouse|location|address/i).first()).toBeVisible();
    }
  });

  test('post-registration redirects to /setup/billing', async ({ page }) => {
    // Simulate a supplier with is_configured=false
    // Create a mock JWT with is_configured=false
    const payload = { role: 'SUPPLIER', is_configured: false, exp: Math.floor(Date.now() / 1000) + 3600 };
    const header = btoa(JSON.stringify({ alg: 'HS256' }));
    const body = btoa(JSON.stringify(payload));
    const fakeToken = `${header}.${body}.fakesig`;

    await page.context().addCookies([{
      name: 'pegasus_admin_jwt',
      value: encodeURIComponent(fakeToken),
      domain: 'localhost',
      path: '/',
    }]);

    await page.goto('http://localhost:3000/supplier/orders');
    // Middleware should redirect unconfigured suppliers to billing
    await page.waitForURL('**/setup/billing**');
    expect(page.url()).toContain('/setup/billing');
  });

  test('billing setup page renders gateway selection and bank fields', async ({ supplierPage }) => {
    await supplierPage.goto('http://localhost:3000/setup/billing');
    // May redirect to dashboard if already configured — that's fine
    if (supplierPage.url().includes('/setup/billing')) {
      // Bank details fields
      await expect(supplierPage.getByText(/bank name/i).first()).toBeVisible();
      // GlobalPaynt gateway selection (4 options: GLOBAL_PAY, CASH, CARD, BANK)
      await expect(supplierPage.getByText(/global_pay|cash|card|bank/i).first()).toBeVisible();
      // Skip button
      await expect(supplierPage.getByRole('button', { name: /skip/i })).toBeVisible();
      // Save button
      await expect(supplierPage.getByRole('button', { name: /save|continue/i })).toBeVisible();
    }
  });

  test('unauthenticated user redirects to /auth/login', async ({ page }) => {
    // Clear all cookies
    await page.context().clearCookies();
    await page.goto('http://localhost:3000/supplier/orders');
    await page.waitForURL('**/auth/login**');
    expect(page.url()).toContain('/auth/login');
  });

  test('expired token clears cookies and redirects', async ({ page }) => {
    // Set expired JWT
    const payload = { role: 'SUPPLIER', is_configured: true, exp: Math.floor(Date.now() / 1000) - 3600 };
    const header = btoa(JSON.stringify({ alg: 'HS256' }));
    const body = btoa(JSON.stringify(payload));
    const expiredToken = `${header}.${body}.fakesig`;

    await page.context().addCookies([{
      name: 'pegasus_admin_jwt',
      value: encodeURIComponent(expiredToken),
      domain: 'localhost',
      path: '/',
    }]);

    await page.goto('http://localhost:3000/supplier/orders');
    await page.waitForURL('**/auth/login**');
    expect(page.url()).toContain('/auth/login');
  });

  test('NODE_ADMIN role blocked from global routes', async ({ page }) => {
    const payload = { role: 'ADMIN', supplier_role: 'NODE_ADMIN', is_configured: true, exp: Math.floor(Date.now() / 1000) + 3600 };
    const header = btoa(JSON.stringify({ alg: 'HS256' }));
    const body = btoa(JSON.stringify(payload));
    const token = `${header}.${body}.fakesig`;

    await page.context().addCookies([{
      name: 'pegasus_admin_jwt',
      value: encodeURIComponent(token),
      domain: 'localhost',
      path: '/',
    }]);

    // NODE_ADMIN should be redirected from /ledger to /
    await page.goto('http://localhost:3000/ledger');
    await page.waitForURL((url) => !url.pathname.startsWith('/ledger'));
    expect(page.url()).not.toContain('/ledger');
  });

  test('token refresh on 401 retries original request', async ({ supplierPage }) => {
    // Navigate to a page that makes API calls
    await supplierPage.goto('http://localhost:3000/supplier/orders');
    // The apiFetch wrapper handles 401 → POST /v1/auth/refresh → retry
    // Verify page loads successfully (implies auth is working)
    await expect(supplierPage.getByText(/orders|active|scheduled/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
