"use client";

import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@pegasusx/pulse-ui";
import type { PulseEvent } from "@pegasusx/types";
import { warehouseApi } from "@/lib/warehouse-api";

export default function NetworkPulsePanel({ className }: { className?: string }) {
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePulse();
      setEvents(data.events ?? []);
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">Ops pulse</h3>
        <button type="button" className="portal-btn portal-btn--ghost text-xs" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} />
    </div>
  );
}
