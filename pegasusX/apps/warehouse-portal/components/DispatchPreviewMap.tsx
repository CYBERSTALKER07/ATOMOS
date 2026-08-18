'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef } from 'react';
import type { WarehouseDispatchProposedRoute } from '@pegasusx/types';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-client';
const ROUTE_COLORS = [
  '#1b6ef3',
  '#0f9d58',
  '#db4437',
  '#f4b400',
  '#ab47bc',
  '#00838f',
];

type DispatchPreviewMapProps = {
  routes: WarehouseDispatchProposedRoute[];
  className?: string;
};

type RouteFeatureCollection = GeoJSON.FeatureCollection<GeoJSON.LineString>;

export default function DispatchPreviewMap({ routes, className }: DispatchPreviewMapProps) {
  const t = usePortalT();
  const mapRef = useRef<maplibregl.Map | null>(null);

  const featureCollection = useMemo<RouteFeatureCollection>(() => {
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = [];
    routes.forEach((route, index) => {
      const geometry = route.route_geometry;
      if (!geometry?.coordinates?.length || geometry.coordinates.length < 2) {
        return;
      }
      features.push({
        type: 'Feature',
        properties: {
          color: ROUTE_COLORS[index % ROUTE_COLORS.length],
          label: route.driver_name || route.driver_id || `Route ${index + 1}`,
          source: geometry.source,
        },
        geometry: {
          type: 'LineString',
          coordinates: geometry.coordinates.map((point) => [point.lng, point.lat]),
        },
      });
    });
    return { type: 'FeatureCollection', features };
  }, [routes]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || featureCollection.features.length === 0) {
      return;
    }
    const bounds = new maplibregl.LngLatBounds();
    for (const feature of featureCollection.features) {
      for (const coordinate of feature.geometry.coordinates) {
        bounds.extend(coordinate as [number, number]);
      }
    }
    if (!bounds.isEmpty()) {
      map.fitBounds(bounds, { padding: 48, maxZoom: 14, duration: 500 });
    }
  }, [featureCollection]);

  if (featureCollection.features.length === 0) {
    return (
      <div className={className} style={{ background: 'var(--background)', color: 'var(--muted)' }}>
        <p className="text-sm text-center px-4">{t("warehouse_portal.dispatch_preview_map.text.route_preview_unavailable_until_optimizer_proposes_stops_with_co")}</p>
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
          ...mapInitialViewState(readCachedAuthSession()?.pack),
          zoom: 11,
        }}
        mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
        style={{ width: '100%', height: '100%' }}
        mapLib={maplibregl}
      >
        <NavigationControl position="top-right" showCompass={false} />
        <Source id="dispatch-preview-routes" type="geojson" data={featureCollection}>
          <Layer
            id="dispatch-preview-lines"
            type="line"
            paint={{
              'line-color': ['get', 'color'],
              'line-width': 4,
              'line-opacity': 0.85,
            }}
            layout={{
              'line-cap': 'round',
              'line-join': 'round',
            }}
          />
        </Source>
      </MapGL>
    </div>
  );
}
