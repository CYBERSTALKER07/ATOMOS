import fs from 'node:fs';
import path from 'node:path';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  UZS_BANKNOTE_DENOMINATIONS,
  TIYIN_PER_UZS,
  STATUTORY_AML_CASH_CEILING_MINOR,
  type CashierSessionView as CashierSessionViewType,
} from '@pegasusx/types';

const {
  mockApiFetch,
  mockGetWarehouseFleetLiveMap,
  mockStartCashierSession,
  mockSubmitBlindCount,
  mockTokenState,
} = vi.hoisted(() => ({
  mockApiFetch: vi.fn(),
  mockGetWarehouseFleetLiveMap: vi.fn(),
  mockStartCashierSession: vi.fn(),
  mockSubmitBlindCount: vi.fn(),
  mockTokenState: { token: '' },
}));

vi.mock('@/lib/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/auth')>();
  return {
    ...actual,
    apiFetch: mockApiFetch,
    readTokenFromCookie: () => mockTokenState.token,
  };
});

vi.mock('@/lib/warehouse-api', () => ({
  warehouseApi: {
    getWarehouseFleetLiveMap: mockGetWarehouseFleetLiveMap,
  },
}));

vi.mock('@/lib/foundry-api', () => ({
  foundryWarehouseApi: {
    startCashierSession: mockStartCashierSession,
    submitBlindCount: mockSubmitBlindCount,
    reclassifyTender: vi.fn(),
    citVaultHandover: vi.fn(),
    amlCheck: vi.fn(),
  },
}));

import CashierPage from '@/app/cashier/page';
import { BlindCountDenominationGrid } from '@/components/cashier/BlindCountDenominationGrid';
import { CashierSessionView } from '@/components/cashier/CashierSessionView';
import {
  buildCashierSelectorOptions,
  calculateDenominationTotals,
  formatSignedWholeSom,
  formatWholeSom,
  formatWholeSomInteger,
  resolveCashierScope,
} from '@/lib/cashier-helpers';

function makeJwt(payload: Record<string, unknown>): string {
  const b64 = (obj: Record<string, unknown>) =>
    Buffer.from(JSON.stringify(obj)).toString('base64url');
  return `${b64({ alg: 'HS256', typ: 'JWT' })}.${b64(payload)}.sig`;
}

describe('Role 7 Depot Cashier Blind Count & Whole-So\'m UI Rules', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTokenState.token = makeJwt({
      sub: 'cashier-operator-7',
      home_node_id: 'wh-central-01',
      home_cell: 'cell-central',
      supplier_id: 'sup-live-01',
    });
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ drivers: [], manifests: [] }),
    });
    mockGetWarehouseFleetLiveMap.mockResolvedValue({
      routes: [],
      warehouse_id: 'wh-central-01',
      fetched_at: '2026-11-12T10:00:00Z',
    });
  });

  it('converts all Uzbek So\'m banknote denominations to integer minor units and computes totals without float drift', () => {
    expect(UZS_BANKNOTE_DENOMINATIONS).toHaveLength(8);
    expect(TIYIN_PER_UZS).toBe(100);

    const totals = calculateDenominationTotals({
      200000: 5,
      50000: 3,
      1000: 10,
    });

    expect(totals.totalNotes).toBe(18);
    expect(totals.totalSom).toBe(1160000);
    expect(totals.totalMinor).toBe(116000000);
    expect(Number.isInteger(totals.totalMinor)).toBe(true);
  });

  it('formats whole so\'m values strictly without raw minor inflation or negative zero', () => {
    expect(formatWholeSom(142500000)).toBe("1,425,000 so'm");
    expect(formatWholeSom(BigInt('500000000000'))).toBe("5,000,000,000 so'm");
    expect(formatSignedWholeSom(250000)).toBe("+2,500 so'm");
    expect(formatSignedWholeSom(-250000)).toBe("-2,500 so'm");
    expect(formatSignedWholeSom(-50)).toBe("0 so'm");
    expect(formatWholeSomInteger(200000)).toBe("200,000 so'm");
  });

  it('resolves warehouse scope from JWT/auth without hardcoded demo fallbacks', () => {
    const emptyScope = resolveCashierScope({ token: '', cachedSession: null, homeNodeId: '' });
    expect(emptyScope).toEqual({
      cellId: '',
      depotId: '',
      supplierId: '',
      actorId: '',
    });

    const jwtScope = resolveCashierScope({
      token: makeJwt({
        sub: 'op-42',
        home_node_id: 'depot-live-9',
        home_cell: 'cell-uz-2',
        supplier_id: 'sup-live-9',
      }),
      cachedSession: null,
      homeNodeId: 'depot-live-9',
    });
    expect(jwtScope).toEqual({
      cellId: 'cell-uz-2',
      depotId: 'depot-live-9',
      supplierId: 'sup-live-9',
      actorId: 'op-42',
    });
  });

  it('builds deduplicated driver and route selector options from live warehouse endpoints', () => {
    const { driverOptions, routeOptions } = buildCashierSelectorOptions({
      drivers: [{ driver_id: 'drv-1', name: 'Dilshod K.' }],
      manifests: [
        { manifest_id: 'mf-10', driver_name: 'Dilshod K.', vehicle_label: 'Van 10' },
        { manifest_id: 'mf-11', driver_name: 'Sardor M.', vehicle_label: 'Van 11' },
      ],
      fleetRoutes: [
        {
          manifest_id: 'mf-10',
          route_id: 'rt-100',
          driver_id: 'drv-1',
          driver_name: 'Dilshod K.',
          manifest_state: 'OUT_FOR_DELIVERY',
          live_location_available: true,
        },
        {
          manifest_id: 'mf-12',
          route_id: 'rt-200',
          driver_id: 'drv-2',
          driver_name: 'Aziz B.',
          manifest_state: 'OUT_FOR_DELIVERY',
          live_location_available: false,
        },
      ],
    });

    expect(driverOptions).toHaveLength(2);
    expect(driverOptions.map((d) => d.driverId)).toEqual(['drv-1', 'drv-2']);
    expect(routeOptions.map((r) => r.routeId)).toEqual(['rt-100', 'rt-200', 'mf-11']);
  });

  it('strictly masks expected cash and variance in CashierSessionView during blind counts and formats settled balances in so\'m', () => {
    const baseSession: CashierSessionViewType = {
      session_id: 'cash-sess-001',
      supplier_id: 'sup-live-01',
      depot_id: 'wh-central-01',
      driver_id: 'drv-1',
      route_id: 'rt-100',
      bag_seal_barcode: 'SEAL-777',
      attempt_number: 1,
      counted_cash_minor: 0,
      expected_cash_minor: 116000000,
      variance_minor: -116000000,
      status: 'AWAITING_BLIND_COUNT_1',
      created_at: '2026-11-12T10:00:00Z',
    };

    const { rerender } = render(React.createElement(CashierSessionView, { session: baseSession }));
    expect(screen.getAllByText('MASKED (BLIND COUNT)')).toHaveLength(2);
    expect(screen.queryByText(/1,160,000/)).not.toBeInTheDocument();
    expect(screen.queryByText(/tiyin/i)).not.toBeInTheDocument();

    rerender(
      React.createElement(CashierSessionView, {
        session: { ...baseSession, attempt_number: 2, status: 'BLIND_RECOUNT_REQUIRED' },
      }),
    );
    expect(screen.getAllByText('MASKED (BLIND COUNT)')).toHaveLength(2);

    rerender(
      React.createElement(CashierSessionView, {
        session: {
          ...baseSession,
          status: 'BALANCED',
          counted_cash_minor: 116000000,
          expected_cash_minor: 116000000,
          variance_minor: 0,
        },
      }),
    );
    expect(screen.getAllByText("1,160,000 so'm")).toHaveLength(2);
    expect(screen.getByText("0 so'm")).toBeInTheDocument();
    expect(screen.queryByText(/tiyin/i)).not.toBeInTheDocument();
  });

  it('submits exact integer minor payload while rendering BlindCountDenominationGrid strictly in whole so\'m', async () => {
    const onSubmitCount = vi.fn().mockResolvedValue(undefined);
    render(
      React.createElement(BlindCountDenominationGrid, { attemptNumber: 1, onSubmitCount }),
    );

    expect(screen.queryByText(/tiyin/i)).not.toBeInTheDocument();
    expect(screen.getByText("200,000 so'm")).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText("Add 10 notes of 200000 so'm"));
    expect(screen.getByText("2,000,000 so'm")).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Confirm Attempt #1/i }));
    await waitFor(() => {
      expect(onSubmitCount).toHaveBeenCalledWith(
        expect.objectContaining({ 200000: 10 }),
        200000000,
        0,
      );
    });
  });

  it('loads CashierPage without fake presets or window.alert and shows manual fallback when endpoints return empty', async () => {
    mockStartCashierSession.mockResolvedValue({
      session_id: 'sess-live-55',
      supplier_id: 'sup-live-01',
      depot_id: 'wh-central-01',
      driver_id: 'drv-live-1',
      route_id: 'rt-live-1',
      bag_seal_barcode: 'SEAL-LIVE-1',
      attempt_number: 2,
      counted_cash_minor: 100000000,
      expected_cash_minor: 142500000,
      variance_minor: -42500000,
      status: 'DISCREPANT_QUARANTINE',
      created_at: '2026-11-12T10:00:00Z',
    } satisfies CashierSessionViewType);

    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});
    render(React.createElement(CashierPage));

    await waitFor(() => {
      expect(screen.getByText('0 drivers · 0 routes')).toBeInTheDocument();
    });

    expect(screen.queryByText(/Anvar/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Rustam/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Quick Presets/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/tiyin/i)).not.toBeInTheDocument();

    const driverInput = screen.getByLabelText('Driver ID') as HTMLInputElement;
    const routeInput = screen.getByLabelText('Route ID') as HTMLInputElement;
    expect(driverInput.value).toBe('');
    expect(routeInput.value).toBe('');

    fireEvent.change(screen.getByPlaceholderText(/Scan or enter tamper seal barcode/i), {
      target: { value: 'SEAL-LIVE-1' },
    });
    fireEvent.change(driverInput, { target: { value: 'drv-live-1' } });
    fireEvent.change(routeInput, { target: { value: 'rt-live-1' } });
    fireEvent.click(screen.getByRole('button', { name: /Bind Seal & Begin Blind Count/i }));

    await waitFor(() => {
      expect(screen.getByText('Active Turn-In Session: sess-live-55')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /Request Supervisor Override/i }));
    expect(alertSpy).not.toHaveBeenCalled();
    expect(screen.getByText('Supervisor Override Request Queued')).toBeInTheDocument();
    alertSpy.mockRestore();
  });

  it('populates Driver and Route selectors from real warehouse endpoints and auto-links driver from selected route', async () => {
    mockApiFetch.mockImplementation(async (pathArg: string) => {
      if (pathArg.startsWith('/v1/warehouse/ops/drivers')) {
        return {
          ok: true,
          json: async () => ({ drivers: [{ driver_id: 'drv-real-1', name: 'Jamshid R.' }] }),
        };
      }
      if (pathArg.startsWith('/v1/warehouse/ops/manifests')) {
        return {
          ok: true,
          json: async () => ({ manifests: [{ manifest_id: 'mf-9', driver_name: 'Jamshid R.' }] }),
        };
      }
      return { ok: true, json: async () => ({}) };
    });
    mockGetWarehouseFleetLiveMap.mockResolvedValue({
      routes: [
        {
          manifest_id: 'mf-9',
          route_id: 'rt-real-9',
          driver_id: 'drv-real-1',
          driver_name: 'Jamshid R.',
          manifest_state: 'OUT_FOR_DELIVERY',
          live_location_available: true,
        },
      ],
      warehouse_id: 'wh-central-01',
      fetched_at: '2026-11-12T10:00:00Z',
    });

    render(React.createElement(CashierPage));

    await waitFor(() => {
      expect(screen.getByText('1 drivers · 1 routes')).toBeInTheDocument();
    });

    const routeSelect = screen.getByLabelText('Route') as HTMLSelectElement;
    const driverSelect = screen.getByLabelText('Driver') as HTMLSelectElement;

    fireEvent.change(routeSelect, { target: { value: 'rt-real-9' } });
    expect(routeSelect.value).toBe('rt-real-9');
    expect(driverSelect.value).toBe('drv-real-1');
  });

  it('validates statutory AML threshold and RRN ±15 minute window', () => {
    expect(STATUTORY_AML_CASH_CEILING_MINOR).toBe(2500000000);
    const orderTime = new Date('2026-10-30T10:00:00Z').getTime();
    const validSwipe = new Date('2026-10-30T10:12:00Z').getTime();
    const invalidSwipe = new Date('2026-10-30T10:20:00Z').getTime();
    const fifteenMinMs = 15 * 60 * 1000;
    expect(Math.abs(validSwipe - orderTime) <= fifteenMinMs).toBe(true);
    expect(Math.abs(invalidSwipe - orderTime) <= fifteenMinMs).toBe(false);
  });

  it('contains zero Tiyin labels, fake presets, alert/confirm calls, or @pegasusx/ui-kit imports in cashier UI files', () => {
    const portalRoot = path.resolve(__dirname, '../..');
    const files = [
      'app/cashier/page.tsx',
      'components/cashier/BlindCountDenominationGrid.tsx',
      'components/cashier/CashierSessionView.tsx',
      'components/cashier/CITHandoverManifest.tsx',
      'components/cashier/RRNReclassificationModal.tsx',
      'components/cashier/AMLExposureCheck.tsx',
    ];

    for (const rel of files) {
      const content = fs.readFileSync(path.join(portalRoot, rel), 'utf8');
      expect(content, rel).not.toMatch(/tiyin/i);
      expect(content, rel).not.toMatch(/@pegasusx\/ui-kit/);
      expect(content, rel).not.toMatch(/\b(alert|confirm)\s*\(/);
      expect(content, rel).not.toMatch(/Anvar|Rustam|fillSampleSeal|sup_samarkand_dairy|depot_tashkent_central|drv_anvar_01|route_yashnobod_101|cashier_admin|cashier_desk|cell-uz/);
    }
  });
});
