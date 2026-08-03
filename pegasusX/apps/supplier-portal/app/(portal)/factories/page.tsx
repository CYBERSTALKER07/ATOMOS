"use client";

import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyFactory, SupplierTopologyUpdateRequest, SupplierTopologyWarehouse } from "@pegasusx/types";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { FactoryForm } from "./components/FactoryForm";
import { FactoryList } from "./components/FactoryList";

const api = createSupplierApi();

export default function FactoriesPage() {
  const [warehouses, setWarehouses] = useState<SupplierTopologyWarehouse[]>([]);
  const [factories, setFactories] = useState<SupplierTopologyFactory[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);

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
  }, [refreshTick]);

  const addFactory = async (name: string, location: LocationValue) => {
    if (warehouses.length === 0) {
      throw new Error("Add at least one warehouse before creating factories.");
    }
    
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
          name,
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number.parseFloat(location.lat),
          lng: Number.parseFloat(location.lng),
          is_active: true,
        },
      ],
    };
    
    await api.updateSupplierTopology(body);
    setShowForm(false);
    load();
  };

  return (
    <PageChrome
      icon="factory"
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
      {showForm && (
        <FactoryForm
          onSave={addFactory}
          onCancel={() => setShowForm(false)}
        />
      )}

      {!showForm && (
        <FactoryList 
          factories={factories} 
          onAddFirst={() => setShowForm(true)} 
        />
      )}
    </PageChrome>
  );
}
