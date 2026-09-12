"use client";

import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@pegasusx/pulse-ui";
import type { PulseEvent } from "@pegasusx/types";
import { apiFetch } from "../lib/auth";

export default function NetworkPulsePanel({ className }: { className?: string }) {
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch("/v1/retailer/pulse");
      if (!res.ok) throw new Error("pulse_failed");
      const data = (await res.json()) as { events: PulseEvent[] };
      setEvents(data.events ?? []);
    } catch {
      setError("pulse_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className={className}>
      <PulseTimeline events={events} loading={loading} error={error} />
    </div>
  );
}
