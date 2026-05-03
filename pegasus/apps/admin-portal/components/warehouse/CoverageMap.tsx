'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import MapGL, { Marker, Source, Layer, NavigationControl, type MapRef } from 'react-map-gl/maplibre';
import type { MapLayerMouseEvent } from 'react-map-gl/maplibre';
import 'maplibre-gl/dist/maplibre-gl.css';
import { apiFetch } from '@/lib/auth';

const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';
const TASHKENT = { latitude: 41.2995, longitude: 69.2401 };

/* ── Types ─────────────────────────────────────────────────────────────── */

interface ZoneRetailer {
  retailer_id: string;
  name: string;
  shop_name: string;
  latitude: number;
  longitude: number;
  distance_km: number;
  orders_last_7: number;
}

interface OverlapWarning {
  warehouse_id: string;
  warehouse_name: string;
  overlap_retailers: number;
  distance_km: number;
}

interface ZonePreviewData {
  retailer_count: number;
  recent_orders_day: number;
  retailers_in_zone: ZoneRetailer[];
  overlap_warnings: OverlapWarning[];
}

interface CoverageMapProps {
  latitude: number;
  longitude: number;
  radiusKm: number;
  addressText: string;
  excludeWarehouseId?: string;
  onLocationChange: (lat: number, lng: number, address: string) => void;
}

/* ── Circle GeoJSON Generator ──────────────────────────────────────────── */

function createCircleGeoJSON(lat: number, lng: number, radiusKm: number, points = 64) {
  const coords: [number, number][] = [];
  const distRadians = radiusKm / 6371.0;
  const latRad = (lat * Math.PI) / 180;
  const lngRad = (lng * Math.PI) / 180;

  for (let i = 0; i <= points; i++) {
    const angle = (i / points) * 2 * Math.PI;
    const cLat = Math.asin(
      Math.sin(latRad) * Math.cos(distRadians) +
      Math.cos(latRad) * Math.sin(distRadians) * Math.cos(angle)
    );
    const cLng = lngRad + Math.atan2(
      Math.sin(angle) * Math.sin(distRadians) * Math.cos(latRad),
      Math.cos(distRadians) - Math.sin(latRad) * Math.sin(cLat)
    );
    coords.push([(cLng * 180) / Math.PI, (cLat * 180) / Math.PI]);
  }

  return {
    type: 'FeatureCollection' as const,
    features: [{
      type: 'Feature' as const,
      properties: {},
      geometry: {
        type: 'Polygon' as const,
        coordinates: [coords],
      },
    }],
  };
}

function retailersToGeoJSON(retailers: ZoneRetailer[]) {
  return {
    type: 'FeatureCollection' as const,
    features: retailers.map(r => ({
      type: 'Feature' as const,
      properties: {
        id: r.retailer_id,
        name: r.name,
        shop_name: r.shop_name,
        orders_last_7: r.orders_last_7,
        distance_km: r.distance_km,
      },
      geometry: {
        type: 'Point' as const,
        coordinates: [r.longitude, r.latitude],
      },
    })),
  };
}

/* ── Component ─────────────────────────────────────────────────────────── */

export default function CoverageMap({
  latitude,
  longitude,
  radiusKm,
  addressText,
  excludeWarehouseId,
  onLocationChange,
}: CoverageMapProps) {
  const mapRef = useRef<MapRef>(null);
  const [preview, setPreview] = useState<ZonePreviewData | null>(null);
  const [loading, setLoading] = useState(false);
  const [hoveredRetailer, setHoveredRetailer] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const lat = latitude || TASHKENT.latitude;
  const lng = longitude || TASHKENT.longitude;

  /* ── Fetch zone preview (debounced) ──────────────────────────────── */
  const fetchPreview = useCallback(async (lt: number, ln: number, r: number) => {
    if (lt === 0 && ln === 0) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({
        lat: lt.toFixed(6),
        lng: ln.toFixed(6),
        radius_km: r.toString(),
      });
      if (excludeWarehouseId) params.set('exclude_warehouse_id', excludeWarehouseId);

      const res = await apiFetch(`/v1/supplier/zone-preview?${params}`);
      if (res.ok) {
        const data = await res.json();
        setPreview(data);
      }
    } catch (e) {
      console.error('[CoverageMap] preview fetch error:', e);
    } finally {
      setLoading(false);
    }
  }, [excludeWarehouseId]);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => fetchPreview(lat, lng, radiusKm), 400);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [lat, lng, radiusKm, fetchPreview]);

  /* ── Map click handler ───────────────────────────────────────────── */
  const handleMapClick = useCallback((e: MapLayerMouseEvent) => {
    onLocationChange(e.lngLat.lat, e.lngLat.lng, addressText);
  }, [addressText, onLocationChange]);

  /* ── Share location ──────────────────────────────────────────────── */
  const handleShareLocation = useCallback(() => {
    if (!navigator.geolocation) return;
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        onLocationChange(pos.coords.latitude, pos.coords.longitude, addressText);
        mapRef.current?.flyTo({ center: [pos.coords.longitude, pos.coords.latitude], zoom: 13, duration: 800 });
        setLocating(false);
      },
      () => setLocating(false),
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, [addressText, onLocationChange]);

  /* ── GeoJSON sources ─────────────────────────────────────────────── */
  const circleGeoJSON = useMemo(
    () => createCircleGeoJSON(lat, lng, radiusKm),
    [lat, lng, radiusKm]
  );

  const retailerGeoJSON = useMemo(
    () => retailersToGeoJSON(preview?.retailers_in_zone ?? []),
    [preview]
  );

  const hasLocation = latitude !== 0 || longitude !== 0;

  return (
    <div className="space-y-3">
      {/* Map */}
      <div className="relative rounded-2xl overflow-hidden" style={{ height: 320, border: '2px solid var(--border)' }}>
        <MapGL
          ref={mapRef}
          initialViewState={{ latitude: lat, longitude: lng, zoom: 12 }}
          style={{ width: '100%', height: '100%' }}
          mapStyle={MAP_STYLE}
          onClick={handleMapClick}
          interactiveLayerIds={['retailer-dots']}
          onMouseEnter={(e) => {
            if (e.features?.[0]?.properties?.name) {
              setHoveredRetailer(e.features[0].properties.name as string);
            }
          }}
          onMouseLeave={() => setHoveredRetailer(null)}
        >
          <NavigationControl position="top-right" showCompass={false} />

          {/* Coverage circle fill */}
          {hasLocation && (
            <Source id="coverage-circle" type="geojson" data={circleGeoJSON}>
              <Layer
                id="coverage-fill"
                type="fill"
                paint={{
                  'fill-color': '#6750a4',
                  'fill-opacity': 0.12,
                }}
              />
              <Layer
                id="coverage-outline"
                type="line"
                paint={{
                  'line-color': '#6750a4',
                  'line-width': 2,
                  'line-dasharray': [4, 2],
                }}
              />
            </Source>
          )}

          {/* Retailer dots */}
          <Source id="retailer-points" type="geojson" data={retailerGeoJSON}>
            <Layer
              id="retailer-dots"
              type="circle"
              paint={{
                'circle-radius': 5,
                'circle-color': '#4caf50',
                'circle-opacity': 0.85,
                'circle-stroke-width': 1,
                'circle-stroke-color': '#ffffff',
              }}
            />
          </Source>

          {/* Warehouse pin */}
          {hasLocation && (
            <Marker latitude={lat} longitude={lng} anchor="bottom">
              <svg width="36" height="36" viewBox="0 0 24 24" fill="#6750a4">
                <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
              </svg>
            </Marker>
          )}
        </MapGL>

        {/* Share location button */}
        <button
          type="button"
          onClick={handleShareLocation}
          disabled={locating}
          className="absolute bottom-3 right-3 flex items-center gap-2 px-3 py-2 rounded-full text-xs font-semibold transition-all"
          style={{
            background: 'var(--background)',
            color: 'var(--foreground)',
            border: '1px solid var(--border)',
            boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
          }}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 8c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm8.94 3c-.46-4.17-3.77-7.48-7.94-7.94V1h-2v2.06C6.83 3.52 3.52 6.83 3.06 11H1v2h2.06c.46 4.17 3.77 7.48 7.94 7.94V23h2v-2.06c4.17-.46 7.48-3.77 7.94-7.94H23v-2h-2.06zM12 19c-3.87 0-7-3.13-7-7s3.13-7 7-7 7 3.13 7 7-3.13 7-7 7z" />
          </svg>
          {locating ? 'Locating…' : 'My Location'}
        </button>

        {/* Hovered retailer tooltip */}
        {hoveredRetailer && (
          <div
            className="absolute top-3 left-3 px-3 py-1.5 rounded-lg md-typescale-label-small"
            style={{
              background: 'var(--background)',
              color: 'var(--foreground)',
              border: '1px solid var(--border)',
              boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
            }}
          >
            {hoveredRetailer}
          </div>
        )}

        {/* Loading indicator */}
        {loading && (
          <div className="absolute top-3 right-14 px-2 py-1 rounded-full" style={{ background: 'var(--background)', border: '1px solid var(--border)' }}>
            <div className="w-4 h-4 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
          </div>
        )}
      </div>

      {/* Coordinates */}
      {hasLocation && (
        <p className="md-typescale-body-small text-center" style={{ color: 'var(--muted)' }}>
          {lat.toFixed(5)}, {lng.toFixed(5)}
        </p>
      )}

      {/* ── Density Stats Panel ──────────────────────────────────────── */}
      {hasLocation && preview && (
        <div
          className="rounded-xl p-3 space-y-2"
          style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
        >
          {/* Stats row */}
          <div className="grid grid-cols-3 gap-2 text-center">
            <div>
              <p className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
                {preview.retailer_count}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                Retailers
              </p>
            </div>
            <div>
              <p className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
                ~{preview.recent_orders_day}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                Orders/Day
              </p>
            </div>
            <div>
              <p
                className="md-typescale-headline-small"
                style={{ color: preview.overlap_warnings.length > 0 ? 'var(--warning, #fb8c00)' : 'var(--success)' }}
              >
                {preview.overlap_warnings.length}
              </p>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                Overlaps
              </p>
            </div>
          </div>

          {/* Overlap warnings */}
          {preview.overlap_warnings.length > 0 && (
            <div className="space-y-1 pt-1" style={{ borderTop: '1px solid var(--border)' }}>
              {preview.overlap_warnings.map(ow => (
                <div
                  key={ow.warehouse_id}
                  className="flex items-center gap-2 px-2 py-1.5 rounded-lg md-typescale-body-small"
                  style={{ background: 'color-mix(in srgb, var(--warning, #fb8c00) 8%, transparent)' }}
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="var(--warning, #fb8c00)" className="shrink-0">
                    <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" />
                  </svg>
                  <span style={{ color: 'var(--foreground)' }}>
                    Overlaps <strong>{ow.warehouse_name}</strong> — {ow.overlap_retailers} shared retailers, {ow.distance_km.toFixed(1)} km apart
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
