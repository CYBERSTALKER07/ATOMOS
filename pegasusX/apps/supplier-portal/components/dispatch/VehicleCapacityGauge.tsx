"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";

interface VehicleCapacityGaugeProps {
  capacityPercentage?: number | null;
  vehicleCode?: string | null;
}

export const VehicleCapacityGauge: React.FC<VehicleCapacityGaugeProps> = ({
  capacityPercentage = null,
  vehicleCode = null,
}) => {
  const t = usePortalT();
  const hasData = capacityPercentage != null && Number.isFinite(capacityPercentage);
  const pct = hasData ? Math.max(0, Math.min(100, Math.round(capacityPercentage))) : 0;

  return (
    <div className="bg-[#121417] border border-gray-800 rounded-2xl p-5 mb-5 select-none shadow-lg">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-300 tracking-wide uppercase">{t("supplier_portal.dispatch.vehicle_capacity_gauge.text.current_truck_capacity")}</h3>
        <span className="text-xs font-bold text-blue-400 bg-blue-950/60 border border-blue-800/60 px-2.5 py-0.5 rounded-full">
          Volumetric Load Factor
        </span>
      </div>

      {!hasData ? (
        <div className="relative w-full h-44 bg-[#181b20] border border-gray-800 rounded-xl flex items-center justify-center p-4">
          <p className="text-xs text-gray-500 text-center">
            No live capacity for {vehicleCode?.trim() || "this vehicle"} yet.
          </p>
        </div>
      ) : (
        <div className="relative w-full h-44 bg-[#181b20] border border-gray-800 rounded-xl flex items-center justify-center p-4 overflow-hidden">
          <div className="relative w-full max-w-xl h-32 flex items-center">
            <div className="w-24 h-24 bg-[#232832] border-2 border-gray-700 rounded-l-2xl relative flex flex-col justify-between p-2 shrink-0 z-10">
              <div className="w-10 h-8 bg-blue-950/80 border border-blue-600/40 rounded-lg self-end" />
              <div className="w-full h-3 bg-gray-800 rounded" />
              <div className="absolute -bottom-3 left-4 w-7 h-7 rounded-full bg-gray-900 border-4 border-gray-700 shadow-md" />
            </div>

            <div className="flex-1 h-28 bg-[#1a1d24] border-2 border-l-0 border-gray-700 rounded-r-xl relative overflow-hidden flex items-center">
              <div
                className="h-full bg-gradient-to-r from-blue-600 via-blue-500 to-indigo-500 transition-all duration-700 flex items-center justify-center relative shadow-inner"
                style={{ width: `${pct}%` }}
              >
                <div className="absolute inset-0 bg-[radial-gradient(#fff_1px,transparent_1px)] [background-size:12px_12px] opacity-10" />
              </div>

              <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                <span className="text-4xl font-extrabold text-white tracking-wider drop-shadow-[0_2px_10px_rgba(0,0,0,0.8)]">
                  {pct}%
                </span>
              </div>

              <div className="absolute -bottom-3 right-6 flex gap-2 z-10">
                <div className="w-7 h-7 rounded-full bg-gray-900 border-4 border-gray-700 shadow-md" />
                <div className="w-7 h-7 rounded-full bg-gray-900 border-4 border-gray-700 shadow-md" />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
