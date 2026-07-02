'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { PulseTimeline } from '@pegasusx/pulse-ui';
import type { PulseEvent } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';

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

export default function HandoffTimelinePanel({ className }: { className?: string }) {
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePulse();
      const rows = (data.events ?? []).filter(isHandoffEvent);
      setEvents(rows);
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const subtitle = useMemo(() => {
    if (loading) return 'Loading handoff chain…';
    if (events.length === 0) return 'No preorder → dispatch → seal events in the recent pulse window.';
    return `${events.length} handoff event(s) in recent pulse.`;
  }, [events.length, loading]);

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-3">
        <div>
          <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">Handoff timeline</h3>
          <p className="text-xs opacity-60 mt-1">{subtitle}</p>
        </div>
        <button type="button" className="button--secondary rounded-lg px-3 py-1 text-xs" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} />
    </div>
  );
}
