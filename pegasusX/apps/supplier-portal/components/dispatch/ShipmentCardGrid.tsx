"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";
import { VehicleShipmentCard } from "@pegasusx/types";
import { ShipmentCard } from "./ShipmentCard";

interface ShipmentCardGridProps {
  shipments: VehicleShipmentCard[];
  selectedShipmentId?: string;
  onSelectShipment: (shipment: VehicleShipmentCard) => void;
}

export const ShipmentCardGrid: React.FC<ShipmentCardGridProps> = ({
  shipments,
  selectedShipmentId,
  onSelectShipment,
}) => {
  const t = usePortalT();
  if (shipments.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-center border border-dashed border-gray-800 rounded-xl bg-[#121417]/50">
        <svg className="w-12 h-12 text-gray-600 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p className="text-sm font-semibold text-gray-400">{t("supplier_portal.dispatch.shipment_card_grid.text.no_active_shipments_matching_criteria")}</p>
        <p className="text-xs text-gray-600 mt-1">{t("supplier_portal.dispatch.shipment_card_grid.text.adjust_filters_or_select_a_different_partner")}</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 select-none max-h-[calc(100vh-280px)] overflow-y-auto pr-1 custom-scrollbar">
      {shipments.map((shipment) => (
        <ShipmentCard
          key={shipment.id}
          shipment={shipment}
          isSelected={selectedShipmentId === shipment.id}
          onSelect={() => onSelectShipment(shipment)}
        />
      ))}
    </div>
  );
};
