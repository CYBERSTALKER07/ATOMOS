"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { CoverageCityChips } from "@/components/CoverageCityChips";
import { createSupplierApi } from "@/lib/api";
import {
  coverageModeLabel,
  normalizeCoverageMode,
  PIN_TARGET_TYPES,
  pinKey,
} from "@/lib/coverage";
import { usePortalT } from "@/lib/i18n";
import type {
  ServicePin,
  ServicePinTargetType,
  SupplierCRMRetailer,
  SupplierRegion,
  SupplierTopologyCoverageCity,
} from "@pegasusx/types";

const api = createSupplierApi();

export default function WarehouseCoveragePage() {
  const t = usePortalT();
  const params = useParams<{ warehouseId: string }>();
  const warehouseId = String(params.warehouseId || "").trim();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState("COUNTRY_CLOSEST");
  const [cities, setCities] = useState<SupplierTopologyCoverageCity[]>([]);
  const [pins, setPins] = useState<ServicePin[]>([]);
  const [regions, setRegions] = useState<SupplierRegion[]>([]);
  const [retailers, setRetailers] = useState<SupplierCRMRetailer[]>([]);
  const [regionName, setRegionName] = useState("");
  const [draftType, setDraftType] = useState<ServicePinTargetType>("RETAILER");
  const [draftTarget, setDraftTarget] = useState("");
  const [draftPriority, setDraftPriority] = useState("0");

  const load = () => {
    if (!warehouseId) return;
    setLoading(true);
    setError(null);
    Promise.all([
      api.getWarehouseCoverage(warehouseId),
      api.getSupplierRegions().catch(() => ({ items: [] })),
      api.getSupplierCRMRetailers().catch(() => ({ retailers: [] })),
    ])
      .then(([coverage, regionResp, crm]) => {
        setMode(normalizeCoverageMode(coverage.mode));
        setCities(coverage.cities ?? []);
        setPins(coverage.pins ?? []);
        setRegions(regionResp.items ?? []);
        setRetailers(crm.retailers ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_coverage_failed"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [warehouseId]);

  const addPin = () => {
    const targetId = draftTarget.trim();
    if (!targetId) return;
    const next: ServicePin = {
      target_type: draftType,
      target_id: targetId,
      priority: Number.parseInt(draftPriority, 10) || 0,
    };
    if (pins.some((pin) => pinKey(pin) === pinKey(next))) return;
    setPins([...pins, next]);
    setDraftTarget("");
  };

  const saveCoverage = async () => {
    setSaving(true);
    setError(null);
    try {
      const resp = await api.replaceWarehouseCoverage(warehouseId, {
        cities,
        pins: pins.map((pin) => ({
          target_type: pin.target_type,
          target_id: pin.target_id,
          priority: pin.priority ?? 0,
        })),
      });
      setMode(normalizeCoverageMode(resp.mode));
      setCities(resp.cities ?? cities);
      setPins(resp.pins ?? pins);
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_coverage_failed");
    } finally {
      setSaving(false);
    }
  };

  const saveRegions = async () => {
    const name = regionName.trim();
    const items: Array<{ region_id?: string; name: string; country_code?: string }> = regions.map((row) => ({
      region_id: row.region_id,
      name: row.name,
      country_code: row.country_code,
    }));
    if (name) items.push({ name });
    setSaving(true);
    setError(null);
    try {
      const resp = await api.replaceSupplierRegions({ items });
      setRegions(resp.items ?? []);
      setRegionName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_regions_failed");
    } finally {
      setSaving(false);
    }
  };

  const retailerOptions = useMemo(
    () => retailers.map((row) => ({ id: row.retailer_id, label: row.retailer_name || row.retailer_id })),
    [retailers],
  );

  return (
    <PageChrome
      icon="warehouse"
      title={t("supplier_portal.residual.text.distribution_nodes_and_coverage_for_retailer_serviceability")}
      description={`Coverage mode: ${coverageModeLabel(mode)}. Pins beat closest. Cross-country pins are rejected.`}
      loading={loading}
      error={error}
      actions={
        <Link href={"/warehouses" as any} className="md-btn md-btn-outlined px-4 py-2">
          Back to warehouses
        </Link>
      }
    >
      <div className="grid gap-6 lg:grid-cols-2">
        <section className="md-card p-6 space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="md-typescale-title-medium">Cities + pins</h2>
            <span className="md-typescale-label-medium text-[var(--color-md-outline)]">
              {coverageModeLabel(mode)}
            </span>
          </div>
          <CoverageCityChips cities={cities} onChange={setCities} />
          <div className="space-y-2">
            <span className="md-typescale-label-medium">Service pins</span>
            <p className="text-xs text-[var(--muted)]">
              LOCATION, RETAILER, then REGION, then CITY. Empty pins fall back to closest same-country warehouse.
            </p>
            <ul className="space-y-2">
              {pins.map((pin) => (
                <li key={pinKey(pin)} className="flex items-center justify-between gap-2 text-sm">
                  <span>
                    {pin.target_type} · {pin.target_id}
                    {pin.priority ? ` · p${pin.priority}` : ""}
                  </span>
                  <button
                    type="button"
                    className="md-btn md-btn-text px-2"
                    onClick={() => setPins(pins.filter((row) => pinKey(row) !== pinKey(pin)))}
                  >
                    Remove
                  </button>
                </li>
              ))}
            </ul>
            <div className="grid gap-2 sm:grid-cols-4">
              <select
                className="md-input"
                value={draftType}
                onChange={(e) => setDraftType(e.target.value as ServicePinTargetType)}
              >
                {PIN_TARGET_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
              {draftType === "RETAILER" ? (
                <select className="md-input sm:col-span-2" value={draftTarget} onChange={(e) => setDraftTarget(e.target.value)}>
                  <option value="">Select retailer</option>
                  {retailerOptions.map((row) => (
                    <option key={row.id} value={row.id}>
                      {row.label}
                    </option>
                  ))}
                </select>
              ) : draftType === "REGION" ? (
                <select className="md-input sm:col-span-2" value={draftTarget} onChange={(e) => setDraftTarget(e.target.value)}>
                  <option value="">Select region</option>
                  {regions.map((row) => (
                    <option key={row.region_id} value={row.region_id}>
                      {row.name}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  className="md-input sm:col-span-2"
                  value={draftTarget}
                  placeholder={draftType === "LOCATION" ? "location_id" : "City name"}
                  onChange={(e) => setDraftTarget(e.target.value)}
                />
              )}
              <input
                className="md-input"
                value={draftPriority}
                placeholder="Priority"
                onChange={(e) => setDraftPriority(e.target.value)}
              />
            </div>
            <button type="button" className="md-btn md-btn-outlined px-3 py-2" onClick={addPin}>
              Add pin
            </button>
          </div>
          <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void saveCoverage()}>
            {saving ? "Saving…" : "Save coverage"}
          </button>
        </section>

        <section className="md-card p-6 space-y-4">
          <h2 className="md-typescale-title-medium">Supplier regions</h2>
          <p className="text-xs text-[var(--muted)]">
            Named groups inside this pack country. Not the unused global Regions table.
          </p>
          <ul className="space-y-2">
            {regions.map((row) => (
              <li key={row.region_id} className="flex items-center justify-between gap-2 text-sm">
                <span>
                  {row.name} · {row.country_code}
                </span>
                <button
                  type="button"
                  className="md-btn md-btn-text px-2"
                  onClick={() => setRegions(regions.filter((item) => item.region_id !== row.region_id))}
                >
                  Remove
                </button>
              </li>
            ))}
          </ul>
          <div className="flex gap-2">
            <input
              className="md-input flex-1"
              value={regionName}
              placeholder="Tashkent metro"
              onChange={(e) => setRegionName(e.target.value)}
            />
            <button type="button" className="md-btn md-btn-outlined px-3" disabled={saving} onClick={() => void saveRegions()}>
              Save regions
            </button>
          </div>
        </section>
      </div>
    </PageChrome>
  );
}
