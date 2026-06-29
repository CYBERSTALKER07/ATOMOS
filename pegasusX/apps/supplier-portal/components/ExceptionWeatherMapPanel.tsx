"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { ExceptionMapCell } from "@pegasusx/types";
import MapGL, { Layer, NavigationControl, Source } from "react-map-gl/maplibre";
import "maplibre-gl/dist/maplibre-gl.css";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();
const DEFAULT_CENTER = { longitude: 69.2401, latitude: 41.2995, zoom: 10 };

const SEVERITY_COLOR: Record<string, string> = {
  low: "#64748b",
  medium: "#f59e0b",
  high: "#dc2626",
};

type ExceptionWeatherMapPanelProps = {
  className?: string;
};

export default function ExceptionWeatherMapPanel({ className }: ExceptionWeatherMapPanelProps) {
  const [cells, setCells] = useState<ExceptionMapCell[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getSupplierExceptionMap({ window_hours: 24 });
      setCells(data.cells ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load exception map");
      setCells([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const pointCollection = useMemo<GeoJSON.FeatureCollection<GeoJSON.Point>>(() => ({
    type: "FeatureCollection",
    features: cells
      .filter((cell) => cell.lat !== 0 || cell.lng !== 0)
      .map((cell) => ({
        type: "Feature",
        properties: {
          severity: cell.severity,
          total: cell.counts?.total ?? 0,
          h3: cell.h3_cell,
        },
        geometry: {
          type: "Point",
          coordinates: [cell.lng, cell.lat],
        },
      })),
  }), [cells]);

  return (
    <div className={className}>
      <div className="flex items-center justify-between gap-3 px-5 py-3 border-b" style={{ borderColor: "var(--desk-border)" }}>
        <div>
          <h3 className="text-sm font-semibold uppercase tracking-wide opacity-70">Exception weather</h3>
          <p className="text-xs opacity-60 mt-1">H3 clusters for shop-closed, delays, and manifest gate issues (24h)</p>
        </div>
        <button type="button" className="portal-btn portal-btn--ghost text-xs" onClick={() => void load()}>
          Refresh
        </button>
      </div>

      {loading ? (
        <p className="text-sm text-center py-10 opacity-60">Loading exception map…</p>
      ) : error ? (
        <p className="text-sm text-center py-10" style={{ color: "var(--desk-danger)" }}>{error}</p>
      ) : cells.length === 0 ? (
        <p className="text-sm text-center py-10 opacity-60">No exception hotspots in the last 24 hours.</p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr] min-h-[320px]">
          <div className="min-h-[280px] rounded-lg overflow-hidden border" style={{ borderColor: "var(--desk-border)" }}>
            <MapGL
              initialViewState={DEFAULT_CENTER}
              mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
              style={{ width: "100%", height: "100%", minHeight: 280 }}
            >
              <NavigationControl position="top-right" showCompass={false} />
              <Source id="exception-cells" type="geojson" data={pointCollection}>
                <Layer
                  id="exception-cells-circle"
                  type="circle"
                  paint={{
                    "circle-radius": ["interpolate", ["linear"], ["get", "total"], 1, 8, 5, 16, 10, 24],
                    "circle-color": [
                      "match",
                      ["get", "severity"],
                      "high",
                      SEVERITY_COLOR.high,
                      "medium",
                      SEVERITY_COLOR.medium,
                      SEVERITY_COLOR.low,
                    ],
                    "circle-opacity": 0.75,
                    "circle-stroke-width": 1,
                    "circle-stroke-color": "#ffffff",
                  }}
                />
              </Source>
            </MapGL>
          </div>

          <div className="overflow-y-auto max-h-[320px] space-y-2 p-2">
            {cells.map((cell) => (
              <Link
                key={cell.h3_cell}
                href={cell.deep_link}
                className="block rounded-lg border p-3 hover:border-[var(--desk-accent)] transition-colors"
                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide">{cell.severity}</span>
                  <span className="text-sm font-semibold tabular-nums">{cell.counts?.total ?? 0}</span>
                </div>
                <div className="mt-2 flex flex-wrap gap-2 text-[11px] opacity-70">
                  <span>shop-closed {cell.counts?.shop_closed ?? 0}</span>
                  <span>delayed {cell.counts?.delayed ?? 0}</span>
                  <span>gate {cell.counts?.manifest_gate ?? 0}</span>
                </div>
                {cell.sample_order_ids?.length ? (
                  <p className="mt-2 text-[11px] font-mono opacity-60 truncate">
                    {cell.sample_order_ids.join(", ")}
                  </p>
                ) : null}
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
