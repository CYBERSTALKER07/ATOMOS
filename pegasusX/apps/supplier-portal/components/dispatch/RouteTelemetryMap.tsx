"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";

interface RouteTelemetryMapProps {
  vehicleCode?: string | null;
  etaSeconds?: number | null;
  distanceMilesLeft?: number | null;
  onChangeRoute?: () => void;
  /** When false/omitted with no ETA, show empty state instead of decorative map. */
  hasLiveRoute?: boolean;
}

export const RouteTelemetryMap: React.FC<RouteTelemetryMapProps> = ({
  vehicleCode = null,
  etaSeconds = null,
  distanceMilesLeft = null,
  onChangeRoute,
  hasLiveRoute,
}) => {
  const t = usePortalT();
  const live =
    hasLiveRoute === true ||
    (hasLiveRoute !== false && etaSeconds != null && Number.isFinite(etaSeconds));

  const formatTime = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hrs.toString().padStart(2, "0")}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  return (
    <div className="bg-[#121417] border border-gray-800 rounded-2xl p-5 mb-5 select-none shadow-lg">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-300 tracking-wide uppercase">{t("supplier_portal.analytics.route_performance.text.route")}</h3>
        <div className="flex items-center gap-4">
          {live && etaSeconds != null ? (
            <span className="text-sm font-bold text-white">{formatTime(etaSeconds)}</span>
          ) : (
            <span className="text-xs text-gray-500">{t("supplier_portal.dispatch.route_telemetry_map.text.no_live_eta")}</span>
          )}
          {live && distanceMilesLeft != null ? (
            <span className="text-xs text-gray-400 font-medium">{distanceMilesLeft} mi. left</span>
          ) : null}
          {onChangeRoute ? (
            <button
              onClick={onChangeRoute}
              className="px-3 py-1 bg-gray-800 hover:bg-gray-700 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1 border border-gray-700"
            >
              Change Route
            </button>
          ) : null}
        </div>
      </div>

      {!live ? (
        <div className="relative w-full h-64 bg-[#14171d] rounded-xl border border-gray-800 flex items-center justify-center px-6">
          <p className="text-xs text-gray-500 text-center">
            No live route telemetry
            {vehicleCode?.trim() ? ` for ${vehicleCode.trim()}` : ""}.
            Tracking appears when the driver is on an active dispatch.
          </p>
        </div>
      ) : (
        <div className="relative w-full h-64 bg-[#14171d] rounded-xl border border-gray-800 overflow-hidden">
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <p className="text-sm font-semibold text-white">
              {vehicleCode?.trim() || "Vehicle"} · live
            </p>
            <p className="text-xs text-gray-400">
              Map tiles / polyline bind when route GPS stream is available for this dispatch.
            </p>
          </div>
        </div>
      )}
    </div>
  );
};
