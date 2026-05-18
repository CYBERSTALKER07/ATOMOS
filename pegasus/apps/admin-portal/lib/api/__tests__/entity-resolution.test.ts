import { beforeEach, describe, expect, it, vi } from 'vitest';

const { apiFetchNoQueueMock } = vi.hoisted(() => ({
  apiFetchNoQueueMock: vi.fn(),
}));

vi.mock('@/lib/auth', () => ({
  apiFetchNoQueue: apiFetchNoQueueMock,
}));

import {
  EntityResolutionRequestError,
  explainSupplierEntity,
  resolveSupplierEntity,
} from '../entity-resolution';

describe('entity-resolution api helper', () => {
  beforeEach(() => {
    apiFetchNoQueueMock.mockReset();
  });

  it('returns resolve result from status-ok envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'ok',
          result: {
            scope_supplier_id: 'sup-1',
            requested_type: 'ANY',
            query: 'ord',
            resolved: {
              node_id: 'ORDER:ord-1',
              entity_type: 'ORDER',
              entity_id: 'ord-1',
              label: 'ord-1',
              score: 0.98,
              confidence_class: 'HIGH',
              deterministic: true,
            },
            candidates: [],
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    const result = await resolveSupplierEntity({
      entity_type: 'ANY',
      query: 'ord',
      max_candidates: 5,
    });

    expect(result.scope_supplier_id).toBe('sup-1');
    expect(result.resolved?.entity_id).toBe('ord-1');
    expect(apiFetchNoQueueMock).toHaveBeenCalledWith(
      '/v1/supplier/entity-resolution/resolve',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('throws structured request error for non-2xx responses', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ error: 'invalid_request' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(
      resolveSupplierEntity({ entity_type: 'ANY', query: 'bad' }),
    ).rejects.toBeInstanceOf(EntityResolutionRequestError);

    await expect(
      resolveSupplierEntity({ entity_type: 'ANY', query: 'bad' }),
    ).rejects.toMatchObject({ status: 400 });
  });

  it('throws on malformed success envelope', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'ok' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );

    await expect(
      explainSupplierEntity({ entity_type: 'ORDER', entity_id: 'ord-1' }),
    ).rejects.toThrow('Entity resolution response payload is malformed');
  });

  it('calls explain endpoint with expected payload', async () => {
    apiFetchNoQueueMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'ok',
          result: {
            scope_supplier_id: 'sup-1',
            source: {
              node_id: 'ORDER:ord-1',
              entity_type: 'ORDER',
              entity_id: 'ord-1',
              label: 'ord-1',
              score: 1,
              confidence_class: 'HIGH',
              deterministic: true,
            },
            projection: { nodes: [], edges: [] },
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );

    await explainSupplierEntity({ entity_type: 'ORDER', entity_id: 'ord-1' });

    expect(apiFetchNoQueueMock).toHaveBeenCalledWith(
      '/v1/supplier/entity-resolution/explain',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ entity_type: 'ORDER', entity_id: 'ord-1' }),
      }),
    );
  });
});
