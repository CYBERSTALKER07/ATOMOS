"use client";

import { usePortalT } from "@/lib/i18n";
import { useState } from "react";
import { AUTH_COUNTRIES } from "@pegasusx/ui-kit/auth";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

interface FactoryFormProps {
  onSave: (name: string, location: LocationValue, extras: { country_code: string }) => Promise<void>;
  onCancel: () => void;
}

const DEFAULT_LOCATION: LocationValue = {
  address: "",
  lat: "41.3111",
  lng: "69.2797",
};

export function FactoryForm({ onSave, onCancel }: FactoryFormProps) {
  const t = usePortalT();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [country, setCountry] = useState("");

  const handleSave = async () => {
    const trimmed = name.trim();
    const latValue = Number.parseFloat(location.lat);
    const lngValue = Number.parseFloat(location.lng);
    
    if (!trimmed || !location.address.trim() || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError(t("supplier_portal.residual.text.name_and_address_are_required"));
      return;
    }
    
    setSaving(true);
    setError(null);
    try {
      await onSave(trimmed, location, { country_code: country });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.save_factory_failed"));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="md-card p-6 space-y-4 mb-6">
      <h2 className="md-typescale-title-medium">{t("supplier_portal.factories.components.factory_form.text.add_factory")}</h2>
      {error && <div className="text-red-600 md-typescale-body-medium">{error}</div>}
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">{t("supplier_portal.analytics.knowledge_graph.text.name")}</span>
        <input className="md-input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("supplier_portal.factories.components.factory_form.text.main_factory")} />
      </label>
      <LocationPicker value={location} onChange={setLocation} label={t("supplier_portal.residual.text.factory_address")} />
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Country</span>
        <select className="md-input w-full max-w-xs" value={country} onChange={(e) => setCountry(e.target.value)}>
          <option value="">Unset</option>
          {AUTH_COUNTRIES.map((c) => (
            <option key={c.code} value={c.code}>{c.name} ({c.code})</option>
          ))}
        </select>
      </label>
      <div className="flex gap-2">
        <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void handleSave()}>
          {saving ? "Saving…" : "Save factory"}
        </button>
        <button type="button" className="md-btn md-btn-text px-4 py-2" disabled={saving} onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
}
