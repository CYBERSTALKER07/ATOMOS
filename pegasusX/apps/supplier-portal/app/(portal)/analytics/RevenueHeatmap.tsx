'use client';

import { useMemo } from 'react';
import MapGL, { Source, Layer, NavigationControl } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { cellToBoundary } from 'h3-js';
import { mapInitialViewState, readCachedAuthSession } from '@pegasusx/api-client';

type RevenueHeatmapProps = {
  className?: string;
  cells?: { h3: string; revenue: number }[];
};

export default function RevenueHeatmap({ className, cells = [] }: RevenueHeatmapProps) {
  const geojsonData = useMemo<GeoJSON.FeatureCollection<GeoJSON.Polygon>>(() => {
    const features: GeoJSON.Feature<GeoJSON.Polygon>[] = cells.map((item) => {
      const boundary = cellToBoundary(item.h3, true);
      boundary.push(boundary[0]);
      const intensity = Math.min(1, item.revenue / 120000);
      return {
        type: 'Feature',
        properties: {
          revenue: item.revenue,
          intensity,
        },
        geometry: {
          type: 'Polygon',
          coordinates: [boundary],
        },
      };
    });

    return { type: 'FeatureCollection', features };
  }, [cells]);

  return (
    <div className={className} style={{ position: 'relative' }}>
      <MapGL
        initialViewState={mapInitialViewState(readCachedAuthSession()?.pack, 10)}
        mapStyle="https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json"
        style={{ width: '100%', height: '100%', borderRadius: '12px' }}
        mapLib={maplibregl}
      >
        <NavigationControl position="top-right" showCompass={false} />

        {geojsonData.features.length > 0 && (
          <Source id="h3-revenue-source" type="geojson" data={geojsonData}>
            <Layer
              id="h3-revenue-layer"
              type="fill"
              paint={{
                'fill-color': [
                  'interpolate',
                  ['linear'],
                  ['get', 'intensity'],
                  0, 'rgba(0, 255, 0, 0.1)',
                  0.5, 'rgba(255, 255, 0, 0.5)',
                  1, 'rgba(255, 0, 0, 0.8)',
                ],
                'fill-opacity': 0.8,
                'fill-outline-color': 'rgba(255, 255, 255, 0.2)',
              }}
            />
          </Source>
        )}
      </MapGL>
      {cells.length === 0 ? (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-xl bg-black/40">
          <p className="text-xs text-gray-300 text-center px-4">
            No H3 revenue density yet. Heatmap fills when analytics density signals exist.
          </p>
        </div>
      ) : null}
    </div>
  );
}
