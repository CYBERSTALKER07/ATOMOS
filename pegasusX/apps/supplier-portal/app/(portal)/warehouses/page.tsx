"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyWarehouse, SupplierTopologyUpdateRequest } from "@pegasusx/types";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { WarehouseForm } from "./components/WarehouseForm";
import { WarehouseList } from "./components/WarehouseList";

const api = createSupplierApi();

export default function WarehousesPage() {
  const t = usePortalT();
  const [topology, setTopology] = useState<{ warehouses: SupplierTopologyWarehouse[]; factories: SupplierTopologyUpdateRequest["factories"] } | null>(null);
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
      .then((t) => setTopology({ warehouses: t.warehouses, factories: t.factories }))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_warehouses_failed")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [refreshTick]);

  const addWarehouse = async (name: string, location: LocationValue, radius: number) => {
    if (!topology) return;
    
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
          name,
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number.parseFloat(location.lat),
          lng: Number.parseFloat(location.lng),
          coverage_radius_km: radius,
          is_active: true,
          is_on_shift: true,
          transfer_mode: "TRUCK",
        },
      ],
      factories: topology.factories,
    };
    
    await api.updateSupplierTopology(body);
    setShowForm(false);
    load();
  };

  const warehouses = topology?.warehouses ?? [];

  return (
    <PageChrome
      icon="warehouse"
      title={t("portal.nav.warehouses")}
      description={t("supplier_portal.residual.text.distribution_nodes_and_coverage_for_retailer_serviceability")}
      loading={loading}
      error={error}
      empty={warehouses.length === 0 && !showForm}
      emptyMessage={t("supplier_portal.residual.text.no_warehouses_yet_add_your_first_distribution_node")}
      actions={
        <button type="button" className="md-btn md-btn-filled md-typescale-label-large px-4 py-2" onClick={() => setShowForm(true)}>
          Add warehouse
        </button>
      }
    >
      {showForm && (
        <WarehouseForm
          onSave={addWarehouse}
          onCancel={() => setShowForm(false)}
        />
      )}

      {!showForm && (
        <WarehouseList 
          warehouses={warehouses} 
          onAddFirst={() => setShowForm(true)} 
        />
      )}
    </PageChrome>
  );
}
