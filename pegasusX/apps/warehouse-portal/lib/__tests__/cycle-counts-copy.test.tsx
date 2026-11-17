import React from 'react';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createTranslator, messageCatalogs } from '@pegasusx/i18n';

const { apiFetchMock, toastMock } = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
  toastMock: vi.fn(),
}));

vi.mock('@/lib/i18n', () => ({
  usePortalT: () => createTranslator('en'),
}));

vi.mock('@/components/PageTransition', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock('@/components/PageChrome', () => ({
  PageChrome: ({
    title,
    description,
    children,
  }: {
    title: string;
    description?: string;
    children: React.ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p data-testid="page-description">{description}</p> : null}
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

import CycleCountsPage from '../../app/cycle-counts/page';

function jsonResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

const here = dirname(fileURLToPath(import.meta.url));
const cycleCountsPath = join(here, '../../app/cycle-counts/page.tsx');

describe('CycleCountsPage lot-mutating copy and approval workflow', () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    toastMock.mockReset();
  });

  afterEach(() => {
    cleanup();
  });

  it('contains zero stale stub-only / non-lot-mutating copy in TSX and all i18n locales', () => {
    const src = readFileSync(cycleCountsPath, 'utf8');
    expect(src).not.toMatch(/stub_counts|stub_only|does_not_ch|no_lot/i);

    const forbidden = /stub counts|stub-only|no lot mutation|does not change lots|partiyalar o'zgarmaydi|partiyalarni hali o'zgartirmaydi|партии не изменяются|без изменения партий/i;
    for (const locale of ['en', 'ru', 'uz'] as const) {
      const cat = messageCatalogs[locale];
      const keysToCheck = [
        'warehouse_portal.cycle_counts.text.cycle_counts_variance_adjustments_description',
        'warehouse_portal.cycle_counts.text.pending_rows_created_on_variance_approve_mutates_lots',
        'warehouse_portal.cycle_counts.text.pending_rows_are_created_on_variance_approve_is_stub_only_no_lot',
        'warehouse_portal.residual.text.8_7_wave_1c_stub_counts_variance_adjustments_approve_does_not_ch',
      ];
      for (const key of keysToCheck) {
        expect(cat[key], `${locale}.${key} missing`).toBeTruthy();
        expect(cat[key], `${locale}.${key} has stale stub copy`).not.toMatch(forbidden);
      }
    }
  });

  it('renders lot-mutating description and approves PENDING inventory adjustments with Idempotency-Key', async () => {
    const requests: Array<{ path: string; method: string; headers: Record<string, string> }> = [];
    let approved = false;

    apiFetchMock.mockImplementation(async (path: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET';
      const headers = (init?.headers ?? {}) as Record<string, string>;
      requests.push({ path, method, headers });

      if (path === '/v1/warehouse/ops/cycle-counts' && method === 'GET') {
        return jsonResponse({
          cycle_counts_enabled: true,
          counts: [
            {
              count_id: 'cnt-10000001',
              warehouse_id: 'wh-1',
              location_id: 'A-01',
              product_id: 'sku-milk-1',
              expected_qty: 20,
              counted_qty: 18,
              variance_qty: -2,
              status: 'SUBMITTED',
            },
          ],
        });
      }
      if (path === '/v1/warehouse/ops/inventory-adjustments' && method === 'GET') {
        return jsonResponse({
          adjustments: [
            {
              adjustment_id: 'adj-20000001',
              warehouse_id: 'wh-1',
              product_id: 'sku-milk-1',
              delta_qty: -2,
              status: approved ? 'APPROVED' : 'PENDING',
            },
          ],
        });
      }
      if (path === '/v1/warehouse/ops/inventory-adjustments/adj-20000001/approve' && method === 'POST') {
        approved = true;
        return jsonResponse({ adjustment_id: 'adj-20000001', status: 'APPROVED' });
      }
      return jsonResponse({ error: 'unexpected' }, 404);
    });

    render(<CycleCountsPage />);

    const desc = await screen.findByTestId('page-description');
    expect(desc.textContent).toContain('atomically updates stock lots');
    expect(desc.textContent).not.toMatch(/stub/i);

    expect(screen.getByText(/atomically updates StockLots and inventory rollups/i)).toBeInTheDocument();

    fireEvent.click(await screen.findByRole('button', { name: 'Approve' }));

    await waitFor(() => {
      expect(
        requests.some(
          (r) => r.path === '/v1/warehouse/ops/inventory-adjustments/adj-20000001/approve' && r.method === 'POST',
        ),
      ).toBe(true);
    });

    const approveReq = requests.find(
      (r) => r.path === '/v1/warehouse/ops/inventory-adjustments/adj-20000001/approve',
    );
    expect(approveReq?.headers['Idempotency-Key']).toMatch(/^adj-approve-adj-20000001-/);
    expect(toastMock).toHaveBeenCalledWith('Adjustment approved', 'success');
  });
});
