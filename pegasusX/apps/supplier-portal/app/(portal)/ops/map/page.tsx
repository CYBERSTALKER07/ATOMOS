"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { usePolling } from "@pegasusx/api-client";
import type { TwinOpsRouteView } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import LiveOpsMap, { LiveOpsSidePanel } from "@/components/LiveOpsMap";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();
const POLL_MS = 20_000;

export default function LiveOpsMapPage() {
  const [routes, setRoutes] = useState<TwinOpsRouteView[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [zoneH3, setZoneH3] = useState("");
  const [delayedOnly, setDelayedOnly] = useState(false);
  const [shopClosedOnly, setShopClosedOnly] = useState(false);
  const [driverId, setDriverId] = useState("");
  const [selected, setSelected] = useState<TwinOpsRouteView | null>(null);
  const selectedIdRef = useRef<string | null>(null);
  selectedIdRef.current = selected?.RouteID ?? null;

  const refresh = useCallback(
    async (silent = false) => {
      if (!silent) setLoading(true);
      try {
        const data = await api.listTwinActiveRoutes({
          zoneH3: zoneH3.trim() || undefined,
          delayedOnly,
          shopClosedOnly,
          driverId: driverId.trim() || undefined,
        });
        setRoutes(data ?? []);
        setError(null);
        const id = selectedIdRef.current;
        if (id) {
          setSelected(data.find((r) => r.RouteID === id) ?? null);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "twin_routes_failed");
      } finally {
        if (!silent) setLoading(false);
      }
    },
    [zoneH3, delayedOnly, shopClosedOnly, driverId],
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh(routes.length > 0);
    },
    POLL_MS,
    [refresh, routes.length],
    { hiddenIntervalMs: 60_000 },
  );

  return (
    <PageChrome
      icon="dispatch"
      title="Live ops map"
      description="Active routes from the digital twin — read-only visibility with ETA confidence colouring."
      loading={loading && routes.length === 0}
      skeletonVariant="dashboard"
      error={error && routes.length === 0 ? error : null}
    >
      <div className="flex flex-wrap gap-3 mb-4 px-1">
        <input
          className="md-input min-w-[140px]"
          placeholder="Zone H3"
          value={zoneH3}
          onChange={(e) => setZoneH3(e.target.value)}
        />
        <input
          className="md-input min-w-[140px]"
          placeholder="Driver ID"
          value={driverId}
          onChange={(e) => setDriverId(e.target.value)}
        />
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={delayedOnly} onChange={(e) => setDelayedOnly(e.target.checked)} />
          Delayed only
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={shopClosedOnly} onChange={(e) => setShopClosedOnly(e.target.checked)} />
          Shop-closed open
        </label>
        <button type="button" className="md-btn md-btn-outlined" onClick={() => void refresh()}>
          Refresh
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4 min-h-[70vh]">
        <LiveOpsMap
          className="min-h-[60vh] rounded-xl overflow-hidden border"
          routes={routes}
          loading={loading}
          error={error}
          selectedRouteId={selected?.RouteID}
          onSelectRoute={setSelected}
        />
        <div
          className="rounded-xl border min-h-[320px] lg:min-h-[60vh]"
          style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface)" }}
        >
          <LiveOpsSidePanel route={selected} />
        </div>
      </div>
    </PageChrome>
  );
}
