"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

import { CoverageCityChips } from "@/components/CoverageCityChips";
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";
import { packMapCenter } from "@pegasusx/api-core";
import type { SupplierTopologyCoverageCity } from "@pegasusx/types";

interface WarehouseFormProps {
  onSave: (
    name: string,
    location: LocationValue,
    radius: number,
    extras: {
      country_code: string;
      coverage_cities: SupplierTopologyCoverageCity[];
      primary_factory_id: string;
    },
  ) => Promise<void>;
  onCancel: () => void;
  factoryOptions?: Array<{ id: string; name: string }>;
}

function defaultWarehouseLocation(pack: { map_center_lat?: number; map_center_lng?: number; status?: string } | null | undefined): LocationValue {
  const c = packMapCenter(pack);
  return {
    address: "",
    lat: c ? String(c.lat) : "",
    lng: c ? String(c.lng) : "",
  };
}

export function WarehouseForm({ onSave, onCancel, factoryOptions = [] }: WarehouseFormProps) {
  const t = usePortalT();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const { pack } = useSupplierPaymentCatalog();
  const [location, setLocation] = useState<LocationValue>(() => defaultWarehouseLocation(null));
  const [radius, setRadius] = useState("50");
  const country = pack?.code || "";
  useEffect(() => {
    const c = packMapCenter(pack);
    if (!c) return;
    setLocation((prev) => {
      if (prev.lat || prev.lng) return prev;
      return { ...prev, lat: String(c.lat), lng: String(c.lng) };
    });
  }, [pack?.code, pack?.map_center_lat, pack?.map_center_lng]);
  const [cities, setCities] = useState<SupplierTopologyCoverageCity[]>([]);
  const [primaryFactory, setPrimaryFactory] = useState("");

  const handleSave = async () => {
    const trimmed = name.trim();
    const latValue = Number.parseFloat(location.lat);
    const lngValue = Number.parseFloat(location.lng);
    const radiusValue = Number.parseFloat(radius);
    
    if (!trimmed || !location.address.trim() || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError(t("supplier_portal.residual.text.name_and_address_are_required"));
      return;
    }
    
    setSaving(true);
    setError(null);
    try {
      await onSave(trimmed, location, Number.isFinite(radiusValue) && radiusValue > 0 ? radiusValue : 50, {
        country_code: country,
        coverage_cities: cities,
        primary_factory_id: primaryFactory,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.save_warehouse_failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="md-card p-6 space-y-4 mb-6">
      <h2 className="md-typescale-title-medium">{t("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse")}</h2>
      {error && <div className="text-red-600 md-typescale-body-medium">{error}</div>}
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">{t("supplier_portal.analytics.knowledge_graph.text.name")}</span>
        <input className="md-input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("supplier_portal.warehouses.components.warehouse_form.text.main_warehouse")} />
      </label>
      <LocationPicker value={location} onChange={setLocation} label={t("supplier_portal.residual.text.warehouse_address")} />
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Country</span>
        <input className="md-input w-full max-w-xs" value={country} readOnly />
        <p className="text-xs text-[var(--muted)]">Pack country is locked. Cross-market warehouses are rejected.</p>
      </label>
      <CoverageCityChips cities={cities} onChange={setCities} />
      {factoryOptions.length > 0 ? (
        <label className="block space-y-1">
          <span className="md-typescale-label-medium">Primary factory</span>
          <select className="md-input w-full max-w-xs" value={primaryFactory} onChange={(e) => setPrimaryFactory(e.target.value)}>
            <option value="">Closest factory (default)</option>
            {factoryOptions.map((f) => (
              <option key={f.id} value={f.id}>{f.name}</option>
            ))}
          </select>
        </label>
      ) : null}
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">{t("supplier_portal.warehouses.components.warehouse_form.text.coverage_km")}</span>
        <input className="md-input w-full max-w-xs" value={radius} onChange={(e) => setRadius(e.target.value)} />
      </label>
      <div className="flex gap-2">
        <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void handleSave()}>
          {saving ? "Saving…" : "Save warehouse"}
        </button>
        <button type="button" className="md-btn md-btn-text px-4 py-2" disabled={saving} onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}
