"use client";

import React, { useState } from "react";
import { useAdminWS } from "@/lib/ws-provider";
import { useAdminToken } from "@/lib/session";
import { Keyboard } from "lucide-react";

export function AdminHeader({
  onOpenSidebar,
  onOpenCheatsheet,
}: {
  onOpenSidebar: () => void;
  onOpenCheatsheet?: () => void;
}) {
  const { status, lastHeartbeat, lastEventTime, reconnect } = useAdminWS();
  const { logout } = useAdminToken();
  const [showWsPopover, setShowWsPopover] = useState(false);

  const statusBadge = () => {
    switch (status) {
      case "connected":
        return {
          label: "Live Stream",
          dotColor: "bg-emerald-500",
          textColor: "text-emerald-700",
          bgColor: "bg-emerald-50 border-emerald-200",
        };
      case "reconnecting":
      case "connecting":
        return {
          label: "Reconnecting",
          dotColor: "bg-amber-500 animate-pulse",
          textColor: "text-amber-700",
          bgColor: "bg-amber-50 border-amber-200",
        };
      case "disconnected":
      default:
        return {
          label: "Offline Mode",
          dotColor: "bg-red-500",
          textColor: "text-red-700",
          bgColor: "bg-red-50 border-red-200",
        };
    }
  };

  const badge = statusBadge();

  return (
    <header
      className="sticky top-0 z-30 flex h-16 items-center justify-between border-b bg-white/90 px-4 backdrop-blur sm:px-6 select-none"
      data-tauri-drag-region
    >
      <div className="flex items-center gap-3" data-tauri-no-drag>
        <button
          type="button"
          onClick={onOpenSidebar}
          className="rounded-lg p-2 text-gray-500 hover:bg-gray-100 lg:hidden"
          aria-label="Open sidebar"
        >
          <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <span className="text-xs font-semibold uppercase tracking-wider text-gray-600 hidden sm:inline-block">
          Network Operations Center · Central Control
        </span>
      </div>

      <div className="flex items-center gap-3" data-tauri-no-drag>
        {/* Native Desktop Telemetry Pill */}
        <div className="hidden xl:inline-flex items-center gap-2 px-2.5 py-1 rounded-full border border-gray-200 bg-gray-100 text-[11px] font-mono text-gray-700">
          <span className="flex items-center gap-1 text-emerald-600 font-semibold">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
            SPK-UZ: 8ms
          </span>
          <span className="text-gray-300">|</span>
          <span className="text-cyan-600 font-semibold">SQLITE OK</span>
        </div>

        {/* Keyboard Shortcuts Trigger Button */}
        {onOpenCheatsheet && (
          <button
            type="button"
            onClick={onOpenCheatsheet}
            className="hidden sm:inline-flex items-center justify-center h-8 w-8 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-100 transition-colors"
            title="Keyboard Shortcuts (?)"
            aria-label="Keyboard Shortcuts"
          >
            <Keyboard size={16} />
          </button>
        )}

        {/* Interactive WebSocket Telemetry Popover (UA-14) */}
        <div className="relative">
          <button
            type="button"
            onClick={() => setShowWsPopover((v) => !v)}
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors ${badge.bgColor} ${badge.textColor}`}
          >
            <span className={`h-2 w-2 rounded-full ${badge.dotColor}`} />
            <span>{badge.label}</span>
          </button>

          {showWsPopover && (
            <div className="absolute right-0 mt-2 w-72 rounded-xl border bg-white p-4 shadow-xl z-50 text-xs space-y-3">
              <div className="flex items-center justify-between border-b pb-2">
                <span className="font-semibold text-gray-900">Event Stream Telemetry</span>
                <span className="font-mono text-[10px] text-gray-400">/v1/ws</span>
              </div>
              <div className="space-y-1.5 text-gray-600">
                <div className="flex justify-between">
                  <span>Connection State:</span>
                  <span className="font-semibold capitalize text-gray-900">{status}</span>
                </div>
                <div className="flex justify-between">
                  <span>Last Heartbeat:</span>
                  <span className="font-mono text-gray-800">
                    {lastHeartbeat ? lastHeartbeat.toLocaleTimeString() : "None"}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Last Event Received:</span>
                  <span className="font-mono text-gray-800">
                    {lastEventTime ? lastEventTime.toLocaleTimeString() : "None"}
                  </span>
                </div>
              </div>

              <div className="pt-2 border-t flex justify-end">
                <button
                  type="button"
                  onClick={() => {
                    reconnect();
                    setShowWsPopover(false);
                  }}
                  className="rounded bg-indigo-50 px-3 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-100"
                >
                  Force Reconnect
                </button>
              </div>
            </div>
          )}
        </div>

        {/* User Session Signout */}
        <button
          type="button"
          onClick={() => void logout()}
          className="rounded border px-3 py-1 text-xs font-medium text-gray-700 hover:bg-gray-100 transition-colors"
        >
          Sign Out
        </button>
      </div>
    </header>
  );
}
