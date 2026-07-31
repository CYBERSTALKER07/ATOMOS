"use client";

import React, { useState } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { DispatchSidebarNav } from "../../../supplier-portal/components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../../supplier-portal/components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../../supplier-portal/components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../../supplier-portal/components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../../supplier-portal/components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../../supplier-portal/components/dispatch/CargoPhotoCarousel";

export default function WarehouseTrackingPage() {
  const [partnerFilters] = useState<PartnerFilterMetric[]>([
    { id: "p1", name: "Lockman", count: 24 },
    { id: "p2", name: "Mertz LLC", count: 22 },
    { id: "p3", name: "Corkery", count: 8 },
    { id: "p4", name: "Kuhn and Sons", count: 5 },
  ]);

  const [selectedPartnerId, setSelectedPartnerId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<"ALL" | "ACTIVE" | "INACTIVE">("ALL");

  const [shipments] = useState<VehicleShipmentCard[]>([
    {
      id: "v-752069247",
      code: "SD-752069247",
      status: "ON_ROUTE",
      vehicle_type: "SEMI_TRUCK",
      eta_seconds: 5035,
      distance_miles_left: 38,
      stops_count: 5,
      stops_summary: ["2821 Keelie Hills", "36716 Audreanne Date", "399 Lorine Island", "0732 Allen Crossing"],
      driver_name: "John Miller",
      driver_phone: "+1 555-0199",
      partner_id: "p1",
      partner_name: "Lockman",
    },
    {
      id: "v-113949207",
      code: "AL-113949207",
      status: "WAITING",
      vehicle_type: "VAN",
      eta_seconds: 8233,
      distance_miles_left: 64,
      stops_count: 5,
      stops_summary: ["42047 Verta Ridge", "22920 Shondra Street", "6722 Locascio Mount"],
      driver_name: "Marcus Vance",
      driver_phone: "+1 555-0189",
      partner_id: "p2",
      partner_name: "Mertz LLC",
    },
  ]);

  const [selectedShipment, setSelectedShipment] = useState<VehicleShipmentCard>(shipments[0]);

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
          totalCount={46}
          activeCount={20}
          inactiveCount={26}
        />

        <ShipmentCardGrid
          shipments={shipments}
          selectedShipmentId={selectedShipment?.id}
          onSelectShipment={setSelectedShipment}
        />
      </main>

      <aside className="w-[580px] bg-[#0c0e11] p-6 overflow-y-auto custom-scrollbar flex flex-col justify-between">
        <div>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-bold text-white tracking-wide">Warehouse Dock Dispatch ({selectedShipment?.code})</h2>
            <button className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold rounded-lg">
              Verify Dock Loading
            </button>
          </div>

          <VehicleCapacityGauge capacityPercentage={59} vehicleCode={selectedShipment?.code} />
          <RouteTelemetryMap vehicleCode={selectedShipment?.code} />
          <CargoPhotoCarousel />
        </div>
      </aside>
    </div>
  );
}
