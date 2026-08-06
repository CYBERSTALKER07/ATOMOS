'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import { PulseTimeline } from '@pegasusx/pulse-ui';
import type { PulseEvent } from '@pegasusx/types';
import { apiFetch } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';

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
  const t = usePortalT();
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/factory/pulse');
      if (!res.ok) throw new Error('pulse_failed');
      const data = (await res.json()) as { events?: PulseEvent[] };
      setEvents((data.events ?? []).filter(isHandoffEvent));
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useFactorySessionReconcile(() => {
    void load();
  });

  useEffect(() => {
    const handler = () => {
      void load();
    };
    window.addEventListener('factory-pulse-refresh', handler);
    return () => window.removeEventListener('factory-pulse-refresh', handler);
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
          <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">{t("factory_portal.loading_bay.text.handoff_timeline")}</h3>
          <p className="text-xs opacity-60 mt-1">{subtitle}</p>
        </div>
        <button type="button" className="desk-btn-ghost text-xs px-2 py-1" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} />
    </div>
  );
}
