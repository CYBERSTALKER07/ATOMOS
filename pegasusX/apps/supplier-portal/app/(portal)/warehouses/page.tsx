"use client";

import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyWarehouse, SupplierTopologyUpdateRequest } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

const DEFAULT_LAT = 41.2995;
const DEFAULT_LNG = 69.2401;

export default function WarehousesPage() {
  const [topology, setTopology] = useState<{ warehouses: SupplierTopologyWarehouse[]; factories: SupplierTopologyUpdateRequest["factories"] } | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [name, setName] = useState("");
  const [lat, setLat] = useState(String(DEFAULT_LAT));
  const [lng, setLng] = useState(String(DEFAULT_LNG));
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
    const latValue = Number.parseFloat(lat);
    const lngValue = Number.parseFloat(lng);
    const radiusValue = Number.parseFloat(radius);
    if (!trimmed || !Number.isFinite(latValue) || !Number.isFinite(lngValue)) {
      setError("Name and coordinates are required.");
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
            lat: w.lat,
            lng: w.lng,
            coverage_radius_km: w.coverage_radius_km,
            is_active: w.is_active,
            is_on_shift: w.is_on_shift,
            transfer_mode: w.transfer_mode,
          })),
          {
            name: trimmed,
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
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "save_warehouse_failed");
    } finally {
      setSaving(false);
    }
  };

  const warehouses = topology?.warehouses ?? [];

  return (
    <PortalSurface
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
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Latitude</span>
              <input className="md-input w-full" value={lat} onChange={(e) => setLat(e.target.value)} />
            </label>
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Longitude</span>
              <input className="md-input w-full" value={lng} onChange={(e) => setLng(e.target.value)} />
            </label>
            <label className="block space-y-1">
              <span className="md-typescale-label-medium">Coverage km</span>
              <input className="md-input w-full" value={radius} onChange={(e) => setRadius(e.target.value)} />
            </label>
          </div>
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
                {w.lat}, {w.lng} · Radius {w.coverage_radius_km} km · {w.is_on_shift ? "On shift" : "Off shift"} ·{" "}
                {w.is_active ? "Active" : "Inactive"}
              </div>
            </li>
          ))}
        </ul>
      )}
    </PortalSurface>
  );
}
