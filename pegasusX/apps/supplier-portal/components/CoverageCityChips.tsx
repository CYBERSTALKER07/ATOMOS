"use client";

import { useState } from "react";
import type { SupplierTopologyCoverageCity } from "@pegasusx/types";
import { forwardGeocode } from "@/lib/geocode";

type Props = {
  cities: SupplierTopologyCoverageCity[];
  onChange: (cities: SupplierTopologyCoverageCity[]) => void;
};

export function CoverageCityChips({ cities, onChange }: Props) {
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const add = async () => {
    const name = draft.trim();
    if (!name) return;
    setBusy(true);
    setError(null);
    try {
      const loc = await forwardGeocode(name);
      if (!loc) {
        setError("City not found");
        return;
      }
      if (cities.some((c) => c.name.toLowerCase() === name.toLowerCase())) {
        setDraft("");
        return;
      }
      onChange([...cities, { name, lat: loc.lat, lng: loc.lng }]);
      setDraft("");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-2">
      <span className="md-typescale-label-medium">Coverage cities</span>
      <p className="text-xs text-[var(--muted)]">
        Empty = whole warehouse country. Cities become compacted H3 cells at checkout.
      </p>
      <div className="flex flex-wrap gap-2">
        {cities.map((city) => (
          <button
            key={city.name}
            type="button"
            className="md-btn md-btn-outlined px-2 py-1 text-sm"
            onClick={() => onChange(cities.filter((c) => c.name !== city.name))}
          >
            {city.name} ×
          </button>
        ))}
      </div>
      <div className="flex gap-2">
        <input
          className="md-input flex-1"
          value={draft}
          placeholder="Add city (geocoded)"
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              void add();
            }
          }}
        />
        <button type="button" className="md-btn md-btn-outlined px-3" disabled={busy} onClick={() => void add()}>
          {busy ? "…" : "Add"}
        </button>
      </div>
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
    </div>
  );
}
