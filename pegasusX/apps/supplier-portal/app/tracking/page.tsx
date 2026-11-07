"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { DispatchSidebarNav } from "../../components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../components/dispatch/CargoPhotoCarousel";

const api = createSupplierApi();

export default function FleetTrackingPage() {
  const t = usePortalT();
  const [partnerFilters, setPartnerFilters] = useState<PartnerFilterMetric[]>([]);
  const [selectedPartnerId, setSelectedPartnerId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<"ALL" | "ACTIVE" | "INACTIVE">("ALL");
  const [searchQuery, setSearchQuery] = useState("");
  const [shipments, setShipments] = useState<VehicleShipmentCard[]>([]);
  const [selectedShipment, setSelectedShipment] = useState<VehicleShipmentCard | null>(null);
  const [inspectorTab, setInspectorTab] = useState<"shipping" | "vehicle" | "documents" | "company" | "billing">("shipping");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [counts, setCounts] = useState({ total: 0, active: 0, inactive: 0 });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const overview = await api.getFleetDispatchOverview("supplier", {
        partner_id: selectedPartnerId,
        status_filter: statusFilter === "ALL" ? undefined : statusFilter,
        search_query: searchQuery.trim() || undefined,
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
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_fleet_tracking"));
    } finally {
      setLoading(false);
    }
  }, [selectedPartnerId, statusFilter, searchQuery]);

  useEffect(() => {
    void load();
  }, [load]);

  const filteredShipments = useMemo(() => {
    return shipments.filter((s) => {
      if (selectedPartnerId && s.partner_id !== selectedPartnerId) return false;
      if (statusFilter === "ACTIVE" && s.status !== "ON_ROUTE") return false;
      if (statusFilter === "INACTIVE" && s.status !== "WAITING") return false;
      if (searchQuery.trim() !== "") {
        const q = searchQuery.toLowerCase();
        const matchCode = s.code.toLowerCase().includes(q);
        const matchDriver = s.driver_name?.toLowerCase().includes(q);
        if (!matchCode && !matchDriver) return false;
      }
      return true;
    });
  }, [shipments, selectedPartnerId, statusFilter, searchQuery]);

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
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
        />

        {loading && shipments.length === 0 ? (
          <p className="text-sm text-gray-500 mt-8">{t("supplier_portal.tracking.text.loading_live_fleet")}</p>
        ) : error && shipments.length === 0 ? (
          <p className="text-sm text-red-400 mt-8">{error}</p>
        ) : filteredShipments.length === 0 ? (
          <p className="text-sm text-gray-500 mt-8">
            No active fleet shipments. Cards appear when drivers are on dispatch.
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
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-bold text-white tracking-wide">
                {selectedShipment?.code || "No vehicle selected"}
              </h2>
              {selectedShipment ? (
                <span className="flex items-center gap-1.5 bg-green-950/60 text-green-400 border border-green-800/60 px-2.5 py-0.5 rounded-full text-xs font-semibold">
                  <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
                  {selectedShipment.status === "ON_ROUTE" ? "On Route" : "Waiting"}
                </span>
              ) : null}
            </div>
          </div>

          <div className="flex items-center gap-6 border-b border-gray-800 pb-2 mb-5 text-xs font-medium">
            {(["shipping", "vehicle", "documents", "company", "billing"] as const).map((tab) => (
              <button
                key={tab}
                type="button"
                onClick={() => setInspectorTab(tab)}
                className={`pb-2 transition-colors relative capitalize ${
                  inspectorTab === tab ? "text-white font-bold" : "text-gray-400 hover:text-white"
                }`}
              >
                {tab === "shipping" ? "Shipping Info" : tab === "vehicle" ? "Vehicle Info" : tab}
                {inspectorTab === tab && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 rounded-full" />
                )}
              </button>
            ))}
          </div>

          <VehicleCapacityGauge
            capacityPercentage={null}
            vehicleCode={selectedShipment?.code ?? null}
          />
          <RouteTelemetryMap
            vehicleCode={selectedShipment?.code ?? null}
            etaSeconds={selectedShipment?.eta_seconds ?? null}
            distanceMilesLeft={selectedShipment?.distance_miles_left ?? null}
            hasLiveRoute={Boolean(selectedShipment)}
          />
          <CargoPhotoCarousel photos={[]} />
        </div>

        <div className="pt-4 border-t border-gray-800 text-xs text-gray-500 flex justify-between items-center">
          <span>{t("supplier_portal.tracking.text.live_fleet_from_dispatch_api")}</span>
          <button type="button" className="text-gray-400 hover:text-white" onClick={() => void load()}>
            Refresh
          </button>
        </div>
      </aside>
    </div>
  );
}
