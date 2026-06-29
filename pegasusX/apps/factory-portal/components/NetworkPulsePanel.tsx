"use client";

import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@pegasusx/pulse-ui";
import type { PulseEvent } from "@pegasusx/types";
import { apiFetch } from "@/lib/auth";

export default function NetworkPulsePanel({ className }: { className?: string }) {
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch("/v1/factory/pulse");
      if (!res.ok) throw new Error("pulse_failed");
      const data = (await res.json()) as { events: PulseEvent[] };
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

  useEffect(() => {
    const handler = () => {
      void load();
    };
    window.addEventListener("factory-pulse-refresh", handler);
    return () => window.removeEventListener("factory-pulse-refresh", handler);
  }, [load]);

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">Network pulse</h3>
        <button type="button" className="desk-btn-ghost text-xs px-2 py-1" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} />
    </div>
  );

}
