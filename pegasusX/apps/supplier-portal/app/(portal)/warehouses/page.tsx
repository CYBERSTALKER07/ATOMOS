"use client";

import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyWarehouse, SupplierTopologyUpdateRequest } from "@pegasusx/types";
import { LocationPicker, type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

const DEFAULT_LOCATION: LocationValue = {
  address: "",
  lat: "41.2995",
  lng: "69.2401",
};

export default function WarehousesPage() {
  const [topology, setTopology] = useState<{ warehouses: SupplierTopologyWarehouse[]; factories: SupplierTopologyUpdateRequest["factories"] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [location, setLocation] = useState<LocationValue>(DEFAULT_LOCATION);
  const [radius, setRadius] = useState("50");

  const load = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierTopology()
      .then((t) => setTopology({ warehouses: t.warehouses, factories: t.factories }))
      .catch((err) => setError(err instanceof Error ? err.message : "load_warehouses_failed"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const addWarehouse = async () => {
    if (!topology) return;
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
      const body: SupplierTopologyUpdateRequest = {
        warehouses: [
          ...topology.warehouses.map((w) => ({
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
          {
            name: trimmed,
            address: location.address.trim(),
            place_id: location.place_id,
            lat: latValue,
            lng: lngValue,
            coverage_radius_km: Number.isFinite(radiusValue) && radiusValue > 0 ? radiusValue : 50,
            is_active: true,
            is_on_shift: true,
            transfer_mode: "TRUCK",
          },
        ],
        factories: topology.factories,
      };
      await api.updateSupplierTopology(body);
      setShowForm(false);
      setName("");
      setLocation(DEFAULT_LOCATION);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_warehouse_failed");
    } finally {
      setSaving(false);
    }
  };

  const warehouses = topology?.warehouses ?? [];

  return (
    <PageChrome
      icon="warehouse"
      title="Warehouses"
      description="Distribution nodes and coverage for retailer serviceability."
      loading={loading}
      error={error}
      empty={warehouses.length === 0 && !showForm}
      emptyMessage="No warehouses yet. Add your first distribution node."
      actions={
        <button type="button" className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setShowForm(true)}>
          Add warehouse
        </button>
      }
    >
      {showForm ? (
        <div className="md-card p-6 space-y-4 mb-6">
          <h2 className="md-typescale-title-medium">Add warehouse</h2>
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
            <button type="button" className="md-btn md-btn-filled px-4 py-2" disabled={saving} onClick={() => void addWarehouse()}>
              {saving ? "Saving…" : "Save warehouse"}
            </button>
            <button type="button" className="md-btn md-btn-text px-4 py-2" onClick={() => setShowForm(false)}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {warehouses.length === 0 && !showForm ? (
        <div className="flex flex-col items-center justify-center py-16 gap-4">
          <p className="md-typescale-body-medium text-[var(--color-md-outline)]">Add your first warehouse to start fulfilling orders.</p>
          <button type="button" className="md-btn md-btn-filled px-6 py-3" onClick={() => setShowForm(true)}>
            Add first warehouse
          </button>
        </div>
      ) : (
        <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
          {warehouses.map((w) => (
            <li key={w.warehouse_id || w.name} className="p-4 md-typescale-body-medium">
              <div className="font-medium">{w.name}</div>
              <div className="text-[var(--color-md-outline)] text-sm mt-1">
                {(w.address || "Coordinates on file").toString()} · Radius {w.coverage_radius_km} km · {w.is_on_shift ? "On shift" : "Off shift"} ·{" "}
                {w.is_active ? "Active" : "Inactive"}
              </div>
            </li>
          ))}
        </ul>
      )}
    </PageChrome>
  );
}
