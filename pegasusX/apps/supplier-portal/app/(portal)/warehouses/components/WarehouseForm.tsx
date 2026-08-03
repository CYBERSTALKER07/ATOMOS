import { useState } from "react";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";

interface WarehouseFormProps {
  onSave: (name: string, location: LocationValue, radius: number) => Promise<void>;
  onCancel: () => void;
}

const DEFAULT_LOCATION: LocationValue = {
  address: "",
  lat: "41.2995",
  lng: "69.2401",
};

export function WarehouseForm({ onSave, onCancel }: WarehouseFormProps) {
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [radius, setRadius] = useState("50");

  const handleSave = async () => {
    const trimmed = name.trim();
    const latValue = Number.parseFloat(location.lat);
    const lngValue = Number.parseFloat(location.lng);
    const radiusValue = Number.parseFloat(radius);
    
    if (!trimmed || !location.address.trim() || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError("Name and address are required.");
      return;
    }
    
    setSaving(true);
    setError(null);
    try {
      await onSave(trimmed, location, Number.isFinite(radiusValue) && radiusValue > 0 ? radiusValue : 50);
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_warehouse_failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="md-card p-6 space-y-4 mb-6">
      <h2 className="md-typescale-title-medium">Add warehouse</h2>
      {error && <div className="text-red-600 md-typescale-body-medium">{error}</div>}
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Name</span>
        <input className="md-input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder="Main warehouse" />
      </label>
      <LocationPicker value={location} onChange={setLocation} label="Warehouse address" />
      <label className="block space-y-1">
        <span className="md-typescale-label-medium">Coverage km</span>
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
