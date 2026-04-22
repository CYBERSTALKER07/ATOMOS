/**
 * V.O.I.D. Playwright — API Fixtures
 *
 * Pre-authenticated APIRequestContext instances for each role.
 * Use for API-only tests (Driver, Payloader) and cross-role integration.
 */
import { test as base, type APIRequestContext } from '@playwright/test';
import { API, supplierLogin, retailerLogin, factoryLogin, warehouseLogin, driverLogin, payloaderLogin } from './auth';

type APIFixtures = {
  supplierAPI: APIRequestContext;
  retailerAPI: APIRequestContext;
  factoryAPI: APIRequestContext;
  warehouseAPI: APIRequestContext;
  driverAPI: APIRequestContext;
  payloaderAPI: APIRequestContext;
};

export const test = base.extend<APIFixtures>({
  supplierAPI: async ({ playwright, request }, use) => {
    const { token } = await supplierLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
  retailerAPI: async ({ playwright, request }, use) => {
    const { token } = await retailerLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
  factoryAPI: async ({ playwright, request }, use) => {
    const { token } = await factoryLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
  warehouseAPI: async ({ playwright, request }, use) => {
    const { token } = await warehouseLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
  driverAPI: async ({ playwright, request }, use) => {
    const { token } = await driverLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
  payloaderAPI: async ({ playwright, request }, use) => {
    const { token } = await payloaderLogin(request);
    const ctx = await playwright.request.newContext({
      baseURL: API,
      extraHTTPHeaders: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    });
    await use(ctx);
    await ctx.dispose();
  },
});

export { expect } from '@playwright/test';
