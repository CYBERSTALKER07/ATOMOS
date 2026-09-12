"use client";

import { usePortalT } from "@/lib/i18n";
import React from "react";

interface DispatchSidebarNavProps {
  activeTab?: string;
  onTabSelect?: (tab: string) => void;
  onRequestClick?: () => void;
}

export const DispatchSidebarNav: React.FC<DispatchSidebarNavProps> = ({
  activeTab = "tracking",
  onTabSelect,
  onRequestClick,
}) => {
  const t = usePortalT();
  return (
    <aside className="w-64 bg-[#121417] text-gray-300 flex flex-col h-full border-r border-gray-800 p-4 select-none">
      {/* Brand Header */}
      <div className="flex items-center gap-3 px-2 py-3 mb-6">
        <div className="w-9 h-9 rounded-full bg-blue-600 flex items-center justify-center text-white font-bold text-lg shadow-md shadow-blue-500/20">
          ➔
        </div>
        <div>
          <h1 className="font-semibold text-white text-base leading-tight">{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.right_direction")}</h1>
          <p className="text-xs text-gray-500">{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.since_2022")}</p>
        </div>
      </div>

      {/* Main Navigation List */}
      <nav className="flex-1 space-y-1 text-sm font-medium">
        <button
          onClick={() => onTabSelect?.("dashboard")}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors ${
            activeTab === "dashboard" ? "bg-blue-600 text-white" : "hover:bg-gray-800/60 text-gray-400 hover:text-white"
          }`}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
          </svg>
          Dashboard
        </button>

        <button
          onClick={() => onTabSelect?.("chats")}
          className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg transition-colors ${
            activeTab === "chats" ? "bg-blue-600 text-white" : "hover:bg-gray-800/60 text-gray-400 hover:text-white"
          }`}
        >
          <div className="flex items-center gap-3">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            Chats
          </div>
          <span className="bg-blue-600 text-white text-xs px-2 py-0.5 rounded-full font-semibold">5</span>
        </button>

        <button
          onClick={() => onTabSelect?.("partners")}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors ${
            activeTab === "partners" ? "bg-blue-600 text-white" : "hover:bg-gray-800/60 text-gray-400 hover:text-white"
          }`}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          Partners
        </button>

        <button
          onClick={() => onTabSelect?.("tracking")}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors ${
            activeTab === "tracking" ? "bg-blue-600 text-white shadow-lg shadow-blue-600/30" : "hover:bg-gray-800/60 text-gray-400 hover:text-white"
          }`}
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7" />
          </svg>
          Tracking
        </button>

        {/* Collapsible Requests Sub-menu */}
        <div className="pt-2">
          <div className="flex items-center justify-between px-3 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
            <span>{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.requests")}</span>
            <span className="bg-gray-800 text-gray-300 text-xs px-2 py-0.5 rounded-full font-medium">3</span>
          </div>
          <div className="pl-4 space-y-1 mt-1 border-l border-gray-800/60 ml-3">
            <button className="w-full text-left px-3 py-1.5 rounded text-gray-400 hover:text-white hover:bg-gray-800/40 text-xs">{t("portal.nav.trucks")}</button>
            <button className="w-full flex items-center justify-between px-3 py-1.5 rounded text-gray-400 hover:text-white hover:bg-gray-800/40 text-xs">
              <span>{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.cargos")}</span>
              <span className="bg-blue-600 text-white text-[10px] px-1.5 py-0.2 rounded-full">2</span>
            </button>
            <button className="w-full text-left px-3 py-1.5 rounded text-gray-400 hover:text-white hover:bg-gray-800/40 text-xs">{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.repair")}</button>
            <button className="w-full text-left px-3 py-1.5 rounded text-gray-400 hover:text-white hover:bg-gray-800/40 text-xs">{t("portal.nav.drivers")}</button>
            <button className="w-full flex items-center justify-between px-3 py-1.5 rounded text-gray-400 hover:text-white hover:bg-gray-800/40 text-xs">
              <span>{t("supplier_portal.dispatch.dispatch_sidebar_nav.text.reports")}</span>
              <span className="bg-blue-600 text-white text-[10px] px-1.5 py-0.2 rounded-full">1</span>
            </button>
          </div>
        </div>

        <button
          onClick={() => onTabSelect?.("analysis")}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-800/60 text-gray-400 hover:text-white transition-colors"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          Analysis
        </button>

        <button
          onClick={() => onTabSelect?.("history")}
          className="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-gray-800/60 text-gray-400 hover:text-white transition-colors"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          History
        </button>
      </nav>

      {/* Bottom Actions & Create New Request Button */}
      <div className="pt-4 border-t border-gray-800 space-y-3">
        <div className="flex items-center justify-around bg-gray-900/80 p-2 rounded-lg text-gray-400">
          <span className="hover:text-white cursor-pointer title='Trucks'">🚚</span>
          <span className="hover:text-white cursor-pointer title='Cargo'">📦</span>
          <span className="hover:text-white cursor-pointer title='Repair'">🛠️</span>
          <span className="hover:text-white cursor-pointer title='Drivers'">👤</span>
          <span className="hover:text-white cursor-pointer title='Reports'">📊</span>
        </div>

        <button
          onClick={onRequestClick}
          className="w-full py-4 border-2 border-dashed border-gray-700 hover:border-blue-500 rounded-xl flex flex-col items-center justify-center text-xs font-semibold text-gray-400 hover:text-blue-400 transition-all bg-gray-900/40 hover:bg-blue-950/20"
        >
          <div className="w-7 h-7 rounded-full bg-blue-600 text-white flex items-center justify-center mb-1 text-sm font-bold shadow-md shadow-blue-600/30">
            +
          </div>
          Create new Request
        </button>
      </div>
    </aside>
  );
};
