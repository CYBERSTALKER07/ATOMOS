/**
 * V.O.I.D. Playwright — Auth Fixtures
 *
 * Provides authenticated browser contexts for every role in the ecosystem.
 * Each fixture logs in via the backend API, sets the correct cookie,
 * and returns a ready-to-use Page object.
 */
import { test as base, type Page, type BrowserContext, type APIRequestContext } from '@playwright/test';

const env = (globalThis as typeof globalThis & {
  process?: { env?: Record<string, string | undefined> };
}).process?.env ?? {};

const API = env.API_BASE_URL || 'http://localhost:8080';

/* ── Credential types per role ── */
export interface SupplierCredentials { email: string; password: string }
export interface RetailerCredentials { phone_number: string; password: string }
export interface FactoryCredentials { phone: string; password: string }
export interface WarehouseCredentials { phone: string; pin: string }
export interface DriverCredentials { phone: string; pin: string }
export interface PayloaderCredentials { phone: string; pin: string }

/* ── Default test credentials (override via env) ── */
const SUPPLIER_CREDS: SupplierCredentials = {
  email: env.TEST_SUPPLIER_EMAIL || 'info@pegasusbeverages.uz',
  password: env.TEST_SUPPLIER_PASSWORD || 'password123',
};
const RETAILER_CREDS: RetailerCredentials = {
  phone_number: env.TEST_RETAILER_PHONE || '+998901234567',
  password: env.TEST_RETAILER_PASSWORD || 'password123',
};
const FACTORY_CREDS: FactoryCredentials = {
  phone: env.TEST_FACTORY_PHONE || '+998901234568',
  password: env.TEST_FACTORY_PASSWORD || 'TestPass123!',
};
const WAREHOUSE_CREDS: WarehouseCredentials = {
  phone: env.TEST_WAREHOUSE_PHONE || '+998901234569',
  pin: env.TEST_WAREHOUSE_PIN || '123456',
};
const DRIVER_CREDS: DriverCredentials = {
  phone: env.TEST_DRIVER_PHONE || '+998909876543',
  pin: env.TEST_DRIVER_PIN || '123456',
};
const PAYLOADER_CREDS: PayloaderCredentials = {
  phone: env.TEST_PAYLOADER_PHONE || '+998905551234',
  pin: env.TEST_PAYLOADER_PIN || '654321',
};

/* ── Login helpers (API-level, returns JWT) ── */
async function loginViaAPI(
  request: APIRequestContext,
  endpoint: string,
  body: object,
): Promise<{ token: string; [key: string]: unknown }> {
  const res = await request.post(`${API}${endpoint}`, { data: body });
  if (!res.ok()) {
    const text = await res.text();
    throw new Error(`Login failed [${res.status()}] ${endpoint}: ${text}`);
  }
  return res.json();
}

async function supplierLogin(request: APIRequestContext, creds = SUPPLIER_CREDS) {
  return loginViaAPI(request, '/v1/auth/admin/login', creds);
}

async function retailerLogin(request: APIRequestContext, creds = RETAILER_CREDS) {
  return loginViaAPI(request, '/v1/auth/retailer/login', creds);
}

async function factoryLogin(request: APIRequestContext, creds = FACTORY_CREDS) {
  return loginViaAPI(request, '/v1/auth/factory/login', creds);
}

async function warehouseLogin(request: APIRequestContext, creds = WAREHOUSE_CREDS) {
  return loginViaAPI(request, '/v1/auth/warehouse/login', creds);
}

async function driverLogin(request: APIRequestContext, creds = DRIVER_CREDS) {
  return loginViaAPI(request, '/v1/auth/driver/login', creds);
}

async function payloaderLogin(request: APIRequestContext, creds = PAYLOADER_CREDS) {
  return loginViaAPI(request, '/v1/auth/payloader/login', creds);
}

/* ── Cookie setter: injects JWT into browser context ── */
async function setAuthCookie(
  context: BrowserContext,
  cookieName: string,
  token: string,
  domain: string,
) {
  await context.addCookies([{
    name: cookieName,
    value: encodeURIComponent(token),
    domain,
    path: '/',
    httpOnly: false,
    secure: false,
    sameSite: 'Lax',
  }]);
}

/* ── Extended test fixtures ── */
type AuthFixtures = {
  supplierPage: Page;
  retailerPage: Page;
  factoryPage: Page;
  warehousePage: Page;
  supplierToken: string;
  retailerToken: string;
  factoryToken: string;
  warehouseToken: string;
  driverToken: string;
  payloaderToken: string;
  authedAPI: {
    supplier: APIRequestContext;
    retailer: APIRequestContext;
    factory: APIRequestContext;
    warehouse: APIRequestContext;
    driver: APIRequestContext;
    payloader: APIRequestContext;
  };
};

export const test = base.extend<AuthFixtures>({
  /* ── Supplier (admin-portal) ── */
  supplierToken: async ({ request }, use) => {
    const data = await supplierLogin(request);
    await use(data.token);
  },
  supplierPage: async ({ browser, supplierToken }, use) => {
    const context = await browser.newContext();
    await setAuthCookie(context, 'pegasus_admin_jwt', supplierToken, 'localhost');
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  /* ── Retailer (retailer-app-desktop) ── */
  retailerToken: async ({ request }, use) => {
    const data = await retailerLogin(request);
    await use(data.token);
  },
  retailerPage: async ({ browser, retailerToken }, use) => {
    const context = await browser.newContext();
    await setAuthCookie(context, 'pegasus_retailer_jwt', retailerToken, 'localhost');
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  /* ── Factory (factory-portal) ── */
  factoryToken: async ({ request }, use) => {
    const data = await factoryLogin(request);
    await use(data.token);
  },
  factoryPage: async ({ browser, factoryToken }, use) => {
    const context = await browser.newContext();
    await setAuthCookie(context, 'pegasus_factory_jwt', factoryToken, 'localhost');
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  /* ── Warehouse (warehouse-portal) ── */
  warehouseToken: async ({ request }, use) => {
    const data = await warehouseLogin(request);
    await use(data.token);
  },
  warehousePage: async ({ browser, warehouseToken }, use) => {
    const context = await browser.newContext();
    await setAuthCookie(context, 'pegasus_warehouse_jwt', warehouseToken, 'localhost');
    const page = await context.newPage();
    await use(page);
    await context.close();
  },

  /* ── API-only tokens (Driver + Payloader) ── */
  driverToken: async ({ request }, use) => {
    const data = await driverLogin(request);
    await use(data.token);
  },
  payloaderToken: async ({ request }, use) => {
    const data = await payloaderLogin(request);
    await use(data.token);
  },

  /* ── Authenticated API request contexts (all roles) ── */
  authedAPI: async ({ playwright, supplierToken, retailerToken, factoryToken, warehouseToken, driverToken, payloaderToken }, use) => {
    const mkCtx = (token: string) =>
      playwright.request.newContext({
        baseURL: API,
        extraHTTPHeaders: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      });

    const [supplier, retailer, factory, warehouse, driver, payloader] = await Promise.all([
      mkCtx(supplierToken),
      mkCtx(retailerToken),
      mkCtx(factoryToken),
      mkCtx(warehouseToken),
      mkCtx(driverToken),
      mkCtx(payloaderToken),
    ]);

    await use({ supplier, retailer, factory, warehouse, driver, payloader });

    await Promise.all([
      supplier.dispose(),
      retailer.dispose(),
      factory.dispose(),
      warehouse.dispose(),
      driver.dispose(),
      payloader.dispose(),
    ]);
  },
});

export { expect } from '@playwright/test';

/* ── Re-export login helpers for direct use ── */
export {
  supplierLogin,
  retailerLogin,
  factoryLogin,
  warehouseLogin,
  driverLogin,
  payloaderLogin,
  setAuthCookie,
  API,
  SUPPLIER_CREDS,
  RETAILER_CREDS,
  FACTORY_CREDS,
  WAREHOUSE_CREDS,
  DRIVER_CREDS,
  PAYLOADER_CREDS,
};
