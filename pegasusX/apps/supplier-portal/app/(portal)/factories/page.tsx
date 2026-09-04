"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyFactory, SupplierTopologyUpdateRequest, SupplierTopologyWarehouse } from "@pegasusx/types";
import { factoryToTopologyInput, warehouseToTopologyInput } from "@/lib/topology";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { FactoryForm } from "./components/FactoryForm";
import { FactoryList } from "./components/FactoryList";

const api = createSupplierApi();

export default function FactoriesPage() {
  const t = usePortalT();
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
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_factories_failed")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [refreshTick]);

  const addFactory = async (name: string, location: LocationValue, extras: { country_code: string }) => {
    if (warehouses.length === 0) {
      throw new Error("Add at least one warehouse before creating factories.");
    }
    
    const body: SupplierTopologyUpdateRequest = {
      warehouses: warehouses.map(warehouseToTopologyInput),
      factories: [
        ...factories.map(factoryToTopologyInput),
        {
          name,
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number.parseFloat(location.lat),
          lng: Number.parseFloat(location.lng),
          country_code: extras.country_code || undefined,
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
      title={t("portal.nav.factories")}
      description={t("supplier_portal.residual.text.production_nodes_for_manifests_and_warehouse_replenishment")}
      loading={loading}
      error={error}
      empty={factories.length === 0 && !showForm}
      emptyMessage={t("supplier_portal.residual.text.no_factories_yet_add_a_production_node_linked_to_your_warehouse_")}
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
