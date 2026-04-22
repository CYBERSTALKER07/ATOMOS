'use client';

import { useState, useRef, useCallback, useEffect } from 'react';
import { reverseGeocode, forwardGeocode } from '../../lib/google-maps';

/**
 * Interactive location picker with map + address search + GPS detect.
 * Uses MapLibre GL (already in the project) for the map, and Google Maps
 * Geocoding API for reverse/forward geocoding.
 */

interface LocationPickerProps {
  lat: string;
  lng: string;
  address: string;
  onLocationChange: (lat: string, lng: string, address: string) => void;
}

export default function LocationPicker({ lat, lng, address, onLocationChange }: LocationPickerProps) {
  const mapContainerRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const markerRef = useRef<maplibregl.Marker | null>(null);

  const [locating, setLocating] = useState(false);
  const [searching, setSearching] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [locErr, setLocErr] = useState('');

  const currentLat = lat ? parseFloat(lat) : 41.2995;
  const currentLng = lng ? parseFloat(lng) : 69.2401;

  // Update marker position and reverse geocode
  const updatePosition = useCallback(async (newLat: number, newLng: number) => {
    onLocationChange(String(newLat), String(newLng), address);
    try {
      const result = await reverseGeocode(newLat, newLng);
      onLocationChange(String(newLat), String(newLng), result.address);
    } catch {
      onLocationChange(String(newLat), String(newLng), `${newLat.toFixed(6)}, ${newLng.toFixed(6)}`);
    }
  }, [address, onLocationChange]);

  // Initialize map
  useEffect(() => {
    if (!mapContainerRef.current || mapRef.current) return;

    let cancelled = false;

    import('maplibre-gl').then((maplibregl) => {
      if (cancelled || !mapContainerRef.current) return;

      // Use a free tile source — OpenStreetMap raster tiles
      const map = new maplibregl.default.Map({
        container: mapContainerRef.current,
        style: {
          version: 8,
          sources: {
            osm: {
              type: 'raster',
              tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
              tileSize: 256,
              attribution: '&copy; OpenStreetMap contributors',
            },
          },
          layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
        },
        center: [currentLng, currentLat],
        zoom: lat ? 14 : 10,
      });

      map.addControl(new maplibregl.default.NavigationControl(), 'top-right');

      // Create draggable marker
      const marker = new maplibregl.default.Marker({ draggable: true, color: '#4F46E5' })
        .setLngLat([currentLng, currentLat])
        .addTo(map);

      marker.on('dragend', () => {
        const lngLat = marker.getLngLat();
        updatePosition(lngLat.lat, lngLat.lng);
      });

      // Click on map to move marker
      map.on('click', (e: maplibregl.MapMouseEvent) => {
        marker.setLngLat(e.lngLat);
        updatePosition(e.lngLat.lat, e.lngLat.lng);
      });

      mapRef.current = map;
      markerRef.current = marker;
    });

    return () => {
      cancelled = true;
      mapRef.current?.remove();
      mapRef.current = null;
      markerRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync marker when lat/lng change externally
  useEffect(() => {
    if (markerRef.current && lat && lng) {
      const newLat = parseFloat(lat);
      const newLng = parseFloat(lng);
      const current = markerRef.current.getLngLat();
      if (Math.abs(current.lat - newLat) > 0.0001 || Math.abs(current.lng - newLng) > 0.0001) {
        markerRef.current.setLngLat([newLng, newLat]);
        mapRef.current?.flyTo({ center: [newLng, newLat], zoom: 14, duration: 800 });
      }
    }
  }, [lat, lng]);

  // GPS detect
  const detectLocation = () => {
    if (!navigator.geolocation) {
      setLocErr('Geolocation not supported by your browser.');
      return;
    }
    setLocating(true);
    setLocErr('');
    navigator.geolocation.getCurrentPosition(
      pos => {
        const { latitude, longitude } = pos.coords;
        updatePosition(latitude, longitude);
        markerRef.current?.setLngLat([longitude, latitude]);
        mapRef.current?.flyTo({ center: [longitude, latitude], zoom: 14, duration: 800 });
        setLocating(false);
      },
      err => {
        setLocating(false);
        setLocErr(err.message || 'Could not get your location.');
      },
      { enableHighAccuracy: true, timeout: 10000 }
    );
  };

  // Address search
  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setSearching(true);
    try {
      const result = await forwardGeocode(searchQuery);
      if (result) {
        onLocationChange(String(result.lat), String(result.lng), result.address);
        markerRef.current?.setLngLat([result.lng, result.lat]);
        mapRef.current?.flyTo({ center: [result.lng, result.lat], zoom: 14, duration: 800 });
      }
    } catch {
      // Silently fail — user can try again
    } finally {
      setSearching(false);
    }
  };

  return (
    <div className="space-y-3">
      {/* Search bar */}
      <div className="flex gap-2">
        <input
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleSearch()}
          placeholder="Search address..."
          className="md-input-outlined flex-1"
        />
        <button
          type="button"
          onClick={handleSearch}
          disabled={searching || !searchQuery.trim()}
          className="md-btn md-btn-tonal px-4 flex items-center gap-1.5 shrink-0"
        >
          {searching ? (
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" /><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>
          )}
          Search
        </button>
        <button
          type="button"
          onClick={detectLocation}
          disabled={locating}
          className="md-btn md-btn-tonal px-4 flex items-center gap-1.5 shrink-0"
          title="Use my current GPS location"
        >
          {locating ? (
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none"><circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" /><path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 8c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm8.94 3c-.46-4.17-3.77-7.48-7.94-7.94V1h-2v2.06C6.83 3.52 3.52 6.83 3.06 11H1v2h2.06c.46 4.17 3.77 7.48 7.94 7.94V23h2v-2.06c4.17-.46 7.48-3.77 7.94-7.94H23v-2h-2.06zM12 19c-3.87 0-7-3.13-7-7s3.13-7 7-7 7 3.13 7 7-3.13 7-7 7z"/></svg>
          )}
          GPS
        </button>
      </div>

      {locErr && <p className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>{locErr}</p>}

      {/* Map */}
      <div
        ref={mapContainerRef}
        className="w-full md-shape-md overflow-hidden"
        style={{ height: 280, border: '1px solid var(--border)' }}
      />

      {/* Resolved address */}
      {address && (
        <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" className="inline mr-1" style={{ verticalAlign: '-2px' }}>
            <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/>
          </svg>
          {address}
        </p>
      )}

      {/* Coordinates */}
      {lat && lng && (
        <p className="md-typescale-label-small" style={{ color: 'var(--accent)' }}>
          Coordinates captured: {parseFloat(lat).toFixed(6)}, {parseFloat(lng).toFixed(6)}
        </p>
      )}

      {/* Manual coordinate inputs */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>Latitude</label>
          <input
            type="number"
            step="any"
            value={lat}
            onChange={e => {
              const v = e.target.value;
              onLocationChange(v, lng, address);
            }}
            placeholder="41.2995"
            className="md-input-outlined w-full"
          />
        </div>
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--foreground)' }}>Longitude</label>
          <input
            type="number"
            step="any"
            value={lng}
            onChange={e => {
              const v = e.target.value;
              onLocationChange(lat, v, address);
            }}
            placeholder="69.2401"
            className="md-input-outlined w-full"
          />
        </div>
      </div>
    </div>
  );
}
