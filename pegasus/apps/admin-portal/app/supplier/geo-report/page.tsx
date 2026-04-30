'use client';

import React, { useEffect, useState, useMemo, useCallback } from 'react';
import dynamic from 'next/dynamic';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import StatsCard from '@/components/StatsCard';
import { cellToBoundary } from 'h3-js';

const MapGL = dynamic(() => import('react-map-gl/maplibre').then(m => m.default), { ssr: false });
const Source = dynamic(() => import('react-map-gl/maplibre').then(m => m.Source), { ssr: false });
const Layer = dynamic(() => import('react-map-gl/maplibre').then(m => m.Layer), { ssr: false });


const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';

/* Color palette for warehouse zones */
const ZONE_COLORS = [
  '#2196F3', '#4CAF50', '#FF9800', '#E91E63', '#9C27B0',
  '#00BCD4', '#FF5722', '#3F51B5', '#8BC34A', '#FFC107',
];

interface WarehouseZone {
  warehouse_id: string;
  warehouse_name: string;
  hex_count: number;
  hexes: string[];
  retailer_count: number;
}

interface GeoReport {
  total_warehouses: number;
  total_hexes: number;
  total_retailers_covered: number;
  unassigned_retailers: number;
  zones: WarehouseZone[];
}

function hexesToGeoJSON(hexes: string[], color: string) {
  return {
    type: 'FeatureCollection' as const,
    features: hexes.map(hex => {
      const boundary = cellToBoundary(hex, true);
      return {
        type: 'Feature' as const,
        properties: { hex, color },
        geometry: {
          type: 'Polygon' as const,
          coordinates: [boundary.concat([boundary[0]])],
        },
      };
    }),
  };
}

export default function GeoReportPage() {
  const [report, setReport] = useState<GeoReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeZoneId, setActiveZoneId] = useState<string | null>(null);
  const [panelOpen, setPanelOpen] = useState(true);

  const fetchReport = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/supplier/geo-report');
      if (!res.ok) throw new Error('Failed to load');
      const data: GeoReport = await res.json();
      setReport(data);
    } catch {
      setError('Failed to load geo report');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchReport(); }, [fetchReport]);

  // GeoJSON for each warehouse zone
  const zoneGeoJSONs = useMemo(() => {
    if (!report) return [];
    return report.zones.map((z, i) => ({
      id: z.warehouse_id,
      name: z.warehouse_name,
      color: ZONE_COLORS[i % ZONE_COLORS.length],
      retailerCount: z.retailer_count,
      hexCount: z.hex_count,
      geojson: hexesToGeoJSON(z.hexes, ZONE_COLORS[i % ZONE_COLORS.length]),
    }));
  }, [report]);

  const zoneRenderRows = useMemo(() => {
    return zoneGeoJSONs.map((zone) => ({
      ...zone,
      dimmed: activeZoneId !== null && activeZoneId !== zone.id,
      active: activeZoneId === zone.id,
    }));
  }, [zoneGeoJSONs, activeZoneId]);

  return (
    <div className="h-full flex flex-col">
      {/* Header bar */}
      <div className="flex items-center justify-between px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <div>
          <h1 className="md-typescale-headline-medium">Coverage Map</h1>
          <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            H3 hexagonal grid — warehouse coverage zones
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setPanelOpen(!panelOpen)}
            className="md-btn md-btn-outlined md-typescale-label-medium px-4 py-2 flex items-center gap-1.5"
          >
            <Icon name={panelOpen ? 'sidebar-close' : 'sidebar-open'} size={16} />
            {panelOpen ? 'Hide Panel' : 'Show Panel'}
          </button>
          <button
            onClick={fetchReport}
            className="md-btn md-btn-tonal md-typescale-label-medium px-4 py-2 flex items-center gap-1.5"
          >
            <Icon name="refresh-cw" size={14} />
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="mx-6 mt-4 flex items-center gap-2 p-3 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
          <Icon name="error" size={16} />
          <span className="md-typescale-body-small">{error}</span>
        </div>
      )}

      {/* Map + panel */}
      <div className="flex-1 flex min-h-0">
        {/* Map */}
        <div className="flex-1 relative">
          {loading ? (
            <div className="absolute inset-0 flex items-center justify-center" style={{ background: 'var(--background)' }}>
              <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>Loading coverage data...</p>
            </div>
          ) : (
            <MapGL
              initialViewState={{ longitude: 69.2401, latitude: 41.2995, zoom: 9 }}
              mapStyle={MAP_STYLE}
              style={{ width: '100%', height: '100%' }}
            >
              {zoneRenderRows.map(zone => {
                return (
                  <Source key={zone.id} id={`zone-${zone.id}`} type="geojson" data={zone.geojson}>
                    <Layer
                      id={`zone-fill-${zone.id}`}
                      type="fill"
                      paint={{ 'fill-color': zone.color, 'fill-opacity': zone.dimmed ? 0.05 : 0.25 }}
                    />
                    <Layer
                      id={`zone-line-${zone.id}`}
                      type="line"
                      paint={{ 'line-color': zone.color, 'line-width': zone.dimmed ? 0.5 : 1.5, 'line-opacity': zone.dimmed ? 0.2 : 0.7 }}
                    />
                  </Source>
                );
              })}
            </MapGL>
          )}

          {/* Legend overlay */}
          {!loading && zoneGeoJSONs.length > 0 && (
            <div className="absolute bottom-4 left-4 md-card p-3 rounded-lg space-y-1.5" style={{ background: 'color-mix(in srgb, var(--background) 90%, transparent)', backdropFilter: 'blur(8px)', maxWidth: 240 }}>
              <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Zones</p>
              {zoneRenderRows.map(z => (
                <div
                  key={z.id}
                  className="flex items-center gap-2 cursor-pointer px-1 py-0.5 rounded"
                  onMouseEnter={() => setActiveZoneId(z.id)}
                  onMouseLeave={() => setActiveZoneId(null)}
                  style={{ background: z.active ? 'var(--surface)' : 'transparent' }}
                >
                  <div className="w-3 h-3 rounded-sm shrink-0" style={{ background: z.color }} />
                  <span className="md-typescale-label-small truncate">{z.name}</span>
                  <span className="ml-auto md-typescale-label-small" style={{ color: 'var(--muted)' }}>{z.hexCount}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Stats panel */}
        {panelOpen && report && (
          <div className="w-80 shrink-0 overflow-y-auto p-4 space-y-4" style={{ borderLeft: '1px solid var(--border)', background: 'var(--background)' }}>
            {/* KPIs */}
            <div className="grid grid-cols-2 gap-3">
              <StatsCard label="Warehouses" value={String(report.total_warehouses)} sub="active" delay={0} />
              <StatsCard label="H3 Cells" value={String(report.total_hexes)} sub="claimed" delay={80} />
              <StatsCard label="Covered" value={String(report.total_retailers_covered)} sub="retailers" accent="var(--success)" delay={160} />
              <StatsCard
                label="Unassigned"
                value={String(report.unassigned_retailers)}
                sub="retailers"
                accent={report.unassigned_retailers > 0 ? 'var(--warning)' : 'var(--success)'}
                delay={240}
              />
            </div>

            {/* Zone breakdown */}
            <div>
              <p className="md-typescale-label-medium mb-3" style={{ color: 'var(--muted)' }}>Zone Breakdown</p>
              <div className="space-y-2">
                {zoneRenderRows.map((z) => (
                  <div
                    key={z.id}
                    className="md-card p-3 rounded-lg cursor-pointer transition-all"
                    style={{
                      border: `1px solid ${z.active ? z.color : 'var(--border)'}`,
                      background: z.active ? 'var(--surface)' : 'transparent',
                    }}
                    onMouseEnter={() => setActiveZoneId(z.id)}
                    onMouseLeave={() => setActiveZoneId(null)}
                  >
                    <div className="flex items-center gap-2 mb-2">
                      <div className="w-3 h-3 rounded-sm" style={{ background: z.color }} />
                      <span className="md-typescale-label-medium font-medium truncate">{z.name}</span>
                    </div>
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      <span>Hexes: <strong style={{ color: 'var(--foreground)' }}>{z.hexCount}</strong></span>
                      <span>Retailers: <strong style={{ color: 'var(--foreground)' }}>{z.retailerCount}</strong></span>
                    </div>
                  </div>
                ))}

                {report.zones.length === 0 && (
                  <div className="text-center py-8">
                    <Icon name="hexagon" size={32} className="mx-auto mb-2" style={{ color: 'var(--muted)' }} />
                    <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>No coverage zones defined yet</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
