"use client";

import { useCallback, useState } from "react";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import { supplierFetch } from "@/lib/auth";

const DEFAULT_POLYGON = {
  type: "Polygon",
  coordinates: [
    [
      [69.24, 41.31],
      [69.28, 41.31],
      [69.28, 41.34],
      [69.24, 41.34],
      [69.24, 41.31],
    ],
  ],
};

export default function ControlTowerCommandPanel({ className = "" }: { className?: string }) {
  const [action, setAction] = useState<"REROUTE" | "FREEZE_DISPATCH" | "PRIORITY_BOOST">("REROUTE");
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const publishOverride = useCallback(async () => {
    setBusy(true);
    setStatus(null);
    try {
      const res = await supplierFetch("/v1/supplier/control-tower/zone-overrides", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action,
          ttl_seconds: 1800,
          polygon_geojson: DEFAULT_POLYGON,
        }),
      });
      if (!res.ok) {
        throw new Error(`override ${res.status}`);
      }
      const row = await res.json();
      setStatus(`Override ${row.override_id} active (${row.action})`);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Override failed");
    } finally {
      setBusy(false);
    }
  }, [action]);

  return (
    <div className={`flex flex-col min-h-[420px] ${className}`}>
      <div
        className="flex flex-wrap items-center gap--2 px-5 py-3 border-b"
        style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
      >
        <span className="md-typescale-title-medium mr-2">Control tower</span>
        <select
          className="portal-input text-sm"
          value={action}
          onChange={(e) => setAction(e.target.value as typeof action)}
        >
          <option value="REROUTE">Reroute</option>
          <option value="FREEZE_DISPATCH">Freeze dispatch</option>
          <option value="PRIORITY_BOOST">Priority boost</option>
        </select>
        <button
          type="button"
          className="portal-btn portal-btn--primary text-sm"
          disabled={busy}
          onClick={() => void publishOverride()}
        >
          {busy ? "Publishing…" : "Publish zone override"}
        </button>
        {status ? (
          <span className="md-typescale-body-small ml-2" style={{ color: "var(--desk-text-secondary)" }}>
            {status}
          </span>
        ) : null}
      </div>
      <FleetLiveMapPanel className="flex-1 min-h-[360px]" />
    </div>
  );
}
