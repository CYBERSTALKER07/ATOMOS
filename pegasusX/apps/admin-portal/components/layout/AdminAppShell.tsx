"use client";

import React, { useState, useEffect, type ReactNode } from "react";
import { AdminAuthGate } from "@/components/auth/AdminAuthGate";
import { AdminSidebar } from "./AdminSidebar";
import { AdminHeader } from "./AdminHeader";
import { useAdminToken } from "@/lib/session";
import { useAdminWS } from "@/lib/ws-provider";
import { api } from "@/lib/api";

export function AdminAppShell({ children }: { children: ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const { token } = useAdminToken();
  const { refreshKey } = useAdminWS();

  const [pendingFlagsCount, setPendingFlagsCount] = useState<number | undefined>();
  const [dlqCount, setDlqCount] = useState<number | undefined>();

  useEffect(() => {
    if (!token) return;
    let active = true;

    async function loadBadges() {
      try {
        const [flags, summary] = await Promise.all([
          api.listPendingFlags(token),
          api.outboxSummary(token),
        ]);
        if (!active) return;
        setPendingFlagsCount(flags.count || flags.items?.length || 0);
        if (summary.dead_letter_available && typeof summary.dead_letter_count === "number") {
          setDlqCount(summary.dead_letter_count);
        }
      } catch {
        // Soft fail
      }
    }

    void loadBadges();
    return () => {
      active = false;
    };
  }, [token, refreshKey]);

  return (
    <AdminAuthGate>
      <div className="flex min-h-screen bg-gray-50 text-gray-900">
        <AdminSidebar
          isOpen={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          pendingFlagsCount={pendingFlagsCount}
          dlqCount={dlqCount}
        />
        <div className="flex flex-1 flex-col overflow-hidden">
          <AdminHeader onOpenSidebar={() => setSidebarOpen(true)} />
          <main className="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">
            <div className="mx-auto max-w-7xl">{children}</div>
          </main>
        </div>
      </div>
    </AdminAuthGate>
  );
}
