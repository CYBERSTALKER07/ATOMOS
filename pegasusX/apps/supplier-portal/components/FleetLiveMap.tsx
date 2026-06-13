'use client';

import { useEffect, useMemo, useRef } from 'react';
import type { SupplierFleetLiveRoute } from '@pegasusx/types';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from '@/lib/use-animated-driver-markers';

const DEFAULT_CENTER: [number, number] = [69.2401, 41.2995];

type FleetLiveMapProps = {
  routes: SupplierFleetLiveRoute[];
  className?: string;
  loading?: boolean;
  error?: string | null;
};

function routeLineFeature(
  route: SupplierFleetLiveRoute,
  index: number,
): GeoJSON.Feature<GeoJSON.LineString> | null {
  const geometry = route.route_geometry;
  if (!geometry?.coordinates?.length || geometry.coordinates.length < 2) {
    return null;
  }
  return {
    type: 'Feature',
    properties: {
      color: FLEET_ROUTE_COLORS[index % FLEET_ROUTE_COLORS.length],
      label: route.driver_name || route.driver_id,
      source: geometry.source,
    },
    geometry: {
      type: 'LineString',
      coordinates: geometry.coordinates.map((point) => [point.lng, point.lat]),
    },
  };
}

export default function FleetLiveMap({ routes, className, loading, error }: FleetLiveMapProps) {
  const mapRef = useRef<maplibregl.Map | null>(null);
  const pointCollection = useAnimatedDriverMarkers(routes);

  const lineCollection = useMemo<GeoJSON.FeatureCollection<GeoJSON.LineString>>(() => {
    const features = routes
      .map((route, index) => routeLineFeature(route, index))
      .filter((feature): feature is GeoJSON.Feature<GeoJSON.LineString> => feature !== null);
    return { type: 'FeatureCollection', features };
  }, [routes]);

  useEffect(() => {
    const map = mapRef.current;
    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;
    for (const feature of lineCollection.features) {
      for (const coordinate of feature.geometry.coordinates) {
        bounds.extend(coordinate as [number, number]);
        hasBounds = true;
      }
    }
    for (const feature of pointCollection.features) {
      bounds.extend(feature.geometry.coordinates as [number, number]);
      hasBounds = true;
    }
    if (map && hasBounds && !bounds.isEmpty()) {
      map.fitBounds(bounds, { padding: 48, maxZoom: 14, duration: 500 });
    }
  }, [lineCollection, pointCollection]);

  if (loading && routes.length === 0) {
    return (
      <div className={className} style={{ color: 'var(--color-md-outline, var(--muted))' }}>
        <p className="text-sm text-center px-4 py-8">Loading live fleet map…</p>
      </div>
    );
  }

  if (error && routes.length === 0) {
    return (
      <div className={className} style={{ color: 'var(--desk-danger, var(--color-md-error))' }}>
        <p className="text-sm text-center px-4 py-8">{error}</p>
      </div>
    );
  }

  if (lineCollection.features.length === 0 && pointCollection.features.length === 0) {
    return (
      <div
        className={className}
        style={{
          background: 'var(--color-md-surface-container, var(--background))',
          color: 'var(--color-md-outline, var(--muted))',
        }}
      >
        <p className="text-sm text-center px-4 py-8">
          No sealed manifests with route geometry are in transit right now.
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
        <NavigationControl position="top-right" showCompass={false} />
        {lineCollection.features.length > 0 ? (
          <Source id="fleet-live-routes" type="geojson" data={lineCollection}>
            <Layer
              id="fleet-live-lines"
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
        ) : null}
        {pointCollection.features.length > 0 ? (
          <Source id="fleet-live-drivers" type="geojson" data={pointCollection}>
            <Layer
              id="fleet-live-driver-points"
              type="circle"
              paint={{
                'circle-color': ['get', 'color'],
                'circle-radius': 7,
                'circle-stroke-color': '#ffffff',
                'circle-stroke-width': 2,
                'circle-opacity': ['case', ['==', ['get', 'stale'], 1], 0.45, 1],
              }}
            />
          </Source>
        ) : null}
      </MapGL>
    </div>
  );
}
