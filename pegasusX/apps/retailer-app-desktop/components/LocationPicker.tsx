"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  autocompleteAddress,
  forwardGeocode,
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
  const [error, setError] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setQuery(value.address);
  }, [value.address]);

  const applyResolved = useCallback(
    (loc: ResolvedLocation) => {
      onChange({
        address: loc.address || loc.formatted_address || value.address,
        lat: String(loc.lat),
        lng: String(loc.lng),
        place_id: loc.place_id,
      });
      setQuery(loc.address || loc.formatted_address || "");
      setOpen(false);
      setSuggestions([]);
    },
    [onChange, value.address],
  );

  const onInputChange = (text: string) => {
    setQuery(text);
    setError(null);
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
    const loc = await resolvePlace(placeId);
    if (loc) {
      applyResolved(loc);
      return;
    }
    const fallback = await forwardGeocode(description);
    if (fallback) applyResolved(fallback);
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

  return (
    <div className="space-y-2">
      <label className="portal-label">{label}</label>
      <div className="relative">
        <input
          className="portal-input w-full"
          placeholder="Start typing street address…"
          value={query}
          onChange={(e) => onInputChange(e.target.value)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
          onFocus={() => suggestions.length > 0 && setOpen(true)}
          autoComplete="street-address"
        />
        {open && suggestions.length > 0 ? (
          <ul className="absolute z-20 mt-1 max-h-48 w-full overflow-auto rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] shadow-lg">
            {suggestions.map((s) => (
              <li key={s.place_id || s.description}>
                <button
                  type="button"
                  className="block w-full px-3 py-2 text-left text-sm hover:bg-[var(--desk-canvas)]"
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
          className="portal-btn portal-btn--ghost text-xs"
          onClick={useMyLocation}
          disabled={locating}
        >
          {locating ? "Locating…" : "Share my location"}
        </button>
        {value.address ? (
          <span className="text-xs text-[var(--desk-text-tertiary)]">Saved for delivery routing</span>
        ) : null}
      </div>
      {error ? <p className="text-xs text-red-500">{error}</p> : null}
    </div>
  );
}
