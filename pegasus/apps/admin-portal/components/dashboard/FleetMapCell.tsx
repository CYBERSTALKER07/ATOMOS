'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { getAdminToken, readTokenFromCookie } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import { MapPin, Wifi, WifiOff } from 'lucide-react';

// ── Types ───────────────────────────────────────────────────────────────────

type DriverPin = {
  driver_id: string;
  name: string;
  lat: number;
  lng: number;
  route_id?: string;
  truck_status?: string;
  last_seen?: string;
};

// ── Fleet Map Cell — The Anchor (2×2) ───────────────────────────────────────
// Most vital live component. Shows real-time GPS positions of all active fleet
// members using MapLibre GL. Falls back to a canvas dot-grid when MapLibre
// fails to load (SSR, missing token, etc.).

export default function FleetMapCell() {
  const mapRef = useRef<HTMLDivElement>(null);
  const [pins, setPins] = useState<DriverPin[]>([]);
  const [isLive, setIsLive] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const mapInstanceRef = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<maplibregl.Marker[]>([]);

  // ── Fetch active fleet positions ──────────────────────────────────────

  const fetchFleet = useCallback(async (signal?: AbortSignal) => {
    try {
      const token = await getAdminToken();
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/fleet/active`, {
        headers: { Authorization: `Bearer ${token}` },
        signal,
      });
      if (res.ok) {
        const data = await res.json();
        // Fleet endpoint may return { drivers: [...] } or [...]
        const list: DriverPin[] = Array.isArray(data)
          ? data
          : data?.drivers ?? data?.fleet ?? [];
        setPins(list.filter((d: DriverPin) => d.lat && d.lng));
        setIsLive(true);
        setError(null);
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        setError('Fleet offline');
        setIsLive(false);
      }
    }
  }, []);

  // ── Polling (10s) ─────────────────────────────────────────────────────

  useEffect(() => {
    const controller = new AbortController();
    fetchFleet(controller.signal);
    const id = setInterval(() => fetchFleet(), 10_000);
    return () => {
      controller.abort();
      clearInterval(id);
    };
  }, [fetchFleet]);

  // ── WebSocket live GPS ────────────────────────────────────────────────

  useEffect(() => {
    if (isTauri()) return; // Tauri handles telemetry via bridge

    let disposed = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let activeSocket: WebSocket | null = null;
    let backoff = 2000;

    const connect = async () => {
      if (disposed) return;
      const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
      const wsBase = apiBase.replace(/^http/, 'ws');
      const token = readTokenFromCookie() || (await getAdminToken().catch(() => null));
      if (!token || disposed) return;

      const url = new URL('/ws/telemetry', wsBase);
      url.searchParams.set('token', token);
      const ws = new WebSocket(url.toString());
      activeSocket = ws;

      ws.onopen = () => { backoff = 2000; };
      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'DRIVER_LOCATION' && msg.driver_id && msg.lat && msg.lng) {
            setPins((prev) => {
              const idx = prev.findIndex((p) => p.driver_id === msg.driver_id);
              const pin: DriverPin = {
                driver_id: msg.driver_id,
                name: msg.driver_name || msg.driver_id,
                lat: msg.lat,
                lng: msg.lng,
                route_id: msg.route_id,
                truck_status: msg.status,
                last_seen: new Date().toISOString(),
              };
              if (idx >= 0) {
                const next = [...prev];
                next[idx] = pin;
                return next;
              }
              return [...prev, pin];
            });
            setIsLive(true);
          }
        } catch { /* ignore non-JSON */ }
      };
      ws.onclose = () => {
        if (!disposed) {
          setIsLive(false);
          reconnectTimer = setTimeout(connect, backoff);
          backoff = Math.min(backoff * 2, 30_000);
        }
      };
      ws.onerror = () => {
        if (!disposed) {
          setIsLive(false);
        }
      };
    };

    void connect();
    return () => {
      disposed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (activeSocket) {
        activeSocket.close();
        activeSocket = null;
      }
    };
  }, []);

  // ── MapLibre GL Rendering ─────────────────────────────────────────────

  useEffect(() => {
    if (!mapRef.current) return;

    let map: maplibregl.Map | null = null;

    const initMap = async () => {
      try {
        const maplibregl = (await import('maplibre-gl')).default;
        await import('maplibre-gl/dist/maplibre-gl.css');

        if (!mapRef.current) return;

        map = new maplibregl.Map({
          container: mapRef.current,
          style: {
            version: 8,
            sources: {
              osm: {
                type: 'raster',
                tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
                tileSize: 256,
                attribution: '© OpenStreetMap',
              },
            },
            layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
            // Dark tint via map background
            glyphs: 'https://demotiles.maplibre.org/font/{fontstack}/{range}.pbf',
          },
          center: [69.24, 41.30], // Default: Tashkent
          zoom: 11,
          attributionControl: false,
        });

        mapInstanceRef.current = map;

        map.addControl(new maplibregl.NavigationControl(), 'top-right');
      } catch {
        setError('Map unavailable');
      }
    };

    void initMap();
    return () => {
      map?.remove();
      mapInstanceRef.current = null;
    };
  }, []);

  // ── Update markers when pins change ───────────────────────────────────

  useEffect(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    const updateMarkers = async () => {
      const maplibregl = (await import('maplibre-gl')).default;

      // Clear old markers
      markersRef.current.forEach((m) => m.remove());
      markersRef.current = [];

      pins.forEach((pin) => {
        const el = document.createElement('div');
        el.style.cssText = `
          width: 12px; height: 12px; border-radius: 50%;
          background: var(--foreground, #fff);
          border: 2px solid var(--background, #000);
          cursor: pointer;
        `;
        el.title = `${pin.name}${pin.route_id ? ` · ${pin.route_id}` : ''}`;

        const marker = new maplibregl.Marker({ element: el })
          .setLngLat([pin.lng, pin.lat])
          .addTo(map);
        markersRef.current.push(marker);
      });

      // Fit bounds if we have pins
      if (pins.length > 1) {
        const bounds = new maplibregl.LngLatBounds();
        pins.forEach((p) => bounds.extend([p.lng, p.lat]));
        map.fitBounds(bounds, { padding: 40, maxZoom: 14 });
      } else if (pins.length === 1) {
        map.flyTo({ center: [pins[0].lng, pins[0].lat], zoom: 13 });
      }
    };

    void updateMarkers();
  }, [pins]);

  return (
    <div className="flex flex-col h-full w-full">
      {/* Header */}
      <div className="bento-card-header">
        <span className="bento-card-title">Fleet GPS</span>
        <div className="flex items-center gap-2">
          <span
            className="md-typescale-label-small tabular-nums"
            style={{ color: 'var(--muted)' }}
          >
            {pins.length} active
          </span>
          {isLive ? (
            <Wifi size={14} style={{ color: 'var(--success)' }} />
          ) : (
            <WifiOff size={14} style={{ color: 'var(--danger)' }} />
          )}
        </div>
      </div>

      {/* Map or fallback */}
      <div ref={mapRef} className="flex-1 min-h-0 relative" style={{ background: 'var(--color-md-surface-container)' }}>
        {error && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-2">
            <MapPin size={24} style={{ color: 'var(--muted)' }} />
            <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
              {error}
            </span>
          </div>
        )}
      </div>

      {/* Live ticker strip */}
      {pins.length > 0 && (
        <div
          className="flex gap-3 px-1 py-2 overflow-x-auto"
          style={{ borderTop: '1px solid var(--border)' }}
        >
          {pins.slice(0, 5).map((pin) => (
            <div
              key={pin.driver_id}
              className="flex items-center gap-1.5 shrink-0"
            >
              <div
                className="w-1.5 h-1.5 rounded-full"
                style={{
                  background:
                    pin.truck_status === 'IN_TRANSIT'
                      ? 'var(--success)'
                      : 'var(--muted)',
                }}
              />
                <span className="md-typescale-label-small truncate max-w-20">
                {pin.name}
              </span>
            </div>
          ))}
          {pins.length > 5 && (
            <span className="md-typescale-label-small shrink-0" style={{ color: 'var(--muted)' }}>
              +{pins.length - 5}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
