import type { EntityResolutionCandidate } from '@pegasus/types/entity-resolution';
import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  hasLocalOrderSearchMatch,
  projectResolvedOrderIDs,
  scheduleOrderResolutionFallback,
} from './search-fallback';

function buildCandidate(entityType: EntityResolutionCandidate['entity_type'], entityID: string): EntityResolutionCandidate {
  return {
    node_id: `${entityType}:${entityID}`,
    entity_type: entityType,
    entity_id: entityID,
    label: entityID,
    score: 0.8,
    confidence_class: 'MEDIUM',
    deterministic: false,
  };
}

describe('supplier orders search fallback helpers', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('detects local match by order id or retailer name', () => {
    const rows = [
      { order_id: 'ord-1001', retailer_id: 'ret-1', retailer_name: 'North Shop' },
      { order_id: 'ord-2002', retailer_id: 'ret-2', retailer_name: 'South Market' },
    ];

    expect(hasLocalOrderSearchMatch('1001', rows)).toBe(true);
    expect(hasLocalOrderSearchMatch('south', rows)).toBe(true);
    expect(hasLocalOrderSearchMatch('unknown', rows)).toBe(false);
    expect(hasLocalOrderSearchMatch('   ', rows)).toBe(false);
  });

  it('projects ORDER and RETAILER candidates into unique order ids', () => {
    const rows = [
      { order_id: 'ord-1', retailer_id: 'ret-1', retailer_name: 'Alpha' },
      { order_id: 'ord-2', retailer_id: 'ret-2', retailer_name: 'Beta' },
      { order_id: 'ord-3', retailer_id: 'ret-2', retailer_name: 'Beta' },
    ];
    const candidates: EntityResolutionCandidate[] = [
      buildCandidate('ORDER', 'ord-9'),
      buildCandidate('RETAILER', 'ret-2'),
      buildCandidate('ORDER', 'ord-9'),
      buildCandidate('ORDER', '   '),
    ];

    expect(projectResolvedOrderIDs(candidates, rows)).toEqual(['ord-9', 'ord-2', 'ord-3']);
  });

  it('returns empty list when no ORDER or RETAILER candidates are present', () => {
    const rows = [{ order_id: 'ord-1', retailer_id: 'ret-1', retailer_name: 'Alpha' }];
    const candidates: EntityResolutionCandidate[] = [
      buildCandidate('VEHICLE', 'veh-1'),
      buildCandidate('DRIVER', 'drv-1'),
    ];

    expect(projectResolvedOrderIDs(candidates, rows)).toEqual([]);
  });

  it('runs fallback resolution only after debounce delay', async () => {
    vi.useFakeTimers();

    const rows = [
      { order_id: 'ord-1', retailer_id: 'ret-1', retailer_name: 'Alpha' },
      { order_id: 'ord-2', retailer_id: 'ret-2', retailer_name: 'Beta' },
    ];
    const resolveEntity = vi.fn().mockResolvedValue({
      candidates: [buildCandidate('RETAILER', 'ret-2')],
    });
    const onResolved = vi.fn();

    scheduleOrderResolutionFallback({
      query: 'beta',
      rows,
      resolveEntity,
      onResolved,
      delayMs: 280,
    });

    vi.advanceTimersByTime(279);
    expect(resolveEntity).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    await Promise.resolve();

    expect(resolveEntity).toHaveBeenCalledTimes(1);
    expect(onResolved).toHaveBeenCalledWith(['ord-2']);
  });

  it('cancels scheduled fallback before timer fires', async () => {
    vi.useFakeTimers();

    const resolveEntity = vi.fn().mockResolvedValue({ candidates: [] });
    const onResolved = vi.fn();

    const cancel = scheduleOrderResolutionFallback({
      query: 'any',
      rows: [],
      resolveEntity,
      onResolved,
      delayMs: 280,
    });
    cancel();

    vi.advanceTimersByTime(500);
    await Promise.resolve();

    expect(resolveEntity).not.toHaveBeenCalled();
    expect(onResolved).not.toHaveBeenCalled();
  });

  it('drops late async result after cancellation', async () => {
    vi.useFakeTimers();

    let resolveAsync!: (value: { candidates: EntityResolutionCandidate[] }) => void;
    const resolveEntity = vi.fn().mockImplementation(
      () => new Promise<{ candidates: EntityResolutionCandidate[] }>((resolve) => {
        resolveAsync = resolve;
      }),
    );
    const onResolved = vi.fn();

    const cancel = scheduleOrderResolutionFallback({
      query: 'ord-9',
      rows: [{ order_id: 'ord-9', retailer_id: 'ret-9', retailer_name: 'Zeta' }],
      resolveEntity,
      onResolved,
      delayMs: 50,
    });

    vi.advanceTimersByTime(50);
    expect(resolveEntity).toHaveBeenCalledTimes(1);

    cancel();
    resolveAsync({ candidates: [buildCandidate('ORDER', 'ord-9')] });
    await Promise.resolve();

    expect(onResolved).not.toHaveBeenCalled();
  });

  it('resolves to empty order ids when resolver rejects', async () => {
    vi.useFakeTimers();

    const resolveEntity = vi.fn().mockRejectedValue(new Error('resolution failed'));
    const onResolved = vi.fn();

    scheduleOrderResolutionFallback({
      query: 'ord',
      rows: [{ order_id: 'ord-1', retailer_id: 'ret-1', retailer_name: 'Alpha' }],
      resolveEntity,
      onResolved,
      delayMs: 20,
    });

    await vi.advanceTimersByTimeAsync(20);

    expect(resolveEntity).toHaveBeenCalledTimes(1);
    expect(onResolved).toHaveBeenCalledWith([]);
  });
});