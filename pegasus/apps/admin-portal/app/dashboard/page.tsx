"use client";

import React, { useState } from "react";
import { readTokenFromCookie } from "@/lib/auth";
import { usePolling } from "@/lib/usePolling";
import StatsCard from "@/components/StatsCard";

interface DashboardMetrics {
  total_pipeline: number;
  pending_volume: number;
  ai_forecast_volume: number;
}

export default function SupplierDashboard() {
  const [metrics, setMetrics] = useState<DashboardMetrics>({
    total_pipeline: 0,
    pending_volume: 0,
    ai_forecast_volume: 0,
  });
  const [isLive, setIsLive] = useState(false);

  usePolling(async (signal) => {
    try {
      const token = readTokenFromCookie();
      if (!token) { setIsLive(false); return; }
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/dashboard`,
        { headers: { Authorization: `Bearer ${token.trim()}` }, signal },
      );
      if (!response.ok) throw new Error("Matrix disconnected");

      const data = await response.json();
      setMetrics(data);
      setIsLive(true);
    } catch (error) {
      if ((error as Error).name === 'AbortError') return;
      console.error("[SYNC ERROR]", error);
      setIsLive(false);
    }
  }, 5000);

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-8 flex items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-large tracking-tight" style={{ color: 'var(--foreground)' }}>Global Supply</h1>
          <p className="md-typescale-body-large mt-1" style={{ color: 'var(--muted)' }}>Regional Command — Production Portal</p>
        </div>
        <div className="flex items-center gap-3">
          {isLive ? (
            <div
              className="flex items-center gap-1.5 h-7 px-2.5 md-shape-full"
              style={{ border: '1px solid var(--border)', cursor: 'default' }}
            >
              <div className="w-2 h-2 rounded-full animate-pulse" style={{ background: 'var(--success)' }} />
              <span className="md-typescale-label-small">System Live</span>
            </div>
          ) : (
            <div
              className="flex items-center gap-1.5 h-7 px-2.5 md-shape-full"
              style={{ border: '1px solid var(--danger)', cursor: 'default' }}
            >
              <div className="w-2 h-2 rounded-full" style={{ background: 'var(--danger)' }} />
              <span className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>System Offline</span>
            </div>
          )}
          <span className="md-typescale-label-small" style={{ color: 'var(--border)' }}>Telemetry: 5000ms</span>
        </div>
      </header>

      <div className="md-divider mb-8" />

      {/* KPI Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
        <StatsCard
          label="Locked Revenue Pipeline"
          value={metrics.total_pipeline.toLocaleString()}
          sub="UZS"
          accent="var(--accent)"
          delay={0}
        />
        <StatsCard
          label="Required Dispatch Today"
          value={metrics.pending_volume.toLocaleString()}
          sub="Units"
          accent="var(--muted)"
          delay={50}
        />
        <StatsCard
          label="AI Future Forecast"
          value={metrics.ai_forecast_volume.toLocaleString()}
          sub="Orders Pending"
          accent="var(--muted)"
          delay={100}
        />
      </div>
    </div>
  );
}
