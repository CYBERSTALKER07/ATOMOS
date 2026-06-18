const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

export type ResolvedLocation = {
  address: string;
  lat: number;
  lng: number;
  place_id?: string;
};

export async function autocompleteAddress(input: string) {
  const q = encodeURIComponent(input.trim());
  if (!q) return [];
  const res = await fetch(`${API_BASE}/v1/platform/geocode/autocomplete?input=${q}`);
  if (!res.ok) return [];
  const data = (await res.json()) as { predictions?: { place_id: string; description: string }[] };
  return data.predictions ?? [];
}

export async function resolvePlace(placeId: string): Promise<ResolvedLocation | null> {
  const res = await fetch(`${API_BASE}/v1/platform/geocode/place?place_id=${encodeURIComponent(placeId)}`);
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
}

export async function reverseGeocode(lat: number, lng: number): Promise<ResolvedLocation | null> {
  const res = await fetch(
    `${API_BASE}/v1/platform/geocode/reverse?lat=${encodeURIComponent(String(lat))}&lng=${encodeURIComponent(String(lng))}`,
  );
  if (!res.ok) return null;
  return (await res.json()) as ResolvedLocation;
}
