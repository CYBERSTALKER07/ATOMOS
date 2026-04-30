/**
 * Google Maps Geocoding & Places API utilities.
 *
 * Uses the REST API directly (no additional npm package required).
 * The API key must be exposed as NEXT_PUBLIC_GOOGLE_MAPS_KEY.
 *
 * Fallback: If no Google Maps key is configured, falls back to Nominatim (OSM).
 */

const GOOGLE_KEY = process.env.NEXT_PUBLIC_GOOGLE_MAPS_KEY || '';

interface GeocodingResult {
  address: string;
  lat: number;
  lng: number;
  placeId?: string;
}

/**
 * Reverse geocode coordinates to a human-readable address.
 * Uses Google Maps Geocoding API if key is configured, otherwise falls back to Nominatim.
 */
export async function reverseGeocode(lat: number, lng: number): Promise<GeocodingResult> {
  if (GOOGLE_KEY) {
    return reverseGeocodeGoogle(lat, lng);
  }
  return reverseGeocodeNominatim(lat, lng);
}

async function reverseGeocodeGoogle(lat: number, lng: number): Promise<GeocodingResult> {
  const url = `https://maps.googleapis.com/maps/api/geocode/json?latlng=${lat},${lng}&key=${encodeURIComponent(GOOGLE_KEY)}`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Google Geocoding API error: ${res.status}`);
  const data = await res.json();

  if (data.status === 'OK' && data.results?.length > 0) {
    const best = data.results[0];
    return {
      address: best.formatted_address,
      lat,
      lng,
      placeId: best.place_id,
    };
  }

  return { address: `${lat.toFixed(6)}, ${lng.toFixed(6)}`, lat, lng };
}

async function reverseGeocodeNominatim(lat: number, lng: number): Promise<GeocodingResult> {
  const url = `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=json`;
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Nominatim error: ${res.status}`);
  const data = await res.json();

  return {
    address: data.display_name || `${lat.toFixed(6)}, ${lng.toFixed(6)}`,
    lat,
    lng,
  };
}

/**
 * Forward geocode an address string to coordinates.
 * Uses Google Maps Geocoding API if key is configured, otherwise Nominatim.
 */
export async function forwardGeocode(address: string): Promise<GeocodingResult | null> {
  if (!address.trim()) return null;

  if (GOOGLE_KEY) {
    const url = `https://maps.googleapis.com/maps/api/geocode/json?address=${encodeURIComponent(address)}&key=${encodeURIComponent(GOOGLE_KEY)}`;
    const res = await fetch(url);
    if (!res.ok) return null;
    const data = await res.json();
    if (data.status === 'OK' && data.results?.length > 0) {
      const best = data.results[0];
      const loc = best.geometry.location;
      return {
        address: best.formatted_address,
        lat: loc.lat,
        lng: loc.lng,
        placeId: best.place_id,
      };
    }
    return null;
  }

  // Nominatim fallback
  const url = `https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(address)}&format=json&limit=1`;
  const res = await fetch(url);
  if (!res.ok) return null;
  const data = await res.json();
  if (data.length > 0) {
    return {
      address: data[0].display_name,
      lat: parseFloat(data[0].lat),
      lng: parseFloat(data[0].lon),
    };
  }
  return null;
}

/**
 * Check if Google Maps API key is configured.
 */
export function isGoogleMapsConfigured(): boolean {
  return GOOGLE_KEY.length > 0;
}
