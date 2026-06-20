"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
  hasValidCoordinates,
  locationFromResolved,
  resolvePlace,
  reverseGeocode,
  type ResolvedLocation,
} from "@/lib/geocode";

export type LocationValue = {
  address: string;
  lat: string;
  lng: string;
  place_id?: string;
};

type LocationPickerProps = {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  label?: string;
};

export function LocationPicker({ value, onChange, label = "Address" }: LocationPickerProps) {
  const [query, setQuery] = useState(value.address);
  const [suggestions, setSuggestions] = useState<{ place_id: string; description: string }[]>([]);
  const [open, setOpen] = useState(false);
  const [locating, setLocating] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setQuery(value.address);
  }, [value.address]);

  const applyResolved = useCallback(
    (loc: ResolvedLocation, fallbackAddress = "") => {
      const next = locationFromResolved(loc, fallbackAddress || value.address);
      onChange(next);
      setQuery(next.address);
      setOpen(false);
      setSuggestions([]);
      setError(null);
    },
    [onChange, value.address],
  );

  const resolveText = useCallback(
    async (text: string): Promise<boolean> => {
      const trimmed = text.trim();
      if (!trimmed) return false;

      const top = (await autocompleteAddress(trimmed))[0];
      if (top?.place_id?.trim()) {
        const byPlace = await resolvePlace(top.place_id);
        if (byPlace) {
          applyResolved(byPlace, trimmed);
          return true;
        }
      }

      const byAddress = await forwardGeocode(trimmed);
      if (byAddress) {
        applyResolved(byAddress, trimmed);
        return true;
      }

      return false;
    },
    [applyResolved],
  );

  const onInputChange = (text: string) => {
    setQuery(text);
    setError(null);
    onChange({
      ...value,
      address: text,
    });
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(async () => {
      const preds = await autocompleteAddress(text);
      setSuggestions(preds);
      setOpen(preds.length > 0);
    }, 250);
  };

  const pickSuggestion = async (placeId: string, description: string) => {
    setQuery(description);
    setOpen(false);
    setResolving(true);
    setError(null);
    try {
      if (placeId.trim()) {
        const loc = await resolvePlace(placeId);
        if (loc) {
          applyResolved(loc, description);
          return;
        }
      }
      const fallback = await forwardGeocode(description);
      if (fallback) {
        applyResolved(fallback, description);
        return;
      }
      setError("Could not resolve that address. Try another suggestion.");
    } finally {
      setResolving(false);
    }
  };

  const commitTypedAddress = async () => {
    const trimmed = query.trim();
    if (!trimmed) return;
    if (value.address === trimmed && hasValidCoordinates(value.lat, value.lng)) {
      return;
    }
    setResolving(true);
    setError(null);
    try {
      const ok = await resolveText(trimmed);
      if (!ok) {
        setError("Pick an address from the list or refine your search.");
      }
    } finally {
      setResolving(false);
    }
  };

  const useMyLocation = () => {
    if (!navigator.geolocation) {
      setError("Geolocation is not supported in this browser.");
      return;
    }
    setLocating(true);
    setError(null);
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        const loc = await reverseGeocode(pos.coords.latitude, pos.coords.longitude);
        setLocating(false);
        if (loc) applyResolved(loc);
        else setError("Could not resolve your location to an address.");
      },
      () => {
        setLocating(false);
        setError("Location permission denied.");
      },
      { enableHighAccuracy: true, timeout: 12000 },
    );
  };

  const pinned = hasValidCoordinates(value.lat, value.lng);

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-(--muted)">{label}</label>
      <div className="relative">
        <input
          className="w-full rounded-lg border border-(--border) bg-(--surface) px-3 py-2 text-sm"
          placeholder="Start typing street address…"
          value={query}
          onChange={(e) => onInputChange(e.target.value)}
          onBlur={() => {
            setTimeout(() => {
              setOpen(false);
              void commitTypedAddress();
            }, 150);
          }}
          onFocus={() => suggestions.length > 0 && setOpen(true)}
          autoComplete="street-address"
        />
        {open && suggestions.length > 0 ? (
          <ul className="absolute z-20 mt-1 max-h-48 w-full overflow-auto rounded-lg border border-(--border) bg-(--surface) shadow-lg">
            {suggestions.map((s) => (
              <li key={s.place_id || s.description}>
                <button
                  type="button"
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-(--surface-muted)"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => void pickSuggestion(s.place_id, s.description)}
                >
                  {s.description}
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded-md border border-(--border) px-3 py-1.5 text-xs font-medium"
          onClick={useMyLocation}
          disabled={locating || resolving}
        >
          {locating ? "Locating…" : "Share my location"}
        </button>
        {pinned ? (
          <span className="text-xs text-(--muted)">Pinned for dispatch routing</span>
        ) : resolving ? (
          <span className="text-xs text-(--muted)">Resolving address…</span>
        ) : null}
      </div>
      {error ? <p className="text-xs text-red-600">{error}</p> : null}
    </div>
  );
}

/** Resolve free-text address to coordinates when the picker has text but no pin yet. */
export async function resolveLocationValue(input: LocationValue): Promise<LocationValue | null> {
  const address = input.address.trim();
  if (!address) return null;
  if (hasValidCoordinates(input.lat, input.lng)) return input;

  const top = (await autocompleteAddress(address))[0];
  if (top?.place_id?.trim()) {
    const byPlace = await resolvePlace(top.place_id);
    if (byPlace) return locationFromResolved(byPlace, address);
  }

  const byAddress = await forwardGeocode(address);
  if (byAddress) return locationFromResolved(byAddress, address);

  return null;
}
