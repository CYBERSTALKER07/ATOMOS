/**
 * V.O.I.D. Playwright — WebSocket Helpers
 *
 * Utilities for intercepting and asserting WebSocket messages
 * in Playwright browser tests. Each portal has its own WS hub:
 *   - Supplier: /ws/telemetry
 *   - Retailer: /v1/ws/retailer
 *   - Driver:   /v1/ws/driver (API-only, tested via raw WS)
 *   - Warehouse: /ws/warehouse
 */
import { type Page } from '@playwright/test';

export interface WSMessage {
  type: string;
  [key: string]: unknown;
}

/**
 * Collect WebSocket messages matching a given URL pattern.
 * Returns a message collector that accumulates received messages.
 */
export function collectWSMessages(page: Page, urlPattern: string | RegExp) {
  const messages: WSMessage[] = [];
  let resolve: ((msg: WSMessage) => void) | null = null;

  page.on('websocket', (ws) => {
    const url = ws.url();
    const matches = typeof urlPattern === 'string' ? url.includes(urlPattern) : urlPattern.test(url);
    if (!matches) return;

    ws.on('framereceived', (frame) => {
      try {
        const parsed = JSON.parse(frame.payload as string) as WSMessage;
        messages.push(parsed);
        if (resolve) {
          resolve(parsed);
          resolve = null;
        }
      } catch { /* binary frame or non-JSON — skip */ }
    });
  });

  return {
    messages,
    /** Wait for the next message matching the optional type filter */
    waitForMessage(type?: string, timeoutMs = 10_000): Promise<WSMessage> {
      const existing = type ? messages.find((m) => m.type === type) : messages[0];
      if (existing) return Promise.resolve(existing);

      return new Promise((res, rej) => {
        const timer = setTimeout(() => rej(new Error(`WS timeout waiting for ${type || 'any'}`)), timeoutMs);
        resolve = (msg: WSMessage) => {
          if (!type || msg.type === type) {
            clearTimeout(timer);
            res(msg);
          }
        };
      });
    },
    /** Get all messages of a specific type */
    ofType(type: string): WSMessage[] {
      return messages.filter((m) => m.type === type);
    },
    /** Total message count */
    get count() {
      return messages.length;
    },
  };
}

/**
 * Wait for a WebSocket connection to be established on the page.
 */
export function waitForWSConnection(page: Page, urlPattern: string | RegExp, timeoutMs = 10_000): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`WS connect timeout for ${urlPattern}`)), timeoutMs);
    page.on('websocket', (ws) => {
      const url = ws.url();
      const matches = typeof urlPattern === 'string' ? url.includes(urlPattern) : urlPattern.test(url);
      if (matches) {
        clearTimeout(timer);
        resolve();
      }
    });
  });
}

/**
 * Delta-Sync event type constants (matches admin-portal delta-sync.ts)
 */
export const DELTA_EVENTS = {
  ORD_UP: 'ORD_UP',     // Order update
  DRV_UP: 'DRV_UP',     // Driver update
  FLT_GPS: 'FLT_GPS',   // Fleet GPS ping
  WH_LOAD: 'WH_LOAD',   // Warehouse load event
  PAY_UP: 'PAY_UP',     // Payment update
  RTE_UP: 'RTE_UP',     // Route update
  NEG_UP: 'NEG_UP',     // Negotiation update
  CRD_UP: 'CRD_UP',     // Credit update
} as const;

/**
 * Short-key mappings for delta-sync compression (matches patcher.ts)
 */
export const SHORT_KEYS: Record<string, string> = {
  s: 'status',
  l: 'location',
  v: 'volumetric_units',
  t: 'timestamp',
  d: 'driver_id',
  r: 'retailer_id',
  o: 'order_id',
  m: 'manifest_id',
};
