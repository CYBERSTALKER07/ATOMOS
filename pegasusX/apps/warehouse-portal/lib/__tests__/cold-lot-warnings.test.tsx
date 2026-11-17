import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PickWave, StockLot, WarehouseBinLocation } from '@pegasusx/types';
import {
  analyzePickWavePreSealColdState,
  analyzeWarehouseLots,
  COLD_LOT_MIGRATION_WARNING,
} from '../cold-lot-warnings';

const { apiFetchMock, toastMock } = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  toastMock: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock('next/link', () => ({
  default: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

vi.mock('@/lib/i18n', () => ({
  usePortalT: () => (key: string, params?: Record<string, string | number>) => {
    if (!params) return key;
    return `${key}(${Object.entries(params)
      .map(([k, v]) => `${k}=${v}`)
      .join(',')})`;
  },
}));

vi.mock('@/components/PageTransition', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('@/components/PageChrome', () => ({
  PageChrome: ({ title, children }: { title: string; children: React.ReactNode }) => (
    <div>
      <h1>{title}</h1>
      {children}
    </div>
  ),
}));

vi.mock('@/components/EmptyState', () => ({
  default: ({ headline }: { headline: string }) => <p>{headline}</p>,
}));

vi.mock('@/components/Toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

vi.mock('@/lib/auth', () => ({
  apiFetch: apiFetchMock,
}));

import BinsPage from '../../app/bins/page';
import PickWavesPage from '../../app/pick-waves/page';

function jsonResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('cold-lot migration warnings and putaway', () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    toastMock.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it('flags migrated cold lots in staging and ambient bins while clearing chill bins', () => {
    const bins: WarehouseBinLocation[] = [
      { warehouse_id: 'wh-1', location_id: 'recv-default', zone: 'RECV', location_type: 'STAGE', pick_sequence: 0, is_active: true },
      { warehouse_id: 'wh-1', location_id: 'C-01', zone: 'CHILL', location_type: 'PICK', pick_sequence: 1, is_active: true },
      { warehouse_id: 'wh-1', location_id: 'F-01', zone: 'FREEZE', location_type: 'PICK', pick_sequence: 2, is_active: true },
    ];
    const lots: StockLot[] = [
      {
        lot_id: 'lot-milk',
        supplier_id: 'sup-1',
        warehouse_id: 'wh-1',
        product_id: 'sku-milk',
        product_name: 'Kefir 1L',
        location_id: 'recv-default',
        lot_code: 'BATCH-M1',
        expiry_date: '2026-11-16',
        quantity_on_hand: 24,
        quantity_reserved: 0,
        status: 'AVAILABLE',
        requires_cold_chain: true,
        handling_class: 'CHILLED',
        cold_lot_warning: COLD_LOT_MIGRATION_WARNING,
      },
      {
        lot_id: 'lot-icecream',
        supplier_id: 'sup-1',
        warehouse_id: 'wh-1',
        product_id: 'sku-ice',
        product_name: 'Ice Cream',
        location_id: 'C-01',
        lot_code: 'BATCH-I1',
        quantity_on_hand: 10,
        quantity_reserved: 0,
        status: 'AVAILABLE',
        requires_cold_chain: true,
        handling_class: 'FROZEN',
      },
      {
        lot_id: 'lot-yogurt-ok',
        supplier_id: 'sup-1',
        warehouse_id: 'wh-1',
        product_id: 'sku-yog',
        product_name: 'Yogurt',
        location_id: 'C-01',
        lot_code: 'BATCH-Y1',
        quantity_on_hand: 15,
        quantity_reserved: 0,
        status: 'AVAILABLE',
        requires_cold_chain: true,
        handling_class: 'CHILLED',
      },
    ];

    const summary = analyzeWarehouseLots(lots, bins, Date.parse('2026-11-14T00:00:00Z'));
    expect(summary.coldWarningLots.map((a) => a.lot.lot_id)).toEqual(['lot-milk', 'lot-icecream']);
    expect(summary.stagingLots.map((a) => a.lot.lot_id)).toEqual(['lot-milk']);
    expect(summary.expiryAlertLots.map((a) => a.lot.lot_id)).toEqual(['lot-milk']);

    const wave: PickWave = {
      wave_id: 'w-1',
      warehouse_id: 'wh-1',
      supplier_id: 'sup-1',
      manifest_id: 'man-1',
      strategy: 'FEFO',
      status: 'OPEN',
      tasks: [
        {
          task_id: 't-1',
          order_id: 'o-1',
          product_id: 'sku-milk',
          lot_id: 'lot-milk',
          location_id: 'recv-default',
          quantity_requested: 2,
          quantity_picked: 0,
          status: 'PENDING',
          pick_sequence: 1,
        },
      ],
    };
    const preSeal = analyzePickWavePreSealColdState(wave, lots, bins, 0);
    expect(preSeal.hasPreSealWarning).toBe(true);
    expect(preSeal.missingBaseline).toBe(true);
    expect(preSeal.flaggedTasks).toHaveLength(1);
  });

  it('renders cold_lot_migrated_without_live_baseline banner on BinsPage and submits cold putaway', async () => {
    const requests: Array<{ path: string; method: string; headers: Record<string, string>; body?: unknown }> = [];
    let currentLocation = 'recv-default';

    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const headers = (init?.headers ?? {}) as Record<string, string>;
      const parsedBody = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
      requests.push({ path, method, headers, body: parsedBody });

      if (path === '/v1/warehouse/ops/bins' && method === 'GET') {
        return jsonResponse({
          bins: [
            { warehouse_id: 'wh-1', location_id: 'recv-default', zone: 'RECV', location_type: 'STAGE', pick_sequence: 0, is_active: true },
            { warehouse_id: 'wh-1', location_id: 'CHILL-01', zone: 'CHILL', location_type: 'PICK', pick_sequence: 1, is_active: true },
          ],
        });
      }
      if (path === '/v1/warehouse/ops/lots' && method === 'GET') {
        return jsonResponse({
          lots_enabled: true,
          lots: [
            {
              lot_id: 'lot-migrated-1',
              warehouse_id: 'wh-1',
              product_id: 'sku-milk',
              product_name: 'Fresh Milk 1L',
              location_id: currentLocation,
              lot_code: 'UZ-MIG-01',
              quantity_on_hand: 30,
              quantity_reserved: 0,
              status: 'AVAILABLE',
              requires_cold_chain: true,
              handling_class: 'CHILLED',
              required_zone: 'CHILL',
              cold_lot_warning: currentLocation === 'recv-default' ? COLD_LOT_MIGRATION_WARNING : undefined,
            },
          ],
        });
      }
      if (path === '/v1/warehouse/ops/lots/putaway' && method === 'POST') {
        currentLocation = 'CHILL-01';
        return jsonResponse({ lot_id: 'lot-migrated-1', location_id: 'CHILL-01', status: 'AVAILABLE' });
      }
      return jsonResponse({ error: 'not_found' }, 404);
    });

    render(<BinsPage />);

    const banner = await screen.findByTestId('cold-lot-warning-banner');
    expect(banner.textContent).toContain(COLD_LOT_MIGRATION_WARNING);
    expect(banner.textContent).toContain('warehouse_portal.cold_lot.warning_banner');

    const confirmBtn = screen
      .getAllByRole('button', { name: 'warehouse_portal.cold_lot.confirm_cold_putaway' })
      .find((b) => b.className.includes('bg-black'))!;
    fireEvent.click(confirmBtn);

    await waitFor(() => {
      expect(requests.some((r) => r.path === '/v1/warehouse/ops/lots/putaway' && r.method === 'POST')).toBe(true);
    });
    const putawayReq = requests.find((r) => r.path === '/v1/warehouse/ops/lots/putaway');
    expect(putawayReq?.body).toEqual({
      product_id: 'sku-milk',
      location_id: 'CHILL-01',
      quantity: 30,
      lot_code: 'UZ-MIG-01',
    });
    expect(putawayReq?.headers['Idempotency-Key']).toMatch(/^warehouse-putaway-/);
  });

  it('surfaces missing cold-chain baseline warning before payload seal on PickWavesPage and allows logging bay temp', async () => {
    const requests: Array<{ path: string; method: string; headers: Record<string, string>; body?: unknown }> = [];
    let readings: Array<{ reading_id: string; temp_c: number }> = [];

    const wave: PickWave = {
      wave_id: 'wave-cold-1',
      warehouse_id: 'wh-1',
      supplier_id: 'sup-1',
      manifest_id: 'man-cold-99',
      strategy: 'FEFO',
      status: 'READY_TO_SEAL',
      cold_lot_migrated_without_live_baseline: true,
      cold_chain_baseline_missing: true,
      tasks: [
        {
          task_id: 't-1',
          order_id: 'o-1',
          product_id: 'sku-milk',
          lot_id: 'lot-cold-1',
          location_id: 'recv-default',
          quantity_requested: 5,
          quantity_picked: 5,
          status: 'CONFIRMED',
          pick_sequence: 1,
        },
      ],
    };

    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const headers = (init?.headers ?? {}) as Record<string, string>;
      const parsedBody = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
      requests.push({ path, method, headers, body: parsedBody });

      if (path === '/v1/warehouse/ops/pick-waves' && method === 'GET') {
        return jsonResponse({ waves: [wave], pick_waves_enabled: true });
      }
      if (path === '/v1/warehouse/ops/pick-waves/wave-cold-1' && method === 'GET') {
        return jsonResponse({
          ...wave,
          cold_chain_baseline_missing: readings.length === 0,
        });
      }
      if (path === '/v1/warehouse/ops/lots') {
        return jsonResponse({
          lots: [
            {
              lot_id: 'lot-cold-1',
              warehouse_id: 'wh-1',
              product_id: 'sku-milk',
              product_name: 'Kefir',
              location_id: 'recv-default',
              quantity_on_hand: 5,
              quantity_reserved: 5,
              status: 'AVAILABLE',
              requires_cold_chain: true,
              required_zone: 'CHILL',
              cold_lot_warning: COLD_LOT_MIGRATION_WARNING,
            },
          ],
        });
      }
      if (path === '/v1/warehouse/ops/bins') {
        return jsonResponse({ bins: [] });
      }
      if (path === '/v1/warehouse/ops/manifests/man-cold-99/temperature' && method === 'GET') {
        return jsonResponse({ manifest_id: 'man-cold-99', readings });
      }
      if (path === '/v1/warehouse/ops/manifests/man-cold-99/temperature' && method === 'POST') {
        readings = [{ reading_id: 'rd-1', temp_c: 3.5 }];
        return jsonResponse({ reading_id: 'rd-1', manifest_id: 'man-cold-99', temp_c: 3.5 }, 201);
      }
      return jsonResponse({ error: 'unexpected' }, 404);
    });

    render(<PickWavesPage />);
    fireEvent.click(await screen.findByRole('button', { name: 'Tasks' }));

    const banner = await screen.findByTestId('cold-lot-warning-banner');
    expect(banner.textContent).toContain(COLD_LOT_MIGRATION_WARNING);

    fireEvent.change(screen.getByLabelText('Bay baseline temperature Celsius'), {
      target: { value: '3.5' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Log Bay Baseline' }));

    await waitFor(() => {
      expect(
        requests.some(
          (r) => r.path === '/v1/warehouse/ops/manifests/man-cold-99/temperature' && r.method === 'POST',
        ),
      ).toBe(true);
    });
    const tempReq = requests.find(
      (r) => r.path === '/v1/warehouse/ops/manifests/man-cold-99/temperature' && r.method === 'POST',
    );
    expect(tempReq?.body).toEqual({ sensor_id: 'BAY-BASELINE', temp_c: 3.5 });
    expect(tempReq?.headers['Idempotency-Key']).toMatch(/^temp-baseline-man-cold-99-/);
  });
});
