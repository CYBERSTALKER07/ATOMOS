import { beforeEach, describe, expect, it, vi } from 'vitest';

const { apiFetchNoQueueMock } = vi.hoisted(() => ({
  apiFetchNoQueueMock: vi.fn(),
}));

vi.mock('@/lib/auth', () => ({
  apiFetchNoQueue: apiFetchNoQueueMock,
}));

import {
  GraphAnalyticsRequestError,
  querySupplierGraphAnalytics,
} from '../graph-analytics';

describe('graph analytics api helper', () => {
  beforeEach(() => {
    apiFetchNoQueueMock.mockReset();
  });

  it('returns graph query data from analytics envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          timestamp: 1716076800,
          data: {
            query_mode: 'LANE_CAPACITY',
            nodes: [],
            edges: [],
            rows: [],
            pagination: {
              page_size: 50,
              offset: 0,
              returned: 0,
              has_more: false,
            },
            explainability: {
              query_mode: 'LANE_CAPACITY',
              scope_supplier_id: 'sup-1',
              applied_filters: {
                page_size: '50',
                offset: '0',
              },
              data_sources: ['SupplyLanes'],
              generated_at: '2026-05-19T12:00:00Z',
            },
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    const result = await querySupplierGraphAnalytics({
      query_mode: 'LANE_CAPACITY',
      page_size: 50,
      offset: 0,
    });

    expect(result.query_mode).toBe('LANE_CAPACITY');
    expect(apiFetchNoQueueMock).toHaveBeenCalledWith(
      '/v1/supplier/analytics/graph/query',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('throws structured request error for non-2xx responses', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ error: 'invalid_request' }), {
        status: 422,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(
      querySupplierGraphAnalytics({ query_mode: 'SUPPLIER_TIER' }),
    ).rejects.toBeInstanceOf(GraphAnalyticsRequestError);

    await expect(
      querySupplierGraphAnalytics({ query_mode: 'SUPPLIER_TIER' }),
    ).rejects.toMatchObject({ status: 422 });
  });

  it('throws on malformed analytics envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ data: {} }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(
      querySupplierGraphAnalytics({ query_mode: 'PRODUCT_LOCATION_TIME' }),
    ).rejects.toThrow('Graph analytics response payload is malformed');
  });
});
