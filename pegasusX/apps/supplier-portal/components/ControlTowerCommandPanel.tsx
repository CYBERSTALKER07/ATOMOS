"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useState } from "react";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import { supplierFetch } from "@/lib/auth";
import { sessionMapCenter } from "@pegasusx/api-client";

function packZonePolygon() {
  const c = sessionMapCenter();
  if (!c) return null;
  const d = 0.02;
  return {
    type: "Polygon",
    coordinates: [
      [
        [c.lng - d, c.lat - d],
        [c.lng + d, c.lat - d],
        [c.lng + d, c.lat + d],
        [c.lng - d, c.lat + d],
        [c.lng - d, c.lat - d],
      ],
    ],
  };
}

export default function ControlTowerCommandPanel({ className = "" }: { className?: string }) {
  const t = usePortalT();
  const [action, setAction] = useState<"REROUTE" | "FREEZE_DISPATCH" | "PRIORITY_BOOST">("REROUTE");
  const [status, setStatus] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const publishOverride = useCallback(async () => {
    setBusy(true);
    setStatus(null);
    try {
      const polygon = packZonePolygon();
      if (!polygon) {
        setStatus("no pack map center");
        return;
      }
      const res = await supplierFetch("/v1/supplier/control-tower/zone-overrides", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action,
          ttl_seconds: 1800,
          polygon_geojson: polygon,
        }),
      });
      if (!res.ok) {
        throw new Error(`override ${res.status}`);
      }
      const row = await res.json();
      setStatus(`Override ${row.override_id} active (${row.action})`);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : t("supplier_portal.residual.text.override_failed"));
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
        <span className="md-typescale-title-medium mr-2">{t("supplier_portal.control_tower_command_panel.text.control_tower")}</span>
        <select
          className="portal-input text-sm"
          value={action}
          onChange={(e) => setAction(e.target.value as typeof action)}
        >
          <option value="REROUTE">{t("supplier_portal.control_tower_command_panel.text.reroute")}</option>
          <option value="FREEZE_DISPATCH">{t("supplier_portal.control_tower_command_panel.text.freeze_dispatch")}</option>
          <option value="PRIORITY_BOOST">{t("supplier_portal.control_tower_command_panel.text.priority_boost")}</option>
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
