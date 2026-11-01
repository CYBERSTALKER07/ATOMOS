import { factoryApiBaseUrl } from "@/lib/auth";

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
    const res = await fetch(`${factoryApiBaseUrl()}/v1/platform/geocode/autocomplete?input=${q}`, {
      signal: controller.signal,
    });
    if (!res.ok) return [];
    const data = (await res.json()) as { predictions?: AutocompletePrediction[] };
    const predictions = data.predictions ?? [];
    autocompleteCache.set(cacheKey, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: predictions });
    return predictions;
  } catch {
    return [];
  } finally {
    if (activeAutocompleteController === controller) {
      activeAutocompleteController = null;
    }
  }
}

export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null> {
  const id = placeId.trim();
  if (!id) return null;
  const cached = placeCache.get(id);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.data;
  }
  const res = await fetch(`${factoryApiBaseUrl()}/v1/platform/geocode/place?place_id=${encodeURIComponent(id)}`);
  if (!res.ok) return null;
  const resolved = (await res.json()) as ResolvedLocation;
  placeCache.set(id, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: resolved });
  return resolved;
}

export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null> {
  const coordKey = `${lat.toFixed(5)}:${lng.toFixed(5)}`;
  const cached = reverseCache.get(coordKey);
  if (cached && cached.expiresAt > Date.now()) {
    return cached.data;
  }
  const res = await fetch(
    `${factoryApiBaseUrl()}/v1/platform/geocode/reverse?lat=${encodeURIComponent(String(lat))}&lng=${encodeURIComponent(String(lng))}`,
  );
  if (!res.ok) return null;
  const resolved = (await res.json()) as ResolvedLocation;
  reverseCache.set(coordKey, { expiresAt: Date.now() + GEOCODE_CACHE_TTL_MS, data: resolved });
  return resolved;
}

export async function forwardGeocode(address: string): Promise<ResolvedLocation | null> {
  const trimmed = address.trim();
  if (!trimmed) return null;
  const res = await fetch(`${factoryApiBaseUrl()}/v1/platform/geocode/forward`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ address: trimmed }),
  });
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
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
