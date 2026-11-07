"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyCoverageCity, SupplierTopologyResponse, SupplierTopologyUpdateRequest } from "@pegasusx/types";
import { type LocationValue } from "@/components/LocationPicker";
import { PageChrome } from "@/components/PageChrome";
import { WarehouseForm } from "./components/WarehouseForm";
import { WarehouseList } from "./components/WarehouseList";
import { factoryToTopologyInput, warehouseToTopologyInput } from "@/lib/topology";

const api = createSupplierApi();

export default function WarehousesPage() {
  const t = usePortalT();
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [modes, setModes] = useState<Record<string, string>>({});

  const load = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierTopology()
      .then(async (topo) => {
        setTopology(topo);
        const next: Record<string, string> = {};
        await Promise.all(
          (topo.warehouses ?? []).map(async (row) => {
            if (!row.warehouse_id) return;
            try {
              const cov = await api.getWarehouseCoverage(row.warehouse_id);
              next[row.warehouse_id] = cov.mode;
            } catch {
              next[row.warehouse_id] = "COUNTRY_CLOSEST";
            }
          }),
        );
        setModes(next);
      })
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_warehouses_failed")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [refreshTick]);

  const addWarehouse = async (
    name: string,
    location: LocationValue,
    radius: number,
    extras: { country_code: string; coverage_cities: SupplierTopologyCoverageCity[]; primary_factory_id: string },
  ) => {
    if (!topology) return;
    
    const body: SupplierTopologyUpdateRequest = {
      warehouses: [
        ...topology.warehouses.map(warehouseToTopologyInput),
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
          country_code: extras.country_code || undefined,
          coverage_cities: extras.coverage_cities.length ? extras.coverage_cities : undefined,
          primary_factory_id: extras.primary_factory_id || undefined,
          assigned_factory_ids: extras.primary_factory_id ? [extras.primary_factory_id] : undefined,
        },
      ],
      factories: topology.factories.map(factoryToTopologyInput),
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
          factoryOptions={(topology?.factories ?? []).map((f) => ({ id: f.factory_id, name: f.name }))}
        />
      )}

      {!showForm && (
        <WarehouseList
          warehouses={warehouses}
          modes={modes}
          onAddFirst={() => setShowForm(true)}
        />
      )}
    </PageChrome>
  );
}
