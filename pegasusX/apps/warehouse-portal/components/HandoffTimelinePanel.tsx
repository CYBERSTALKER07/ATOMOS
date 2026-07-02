'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { PulseTimeline } from '@pegasusx/pulse-ui';
import type { PulseEvent } from '@pegasusx/types';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';

const HANDOFF_KINDS = new Set([
  'PREORDER',
  'ORDER_ACCEPTED',
  'ORDER_DISPATCHED',
  'MANIFEST_SEALED',
  'MANIFEST_DISPATCHED',
  'DISPATCH',
]);

function isHandoffEvent(event: PulseEvent): boolean {
  const haystack = `${event.kind} ${event.title}`.toUpperCase();
  if (HANDOFF_KINDS.has(event.kind.toUpperCase())) return true;
  return /PREORDER|ACCEPT|DISPATCH|SEAL|MANIFEST/.test(haystack);
}

type HandoffSource = 'warehouse' | 'factory';

export default function HandoffTimelinePanel({
  className,
  source = 'warehouse',
}: {
  className?: string;
  source?: HandoffSource;
}) {
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      if (source === 'factory') {
        const res = await apiFetch('/v1/factory/pulse');
        if (!res.ok) throw new Error('pulse_failed');
        const data = (await res.json()) as { events?: PulseEvent[] };
        setEvents((data.events ?? []).filter(isHandoffEvent));
      } else {
        const data = await warehouseApi.getWarehousePulse();
        setEvents((data.events ?? []).filter(isHandoffEvent));
      }
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [source]);

  useEffect(() => {
    void load();
  }, [load]);

  useWarehouseSessionReconcile(() => {
    if (source === 'factory') return;
    void load();
  });

  useEffect(() => {
    if (source !== 'factory') return;
    const handler = () => {
      void load();
    };
    window.addEventListener('factory-pulse-refresh', handler);
    return () => window.removeEventListener('factory-pulse-refresh', handler);
  }, [load, source]);

  const subtitle = useMemo(() => {
    if (loading) return 'Loading handoff chain…';
    if (events.length === 0) return 'No preorder → dispatch → seal events in the recent pulse window.';
    return `${events.length} handoff event(s) in recent pulse.`;
  }, [events.length, loading]);

  const refreshClass = source === 'factory' ? 'desk-btn-ghost text-xs px-2 py-1' : 'portal-btn portal-btn--ghost text-xs';

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-3">
        <div>
          <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">Handoff timeline</h3>
          <p className="text-xs opacity-60 mt-1">{subtitle}</p>
        </div>
        <button type="button" className={refreshClass} onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} />
    </div>
  );
}
