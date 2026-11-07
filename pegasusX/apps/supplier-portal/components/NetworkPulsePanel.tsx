"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { PulseTimeline } from "@pegasusx/pulse-ui";
import type { PulseEvent } from "@pegasusx/types";
import { supplierFetch } from "@/lib/auth";

export default function NetworkPulsePanel({ className }: { className?: string }) {
  const t = usePortalT();
  const [events, setEvents] = useState<PulseEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await supplierFetch("/v1/supplier/pulse");
      if (!response.ok) throw new Error("pulse_failed");
      const data = (await response.json()) as { events: PulseEvent[] };
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
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">{t("supplier_portal.network_pulse_panel.text.network_pulse")}</h3>
        <button type="button" className="portal-btn portal-btn--ghost text-xs" onClick={() => void load()}>
          Refresh
        </button>
      </div>
      <PulseTimeline events={events} loading={loading} error={error} />
    </div>
  );
}
