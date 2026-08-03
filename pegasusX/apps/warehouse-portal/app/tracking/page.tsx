"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { createWarehouseApi } from "@/lib/api";
import { DispatchSidebarNav } from "../../../supplier-portal/components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../../supplier-portal/components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../../supplier-portal/components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../../supplier-portal/components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../../supplier-portal/components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../../supplier-portal/components/dispatch/CargoPhotoCarousel";

const api = createWarehouseApi();

export default function WarehouseTrackingPage() {
  const [partnerFilters, setPartnerFilters] = useState<PartnerFilterMetric[]>([]);
  const [selectedPartnerId, setSelectedPartnerId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<"ALL" | "ACTIVE" | "INACTIVE">("ALL");
  const [shipments, setShipments] = useState<VehicleShipmentCard[]>([]);
  const [selectedShipment, setSelectedShipment] = useState<VehicleShipmentCard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [counts, setCounts] = useState({ total: 0, active: 0, inactive: 0 });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const overview = await api.getFleetDispatchOverview("warehouse", {
        partner_id: selectedPartnerId,
        status_filter: statusFilter === "ALL" ? undefined : statusFilter,
      });
      const rows = overview.shipments ?? [];
      setShipments(rows);
      setPartnerFilters(overview.partner_filters ?? []);
      setCounts({
        total: overview.total_count ?? rows.length,
        active: overview.active_count ?? rows.filter((s) => s.status === "ON_ROUTE").length,
        inactive: overview.inactive_count ?? rows.filter((s) => s.status !== "ON_ROUTE").length,
      });
      setSelectedShipment((prev) => {
        if (prev && rows.some((r) => r.id === prev.id)) {
          return rows.find((r) => r.id === prev.id) ?? null;
        }
        return rows[0] ?? null;
      });
    } catch (err) {
      setShipments([]);
      setPartnerFilters([]);
      setSelectedShipment(null);
      setCounts({ total: 0, active: 0, inactive: 0 });
      setError(err instanceof Error ? err.message : "Failed to load warehouse tracking");
    } finally {
      setLoading(false);
    }
  }, [selectedPartnerId, statusFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const filteredShipments = useMemo(() => {
    return shipments.filter((s) => {
      if (selectedPartnerId && s.partner_id !== selectedPartnerId) return false;
      if (statusFilter === "ACTIVE" && s.status !== "ON_ROUTE") return false;
      if (statusFilter === "INACTIVE" && s.status !== "WAITING") return false;
      return true;
    });
  }, [shipments, selectedPartnerId, statusFilter]);

  return (
    <div className="flex h-screen bg-[#0a0b0d] text-white overflow-hidden">
      <DispatchSidebarNav activeTab="tracking" />

      <main className="flex-1 border-r border-gray-800 p-6 overflow-y-auto custom-scrollbar">
        <PartnerFilterBar
          partnerFilters={partnerFilters}
          selectedPartnerId={selectedPartnerId}
          onPartnerSelect={setSelectedPartnerId}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          totalCount={counts.total}
          activeCount={counts.active}
          inactiveCount={counts.inactive}
        />

        {loading && shipments.length === 0 ? (
          <p className="text-sm text-gray-500 mt-8">Loading warehouse fleet…</p>
        ) : error && shipments.length === 0 ? (
          <p className="text-sm text-red-400 mt-8">{error}</p>
        ) : filteredShipments.length === 0 ? (
          <p className="text-sm text-gray-500 mt-8">
            No dock dispatches yet. Cards appear when warehouse fleet is on route.
          </p>
        ) : (
          <ShipmentCardGrid
            shipments={filteredShipments}
            selectedShipmentId={selectedShipment?.id}
            onSelectShipment={setSelectedShipment}
          />
        )}
      </main>

      <aside className="w-[580px] bg-[#0c0e11] p-6 overflow-y-auto custom-scrollbar flex flex-col justify-between">
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-white tracking-wide">
              Warehouse Dock Dispatch ({selectedShipment?.code || "—"})
            </h2>
          </div>

          <VehicleCapacityGauge capacityPercentage={null} vehicleCode={selectedShipment?.code ?? null} />
          <RouteTelemetryMap
            vehicleCode={selectedShipment?.code ?? null}
            etaSeconds={selectedShipment?.eta_seconds ?? null}
            distanceMilesLeft={selectedShipment?.distance_miles_left ?? null}
            hasLiveRoute={Boolean(selectedShipment)}
          />
          <CargoPhotoCarousel photos={[]} />
        </div>
      </aside>
    </div>
  );
}
