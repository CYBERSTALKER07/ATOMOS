import { useState } from "react";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

interface FactoryFormProps {
  onSave: (name: string, location: LocationValue) => Promise<void>;
  onCancel: () => void;
}

const DEFAULT_LOCATION: LocationValue = {
  address: "",
  lat: "41.3111",
  lng: "69.2797",
};

export function FactoryForm({ onSave, onCancel }: FactoryFormProps) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);

  const handleSave = async () => {
    const trimmed = name.trim();
    const latValue = Number.parseFloat(location.lat);
    const lngValue = Number.parseFloat(location.lng);
    
    if (!trimmed || !location.address.trim() || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError("Name and address are required.");
      return;
    }
    
    setSaving(true);
    setError(null);
    try {
      await onSave(trimmed, location);
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_factory_failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="md-card p-6 space-y-4 mb-6">
      <h2 className="md-typescale-title-medium">Add factory</h2>
      {error && <div className="text-red-600 md-typescale-body-medium">{error}</div>}
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Name</span>
        <input className="md-input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder="Main factory" />
      </label>
      <LocationPicker value={location} onChange={setLocation} label="Factory address" />
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
