'use client';

import { useState, useCallback, useRef } from 'react';
import MapGL, { Marker, NavigationControl, type MapRef } from 'react-map-gl/maplibre';
import type { MapLayerMouseEvent } from 'react-map-gl/maplibre';
import 'maplibre-gl/dist/maplibre-gl.css';

const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';
const TASHKENT = { latitude: 41.2995, longitude: 69.2401 };

interface MapLocationPickerProps {
  latitude: number;
  longitude: number;
  addressText: string;
  onChange: (lat: number, lng: number, address: string) => void;
}

export default function MapLocationPicker({ latitude, longitude, addressText, onChange }: MapLocationPickerProps) {
  const [marker, setMarker] = useState<{ lat: number; lng: number }>({
    lat: latitude || TASHKENT.latitude,
    lng: longitude || TASHKENT.longitude,
  });
  const [locating, setLocating] = useState(false);
  const mapRef = useRef<MapRef>(null);

  const handleMapClick = useCallback((e: MapLayerMouseEvent) => {
    const { lng, lat } = e.lngLat;
    setMarker({ lat, lng });
    onChange(lat, lng, addressText);
  }, [addressText, onChange]);

  const handleShareLocation = useCallback(() => {
    if (!navigator.geolocation) return;
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const { latitude: lat, longitude: lng } = pos.coords;
        setMarker({ lat, lng });
        onChange(lat, lng, addressText);
        mapRef.current?.flyTo({ center: [lng, lat], zoom: 16, duration: 800 });
        setLocating(false);
      },
      () => setLocating(false),
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, [addressText, onChange]);

  const handleAddressChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(marker.lat, marker.lng, e.target.value);
  }, [marker, onChange]);

  return (
    <div className="space-y-3">
      {/* Map */}
      <div className="relative rounded-2xl overflow-hidden" style={{ height: 240, border: '2px solid var(--border)' }}>
        <MapGL
          ref={mapRef}
          initialViewState={{
            latitude: marker.lat,
            longitude: marker.lng,
            zoom: 14,
          }}
          style={{ width: '100%', height: '100%' }}
          mapStyle={MAP_STYLE}
          onClick={handleMapClick}
        >
          <NavigationControl position="top-right" showCompass={false} />
          <Marker latitude={marker.lat} longitude={marker.lng} anchor="bottom">
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
              <svg width="32" height="32" viewBox="0 0 24 24" fill="var(--accent)">
                <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
              </svg>
            </div>
          </Marker>
        </MapGL>

        {/* Share Location button */}
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
          {locating ? 'Locating...' : 'My Location'}
        </button>
      </div>

      {/* Coordinates display */}
      {(marker.lat !== TASHKENT.latitude || marker.lng !== TASHKENT.longitude) && (
        <p className="md-typescale-body-small text-center" style={{ color: 'var(--muted)' }}>
          {marker.lat.toFixed(5)}, {marker.lng.toFixed(5)}
        </p>
      )}

      {/* Address text input */}
      <div>
        <label className="md-typescale-body-medium block mb-2 font-medium" style={{ color: 'var(--foreground)' }}>
          Warehouse Address
        </label>
        <input
          type="text"
          value={addressText}
          onChange={handleAddressChange}
          placeholder="Street address, city"
          className="w-full px-5 py-4 rounded-2xl md-typescale-body-large outline-none transition-all"
          style={{
            background: 'var(--surface)',
            color: 'var(--foreground)',
            border: '2px solid var(--border)',
            fontSize: '16px',
          }}
          onFocus={e => e.target.style.borderColor = 'var(--accent)'}
          onBlur={e => e.target.style.borderColor = 'var(--border)'}
        />
      </div>

      <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
        Click on the map to place your warehouse pin, or use &quot;My Location&quot; to auto-detect.
      </p>
    </div>
  );
}
