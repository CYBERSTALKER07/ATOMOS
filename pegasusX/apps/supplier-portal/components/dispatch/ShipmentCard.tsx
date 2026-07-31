"use client";

import React from "react";
import { VehicleShipmentCard } from "@pegasusx/types";

interface ShipmentCardProps {
  shipment: VehicleShipmentCard;
  isSelected?: boolean;
  onSelect?: () => void;
}

export const ShipmentCard: React.FC<ShipmentCardProps> = ({
  shipment,
  isSelected = false,
  onSelect,
}) => {
  const formatTime = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return `${hrs.toString().padStart(2, "0")}:${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  const isRoute = shipment.status === "ON_ROUTE";

  return (
    <div
      onClick={onSelect}
      className={`bg-[#121417] border rounded-xl p-4 cursor-pointer transition-all duration-200 relative overflow-hidden group ${
        isSelected
          ? "border-blue-500 ring-2 ring-blue-500/30 bg-[#161a20] shadow-xl shadow-blue-500/10"
          : "border-gray-800/80 hover:border-gray-700 hover:bg-[#16181d]"
      }`}
    >
      {/* Header: Code & Status Pill */}
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-bold text-white text-sm tracking-wide">{shipment.code}</h3>
        <div
          className={`px-2.5 py-0.5 rounded-full text-[11px] font-semibold flex items-center gap-1.5 ${
            isRoute
              ? "bg-green-950/60 text-green-400 border border-green-800/60"
              : "bg-amber-950/60 text-amber-400 border border-amber-800/60"
          }`}
        >
          <span className={`w-1.5 h-1.5 rounded-full ${isRoute ? "bg-green-400 animate-pulse" : "bg-amber-400"}`} />
          {isRoute ? "On Route" : "Waiting"}
        </div>
      </div>

      {/* Stats & Stop list preview */}
      <div className="grid grid-cols-2 gap-2 text-xs mb-3">
        <div>
          <p className="text-gray-200 font-bold">{formatTime(shipment.eta_seconds)}</p>
          <p className="text-gray-500 text-[11px] mt-0.5">{shipment.distance_miles_left} mi. left</p>
        </div>
        <div className="space-y-1 overflow-hidden">
          {shipment.stops_summary.slice(0, 3).map((stop, idx) => (
            <div key={idx} className="flex items-center gap-1 text-gray-400 text-[10px] truncate">
              <span className="w-1 h-1 rounded-full bg-blue-500 shrink-0" />
              <span className="truncate">{stop}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 3D Vehicle Vector Graphic */}
      <div className="h-24 w-full bg-[#181b20] rounded-lg flex items-center justify-center border border-gray-800/40 p-2 relative">
        {shipment.vehicle_type === "SEMI_TRUCK" ? (
          <svg className="w-36 h-20 text-gray-400" viewBox="0 0 200 90" fill="none">
            {/* Trailer Body */}
            <rect x="10" y="20" width="130" height="45" rx="4" fill="#20242c" stroke="#3a3f4d" strokeWidth="2" />
            <path d="M10 20 H140 V65 H10 Z" fill="url(#trailerGrad)" opacity="0.4" />
            <line x1="140" y1="20" x2="140" y2="65" stroke="#3a3f4d" strokeWidth="2" />
            {/* Cab */}
            <path d="M142 35 L175 35 L188 52 L188 65 L142 65 Z" fill="#2c323f" stroke="#485065" strokeWidth="2" />
            {/* Cab Window */}
            <path d="M165 40 L175 40 L183 50 L165 50 Z" fill="#121417" stroke="#3a3f4d" strokeWidth="1" />
            {/* Wheels */}
            <circle cx="35" cy="67" r="9" fill="#121417" stroke="#485065" strokeWidth="3" />
            <circle cx="58" cy="67" r="9" fill="#121417" stroke="#485065" strokeWidth="3" />
            <circle cx="120" cy="67" r="9" fill="#121417" stroke="#485065" strokeWidth="3" />
            <circle cx="168" cy="67" r="9" fill="#121417" stroke="#485065" strokeWidth="3" />
          </svg>
        ) : (
          <svg className="w-32 h-18 text-gray-400" viewBox="0 0 180 80" fill="none">
            {/* Van Body */}
            <path d="M15 25 L120 25 L145 42 L160 45 L160 60 L15 60 Z" fill="#20242c" stroke="#3a3f4d" strokeWidth="2" />
            {/* Van Window */}
            <path d="M125 30 L140 42 L125 42 Z" fill="#121417" stroke="#3a3f4d" strokeWidth="1" />
            {/* Van Wheels */}
            <circle cx="40" cy="62" r="8" fill="#121417" stroke="#485065" strokeWidth="3" />
            <circle cx="135" cy="62" r="8" fill="#121417" stroke="#485065" strokeWidth="3" />
          </svg>
        )}
      </div>
    </div>
  );
};
