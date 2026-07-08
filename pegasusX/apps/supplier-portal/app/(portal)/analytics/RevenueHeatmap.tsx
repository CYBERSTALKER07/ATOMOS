'use client';

import { useMemo } from 'react';
import MapGL, { Source, Layer, NavigationControl } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { cellToBoundary } from 'h3-js';

const DEFAULT_CENTER: [number, number] = [-98.5795, 39.8283]; // USA center for mock data

type RevenueHeatmapProps = {
  className?: string;
};

// Removed Mock Data
const MOCK_H3_REVENUE_DATA: { h3: string; revenue: number }[] = [];

export default function RevenueHeatmap({ className }: RevenueHeatmapProps) {
  const geojsonData = useMemo<GeoJSON.FeatureCollection<GeoJSON.Polygon>>(() => {
    const features: GeoJSON.Feature<GeoJSON.Polygon>[] = MOCK_H3_REVENUE_DATA.map((item) => {
      // Get the boundaries of the hex cell
      const boundary = cellToBoundary(item.h3, true);
      // Close the polygon
      boundary.push(boundary[0]);
      
      // Calculate color intensity based on revenue (max ~120k)
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
  }, []);

  return (
    <div className={className}>
      <MapGL
        initialViewState={{
          longitude: DEFAULT_CENTER[0],
          latitude: DEFAULT_CENTER[1],
          zoom: 3.5,
        }}
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
                  1, 'rgba(255, 0, 0, 0.8)'
                ],
                'fill-opacity': 0.8,
                'fill-outline-color': 'rgba(255, 255, 255, 0.2)'
              }}
            />
          </Source>
        )}
      </MapGL>
    </div>
  );
}
