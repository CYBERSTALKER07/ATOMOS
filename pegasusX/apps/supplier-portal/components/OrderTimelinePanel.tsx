'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import type { OrderTimelineEntry } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';

function formatWhen(iso: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
  } catch {
    return iso;
  }
}

export function OrderTimelinePanel({ orderId }: { orderId: string }) {
  const t = usePortalT();
  const [items, setItems] = useState<OrderTimelineEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const api = createSupplierApi();

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const data = await api.getOrderTimeline(orderId);
      setItems(data.items ?? []);
    } catch {
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [orderId, api]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) return <p className="text-sm text-[var(--desk-text-tertiary)]">{t("supplier_portal.order_timeline_panel.text.loading_timeline")}</p>;
  if (items.length === 0) return <p className="text-sm text-[var(--desk-text-tertiary)]">{t("supplier_portal.order_timeline_panel.text.no_status_history_yet")}</p>;

  return (
    <ol className="space-y-3 border-l border-[var(--desk-border)] pl-4">
      {items.map((entry) => (
        <li key={entry.transition_id} className="relative">
          <span className="absolute -left-[1.35rem] top-1.5 h-2 w-2 rounded-full bg-[var(--desk-accent)]" />
          <p className="text-sm font-medium">
            {entry.previous_status ? `${entry.previous_status} → ${entry.new_status}` : entry.new_status}
          </p>
          <p className="text-xs text-[var(--desk-text-tertiary)]">
            {formatWhen(entry.created_at)}
            {entry.reason ? ` · ${entry.reason}` : ''}
          </p>
        </li>
      ))}
    </ol>
  );
}
