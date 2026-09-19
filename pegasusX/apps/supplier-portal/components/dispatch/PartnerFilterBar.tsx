"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";
import { PartnerFilterMetric } from "@pegasusx/types";

interface PartnerFilterBarProps {
  partnerFilters: PartnerFilterMetric[];
  selectedPartnerId?: string;
  onPartnerSelect: (partnerId?: string) => void;
  statusFilter: "ALL" | "ACTIVE" | "INACTIVE";
  onStatusFilterChange: (status: "ALL" | "ACTIVE" | "INACTIVE") => void;
  totalCount: number;
  activeCount: number;
  inactiveCount: number;
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
}

export const PartnerFilterBar: React.FC<PartnerFilterBarProps> = ({
  partnerFilters,
  selectedPartnerId,
  onPartnerSelect,
  statusFilter,
  onStatusFilterChange,
  totalCount,
  activeCount,
  inactiveCount,
  searchQuery = "",
  onSearchChange,
}) => {
  const t = usePortalT();
  return (
    <div className="space-y-4 mb-4 select-none">
      {/* Title & Search Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white tracking-tight">{t("portal.nav.tracking")}</h2>
        <div className="relative">
          <input
            type="text"
            placeholder={t("supplier_portal.dispatch.partner_filter_bar.text.search_vehicle_or_route")}
            value={searchQuery}
            onChange={(e) => onSearchChange?.(e.target.value)}
            className="bg-[#1a1d21] text-xs text-white placeholder-gray-500 pl-8 pr-3 py-2 rounded-lg border border-gray-800 focus:outline-none focus:border-blue-500 transition-colors w-52"
          />
          <svg className="w-4 h-4 text-gray-500 absolute left-2.5 top-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Filter by Partners */}
      <div>
        <p className="text-xs text-gray-400 font-medium mb-2 uppercase tracking-wider">{t("supplier_portal.dispatch.partner_filter_bar.text.filter_by_partners")}</p>
        <div className="flex flex-wrap gap-2">
          {partnerFilters.map((partner) => {
            const isSelected = selectedPartnerId === partner.id;
            return (
              <button
                key={partner.id}
                onClick={() => onPartnerSelect(isSelected ? undefined : partner.id)}
                className={`px-3 py-1.5 rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all ${
                  isSelected
                    ? "bg-blue-600 text-white shadow-md shadow-blue-600/30"
                    : "bg-[#1a1d21] text-gray-300 hover:bg-gray-800 border border-gray-800"
                }`}
              >
                <span>{partner.name}</span>
                <span className={`text-[10px] px-1.5 py-0.2 rounded ${isSelected ? "bg-blue-700 text-white" : "bg-gray-800 text-gray-400"}`}>
                  {partner.count}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Show Status Filter */}
      <div className="flex items-center gap-2 pt-1">
        <span className="text-xs text-gray-400 font-medium mr-1 uppercase tracking-wider">{t("supplier_portal.dispatch.partner_filter_bar.text.show")}</span>
        <button
          onClick={() => onStatusFilterChange("ALL")}
          className={`px-3 py-1 rounded-md text-xs font-semibold transition-all ${
            statusFilter === "ALL" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "bg-[#1a1d21] text-gray-400 hover:text-white"
          }`}
        >
          All <span className="ml-1 text-[10px] opacity-80">{totalCount}</span>
        </button>
        <button
          onClick={() => onStatusFilterChange("ACTIVE")}
          className={`px-3 py-1 rounded-md text-xs font-semibold transition-all ${
            statusFilter === "ACTIVE" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "bg-[#1a1d21] text-gray-400 hover:text-white"
          }`}
        >
          Active <span className="ml-1 text-[10px] opacity-80">{activeCount}</span>
        </button>
        <button
          onClick={() => onStatusFilterChange("INACTIVE")}
          className={`px-3 py-1 rounded-md text-xs font-semibold transition-all ${
            statusFilter === "INACTIVE" ? "bg-blue-600 text-white shadow-md shadow-blue-600/30" : "bg-[#1a1d21] text-gray-400 hover:text-white"
          }`}
        >
          Inactive <span className="ml-1 text-[10px] opacity-80">{inactiveCount}</span>
        </button>
      </div>
    </div>
  );
};
