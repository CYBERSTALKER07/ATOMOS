"use client";

import React, { useState } from "react";
import { apiFetch } from "@/lib/auth";
import { useSyncHub } from "@/lib/useSyncHub";
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

  useSyncHub("POLL", "default", async (signal) => {
    try {
      const response = await apiFetch('/v1/supplier/dashboard', { signal });
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
    <div className="min-h-full p-6 md:p-10 relative overflow-hidden" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Atmospheric background elements */}
      <div className="absolute top-[-10%] right-[-10%] w-[40%] h-[40%] gradient-orb-sky opacity-20 blur-[100px] pointer-events-none" />
      <div className="absolute bottom-[-10%] left-[-10%] w-[30%] h-[30%] gradient-orb-peach opacity-10 blur-[100px] pointer-events-none" />

      <header className="mb-12 flex flex-col md:flex-row md:items-end justify-between gap-6 relative z-10">
        <div>
          <motion.h1 
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            className="md-typescale-display-small font-bold tracking-tight mb-2"
          >
            Global Supply
          </motion.h1>
          <motion.p 
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.1 }}
            className="md-typescale-body-large opacity-60"
          >
            Regional Command — Production Portal
          </motion.p>
        </div>
        
        <motion.div 
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.2 }}
          className="flex items-center gap-4"
        >
          <div className="flex flex-col items-end hidden md:flex">
            <span className="md-typescale-label-small opacity-40 uppercase tracking-widest">Connection</span>
            <span className="md-typescale-label-medium font-mono text-success">STABLE_5000MS</span>
          </div>
          
          {isLive ? (
            <div
              className="flex items-center gap-2 h-10 px-4 glass-premium rounded-full border-success/20"
              style={{ cursor: 'default' }}
            >
              <div className="w-2 h-2 rounded-full bg-success shadow-[0_0_8px_var(--success)] animate-pulse" />
              <span className="md-typescale-label-medium font-bold text-success">LIVE_SYNC</span>
            </div>
          ) : (
            <div
              className="flex items-center gap-2 h-10 px-4 glass-premium rounded-full border-danger/20"
              style={{ cursor: 'default' }}
            >
              <div className="w-2 h-2 rounded-full bg-danger" />
              <span className="md-typescale-label-medium font-bold text-danger">OFFLINE</span>
            </div>
          )}
        </motion.div>
      </header>

      <div className="md-divider mb-12 opacity-10" />

      {/* KPI Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12 relative z-10">
        <StatsCard
          label="Locked Revenue Pipeline"
          value={metrics.total_pipeline.toLocaleString()}
          sub="TOTAL_UZS_VOLUME"
          accent="var(--color-md-primary)"
          delay={0}
        />
        <StatsCard
          label="Required Dispatch Today"
          value={metrics.pending_volume.toLocaleString()}
          sub="PENDING_UNITS"
          accent="var(--color-md-secondary)"
          delay={100}
        />
        <StatsCard
          label="AI Future Forecast"
          value={metrics.ai_forecast_volume.toLocaleString()}
          sub="PREDICTED_ORDERS"
          accent="var(--color-md-tertiary)"
          delay={200}
        />
      </div>
      
      {/* Placeholder for future sections */}
      <motion.div 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        className="grid grid-cols-1 md:grid-cols-2 gap-6 relative z-10"
      >
        <div className="h-64 glass-premium rounded-3xl p-8 flex flex-col justify-center border-white/5">
           <h3 className="md-typescale-title-large font-bold mb-2">Production Capacity</h3>
           <p className="opacity-60 mb-6">Real-time analysis of assembly lines across all regions.</p>
           <div className="flex gap-2">
             <div className="h-2 flex-1 bg-white/5 rounded-full overflow-hidden">
                <motion.div 
                  initial={{ width: 0 }}
                  animate={{ width: '78%' }}
                  transition={{ delay: 0.8, duration: 1.5 }}
                  className="h-full bg-primary"
                />
             </div>
             <span className="md-typescale-label-small">78%</span>
           </div>
        </div>
        
        <div className="h-64 glass-premium rounded-3xl p-8 flex flex-col justify-center border-white/5">
           <h3 className="md-typescale-title-large font-bold mb-2">Regional Logistics</h3>
           <p className="opacity-60 mb-6">Dispatch efficiency tracking and route optimization.</p>
           <div className="flex gap-2">
             <div className="h-2 flex-1 bg-white/5 rounded-full overflow-hidden">
                <motion.div 
                  initial={{ width: 0 }}
                  animate={{ width: '92%' }}
                  transition={{ delay: 1, duration: 1.5 }}
                  className="h-full bg-secondary"
                />
             </div>
             <span className="md-typescale-label-small">92%</span>
           </div>
        </div>
      </motion.div>
    </div>
  );
}
