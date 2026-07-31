"use client";

import React, { useState, useEffect } from "react";
import { VehicleShipmentCard, PartnerFilterMetric } from "@pegasusx/types";
import { DispatchSidebarNav } from "../../components/dispatch/DispatchSidebarNav";
import { PartnerFilterBar } from "../../components/dispatch/PartnerFilterBar";
import { ShipmentCardGrid } from "../../components/dispatch/ShipmentCardGrid";
import { VehicleCapacityGauge } from "../../components/dispatch/VehicleCapacityGauge";
import { RouteTelemetryMap } from "../../components/dispatch/RouteTelemetryMap";
import { CargoPhotoCarousel } from "../../components/dispatch/CargoPhotoCarousel";

export default function FleetTrackingPage() {
  const [partnerFilters, setPartnerFilters] = useState<PartnerFilterMetric[]>([
    { id: "p1", name: "Lockman", count: 24 },
    { id: "p2", name: "Mertz LLC", count: 22 },
    { id: "p3", name: "Corkery", count: 8 },
    { id: "p4", name: "Kuhn and Sons", count: 5 },
    { id: "p5", name: "Weissnat and Sons", count: 3 },
    { id: "p6", name: "Morissette Inc", count: 2 },
    { id: "p7", name: "Deckow LLC", count: 2 },
  ]);

  const [selectedPartnerId, setSelectedPartnerId] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<"ALL" | "ACTIVE" | "INACTIVE">("ALL");
  const [searchQuery, setSearchQuery] = useState("");

  const [shipments, setShipments] = useState<VehicleShipmentCard[]>([
    {
      id: "v-936383762",
      code: "XR-936383762",
      status: "WAITING",
      vehicle_type: "VAN",
      eta_seconds: 13933,
      distance_miles_left: 36,
      stops_count: 4,
      stops_summary: ["61115 Claudio Walks", "15303 Bohringer Inlet", "457 Saint Veda", "3516 Ryan Valleys"],
      driver_name: "Alex Mercer",
      driver_phone: "+1 555-0142",
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
      stops_summary: ["42047 Verta Ridge", "22920 Shondra Street", "6722 Locascio Mount", "0732 Allen Crossing", "33900 Shonel Street"],
      driver_name: "Marcus Vance",
      driver_phone: "+1 555-0189",
      partner_id: "p2",
      partner_name: "Mertz LLC",
    },
    {
      id: "v-118945307",
      code: "AL-118945307",
      status: "ON_ROUTE",
      vehicle_type: "VAN",
      eta_seconds: 5596,
      distance_miles_left: 90,
      stops_count: 5,
      stops_summary: ["0298 Hermann Corners", "50578 Schuppe Streamway", "88364 Edison Valley", "412 Elenor Way", "4074 Hegmann Pike"],
      driver_name: "Victor Brooks",
      driver_phone: "+1 555-0211",
      partner_id: "p3",
      partner_name: "Corkery",
    },
    {
      id: "v-752069247",
      code: "SD-752069247",
      status: "ON_ROUTE",
      vehicle_type: "SEMI_TRUCK",
      eta_seconds: 5035,
      distance_miles_left: 38,
      stops_count: 5,
      stops_summary: ["2821 Keelie Hills", "36716 Audreanne Date", "399 Lorine Island", "0732 Allen Crossing", "4164 Torrance Plaza"],
      driver_name: "John Miller",
      driver_phone: "+1 555-0199",
      partner_id: "p1",
      partner_name: "Lockman",
    },
    {
      id: "v-752263347",
      code: "SD-752263347",
      status: "ON_ROUTE",
      vehicle_type: "SEMI_TRUCK",
      eta_seconds: 2596,
      distance_miles_left: 98,
      stops_count: 4,
      stops_summary: ["3259 Haley Wells", "34430 Mraz Locks", "50520 Beatty Burg", "61115 Claudio Walks"],
      driver_name: "David Ray",
      driver_phone: "+1 555-0304",
      partner_id: "p4",
      partner_name: "Kuhn and Sons",
    },
    {
      id: "v-916427621",
      code: "XR-916427621",
      status: "ON_ROUTE",
      vehicle_type: "SEMI_TRUCK",
      eta_seconds: 1456,
      distance_miles_left: 112,
      stops_count: 3,
      stops_summary: ["62597 Viviane Harbors", "0732 Allen Crossing", "9667 Huel Drive"],
      driver_name: "Sam West",
      driver_phone: "+1 555-0455",
      partner_id: "p5",
      partner_name: "Weissnat and Sons",
    },
  ]);

  const [selectedShipment, setSelectedShipment] = useState<VehicleShipmentCard>(shipments[3]);
  const [inspectorTab, setInspectorTab] = useState<"shipping" | "vehicle" | "documents" | "company" | "billing">("shipping");

  const filteredShipments = shipments.filter((s) => {
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

  return (
    <div className="flex h-screen bg-[#0a0b0d] text-white overflow-hidden">
      {/* Column 1: Left Navigation Sidebar */}
      <DispatchSidebarNav activeTab="tracking" />

      {/* Column 2: Dispatch Grid & Partner Filters */}
      <main className="flex-1 border-r border-gray-800 p-6 overflow-y-auto custom-scrollbar">
        <PartnerFilterBar
          partnerFilters={partnerFilters}
          selectedPartnerId={selectedPartnerId}
          onPartnerSelect={setSelectedPartnerId}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          totalCount={71}
          activeCount={34}
          inactiveCount={37}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
        />

        <ShipmentCardGrid
          shipments={filteredShipments}
          selectedShipmentId={selectedShipment?.id}
          onSelectShipment={setSelectedShipment}
        />
      </main>

      {/* Column 3: Active Vehicle & Route Inspector */}
      <aside className="w-[580px] bg-[#0c0e11] p-6 overflow-y-auto custom-scrollbar flex flex-col justify-between">
        <div>
          {/* Header Bar */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-bold text-white tracking-wide">{selectedShipment?.code || "SD-752069247"}</h2>
              <span className="flex items-center gap-1.5 bg-green-950/60 text-green-400 border border-green-800/60 px-2.5 py-0.5 rounded-full text-xs font-semibold">
                <span className="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse" />
                {selectedShipment?.status === "ON_ROUTE" ? "On Route" : "Waiting"}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <button className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1 border border-gray-700">
                📞 Call Driver
              </button>
              <button className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1 shadow-md shadow-blue-600/30">
                💬 Chat with Driver
              </button>
            </div>
          </div>

          {/* Sub Navigation Tabs */}
          <div className="flex items-center gap-6 border-b border-gray-800 pb-2 mb-5 text-xs font-medium">
            <button
              onClick={() => setInspectorTab("shipping")}
              className={`pb-2 transition-colors relative ${
                inspectorTab === "shipping" ? "text-white font-bold" : "text-gray-400 hover:text-white"
              }`}
            >
              Shipping Info
              {inspectorTab === "shipping" && <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500 rounded-full" />}
            </button>
            <button
              onClick={() => setInspectorTab("vehicle")}
              className={`pb-2 transition-colors relative ${
                inspectorTab === "vehicle" ? "text-white font-bold" : "text-gray-400 hover:text-white"
              }`}
            >
              Vehicle Info
            </button>
            <button
              onClick={() => setInspectorTab("documents")}
              className={`pb-2 transition-colors relative ${
                inspectorTab === "documents" ? "text-white font-bold" : "text-gray-400 hover:text-white"
              }`}
            >
              Documents
            </button>
            <button
              onClick={() => setInspectorTab("company")}
              className={`pb-2 transition-colors relative ${
                inspectorTab === "company" ? "text-white font-bold" : "text-gray-400 hover:text-white"
              }`}
            >
              Company
            </button>
            <button
              onClick={() => setInspectorTab("billing")}
              className={`pb-2 transition-colors relative ${
                inspectorTab === "billing" ? "text-white font-bold" : "text-gray-400 hover:text-white"
              }`}
            >
              Billing
            </button>
          </div>

          {/* Current Truck Capacity Visualizer */}
          <VehicleCapacityGauge
            capacityPercentage={selectedShipment?.vehicle_type === "SEMI_TRUCK" ? 59 : 78}
            vehicleCode={selectedShipment?.code}
          />

          {/* Route & Telemetry Map */}
          <RouteTelemetryMap
            vehicleCode={selectedShipment?.code}
            etaSeconds={selectedShipment?.eta_seconds}
            distanceMilesLeft={selectedShipment?.distance_miles_left}
          />

          {/* Cargo Photo Reports */}
          <CargoPhotoCarousel />
        </div>

        {/* Route Requests Footer */}
        <div className="pt-4 border-t border-gray-800 text-xs text-gray-500 flex justify-between items-center">
          <span>Route Requests: 0 Pending</span>
          <span className="text-gray-400">PegasusX Live Telemetry Stream</span>
        </div>
      </aside>
    </div>
  );
}
