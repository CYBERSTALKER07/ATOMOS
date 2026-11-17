import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { CycleCount, InventoryAdjustment, PickWave } from '@pegasusx/types';

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
  usePortalT: () => (key: string) => key,
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
  default: ({ headline, body }: { headline: string; body: string }) => (
    <div>
      <p>{headline}</p>
      <p>{body}</p>
    </div>
  ),
}));

vi.mock('@/components/Toast', () => ({
  useToast: () => ({ toast: toastMock }),
}));

vi.mock('@/lib/auth', () => ({
  apiFetch: apiFetchMock,
}));

import PickWavesPage from '../pick-waves/page';
import CycleCountsPage from '../cycle-counts/page';

function jsonResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('PickWavesPage', () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    toastMock.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it('creates waves with UUID key, confirms explicit 0 picked qty as SHORT, and waives shorts', async () => {
    let waveState: PickWave = {
      wave_id: 'wave-11111111',
      warehouse_id: 'wh-1',
      supplier_id: 'sup-1',
      manifest_id: 'manifest-22222222',
      status: 'OPEN',
      strategy: 'FEFO',
      tasks: [
        {
          task_id: 'task-1',
          order_id: 'ord-1',
          product_id: 'SKU-APPLE',
          lot_id: 'lot-aaaa1111',
          location_id: 'A-01',
          quantity_requested: 5,
          quantity_picked: 0,
          pick_sequence: 1,
          status: 'PENDING',
        },
      ],
    };

    const requests: Array<{ path: string; method: string; headers: Record<string, string>; body?: unknown }> = [];

    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const headers = (init?.headers ?? {}) as Record<string, string>;
      const parsedBody = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
      requests.push({ path, method, headers, body: parsedBody });

      if (path === '/v1/warehouse/ops/pick-waves' && method === 'GET') {
        return jsonResponse({ waves: [waveState], pick_waves_enabled: true });
      }
      if (path === '/v1/warehouse/ops/pick-waves' && method === 'POST') {
        return jsonResponse(waveState);
      }
      if (path === '/v1/warehouse/ops/pick-waves/wave-11111111' && method === 'GET') {
        return jsonResponse(waveState);
      }
      if (path === '/v1/warehouse/ops/pick-waves/wave-11111111/tasks/task-1/confirm' && method === 'POST') {
        const qty = (parsedBody as { quantity_picked: number }).quantity_picked;
        waveState = {
          ...waveState,
          status: qty < 5 ? 'IN_PROGRESS' : 'READY_TO_SEAL',
          tasks: waveState.tasks?.map((task) =>
            task.task_id === 'task-1'
              ? { ...task, quantity_picked: qty, status: qty < 5 ? 'SHORT' : 'CONFIRMED' }
              : task,
          ),
        };
        return jsonResponse(waveState);
      }
      if (path === '/v1/warehouse/ops/pick-waves/wave-11111111/waive-shorts' && method === 'POST') {
        waveState = {
          ...waveState,
          status: 'READY_TO_SEAL',
          tasks: waveState.tasks?.map((task) =>
            task.status === 'SHORT' ? { ...task, status: 'SHORT_WAIVED' } : task,
          ),
        };
        return jsonResponse(waveState);
      }
      return jsonResponse({ error: `unexpected ${method} ${path}` }, 404);
    });

    render(<PickWavesPage />);

    await screen.findByText('Tasks');
    fireEvent.click(screen.getByRole('button', { name: 'Tasks' }));

    const qtyInput = await screen.findByLabelText('Picked quantity for SKU-APPLE');
    expect((qtyInput as HTMLInputElement).value).toBe('5');

    // Blank input must be rejected rather than coerced via Number("") === 0.
    fireEvent.change(qtyInput, { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));
    expect(toastMock).toHaveBeenCalledWith('Picked qty required', 'error');
    expect(requests.some((r) => r.path.endsWith('/confirm'))).toBe(false);

    // Explicit 0 picked qty must send quantity_picked: 0 (not coerced to requested 5).
    fireEvent.change(qtyInput, { target: { value: '0' } });
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));

    await waitFor(() => {
      expect(screen.getByText('SHORT')).toBeTruthy();
    });

    const confirmReq = requests.find((r) => r.path.endsWith('/tasks/task-1/confirm'));
    expect(confirmReq).toBeDefined();
    expect(confirmReq?.body).toEqual({ quantity_picked: 0 });
    expect(confirmReq?.headers['Idempotency-Key']).toMatch(/^pick-confirm-task-1-[0-9a-f-]{36}$/i);

    const waiveBtn = await screen.findByRole('button', { name: 'Waive Shorts' });
    fireEvent.click(waiveBtn);

    await waitFor(() => {
      expect(screen.getByText('SHORT_WAIVED')).toBeTruthy();
    });

    const waiveReq = requests.find((r) => r.path.endsWith('/waive-shorts'));
    expect(waiveReq).toBeDefined();
    expect(waiveReq?.method).toBe('POST');
    expect(waiveReq?.headers['Idempotency-Key']).toMatch(/^pick-waive-wave-11111111-[0-9a-f-]{36}$/i);
    expect(screen.queryByRole('button', { name: 'Waive Shorts' })).toBeNull();

    // Create wave uses UUID idempotency key too.
    fireEvent.change(screen.getByLabelText('Manifest ID'), { target: { value: 'manifest-new' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create wave' }));
    await waitFor(() => {
      expect(requests.some((r) => r.path === '/v1/warehouse/ops/pick-waves' && r.method === 'POST')).toBe(true);
    });
    const createReq = requests.find((r) => r.path === '/v1/warehouse/ops/pick-waves' && r.method === 'POST');
    expect(createReq?.headers['Idempotency-Key']).toMatch(/^pick-wave-[0-9a-f-]{36}$/i);
  });

  it('supports Short (0) quick action directly without needing manual edit', async () => {
    const wave: PickWave = {
      wave_id: 'wave-22222222',
      warehouse_id: 'wh-1',
      supplier_id: 'sup-1',
      manifest_id: 'manifest-33333333',
      status: 'OPEN',
      strategy: 'FEFO',
      tasks: [
        {
          task_id: 'task-short-btn',
          order_id: 'ord-2',
          product_id: 'SKU-JUICE',
          lot_id: 'lot-bbbb2222',
          location_id: 'B-02',
          quantity_requested: 8,
          quantity_picked: 0,
          pick_sequence: 1,
          status: 'PENDING',
        },
      ],
    };

    const requests: Array<{ path: string; body?: unknown }> = [];
    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const body = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
      if (method === 'POST') requests.push({ path, body });
      if (path === '/v1/warehouse/ops/pick-waves' && method === 'GET') {
        return jsonResponse({ waves: [wave], pick_waves_enabled: true });
      }
      if (path === '/v1/warehouse/ops/pick-waves/wave-22222222' && method === 'GET') {
        return jsonResponse(wave);
      }
      if (path.endsWith('/confirm') && method === 'POST') {
        return jsonResponse({
          ...wave,
          status: 'IN_PROGRESS',
          tasks: [{ ...wave.tasks![0], quantity_picked: 0, status: 'SHORT' }],
        });
      }
      return jsonResponse({}, 404);
    });

    render(<PickWavesPage />);
    await screen.findByText('Tasks');
    fireEvent.click(screen.getByRole('button', { name: 'Tasks' }));

    const shortBtn = await screen.findByRole('button', { name: 'Short (0)' });
    fireEvent.click(shortBtn);

    await waitFor(() => {
      expect(screen.getByText('SHORT')).toBeTruthy();
    });
    expect(requests).toContainEqual({
      path: '/v1/warehouse/ops/pick-waves/wave-22222222/tasks/task-short-btn/confirm',
      body: { quantity_picked: 0 },
    });
  });
});

describe('CycleCountsPage', () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    toastMock.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it('creates count, submits explicit 0 with variance display, and approves/rejects adjustments', async () => {
    let countsState: CycleCount[] = [];
    let adjustmentsState: InventoryAdjustment[] = [
      {
        adjustment_id: 'adj-reject-1',
        warehouse_id: 'wh-1',
        product_id: 'SKU-SODA',
        delta_qty: -2,
        status: 'PENDING',
      },
    ];

    const requests: Array<{ path: string; method: string; headers: Record<string, string>; body?: unknown }> = [];

    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const headers = (init?.headers ?? {}) as Record<string, string>;
      const parsedBody = typeof init?.body === 'string' ? JSON.parse(init.body) : undefined;
      requests.push({ path, method, headers, body: parsedBody });

      if (path === '/v1/warehouse/ops/cycle-counts' && method === 'GET') {
        return jsonResponse({ counts: countsState, cycle_counts_enabled: true });
      }
      if (path === '/v1/warehouse/ops/inventory-adjustments' && method === 'GET') {
        return jsonResponse({ adjustments: adjustmentsState, cycle_counts_enabled: true });
      }
      if (path === '/v1/warehouse/ops/cycle-counts' && method === 'POST') {
        const created: CycleCount = {
          count_id: 'cc-11111111',
          warehouse_id: 'wh-1',
          location_id: 'BIN-A1',
          product_id: 'SKU-MILK',
          expected_qty: 10,
          status: 'OPEN',
        };
        countsState = [created];
        return jsonResponse(created);
      }
      if (path === '/v1/warehouse/ops/cycle-counts/cc-11111111/submit' && method === 'POST') {
        const submitted: CycleCount = {
          ...countsState[0],
          counted_qty: (parsedBody as { counted_qty: number }).counted_qty,
          variance_qty: (parsedBody as { counted_qty: number }).counted_qty - 10,
          reason_code: (parsedBody as { reason_code?: string }).reason_code,
          status: 'ADJUSTMENT_PENDING',
        };
        countsState = [submitted];
        adjustmentsState = [
          {
            adjustment_id: 'adj-approve-1',
            warehouse_id: 'wh-1',
            product_id: 'SKU-MILK',
            count_id: 'cc-11111111',
            delta_qty: -10,
            status: 'PENDING',
          },
          ...adjustmentsState,
        ];
        return jsonResponse(submitted);
      }
      if (path === '/v1/warehouse/ops/inventory-adjustments/adj-approve-1/approve' && method === 'POST') {
        adjustmentsState = adjustmentsState.map((a) =>
          a.adjustment_id === 'adj-approve-1' ? { ...a, status: 'APPROVED' } : a,
        );
        return jsonResponse({ status: 'APPROVED' });
      }
      if (path === '/v1/warehouse/ops/inventory-adjustments/adj-reject-1/reject' && method === 'POST') {
        adjustmentsState = adjustmentsState.map((a) =>
          a.adjustment_id === 'adj-reject-1' ? { ...a, status: 'REJECTED' } : a,
        );
        return jsonResponse({ status: 'REJECTED' });
      }
      return jsonResponse({ error: `unexpected ${method} ${path}` }, 404);
    });

    render(<CycleCountsPage />);

    await screen.findByText('warehouse_portal.residual.text.no_cycle_counts');

    // Create cycle count
    fireEvent.change(screen.getByLabelText('Location ID'), { target: { value: 'BIN-A1' } });
    fireEvent.change(screen.getByLabelText('Product ID'), { target: { value: 'SKU-MILK' } });
    fireEvent.change(screen.getByLabelText('Expected (optional)'), { target: { value: '10' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create count' }));

    const qtyInput = await screen.findByLabelText('Counted quantity for SKU-MILK');
    const createReq = requests.find((r) => r.path === '/v1/warehouse/ops/cycle-counts' && r.method === 'POST');
    expect(createReq?.body).toEqual({ location_id: 'BIN-A1', product_id: 'SKU-MILK', expected_qty: 10 });
    expect(createReq?.headers['Idempotency-Key']).toMatch(/^cc-[0-9a-f-]{36}$/i);

    // Blank counted qty must be rejected (never coerced to 0).
    fireEvent.change(qtyInput, { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: 'warehouse_portal.cycle_counts.text.submit' }));
    expect(toastMock).toHaveBeenCalledWith('Counted qty required', 'error');
    expect(requests.some((r) => r.path.endsWith('/submit'))).toBe(false);

    // Submit explicit 0 counted qty with reason code.
    fireEvent.change(qtyInput, { target: { value: '0' } });
    fireEvent.change(screen.getByLabelText('Reason code for SKU-MILK'), { target: { value: 'DAMAGED' } });
    fireEvent.click(screen.getByRole('button', { name: 'warehouse_portal.cycle_counts.text.submit' }));

    await waitFor(() => {
      expect(screen.getByText('0/10')).toBeTruthy();
      expect(screen.getByText('Δ-10')).toBeTruthy();
    });

    const submitReq = requests.find((r) => r.path === '/v1/warehouse/ops/cycle-counts/cc-11111111/submit');
    expect(submitReq?.body).toEqual({ counted_qty: 0, reason_code: 'DAMAGED' });
    expect(submitReq?.headers['Idempotency-Key']).toMatch(/^cc-submit-cc-11111111-[0-9a-f-]{36}$/i);

    // Approve first pending adjustment and reject second pending adjustment.
    const approveButtons = await screen.findAllByRole('button', { name: 'Approve' });
    fireEvent.click(approveButtons[0]);

    await waitFor(() => {
      expect(screen.getByText('APPROVED')).toBeTruthy();
    });
    const approveReq = requests.find((r) => r.path.endsWith('/adj-approve-1/approve'));
    expect(approveReq?.headers['Idempotency-Key']).toMatch(/^adj-approve-adj-approve-1-[0-9a-f-]{36}$/i);

    const rejectButton = screen.getByRole('button', { name: 'Reject' });
    fireEvent.click(rejectButton);

    await waitFor(() => {
      expect(screen.getByText('REJECTED')).toBeTruthy();
    });
    const rejectReq = requests.find((r) => r.path.endsWith('/adj-reject-1/reject'));
    expect(rejectReq?.headers['Idempotency-Key']).toMatch(/^adj-reject-adj-reject-1-[0-9a-f-]{36}$/i);
  });
});
