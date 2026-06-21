'use client';

import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';

function formatWhen(iso: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
  } catch {
    return iso;
  }
}

export function OrderTimelinePanel({ orderId }: { orderId: string }) {
  const [items, setItems] = useState<OrderTimelineEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const data = await warehouseApi.getOrderTimeline(orderId);
      setItems(data.items ?? []);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return <p className="text-sm text-[var(--desk-text-tertiary)]">Loading timeline…</p>;
  }
  if (items.length === 0) {
    return <p className="text-sm text-[var(--desk-text-tertiary)]">No status history yet.</p>;
  }

  return (
    <ol className="space-y-3 border-l border-[var(--desk-border)] pl-4">
      {items.map((entry) => (
        <li key={entry.transition_id} className="relative">
          <span className="absolute -left-[1.35rem] top-1.5 h-2 w-2 rounded-full bg-[var(--desk-accent)]" />
          <p className="text-sm font-medium">
            {entry.previous_status ? `${entry.previous_status} → ${entry.new_status}` : entry.new_status}
            {entry.event_kind === 'DELAY' ? ' · Delayed' : null}
          </p>
          <p className="text-xs text-[var(--desk-text-tertiary)]">
            {formatWhen(entry.created_at)}
            {entry.actor_role ? ` · ${entry.actor_role}` : ''}
            {entry.reason ? ` · ${entry.reason}` : ''}
          </p>
          {entry.metadata?.proposed_delivery_date ? (
            <p className="text-xs text-amber-700 mt-1">
              Proposed delivery: {String(entry.metadata.proposed_delivery_date)}
            </p>
          ) : null}
        </li>
      ))}
    </ol>
  );
}
