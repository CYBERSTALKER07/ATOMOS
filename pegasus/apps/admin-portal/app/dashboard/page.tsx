"use client";

import React, { useState } from "react";
import { motion } from "framer-motion";
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
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--desk-bg)', color: 'var(--desk-text-primary)' }}>
      <header className="md-page-header mb-10">
        <div>
          <motion.h1 
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            className="md-typescale-display-small font-light tracking-tight mb-2"
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
            <span className="md-typescale-label-medium font-mono" style={{ color: 'var(--desk-success)' }}>STABLE_5000MS</span>
          </div>
          
          {isLive ? (
            <div
              className="md-chip"
              style={{ cursor: 'default' }}
            >
              <span className="desk-status-dot desk-status-dot--success" />
              <span className="md-typescale-label-small">LIVE_SYNC</span>
            </div>
          ) : (
            <div
              className="md-chip"
              style={{ cursor: 'default', borderColor: 'var(--desk-danger)' }}
            >
              <span className="desk-status-dot desk-status-dot--danger" />
              <span className="md-typescale-label-small" style={{ color: 'var(--desk-danger)' }}>OFFLINE</span>
            </div>
          )}
        </motion.div>
      </header>

      <div className="md-divider mb-10" />

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
        <StatsCard
          label="Locked Revenue Pipeline"
          value={metrics.total_pipeline.toLocaleString()}
          sub="TOTAL_UZS_VOLUME"
          accent="var(--desk-accent)"
          delay={0}
        />
        <StatsCard
          label="Required Dispatch Today"
          value={metrics.pending_volume.toLocaleString()}
          sub="PENDING_UNITS"
          accent="var(--desk-warning)"
          delay={100}
        />
        <StatsCard
          label="AI Future Forecast"
          value={metrics.ai_forecast_volume.toLocaleString()}
          sub="PREDICTED_ORDERS"
          accent="var(--desk-success)"
          delay={200}
        />
      </div>

      <motion.div 
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        className="grid grid-cols-1 md:grid-cols-2 gap-6"
      >
        <div className="h-64 desk-card p-8 flex flex-col justify-center">
           <h3 className="md-typescale-title-large font-light mb-2">Production Capacity</h3>
           <p className="md-typescale-body-medium mb-6" style={{ color: 'var(--desk-text-secondary)' }}>Real-time analysis of assembly lines across all regions.</p>
           <div className="flex gap-2 items-center">
             <div className="h-2 flex-1 rounded-full overflow-hidden" style={{ background: 'var(--desk-surface-alt)' }}>
                <motion.div 
                  initial={{ width: 0 }}
                  animate={{ width: '78%' }}
                  transition={{ delay: 0.8, duration: 1.5 }}
                  className="h-full"
                  style={{ background: 'var(--desk-accent)' }}
                />
             </div>
             <span className="md-typescale-label-small">78%</span>
           </div>
        </div>

        <div className="h-64 desk-card p-8 flex flex-col justify-center">
           <h3 className="md-typescale-title-large font-light mb-2">Regional Logistics</h3>
           <p className="md-typescale-body-medium mb-6" style={{ color: 'var(--desk-text-secondary)' }}>Dispatch efficiency tracking and route optimization.</p>
           <div className="flex gap-2 items-center">
             <div className="h-2 flex-1 rounded-full overflow-hidden" style={{ background: 'var(--desk-surface-alt)' }}>
                <motion.div 
                  initial={{ width: 0 }}
                  animate={{ width: '92%' }}
                  transition={{ delay: 1, duration: 1.5 }}
                  className="h-full"
                  style={{ background: 'var(--desk-success)' }}
                />
             </div>
             <span className="md-typescale-label-small">92%</span>
           </div>
        </div>
      </motion.div>
    </div>
  );
}
