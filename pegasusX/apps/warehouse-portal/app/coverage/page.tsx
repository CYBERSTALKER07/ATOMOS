"use client";

import { useCallback, useEffect, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import PageTransition from "@/components/PageTransition";
import { createWarehouseApi } from "@/lib/api";
import { coverageModeLabel, normalizeCoverageMode, pinKey } from "@/lib/coverage";
import type { ServicePin, SupplierTopologyCoverageCity, WarehouseOpsSupplyFactoryResponse } from "@pegasusx/types";

const api = createWarehouseApi();

export default function WarehouseCoverageViewPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState("COUNTRY_CLOSEST");
  const [country, setCountry] = useState("");
  const [cities, setCities] = useState<SupplierTopologyCoverageCity[]>([]);
  const [pins, setPins] = useState<ServicePin[]>([]);
  const [factory, setFactory] = useState<WarehouseOpsSupplyFactoryResponse | null>(null);
  const [factoryError, setFactoryError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    setFactoryError(null);
    Promise.all([
      api.getWarehouseOpsCoverage(),
      api.getWarehouseOpsSupplyFactory().catch((err: unknown) => {
        setFactoryError(err instanceof Error ? err.message : "factory_unassigned");
        return null;
      }),
    ])
      .then(([coverage, supply]) => {
        setMode(normalizeCoverageMode(coverage.mode));
        setCountry(coverage.country_code || "");
        setCities(coverage.cities ?? []);
        setPins(coverage.pins ?? []);
        setFactory(supply);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_coverage_failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="warehouse"
        title="Coverage and supply"
        description="Pins and cities are set by the supplier. Nearest factory comes from the engine. This warehouse cannot re-pin or pick a factory in another country."
      >
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="md-skeleton md-skeleton-row" />
            ))}
          </div>
        ) : error ? (
          <p className="text-sm text-[var(--danger)]">{error}</p>
        ) : (
          <div className="max-w-2xl space-y-6">
            <section className="rounded-xl border border-[var(--border)] p-4">
              <h2 className="text-sm font-semibold">Coverage</h2>
              <p className="mt-1 text-xs text-[var(--muted)]">
                Mode {coverageModeLabel(mode)}
                {country ? ` · country ${country}` : ""}
              </p>
              <div className="mt-3">
                <p className="text-xs font-medium text-[var(--muted)]">Cities</p>
                {cities.length === 0 ? (
                  <p className="text-sm">Closest same-country warehouse matching (no city cells).</p>
                ) : (
                  <ul className="mt-1 space-y-1 text-sm">
                    {cities.map((city) => (
                      <li key={`${city.name}:${city.lat}:${city.lng}`}>{city.name}</li>
                    ))}
                  </ul>
                )}
              </div>
              <div className="mt-3">
                <p className="text-xs font-medium text-[var(--muted)]">Pins</p>
                {pins.length === 0 ? (
                  <p className="text-sm">No supplier pins on this warehouse.</p>
                ) : (
                  <ul className="mt-1 space-y-1 text-sm">
                    {pins.map((pin) => (
                      <li key={pinKey(pin)}>
                        {pin.target_type} · {pin.target_id}
                        {pin.priority ? ` · priority ${pin.priority}` : ""}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </section>

            <section className="rounded-xl border border-[var(--border)] p-4">
              <h2 className="text-sm font-semibold">Nearest factory</h2>
              {factoryError ? (
                <p className="mt-1 text-sm text-[var(--muted)]">{factoryError}</p>
              ) : factory?.factory_id ? (
                <p className="mt-1 text-sm">
                  {factory.factory_id}
                  <span className="block text-xs text-[var(--muted)]">
                    Engine · {factory.transfer_mode || "truck"} · {factory.country_code}
                  </span>
                </p>
              ) : (
                <p className="mt-1 text-sm text-[var(--muted)]">No same-country factory assigned.</p>
              )}
            </section>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
