const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
  formatted_address?: string;
};

export type AutocompletePrediction = {
  place_id: string;
  description: string;
};

const GEOCODE_CACHE_TTL_MS = 60_000;
const autocompleteCache = new Map<string, { expiresAt: number; data: AutocompletePrediction[] }>();
const placeCache = new Map<string, { expiresAt: number; data: ResolvedLocation }>();
const reverseCache = new Map<string, { expiresAt: number; data: ResolvedLocation }>();
let activeAutocompleteController: AbortController | null = null;

const DEFAULT_PREDICTIONS: AutocompletePrediction[] = [
  { place_id: "tash-chilonzor", description: "Chilonzor District, Tashkent" },
  { place_id: "tash-yunusobod", description: "Yunusobod District, Tashkent" },
  { place_id: "tash-mirzo-ulugbek", description: "Mirzo Ulugbek District, Tashkent" },
  { place_id: "tash-sergeli", description: "Sergeli Industrial Zone, Tashkent" },
  { place_id: "tash-temur", description: "Amir Temur Avenue, Tashkent" },
];

export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]> {
  const trimmed = input.trim();
  if (!trimmed) return [];
  const cacheKey = trimmed.toLowerCase();
  const cached = autocompleteCache.get(cacheKey);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.data;
  }

  activeAutocompleteController?.abort();
  const controller = new AbortController();
  activeAutocompleteController = controller;

  const q = encodeURIComponent(trimmed);
  try {
    const res = await fetch(`${API_BASE}/v1/platform/geocode/autocomplete?input=${q}`, {
      signal: controller.signal,
    });
    if (res.ok) {
      const data = (await res.json()) as { predictions?: AutocompletePrediction[] };
      const predictions = data.predictions ?? [];
      if (predictions.length > 0) {
        autocompleteCache.set(cacheKey, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: predictions });
        return predictions;
      }
    }
  } catch {
    // Offline predictions fallback
  } finally {
    if (activeAutocompleteController === controller) {
      activeAutocompleteController = null;
    }
  }

  const matches = DEFAULT_PREDICTIONS.filter((p) => p.description.toLowerCase().includes(trimmed.toLowerCase()));
  return matches.length > 0 ? matches : [{ place_id: `custom-${trimmed}`, description: `${trimmed}, Tashkent` }];
}

export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null> {
  const id = placeId.trim();
  if (!id) return null;
  const cached = placeCache.get(id);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.data;
  }
  try {
    const res = await fetch(`${API_BASE}/v1/platform/geocode/place?place_id=${encodeURIComponent(id)}`);
    if (res.ok) {
      const resolved = (await res.json()) as ResolvedLocation;
      placeCache.set(id, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: resolved });
      return resolved;
    }
  } catch {
    // Offline resolution fallback
  }

  if (id.includes("chilonzor")) {
    return { address: "Chilonzor District, Tashkent", lat: 41.2721, lng: 69.2045, place_id: id };
  } else if (id.includes("yunusobod")) {
    return { address: "Yunusobod District, Tashkent", lat: 41.3456, lng: 69.2891, place_id: id };
  } else if (id.includes("sergeli")) {
    return { address: "Sergeli Industrial Zone, Tashkent", lat: 41.2234, lng: 69.2178, place_id: id };
  }
  return { address: "Amir Temur Avenue, Tashkent", lat: 41.2995, lng: 69.2401, place_id: id };
}

export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null> {
  const coordKey = `${lat.toFixed(5)}:${lng.toFixed(5)}`;
  const cached = reverseCache.get(coordKey);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.data;
  }
  try {
    const res = await fetch(
      `${API_BASE}/v1/platform/geocode/reverse?lat=${encodeURIComponent(String(lat))}&lng=${encodeURIComponent(String(lng))}`,
    );
    if (res.ok) {
      const resolved = (await res.json()) as ResolvedLocation;
      reverseCache.set(coordKey, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: resolved });
      return resolved;
    }
  } catch {
    // Offline reverse fallback
  }

  if (Math.abs(lat - 41.2995) < 0.25 && Math.abs(lng - 69.2401) < 0.25) {
    return { address: "Amir Temur Ave, Tashkent, Uzbekistan", lat, lng };
  }
  return { address: `Pinned location (${lat.toFixed(4)}, ${lng.toFixed(4)})`, lat, lng };
}

export async function forwardGeocode(address: string): Promise<ResolvedLocation | null> {
  const trimmed = address.trim();
  if (!trimmed) return null;
  try {
    const res = await fetch(`${API_BASE}/v1/platform/geocode/forward`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ address: trimmed }),
    });
    if (res.ok) return (await res.json()) as ResolvedLocation;
  } catch {
    // Offline forward fallback
  }
  return { address: trimmed || "Amir Temur Ave, Tashkent", lat: 41.2995, lng: 69.2401 };
}

export function hasValidCoordinates(lat: string, lng: string): boolean {
  const latN = Number.parseFloat(lat);
  const lngN = Number.parseFloat(lng);
  if (!Number.isFinite(latN) || !Number.isFinite(lngN)) return false;
  if (latN === 0 && lngN === 0) return false;
  return latN >= -90 && latN <= 90 && lngN >= -180 && lngN <= 180;
}

export function locationFromResolved(loc: ResolvedLocation, fallbackAddress = ""): {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
} {
  return {
    address: loc.address || loc.formatted_address || fallbackAddress,
    lat: String(loc.lat),
    lng: String(loc.lng),
    place_id: loc.place_id,
  };
}
