'use client';

import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { apiFetch } from '../lib/auth';

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
      const res = await apiFetch(`/v1/order/${orderId}/timeline`);
      if (res.ok) {
        const data = (await res.json()) as { items?: OrderTimelineEntry[] };
        setItems(data.items ?? []);
      } else {
        setItems([]);
      }
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
    return <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Loading status history…</p>;
  }
  if (items.length === 0) {
    return <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">No status history yet.</p>;
  }

  return (
    <ol className="space-y-3 border-l border-[var(--desk-border)] pl-4">
      {items.map((entry) => (
        <li key={entry.transition_id} className="relative">
          <span className="absolute -left-[1.35rem] top-1.5 h-2 w-2 rounded-full bg-[var(--desk-accent)]" />
          <p className="md-typescale-body-medium font-semibold">
            {entry.previous_status ? `${entry.previous_status} → ${entry.new_status}` : entry.new_status}
            {entry.event_kind === 'DELAY' ? ' · Delayed' : null}
          </p>
          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
            {formatWhen(entry.created_at)}
            {entry.reason ? ` · ${entry.reason}` : ''}
          </p>
          {entry.metadata?.proposed_delivery_date ? (
            <p className="md-typescale-body-small text-amber-700 mt-1">
              Proposed delivery: {String(entry.metadata.proposed_delivery_date)}
            </p>
          ) : null}
        </li>
      ))}
    </ol>
  );
}
