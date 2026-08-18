"use client";

import { usePortalT } from "@/lib/i18n";
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
import { mapInitialViewState, readCachedAuthSession } from "@pegasusx/api-client";
import MapGL, { Marker, NavigationControl } from 'react-map-gl/maplibre';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';

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
  const resolvedLabel = label ?? t("factory_portal.residual.text.address");
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
      setError(t("factory_portal.residual.text.could_not_resolve_that_address_try_another_suggestion"));
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
        setError(t("factory_portal.residual.text.pick_an_address_from_the_list_or_refine_your_search"));
      }
    } finally {
      setResolving(false);
    }
  };

  const useMyLocation = () => {
    if (!navigator.geolocation) {
      setError(t("factory_portal.residual.text.geolocation_is_not_supported_in_this_browser"));
      return;
    }
    setLocating(true);
    setError(null);
    navigator.geolocation.getCurrentPosition(
      async (pos) => {
        const loc = await reverseGeocode(pos.coords.latitude, pos.coords.longitude);
        setLocating(false);
        if (loc) applyResolved(loc);
        else setError(t("factory_portal.residual.text.could_not_resolve_your_location_to_an_address"));
      },
      () => {
        setLocating(false);
        setError(t("factory_portal.residual.text.location_permission_denied"));
      },
      { enableHighAccuracy: true, timeout: 12000 },
    );
  };

  const pinned = hasValidCoordinates(value.lat, value.lng);

  const handleMapClick = async (e: maplibregl.MapMouseEvent) => {
    const lat = e.lngLat.lat;
    const lng = e.lngLat.lng;
    setResolving(true);
    setError(null);
    try {
      const loc = await reverseGeocode(lat, lng);
      if (loc) {
        applyResolved(loc);
      } else {
        onChange({ ...value, lat: String(lat), lng: String(lng) });
        setQuery(`Pinned location (${lat.toFixed(4)}, ${lng.toFixed(4)})`);
      }
    } finally {
      setResolving(false);
    }
  };

  return (
    <div className="space-y-2">
      <label className="portal-label">{resolvedLabel}</label>
      <div className="relative">
        <input
          className="portal-input w-full"
          placeholder={t("factory_portal.location_picker.text.start_typing_street_address")}
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
          className="portal-btn portal-btn--ghost text-xs"
          onClick={useMyLocation}
          disabled={locating || resolving}
        >
          {locating ? "Locating…" : "Share my location"}
        </button>
        {pinned ? (
          <span className="md-helper">{t("factory_portal.location_picker.text.pinned_for_supply_routing")}</span>
        ) : resolving ? (
          <span className="md-helper">{t("factory_portal.location_picker.text.resolving_address")}</span>
        ) : null}
      </div>
      {error ? <p className="md-helper" data-error="true">{error}</p> : null}

      <div className="h-64 w-full mt-2 rounded-xl overflow-hidden border" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
        <MapGL
          initialViewState={{
            ...mapInitialViewState(readCachedAuthSession()?.pack, pinned ? 14 : 10),
            ...(pinned
              ? { longitude: parseFloat(value.lng), latitude: parseFloat(value.lat), zoom: 14 }
              : {}),
          }}
          longitude={pinned ? parseFloat(value.lng) : undefined}
          latitude={pinned ? parseFloat(value.lat) : undefined}
          mapStyle="https://basemaps.cartocdn.com/gl/positron-gl-style/style.json"
          style={{ width: '100%', height: '100%' }}
          mapLib={maplibregl}
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          onClick={(e) => void handleMapClick(e as any)}
          interactiveLayerIds={[]}
          cursor="crosshair"
        >
          <NavigationControl position="top-right" showCompass={false} />
          {pinned && (
            <Marker longitude={parseFloat(value.lng)} latitude={parseFloat(value.lat)} anchor="bottom">
              <div className="flex flex-col items-center">
                <div className="w-4 h-4 rounded-full bg-[var(--color-md-primary)] shadow-lg" />
                <div className="w-1 h-3 bg-[var(--color-md-primary)]" />
              </div>
            </Marker>
          )}
        </MapGL>
      </div>
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
