'use client';

import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import MapGL, { Marker, Source, Layer, NavigationControl, type MapRef } from 'react-map-gl/maplibre';
import type { MapLayerMouseEvent } from 'react-map-gl/maplibre';
import 'maplibre-gl/dist/maplibre-gl.css';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';
const TASHKENT = { latitude: 41.2995, longitude: 69.2401 };

// ── Types ─────────────────────────────────────────────────────────────────────

interface FactoryNode {
  factory_id: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  is_active: boolean;
}

interface WarehouseNode {
  warehouse_id: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  primary_factory_id?: string;
  is_active: boolean;
}

interface RetailerNode {
  id: string;
  name: string;
  shop_name: string;
  lat: number;
  lng: number;
}

interface FactoryNetworkMapProps {
  factories: FactoryNode[];
  onFactoryClick?: (f: FactoryNode) => void;
}

// ── Component ─────────────────────────────────────────────────────────────────

export default function FactoryNetworkMap({ factories, onFactoryClick }: FactoryNetworkMapProps) {
  const mapRef = useRef<MapRef>(null);
  const [warehouses, setWarehouses] = useState<WarehouseNode[]>([]);
  const [retailers, setRetailers] = useState<RetailerNode[]>([]);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [selectedFactoryId, setSelectedFactoryId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Fetch warehouses + retailers on mount
  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const [whRes, rtRes] = await Promise.all([
          apiFetch('/v1/supplier/warehouses'),
          apiFetch('/v1/supplier/retailers/locations'),
        ]);
        if (cancelled) return;
        if (whRes.ok) {
          const whData = await whRes.json();
          setWarehouses((whData.warehouses ?? []).map((w: Record<string, unknown>) => ({
            warehouse_id: w.warehouse_id,
            name: w.name,
            address: w.address ?? '',
            lat: w.lat ?? 0,
            lng: w.lng ?? 0,
            primary_factory_id: w.primary_factory_id ?? '',
            is_active: w.is_active ?? true,
          })));
        }
        if (rtRes.ok) {
          const rtData = await rtRes.json();
          setRetailers(rtData.retailers ?? []);
        }
      } catch {
        // Non-blocking — map still renders with available data
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, []);

  // Connection lines: factory ↔ assigned warehouses (GeoJSON LineString)
  const connectionLines = useMemo(() => {
    const features: GeoJSON.Feature[] = [];
    for (const wh of warehouses) {
      if (!wh.primary_factory_id) continue;
      const factory = factories.find(f => f.factory_id === wh.primary_factory_id);
      if (!factory) continue;
      features.push({
        type: 'Feature',
        properties: {
          factory_id: factory.factory_id,
          warehouse_id: wh.warehouse_id,
          selected: selectedFactoryId === factory.factory_id ? 'yes' : 'no',
        },
        geometry: {
          type: 'LineString',
          coordinates: [
            [factory.lng, factory.lat],
            [wh.lng, wh.lat],
          ],
        },
      });
    }
    return { type: 'FeatureCollection' as const, features };
  }, [factories, warehouses, selectedFactoryId]);

  // Hoverable tooltip
  const hoveredEntity = useMemo(() => {
    if (!hoveredId) return null;
    const f = factories.find(x => x.factory_id === hoveredId);
    if (f) return { type: 'Factory', name: f.name, address: f.address, lat: f.lat, lng: f.lng };
    const w = warehouses.find(x => x.warehouse_id === hoveredId);
    if (w) return { type: 'Warehouse', name: w.name, address: w.address, lat: w.lat, lng: w.lng };
    const r = retailers.find(x => x.id === hoveredId);
    if (r) return { type: 'Retailer', name: r.name || r.shop_name, address: r.shop_name, lat: r.lat, lng: r.lng };
    return null;
  }, [hoveredId, factories, warehouses, retailers]);

  // Center map on content
  const initialView = useMemo(() => {
    const allPoints = [
      ...factories.map(f => ({ lat: f.lat, lng: f.lng })),
      ...warehouses.filter(w => w.lat && w.lng).map(w => ({ lat: w.lat, lng: w.lng })),
    ];
    if (allPoints.length === 0) return { latitude: TASHKENT.latitude, longitude: TASHKENT.longitude, zoom: 10 };
    const avgLat = allPoints.reduce((s, p) => s + p.lat, 0) / allPoints.length;
    const avgLng = allPoints.reduce((s, p) => s + p.lng, 0) / allPoints.length;
    return { latitude: avgLat, longitude: avgLng, zoom: allPoints.length === 1 ? 12 : 9 };
  }, [factories, warehouses]);

  const handleFactoryClick = useCallback((f: FactoryNode) => {
    setSelectedFactoryId(prev => (prev === f.factory_id ? null : f.factory_id));
    onFactoryClick?.(f);
  }, [onFactoryClick]);

  const handleLocate = useCallback(() => {
    if (!navigator.geolocation) return;
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        mapRef.current?.flyTo({ center: [pos.coords.longitude, pos.coords.latitude], zoom: 13, duration: 800 });
      },
      () => {},
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, []);

  return (
    <div className="relative rounded-2xl overflow-hidden" style={{ height: 560, border: '2px solid var(--border)' }}>
      <MapGL
        ref={mapRef}
        initialViewState={initialView}
        style={{ width: '100%', height: '100%' }}
        mapStyle={MAP_STYLE}
      >
        <NavigationControl position="top-right" showCompass={false} />

        {/* Connection lines (factory → warehouse) */}
        <Source id="connections" type="geojson" data={connectionLines}>
          <Layer
            id="connection-lines"
            type="line"
            paint={{
              'line-color': ['case', ['==', ['get', 'selected'], 'yes'], '#ffffff', '#666666'],
              'line-width': ['case', ['==', ['get', 'selected'], 'yes'], 2.5, 1.5],
              'line-opacity': ['case', ['==', ['get', 'selected'], 'yes'], 0.9, 0.4],
              'line-dasharray': [4, 3],
            }}
          />
        </Source>

        {/* Retailer dots */}
        {retailers.map(r => (
          <Marker
            key={`r-${r.id}`}
            latitude={r.lat}
            longitude={r.lng}
            anchor="center"
          >
            <div
              onMouseEnter={() => setHoveredId(r.id)}
              onMouseLeave={() => setHoveredId(null)}
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: 'var(--warning)',
                opacity: 0.6,
                border: '1px solid rgba(0,0,0,0.2)',
                cursor: 'pointer',
              }}
            />
          </Marker>
        ))}

        {/* Warehouse markers */}
        {warehouses.filter(w => w.lat && w.lng).map(w => {
          const isLinked = selectedFactoryId ? w.primary_factory_id === selectedFactoryId : !!w.primary_factory_id;
          return (
            <Marker
              key={`w-${w.warehouse_id}`}
              latitude={w.lat}
              longitude={w.lng}
              anchor="bottom"
            >
              <div
                onMouseEnter={() => setHoveredId(w.warehouse_id)}
                onMouseLeave={() => setHoveredId(null)}
                className="flex flex-col items-center cursor-pointer"
              >
                <div
                  className="flex items-center justify-center rounded-full transition-all"
                  style={{
                    width: 28,
                    height: 28,
                    background: isLinked ? 'var(--accent)' : 'var(--surface)',
                    border: `2px solid ${isLinked ? 'var(--accent-foreground)' : 'var(--border)'}`,
                    opacity: selectedFactoryId && !isLinked ? 0.4 : 1,
                  }}
                >
                  <Icon name="warehouse" size={14} style={{ color: isLinked ? 'var(--accent-foreground)' : 'var(--muted)' }} />
                </div>
              </div>
            </Marker>
          );
        })}

        {/* Factory markers */}
        {factories.map(f => {
          const isSelected = selectedFactoryId === f.factory_id;
          return (
            <Marker
              key={`f-${f.factory_id}`}
              latitude={f.lat}
              longitude={f.lng}
              anchor="bottom"
            >
              <div
                onClick={() => handleFactoryClick(f)}
                onMouseEnter={() => setHoveredId(f.factory_id)}
                onMouseLeave={() => setHoveredId(null)}
                className="flex flex-col items-center cursor-pointer"
              >
                <div
                  className="flex items-center justify-center rounded-lg transition-all"
                  style={{
                    width: 36,
                    height: 36,
                    background: isSelected ? 'var(--foreground)' : 'var(--accent)',
                    border: isSelected ? '3px solid var(--warning)' : '2px solid var(--accent-foreground)',
                    boxShadow: isSelected ? '0 0 12px rgba(255,255,255,0.3)' : 'none',
                  }}
                >
                  <Icon name="factory" size={18} style={{ color: 'var(--accent-foreground)' }} />
                </div>
                <span
                  className="mt-1 px-1.5 py-0.5 rounded text-[10px] font-semibold whitespace-nowrap"
                  style={{ background: 'var(--background)', color: 'var(--foreground)', border: '1px solid var(--border)' }}
                >
                  {f.name}
                </span>
              </div>
            </Marker>
          );
        })}
      </MapGL>

      {/* Hover tooltip */}
      {hoveredEntity && (
        <div
          className="absolute top-4 left-4 rounded-xl px-4 py-3 z-10 md-animate-in"
          style={{ background: 'var(--surface)', border: '1px solid var(--border)', maxWidth: 260 }}
        >
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{hoveredEntity.type}</p>
          <p className="md-typescale-body-medium font-semibold">{hoveredEntity.name}</p>
          {hoveredEntity.address && (
            <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>{hoveredEntity.address}</p>
          )}
          <p className="md-typescale-body-small font-mono mt-1" style={{ color: 'var(--muted)', fontSize: 11 }}>
            {hoveredEntity.lat.toFixed(5)}, {hoveredEntity.lng.toFixed(5)}
          </p>
        </div>
      )}

      {/* Legend */}
      <div
        className="absolute bottom-4 left-4 rounded-xl px-4 py-3 z-10 flex flex-col gap-2"
        style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}
      >
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded" style={{ background: 'var(--accent)' }} />
          <span className="md-typescale-label-small">Factories ({factories.length})</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full" style={{ background: 'var(--surface)', border: '2px solid var(--border)' }} />
          <span className="md-typescale-label-small">Warehouses ({warehouses.length})</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full" style={{ background: 'var(--warning)', opacity: 0.6 }} />
          <span className="md-typescale-label-small">Retailers ({retailers.length})</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-6 border-t-2" style={{ borderColor: '#666', borderStyle: 'dashed' }} />
          <span className="md-typescale-label-small">Supply link</span>
        </div>
      </div>

      {/* My Location button */}
      <button
        type="button"
        onClick={handleLocate}
        className="absolute bottom-4 right-4 flex items-center gap-2 px-3 py-2 rounded-full text-xs font-semibold z-10"
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
        My Location
      </button>

      {/* Loading overlay */}
      {loading && (
        <div className="absolute inset-0 flex items-center justify-center z-20" style={{ background: 'rgba(0,0,0,0.4)' }}>
          <div className="px-4 py-2 rounded-lg" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium">Loading network data...</p>
          </div>
        </div>
      )}
    </div>
  );
}
