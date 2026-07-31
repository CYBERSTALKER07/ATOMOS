"use client";

import React from "react";

interface RouteTelemetryMapProps {
  vehicleCode?: string;
  etaSeconds?: number;
  distanceMilesLeft?: number;
  onChangeRoute?: () => void;
}

export const RouteTelemetryMap: React.FC<RouteTelemetryMapProps> = ({
  vehicleCode = "SD-752069247",
  etaSeconds = 5035,
  distanceMilesLeft = 38,
  onChangeRoute,
}) => {
  const formatTime = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hrs.toString().padStart(2, "0")}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  return (
    <div className="bg-[#121417] border border-gray-800 rounded-2xl p-5 mb-5 select-none shadow-lg">
      {/* Route Header Info */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-300 tracking-wide uppercase">Route</h3>
        <div className="flex items-center gap-4">
          <span className="text-sm font-bold text-white">{formatTime(etaSeconds)}</span>
          <span className="text-xs text-gray-400 font-medium">{distanceMilesLeft} mi. left</span>
          <button
            onClick={onChangeRoute}
            className="px-3 py-1 bg-gray-800 hover:bg-gray-700 text-white text-xs font-semibold rounded-lg transition-colors flex items-center gap-1 border border-gray-700"
          >
            <span>✏️</span> Change Route
          </button>
        </div>
      </div>

      {/* Dark Map Mock Canvas / SVG representation */}
      <div className="relative w-full h-64 bg-[#14171d] rounded-xl border border-gray-800 overflow-hidden group">
        {/* Dark Grid & Water Pattern */}
        <div className="absolute inset-0 bg-[#0e1014] opacity-90">
          <svg className="w-full h-full opacity-30" width="100%" height="100%">
            <defs>
              <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                <path d="M 40 0 L 0 0 0 40" fill="none" stroke="#252a34" strokeWidth="1" />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid)" />
            {/* Water shapes / River mockup */}
            <path d="M 120 0 C 180 100, 150 180, 220 260" fill="none" stroke="#1a2536" strokeWidth="24" />
          </svg>
        </div>

        {/* Map Labels */}
        <div className="absolute top-4 left-4 text-[10px] text-gray-600 font-mono tracking-widest uppercase">
          New Jersey / New York Metro Sector
        </div>

        {/* Polyline Route */}
        <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 500 250">
          {/* Outer glow route */}
          <path
            d="M 100 200 Q 220 180 260 120 T 400 60"
            fill="none"
            stroke="#2563eb"
            strokeWidth="6"
            strokeLinecap="round"
            opacity="0.4"
          />
          {/* Inner core route */}
          <path
            d="M 100 200 Q 220 180 260 120 T 400 60"
            fill="none"
            stroke="#3b82f6"
            strokeWidth="3"
            strokeDasharray="6 4"
            strokeLinecap="round"
          />
          {/* Waypoint Pins */}
          <circle cx="100" cy="200" r="6" fill="#3b82f6" stroke="#ffffff" strokeWidth="2" />
          <circle cx="260" cy="120" r="7" fill="#2563eb" stroke="#ffffff" strokeWidth="2.5" className="animate-ping" />
          <circle cx="260" cy="120" r="7" fill="#3b82f6" stroke="#ffffff" strokeWidth="2.5" />
          <circle cx="400" cy="60" r="6" fill="#10b981" stroke="#ffffff" strokeWidth="2" />
        </svg>

        {/* Driver GPS Floating Marker */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-blue-600 text-white text-[10px] font-bold px-2 py-1 rounded-full shadow-lg shadow-blue-500/50 flex items-center gap-1 border border-white">
          <span className="w-2 h-2 rounded-full bg-white animate-pulse" />
          <span>SD-752069247 (Live GPS)</span>
        </div>

        {/* Map Control Buttons */}
        <div className="absolute bottom-3 right-3 flex flex-col gap-1">
          <button className="w-7 h-7 bg-gray-900/90 hover:bg-gray-800 text-white text-xs font-bold rounded flex items-center justify-center border border-gray-700 shadow">
            +
          </button>
          <button className="w-7 h-7 bg-gray-900/90 hover:bg-gray-800 text-white text-xs font-bold rounded flex items-center justify-center border border-gray-700 shadow">
            -
          </button>
        </div>
      </div>
    </div>
  );
};
