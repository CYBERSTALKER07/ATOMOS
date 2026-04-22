'use client';

import React, { useState, useRef, useCallback } from 'react';
import dynamic from 'next/dynamic';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { cellToBoundary, latLngToCell } from 'h3-js';

const MapGL = dynamic(() => import('react-map-gl/maplibre').then(m => m.default), { ssr: false });
const Source = dynamic(() => import('react-map-gl/maplibre').then(m => m.Source), { ssr: false });
const Layer = dynamic(() => import('react-map-gl/maplibre').then(m => m.Layer), { ssr: false });
const Marker = dynamic(() => import('react-map-gl/maplibre').then(m => m.Marker), { ssr: false });

const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';
const H3_RES_OPTIONS = [
  { value: 7, label: 'Standard (res 7)', sub: '~5.16 km²/hex' },
  { value: 8, label: 'Dense (res 8)', sub: '~0.74 km²/hex' },
];

interface Props {
  warehouseId: string;
  warehouseName: string;
  lat: number;
  lng: number;
  existingHexes: string[];
  onSaved: () => void;
}

interface ConflictItem {
  hex: string;
  warehouse_id: string;
  warehouse_name: string;
}

interface ValidateResult {
  hexes: string[];
  conflicts: ConflictItem[];
  retailer_count: number;
}

/** Convert an array of H3 cell IDs to a GeoJSON FeatureCollection of polygons. */
function hexesToGeoJSON(hexes: string[], color: string) {
  const features = hexes.map(hex => {
    const boundary = cellToBoundary(hex, true); // [lng, lat] for GeoJSON
    return {
      type: 'Feature' as const,
      properties: { hex, color },
      geometry: {
        type: 'Polygon' as const,
        coordinates: [boundary.concat([boundary[0]])], // close ring
      },
    };
  });
  return { type: 'FeatureCollection' as const, features };
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export default function CoverageEditor({ warehouseId, warehouseName, lat, lng, existingHexes, onSaved }: Props) {
  const [resolution, setResolution] = useState(7);
  const [polygon, setPolygon] = useState<[number, number][]>([]); // [lat, lng] vertices
  const [drawing, setDrawing] = useState(false);
  const [previewResult, setPreviewResult] = useState<ValidateResult | null>(null);
  const [saving, setSaving] = useState(false);
  const [validating, setValidating] = useState(false);
  const [error, setError] = useState('');
  const mapRef = useRef<{ getMap: () => unknown } | null>(null);

  // Existing coverage as GeoJSON
  const existingGeoJSON = hexesToGeoJSON(existingHexes, '#4CAF50');

  // Preview hexes GeoJSON
  const previewGeoJSON = previewResult
    ? hexesToGeoJSON(
        previewResult.hexes.filter(h => !previewResult.conflicts.some(c => c.hex === h)),
        '#2196F3',
      )
    : null;

  // Conflict hexes GeoJSON
  const conflictGeoJSON = previewResult
    ? hexesToGeoJSON(
        previewResult.conflicts.map(c => c.hex),
        '#F44336',
      )
    : null;

  // Polygon drawing GeoJSON
  const polygonGeoJSON = polygon.length >= 2
    ? {
        type: 'FeatureCollection' as const,
        features: [{
          type: 'Feature' as const,
          properties: {},
          geometry: {
            type: polygon.length >= 3 ? 'Polygon' as const : 'LineString' as const,
            coordinates: polygon.length >= 3
              ? [[...polygon.map(p => [p[1], p[0]]), [polygon[0][1], polygon[0][0]]]]
              : polygon.map(p => [p[1], p[0]]),
          },
        }],
      }
    : null;

  const handleMapClick = useCallback((e: { lngLat: { lat: number; lng: number } }) => {
    if (!drawing) return;
    setPolygon(prev => [...prev, [e.lngLat.lat, e.lngLat.lng]]);
  }, [drawing]);

  const startDrawing = () => {
    setPolygon([]);
    setPreviewResult(null);
    setError('');
    setDrawing(true);
  };

  const finishDrawing = () => {
    setDrawing(false);
  };

  const clearPolygon = () => {
    setPolygon([]);
    setPreviewResult(null);
    setError('');
    setDrawing(false);
  };

  // Preview / validate
  const handlePreview = async () => {
    if (polygon.length < 3) {
      setError('Draw at least 3 points to define a zone');
      return;
    }
    setValidating(true);
    setError('');
    try {
      const res = await apiFetch('/v1/supplier/warehouses/validate-coverage', {
        method: 'POST',
        body: JSON.stringify({
          polygon: polygon.map(p => [p[0], p[1]]), // [lat, lng]
          h3_resolution: resolution,
          warehouse_id: warehouseId,
        }),
      });
      if (!res.ok) throw new Error('Validation failed');
      const data: ValidateResult = await res.json();
      setPreviewResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Preview failed');
    } finally {
      setValidating(false);
    }
  };

  // Save the zone
  const handleSave = async () => {
    if (polygon.length < 3) return;
    setSaving(true);
    setError('');
    try {
      const res = await apiFetch(`/v1/supplier/warehouses/${warehouseId}/coverage`, {
        method: 'POST',
        body: JSON.stringify({
          polygon: polygon.map(p => [p[0], p[1]]),
          h3_resolution: resolution,
        }),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Save failed');
      }
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  // Auto-generate a circle polygon from the warehouse center
  const generateCircle = () => {
    const radiusKm = 20;
    const points = 24;
    const pts: [number, number][] = [];
    for (let i = 0; i < points; i++) {
      const angle = (i / points) * 2 * Math.PI;
      const dLat = (radiusKm / 111.32) * Math.cos(angle);
      const dLng = (radiusKm / (111.32 * Math.cos(lat * Math.PI / 180))) * Math.sin(angle);
      pts.push([lat + dLat, lng + dLng]);
    }
    setPolygon(pts);
    setDrawing(false);
    setPreviewResult(null);
  };

  // Quick-preview the center cell
  const centerHex = latLngToCell(lat, lng, resolution);

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="p-4 space-y-3" style={{ borderBottom: '1px solid var(--border)' }}>
        {/* Resolution selector */}
        <div className="flex gap-2">
          {H3_RES_OPTIONS.map(opt => (
            <button
              key={opt.value}
              onClick={() => { setResolution(opt.value); setPreviewResult(null); }}
              className="flex-1 px-3 py-2 rounded-lg transition-colors md-typescale-label-small"
              style={{
                background: resolution === opt.value ? 'var(--accent)' : 'var(--surface)',
                color: resolution === opt.value ? 'var(--accent-foreground)' : 'var(--foreground)',
                border: `1px solid ${resolution === opt.value ? 'var(--accent)' : 'var(--border)'}`,
              }}
            >
              <div>{opt.label}</div>
              <div style={{ color: resolution === opt.value ? 'var(--accent-foreground)' : 'var(--muted)', opacity: 0.7, fontSize: '0.65rem' }}>{opt.sub}</div>
            </button>
          ))}
        </div>

        {/* Drawing controls */}
        <div className="flex gap-2">
          {!drawing ? (
            <>
              <Button size="sm" className="button--primary flex-1" onPress={startDrawing}>
                <Icon name="edit" size={14} className="mr-1" />
                Draw Zone
              </Button>
              <Button size="sm" variant="outline" className="flex-1" onPress={generateCircle}>
                20km Circle
              </Button>
            </>
          ) : (
            <>
              <Button size="sm" className="button--primary flex-1" onPress={finishDrawing} isDisabled={polygon.length < 3}>
                Finish ({polygon.length} pts)
              </Button>
              <Button size="sm" variant="outline" onPress={clearPolygon}>
                Clear
              </Button>
            </>
          )}
        </div>

        {/* Status info */}
        <div className="flex items-center gap-4 md-typescale-label-small" style={{ color: 'var(--muted)' }}>
          <span>Existing: {existingHexes.length} hexes</span>
          <span>Center: {centerHex.slice(0, 10)}...</span>
          {polygon.length > 0 && <span>Polygon: {polygon.length} vertices</span>}
        </div>

        {error && (
          <div className="flex items-center gap-2 p-2 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
            <Icon name="error" size={14} />
            <p className="md-typescale-label-small">{error}</p>
          </div>
        )}
      </div>

      {/* Map */}
      <div className="flex-1 relative" style={{ minHeight: 350 }}>
        <MapGL
          ref={mapRef as React.Ref<never>}
          initialViewState={{ longitude: lng, latitude: lat, zoom: 10 }}
          mapStyle={MAP_STYLE}
          style={{ width: '100%', height: '100%' }}
          onClick={handleMapClick}
          cursor={drawing ? 'crosshair' : 'grab'}
        >
          {/* Warehouse marker */}
          <Marker longitude={lng} latitude={lat} anchor="center">
            <div className="w-6 h-6 rounded-full flex items-center justify-center" style={{ background: 'var(--accent)', border: '2px solid white' }}>
              <Icon name="warehouse" size={12} className="text-white" />
            </div>
          </Marker>

          {/* Existing coverage (green) */}
          {existingHexes.length > 0 && (
            <Source id="existing-hexes" type="geojson" data={existingGeoJSON}>
              <Layer id="existing-hex-fill" type="fill" paint={{ 'fill-color': '#4CAF50', 'fill-opacity': 0.2 }} />
              <Layer id="existing-hex-line" type="line" paint={{ 'line-color': '#4CAF50', 'line-width': 1.5, 'line-opacity': 0.6 }} />
            </Source>
          )}

          {/* Drawing polygon outline */}
          {polygonGeoJSON && (
            <Source id="drawing-polygon" type="geojson" data={polygonGeoJSON as GeoJSON.FeatureCollection}>
              {polygon.length >= 3 ? (
                <>
                  <Layer id="draw-fill" type="fill" paint={{ 'fill-color': '#FFC107', 'fill-opacity': 0.15 }} />
                  <Layer id="draw-line" type="line" paint={{ 'line-color': '#FFC107', 'line-width': 2, 'line-dasharray': [2, 2] }} />
                </>
              ) : (
                <Layer id="draw-line" type="line" paint={{ 'line-color': '#FFC107', 'line-width': 2, 'line-dasharray': [2, 2] }} />
              )}
            </Source>
          )}

          {/* Preview hexes (blue) */}
          {previewGeoJSON && previewGeoJSON.features.length > 0 && (
            <Source id="preview-hexes" type="geojson" data={previewGeoJSON}>
              <Layer id="preview-hex-fill" type="fill" paint={{ 'fill-color': '#2196F3', 'fill-opacity': 0.3 }} />
              <Layer id="preview-hex-line" type="line" paint={{ 'line-color': '#2196F3', 'line-width': 1.5 }} />
            </Source>
          )}

          {/* Conflict hexes (red) */}
          {conflictGeoJSON && conflictGeoJSON.features.length > 0 && (
            <Source id="conflict-hexes" type="geojson" data={conflictGeoJSON}>
              <Layer id="conflict-hex-fill" type="fill" paint={{ 'fill-color': '#F44336', 'fill-opacity': 0.4 }} />
              <Layer id="conflict-hex-line" type="line" paint={{ 'line-color': '#F44336', 'line-width': 2 }} />
            </Source>
          )}

          {/* Drawing vertex markers */}
          {polygon.map((pt, i) => (
            <Marker key={i} longitude={pt[1]} latitude={pt[0]} anchor="center">
              <div className="w-3 h-3 rounded-full" style={{ background: '#FFC107', border: '1px solid white' }} />
            </Marker>
          ))}
        </MapGL>

        {drawing && (
          <div className="absolute top-3 left-3 px-3 py-1.5 rounded-lg md-typescale-label-small" style={{ background: 'var(--backdrop)', color: 'white' }}>
            Click map to add vertices
          </div>
        )}
      </div>

      {/* Preview results + action bar */}
      <div className="p-4 space-y-3" style={{ borderTop: '1px solid var(--border)' }}>
        {previewResult && (
          <div className="grid grid-cols-3 gap-3 text-center">
            <div className="md-card p-2">
              <p className="md-typescale-headline-small" style={{ color: 'var(--accent)', fontVariantNumeric: 'tabular-nums' }}>
                {previewResult.hexes.length}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Hexes</p>
            </div>
            <div className="md-card p-2">
              <p className="md-typescale-headline-small" style={{ color: 'var(--foreground)', fontVariantNumeric: 'tabular-nums' }}>
                {previewResult.retailer_count}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Retailers</p>
            </div>
            <div className="md-card p-2">
              <p className="md-typescale-headline-small" style={{ color: previewResult.conflicts.length > 0 ? 'var(--danger)' : 'var(--success)', fontVariantNumeric: 'tabular-nums' }}>
                {previewResult.conflicts.length}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Conflicts</p>
            </div>
          </div>
        )}

        {previewResult && previewResult.conflicts.length > 0 && (
          <div className="p-2 rounded-lg md-typescale-label-small" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
            Overlaps with: {[...new Set(previewResult.conflicts.map(c => c.warehouse_name))].join(', ')}
          </div>
        )}

        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            className="flex-1"
            onPress={handlePreview}
            isPending={validating}
            isDisabled={polygon.length < 3}
          >
            Preview
          </Button>
          <Button
            size="sm"
            className="button--primary flex-1"
            onPress={handleSave}
            isPending={saving}
            isDisabled={polygon.length < 3 || (previewResult !== null && previewResult.conflicts.length > 0)}
          >
            Save Zone
          </Button>
        </div>
      </div>
    </div>
  );
}
