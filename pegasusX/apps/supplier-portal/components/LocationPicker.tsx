"use client";

import { usePortalT } from "@/lib/i18n";
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

export function LocationPicker({ value, onChange, label }: LocationPickerProps) {
  const t = usePortalT();
  const resolvedLabel = label ?? t("supplier_portal.residual.text.address");
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
      setError(t("supplier_portal.residual.text.geolocation_is_not_supported_in_this_browser"));
      return;
    }
    setLocating(true);
    setError(null);
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        const loc = await reverseGeocode(pos.coords.latitude, pos.coords.longitude);
        setLocating(false);
        if (loc) applyResolved(loc);
        else setError(t("supplier_portal.residual.text.could_not_resolve_your_location_to_an_address"));
      },
      () => {
        setLocating(false);
        setError(t("supplier_portal.residual.text.location_permission_denied"));
      },
      { enableHighAccuracy: true, timeout: 12000 },
    );
  };

  return (
    <div className="space-y-2">
      <label className="text-xs font-medium text-(--muted)">{resolvedLabel}</label>
      <div className="relative">
        <input
          className="w-full rounded-lg border border-(--border) bg-(--surface) px-3 py-2 text-sm"
          placeholder={t("supplier_portal.location_picker.text.start_typing_street_address")}
          value={query}
          onChange={(e) => onInputChange(e.target.value)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
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
          disabled={locating}
        >
          {locating ? "Locating…" : "Share my location"}
        </button>
        {value.address ? (
          <span className="text-xs text-(--muted)">{t("supplier_portal.location_picker.text.pinned_for_dispatch_routing")}</span>
        ) : null}
      </div>
      {error ? <p className="text-xs text-red-600">{error}</p> : null}
    </div>
  );
}
