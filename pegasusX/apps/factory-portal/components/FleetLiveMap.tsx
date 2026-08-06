'use client';


import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef } from 'react';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import type { FactoryFleetLiveRoute } from '@/lib/use-factory-fleet-live-map';

const DEFAULT_CENTER: [number, number] = [69.2401, 41.2995];
const COLORS = ['#2563eb', '#059669', '#d97706', '#dc2626', '#7c3aed'];

type Props = {
  routes: FactoryFleetLiveRoute[];
  className?: string;
  loading?: boolean;
  error?: string | null;
};

export default function FleetLiveMap({ routes, className, loading, error }: Props) {
  const t = usePortalT();
  const mapRef = useRef<maplibregl.Map | null>(null);

  useEffect(() => {
    void import('maplibre-gl/dist/maplibre-gl.css');
  }, []);

  const points = useMemo(() => {
    const features = routes
      .map((route, index) => {
        const loc = route.driver_location;
        if (!loc) return null;
        const lng = loc.lng || loc.longitude || 0;
        const lat = loc.lat || loc.latitude || 0;
        if (!lat && !lng) return null;
        return {
          type: 'Feature' as const,
          properties: {
            color: COLORS[index % COLORS.length],
            label: route.driver_name || route.driver_id,
            stale: route.location_stale ? 1 : 0,
          },
          geometry: {
            type: 'Point' as const,
            coordinates: [lng, lat] as [number, number],
          },
        };
      })
      .filter((f): f is NonNullable<typeof f> => f !== null);
    return { type: 'FeatureCollection' as const, features };
  }, [routes]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || points.features.length === 0) return;
    const bounds = new maplibregl.LngLatBounds();
    for (const f of points.features) {
      bounds.extend(f.geometry.coordinates as [number, number]);
    }
    if (!bounds.isEmpty()) {
      map.fitBounds(bounds, { padding: 48, maxZoom: 14, duration: 500 });
    }
  }, [points]);

  if (loading && routes.length === 0) {
    return (
      <div className={className}>
        <p className="text-sm text-center px-4 py-8" style={{ color: 'var(--muted)' }}>
          Loading live fleet map…
        </p>
      </div>
    );
  }

  if (error && routes.length === 0) {
    return (
      <div className={className}>
        <p className="text-sm text-center px-4 py-8" style={{ color: 'var(--danger)' }}>
          {error}
        </p>
      </div>
    );
  }

  if (points.features.length === 0) {
    return (
      <div className={className}>
        <p className="text-sm text-center px-4 py-8" style={{ color: 'var(--muted)' }}>
          No sealed/dispatched factory drivers with live GPS right now.
        </p>
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
        style={{ width: '100%', height: '100%' }}
        mapLib={maplibregl}
      >
        <NavigationControl position="top-right" />
        <Source id="factory-fleet-drivers" type="geojson" data={points}>
          <Layer
            id="factory-fleet-driver-points"
            type="circle"
            paint={{
              'circle-color': ['get', 'color'],
              'circle-radius': 8,
              'circle-stroke-color': '#ffffff',
              'circle-stroke-width': 2,
              'circle-opacity': ['case', ['==', ['get', 'stale'], 1], 0.45, 1],
            }}
          />
        </Source>
      </MapGL>
    </div>
  );
}
