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

export async function autocompleteAddress(input: string): Promise<AutocompletePrediction[]> {
  const q = encodeURIComponent(input.trim());
  if (!q) return [];
  const res = await fetch(`${factoryApiBaseUrl}/v1/platform/geocode/autocomplete?input=${q}`);
  if (!res.ok) return [];
  const data = (await res.json()) as { predictions?: AutocompletePrediction[] };
  return data.predictions ?? [];
}

export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null> {
  const res = await fetch(`${factoryApiBaseUrl}/v1/platform/geocode/place?place_id=${encodeURIComponent(placeId)}`);
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
}

export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null> {
  const res = await fetch(
    `${factoryApiBaseUrl}/v1/platform/geocode/reverse?lat=${encodeURIComponent(String(lat))}&lng=${encodeURIComponent(String(lng))}`,
  );
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
}

export async function forwardGeocode(address: string): Promise<ResolvedLocation | null> {
  const res = await fetch(`${factoryApiBaseUrl}/v1/platform/geocode/forward`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ address }),
  });
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
}
