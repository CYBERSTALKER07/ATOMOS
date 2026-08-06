"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef, useState } from "react";
import type { TwinOpsRouteView } from "@pegasusx/types";
import MapGL, { Layer, NavigationControl, Source } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import { useMapLibreTeardown } from "@pegasusx/ui-kit/desktop";

const DEFAULT_CENTER: [number, number] = [69.2401, 41.2995];

const LATENESS_COLORS: Record<string, string> = {
  green: "#0f9d58",
  amber: "#f4b400",
  red: "#db4437",
};

type LiveOpsMapProps = {
  routes: TwinOpsRouteView[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  selectedRouteId?: string | null;
  onSelectRoute?: (route: TwinOpsRouteView | null) => void;
};

export default function LiveOpsMap({
  routes,
  className,
  loading,
  error,
  selectedRouteId,
  onSelectRoute,
}: LiveOpsMapProps) {
  const t = usePortalT();
  const mapRef = useRef<maplibregl.Map | null>(null);
  useMapLibreTeardown(mapRef);

  useEffect(() => {
    void import("maplibre-gl/dist/maplibre-gl.css");
  }, []);

  const pointCollection = useMemo<GeoJSON.FeatureCollection<GeoJSON.Point>>(() => {
    const features: GeoJSON.Feature<GeoJSON.Point>[] = [];
    for (const route of routes) {
      if (route.CurrentLat == null || route.CurrentLng == null) continue;
      features.push({
        type: "Feature",
        properties: {
          routeId: route.RouteID,
          color: LATENESS_COLORS[route.lateness] ?? LATENESS_COLORS.green,
          selected: route.RouteID === selectedRouteId ? 1 : 0,
        },
        geometry: {
          type: "Point",
          coordinates: [route.CurrentLng, route.CurrentLat],
        },
      });
    }
    return { type: "FeatureCollection", features };
  }, [routes, selectedRouteId]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || pointCollection.features.length === 0) return;
    const bounds = new maplibregl.LngLatBounds();
    for (const feature of pointCollection.features) {
      bounds.extend(feature.geometry.coordinates as [number, number]);
    }
    if (!bounds.isEmpty()) {
      map.fitBounds(bounds, { padding: 64, maxZoom: 13, duration: 500 });
    }
  }, [pointCollection]);

  if (loading && routes.length === 0) {
    return (
      <div className={className} style={{ color: "var(--color-md-outline, var(--muted))" }}>
        <p className="text-sm text-center px-4 py-8">{t("supplier_portal.live_ops_map.text.loading_live_routes")}</p>
      </div>
    );
  }

  if (error && routes.length === 0) {
    return (
      <div className={className} style={{ color: "var(--desk-danger)" }}>
        <p className="text-sm text-center px-4 py-8">{error}</p>
      </div>
    );
  }

  if (pointCollection.features.length === 0) {
    return (
      <div
        className={className}
        style={{
          background: "var(--color-md-surface-container, var(--background))",
          color: "var(--color-md-outline, var(--muted))",
        }}
      >
        <p className="text-sm text-center px-4 py-8">{t("supplier_portal.live_ops_map.text.no_active_routes_from_the_digital_twin_right_now")}</p>
      </div>
    );
  }

  return (
    <div className={className}>
      <MapGL
        ref={(ref) => {
          mapRef.current = ref?.getMap() ?? null;
        }}
        initialViewState={{
          longitude: DEFAULT_CENTER[0],
          latitude: DEFAULT_CENTER[1],
          zoom: 11,
        }}
        mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
        style={{ width: "100%", height: "100%" }}
        mapLib={maplibregl}
        onClick={(evt) => {
          if (!onSelectRoute) return;
          const feature = evt.features?.[0];
          const routeId = feature?.properties?.routeId as string | undefined;
          if (!routeId) {
            onSelectRoute(null);
            return;
          }
          const route = routes.find((r) => r.RouteID === routeId);
          onSelectRoute(route ?? null);
        }}
        interactiveLayerIds={["live-ops-route-points"]}
      >
        <NavigationControl position="top-right" />
        <Source id="live-ops-routes" type="geojson" data={pointCollection}>
          <Layer
            id="live-ops-route-points"
            type="circle"
            paint={{
              "circle-color": ["get", "color"],
              "circle-radius": ["case", ["==", ["get", "selected"], 1], 12, 8],
              "circle-stroke-color": "#ffffff",
              "circle-stroke-width": 2,
            }}
          />
        </Source>
      </MapGL>
    </div>
  );
}

export function LiveOpsSidePanel({ route }: { route: TwinOpsRouteView | null }) {
  if (!route) {
    return (
      <div className="p-4 text-sm" style={{ color: "var(--desk-text-secondary)" }}>
        Select a route marker to inspect driver, stops, inventory, and open exceptions.
      </div>
    );
  }

  return (
    <div className="p-4 flex flex-col gap-4 text-sm overflow-y-auto h-full">
      <div>
        <h3 className="font-semibold text-base">{route.driver_name || route.DriverID}</h3>
        <p style={{ color: "var(--desk-text-secondary)" }}>
          Route <span className="font-mono">{route.RouteID}</span>
          {route.driver_score != null ? ` · Score ${route.driver_score}` : ""}
        </p>
        <p>
          Lateness:{" "}
          <span
            style={{
              color: LATENESS_COLORS[route.lateness] ?? LATENESS_COLORS.green,
              fontWeight: 600,
            }}
          >
            {route.lateness}
          </span>
          · {route.RemainingStops ?? 0} stops remaining
        </p>
      </div>

      <section>
        <h4 className="font-medium mb-2">{t("supplier_portal.live_ops_map.text.remaining_stops")}</h4>
        <ul className="space-y-2">
          {(route.Stops ?? [])
            .filter((s) => s.Status !== "COMPLETED" && s.Status !== "CANCELLED")
            .map((stop) => (
              <li key={stop.StopID} className="border rounded p-2" style={{ borderColor: "var(--desk-border)" }}>
                <div className="font-mono text-xs">{stop.StopID}</div>
                <div>
                  ETA window: {formatTime(stop.WindowStart)} – {formatTime(stop.WindowEnd)}
                </div>
                <div>Predicted: {formatTime(stop.PredictedArrival)}</div>
              </li>
            ))}
        </ul>
      </section>

      <section>
        <h4 className="font-medium mb-2">{t("supplier_portal.live_ops_map.text.vehicle_inventory")}</h4>
        {(route.Inventory ?? []).length === 0 ? (
          <p style={{ color: "var(--desk-text-secondary)" }}>{t("supplier_portal.live_ops_map.text.no_inventory_rows")}</p>
        ) : (
          <ul className="space-y-1 font-mono text-xs">
            {(route.Inventory ?? []).map((row) => (
              <li key={row.Sku}>{row.Sku}: {row.QtyOnVehicle}</li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h4 className="font-medium mb-2">{t("supplier_portal.live_ops_map.text.open_exceptions")}</h4>
        {(route.exceptions ?? []).length === 0 ? (
          <p style={{ color: "var(--desk-text-secondary)" }}>{t("supplier_portal.live_ops_map.text.none_on_this_route")}</p>
        ) : (
          <ul className="space-y-2">
            {(route.exceptions ?? []).map((exc, idx) => (
              <li key={`${exc.type}-${idx}`} className="border rounded p-2" style={{ borderColor: "var(--desk-border)" }}>
                <div className="font-medium">{exc.type}</div>
                {exc.order_id ? <div className="font-mono text-xs">{exc.order_id}</div> : null}
                {exc.status ? <div>{exc.status}</div> : null}
                {exc.detail ? <div className="text-xs opacity-80">{exc.detail}</div> : null}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function formatTime(value?: string): string {
  if (!value) return "—";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString();
}
