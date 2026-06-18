"use client";

import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyFactory, SupplierTopologyUpdateRequest, SupplierTopologyWarehouse } from "@pegasusx/types";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

const DEFAULT_LOCATION: LocationValue = {
  address: "",
  lat: "41.3111",
  lng: "69.2797",
};

export default function FactoriesPage() {
  const [warehouses, setWarehouses] = useState<SupplierTopologyWarehouse[]>([]);
  const [factories, setFactories] = useState<SupplierTopologyFactory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);

  const load = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierTopology()
      .then((t) => {
        setWarehouses(t.warehouses);
        setFactories(t.factories);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_factories_failed"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const addFactory = async () => {
    const trimmed = name.trim();
    const latValue = Number.parseFloat(location.lat);
    const lngValue = Number.parseFloat(location.lng);
    if (!trimmed || !location.address.trim() || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError("Name and address are required.");
      return;
    }
    if (warehouses.length === 0) {
      setError("Add at least one warehouse before creating factories.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const body: SupplierTopologyUpdateRequest = {
        warehouses: warehouses.map((w) => ({
          warehouse_id: w.warehouse_id,
          name: w.name,
          address: w.address,
          place_id: w.place_id,
          lat: w.lat,
          lng: w.lng,
          coverage_radius_km: w.coverage_radius_km,
          is_active: w.is_active,
          is_on_shift: w.is_on_shift,
          transfer_mode: w.transfer_mode,
        })),
        factories: [
          ...factories.map((f) => ({
            factory_id: f.factory_id,
            name: f.name,
            address: f.address,
            place_id: f.place_id,
            lat: f.lat,
            lng: f.lng,
            is_active: f.is_active,
          })),
          {
            name: trimmed,
            address: location.address.trim(),
            place_id: location.place_id,
            lat: latValue,
            lng: lngValue,
            is_active: true,
          },
        ],
      };
      await api.updateSupplierTopology(body);
      setShowForm(false);
      setName("");
      setLocation(DEFAULT_LOCATION);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_factory_failed");
    } finally {
      setSaving(false);
    }
  };

  return (
    <PortalSurface
      title="Factories"
      description="Production nodes for manifests and warehouse replenishment."
      loading={loading}
      error={error}
      empty={factories.length === 0 && !showForm}
      emptyMessage="No factories yet. Add a production node linked to your warehouse network."
      actions={
        <button type="button" className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setShowForm(true)}>
          Add factory
        </button>
      }
    >
      {showForm ? (
        <div className="md-card p-6 space-y-4 mb-6">
          <h2 className="md-typescale-title-medium">Add factory</h2>
          <label className="block space-y-1">
            <span className="md-typescale-label-medium">Name</span>
            <input className="md-input w-full" value={name} onChange={(e) => setName(e.target.value)} placeholder="Main factory" />
          </label>
          <LocationPicker value={location} onChange={setLocation} label="Factory address" />
          <div className="flex gap-2">
            <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void addFactory()}>
              {saving ? "Saving…" : "Save factory"}
            </button>
            <button type="button" className="md-btn md-btn-text px-4 py-2" onClick={() => setShowForm(false)}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {factories.length === 0 && !showForm ? (
        <div className="flex flex-col items-center justify-center py-16 gap-4">
          <p className="md-typescale-body-medium text-[var(--color-md-outline)]">Add your first factory to link production to warehouses.</p>
          <button type="button" className="md-btn md-btn-filled px-6 py-3" onClick={() => setShowForm(true)}>
            Add first factory
          </button>
        </div>
      ) : (
        <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
          {factories.map((f) => (
            <li key={f.factory_id || f.name} className="p-4 md-typescale-body-medium">
              <div className="font-medium">{f.name}</div>
              <div className="text-[var(--color-md-outline)] text-sm mt-1">
                {(f.address || "Coordinates on file").toString()} · {f.is_active ? "Active" : "Inactive"}
              </div>
            </li>
          ))}
        </ul>
      )}
    </PortalSurface>
  );
}
