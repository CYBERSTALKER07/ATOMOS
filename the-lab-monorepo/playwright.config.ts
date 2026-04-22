import { defineConfig, devices } from '@playwright/test';

const API_BASE = process.env.API_BASE_URL || 'http://localhost:8080';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],
  timeout: 30_000,
  expect: { timeout: 5_000 },

  use: {
    baseURL: API_BASE,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  projects: [
    /* ── Web Portal: Admin (Supplier) ── */
    {
      name: 'admin-portal',
      testDir: './tests/supplier',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:3000',
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'admin-portal-mobile',
      testDir: './tests/supplier',
      testMatch: /.*@responsive.*/,
      use: {
        ...devices['iPhone 15 Pro'],
        baseURL: 'http://localhost:3000',
      },
    },
    {
      name: 'admin-portal-tablet',
      testDir: './tests/supplier',
      testMatch: /.*@responsive.*/,
      use: {
        ...devices['iPad Pro 11'],
        baseURL: 'http://localhost:3000',
      },
    },

    /* ── Web Portal: Retailer Desktop ── */
    {
      name: 'retailer-desktop',
      testDir: './tests/retailer',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:3001',
        viewport: { width: 1920, height: 1080 },
      },
    },
    {
      name: 'retailer-mobile',
      testDir: './tests/retailer',
      testMatch: /.*@responsive.*/,
      use: {
        ...devices['Galaxy S24'],
        baseURL: 'http://localhost:3001',
      },
    },

    /* ── Web Portal: Factory ── */
    {
      name: 'factory-portal',
      testDir: './tests/factory',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:3002',
        viewport: { width: 1920, height: 1080 },
      },
    },

    /* ── Web Portal: Warehouse ── */
    {
      name: 'warehouse-portal',
      testDir: './tests/warehouse',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: 'http://localhost:3003',
        viewport: { width: 1920, height: 1080 },
      },
    },

    /* ── API-only: Driver + Payloader (no browser) ── */
    {
      name: 'api',
      testDir: './tests',
      testMatch: /.*-api\/.*\.spec\.ts/,
      use: {
        baseURL: API_BASE,
      },
    },

    /* ── Cross-Role Integration ── */
    {
      name: 'cross-role',
      testDir: './tests/cross-role',
      use: {
        baseURL: API_BASE,
      },
    },
  ],

  /* Dev server startup — start all 4 portals */
  webServer: [
    {
      command: 'cd apps/admin-portal && npm run dev',
      port: 3000,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
    },
    {
      command: 'cd apps/retailer-app-desktop && npm run dev',
      port: 3001,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
    },
    {
      command: 'cd apps/factory-portal && npm run dev',
      port: 3002,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
    },
    {
      command: 'cd apps/warehouse-portal && npm run dev',
      port: 3003,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
    },
  ],
});
