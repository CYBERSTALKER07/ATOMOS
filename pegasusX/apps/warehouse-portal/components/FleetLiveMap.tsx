'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useMemo, useRef } from 'react';
import type { WarehouseFleetLiveRoute } from '@pegasusx/types';
import MapGL, { Layer, NavigationControl, Source } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import { useMapLibreTeardown } from '@pegasusx/ui-kit/desktop';
import { FLEET_ROUTE_COLORS, useAnimatedDriverMarkers } from '@/lib/use-animated-driver-markers';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-client';
const MAP_PITCH_3D = 50;

type FleetLiveMapProps = {
  routes: WarehouseFleetLiveRoute[];
  className?: string;
  loading?: boolean;
  error?: string | null;
  enable3DView?: boolean;
};

function routeLineFeature(
  route: WarehouseFleetLiveRoute,
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

export default function FleetLiveMap({
  routes,
  className,
  loading,
  error,
  enable3DView = false,
}: FleetLiveMapProps) {
  const t = usePortalT();
  const mapRef = useRef<maplibregl.Map | null>(null);
  const pointCollection = useAnimatedDriverMarkers(routes);

  useMapLibreTeardown(mapRef);

  useEffect(() => {
    void import('maplibre-gl/dist/maplibre-gl.css');
  }, []);

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

  useEffect(() => {
    const map = mapRef.current;
    if (!map) return;
    map.easeTo({ pitch: enable3DView ? MAP_PITCH_3D : 0, duration: 300 });
  }, [enable3DView]);

  if (loading && routes.length === 0) {
    return (
      <div className={className} style={{ color: 'var(--muted)' }}>
        <p className="text-sm text-center px-4 py-8">{t("warehouse_portal.fleet_live_map.text.loading_live_fleet_map")}</p>
      </div>
    );
  }

  if (error && routes.length === 0) {
    return (
      <div className={className} style={{ color: 'var(--danger)' }}>
        <p className="text-sm text-center px-4 py-8">{error}</p>
      </div>
    );
  }

  if (lineCollection.features.length === 0 && pointCollection.features.length === 0) {
    return (
      <div className={className} style={{ background: 'var(--background)', color: 'var(--muted)' }}>
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
          ...mapInitialViewState(readCachedAuthSession()?.pack),
          zoom: 11,
          pitch: 0,
        }}
        mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
        style={{ width: '100%', height: '100%' }}
        mapLib={maplibregl}
        maxPitch={enable3DView ? 60 : 0}
      >
        <NavigationControl position="top-right" showCompass={enable3DView} />
        {lineCollection.features.length > 0 ? (
          <Source id="warehouse-fleet-live-routes" type="geojson" data={lineCollection}>
            <Layer
              id="warehouse-fleet-live-lines"
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
          <Source
            id="warehouse-fleet-live-drivers"
            type="geojson"
            data={pointCollection}
            cluster={true}
            clusterMaxZoom={14}
            clusterRadius={40}
          >
            <Layer
              id="warehouse-fleet-live-driver-clusters"
              type="circle"
              filter={['has', 'point_count']}
              paint={{
                'circle-color': '#111827',
                'circle-radius': ['step', ['get', 'point_count'], 14, 10, 18, 30, 22],
                'circle-stroke-color': '#ffffff',
                'circle-stroke-width': 2,
              }}
            />
            <Layer
              id="warehouse-fleet-live-driver-cluster-count"
              type="symbol"
              filter={['has', 'point_count']}
              layout={{
                'text-field': '{point_count_abbreviated}',
                'text-size': 11,
              }}
              paint={{
                'text-color': '#ffffff',
              }}
            />
            <Layer
              id="warehouse-fleet-live-driver-points"
              type="circle"
              filter={['!', ['has', 'point_count']]}
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
