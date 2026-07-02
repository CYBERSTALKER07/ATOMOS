"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import type { PlanningSignalIngestStatus } from "@pegasusx/types";

const api = createSupplierApi();

function formatLag(seconds: number): string {
  if (seconds <= 0) return "live";
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}

export default function SignalIngestOpsPanel() {
  const [status, setStatus] = useState<PlanningSignalIngestStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getPlanningSignalStatus();
      setStatus(resp);
    } catch (err) {
      setError(err instanceof Error ? err.message : "signal_status_unavailable");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useSupplierSessionReconcile(() => {
    void load();
  });

  return (
    <section className="desk-card p-6 mt-6">
      <div className="flex items-center justify-between gap-3 mb-4">
        <div>
          <h2 className="bento-card-title">Signal ingest ops</h2>
          <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
            Kafka collect → ai-worker projection — no ML inference on the hot path.
          </p>
        </div>
        <button type="button" className="portal-btn portal-btn--ghost text-xs" onClick={() => void load()}>
          Refresh
        </button>
      </div>

      {error ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : loading && !status ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          Loading ingest health…
        </p>
      ) : status ? (
        <>
          <KpiStatGrid columns={4}>
            <KpiStatCard label="Projections" value={status.projection_count} sub="PlanningSignalProjections rows" />
            <KpiStatCard
              label="Ingest lag"
              value={formatLag(status.lag_seconds)}
              sub={status.last_ingest_at ? `Last ${new Date(status.last_ingest_at).toLocaleString()}` : "No ingests yet"}
            />
            <KpiStatCard
              label="Baseline rows"
              value={status.baseline_rows_from_signals}
              sub="From signal_ingest source"
            />
            <KpiStatCard
              label="Pipeline"
              value={status.healthy ? "Healthy" : "Stale"}
              sub={status.topic}
            />
          </KpiStatGrid>
        </>
      ) : null}
    </section>
  );
}
