'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';
import { motion } from 'framer-motion';

interface AnalyticsOverview {
  daily_activity: unknown[];
  transfers_total: number;
  manifests_active: number;
  exception_queue: number;
  avg_lead_time_mins: number;
}

export default function FactoryAnalyticsPage() {
  const [overview, setOverview] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/analytics/overview');
      if (!res.ok) {
        setError(`Unable to load analytics overview (${res.status}).`);
        return;
      }
      setOverview(await res.json());
    } catch {
      setError('Unable to load analytics overview right now.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const kpis = overview
    ? [
        { label: 'Transfers Total', value: overview.transfers_total, icon: 'transfers' },
        { label: 'Active Manifests', value: overview.manifests_active, icon: 'manifests' },
        { label: 'Exception Queue', value: overview.exception_queue, icon: 'warning', danger: overview.exception_queue > 0 },
        { label: 'Avg Lead Time (min)', value: overview.avg_lead_time_mins, icon: 'analytics' },
      ]
    : [];

  return (
    <PageTransition>
      <div className="space-y-4 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Analytics Overview</h1>
            <p className="mt-1 text-sm text-[var(--muted)]">
              Factory throughput, manifest pressure, and exception queue from the analytics endpoint.
            </p>
          </div>
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => {
              setLoading(true);
              void load();
            }}
            className="button--secondary flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm hover-lift active-press"
          >
            <Icon name="refresh" size={16} /> Refresh
          </motion.button>
        </div>

        {loading ? (
          <FactoryPageState
            kind="loading"
            skeleton={
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                {Array.from({ length: 4 }).map((_, i) => (
                  <div key={i} className="md-skeleton h-28 rounded-xl" />
                ))}
              </div>
            }
          />
        ) : error && !overview ? (
          <FactoryPageState
            kind="error"
            headline="Unable to load analytics overview"
            body={error}
            actionLabel="Retry"
            onAction={() => {
              setLoading(true);
              void load();
            }}
          />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {kpis.map((kpi) => (
              <motion.div
                key={kpi.label}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-[var(--muted)]">{kpi.label}</span>
                  <Icon name={kpi.icon} size={18} className="text-[var(--muted)]" />
                </div>
                <div className="mt-4 flex items-center gap-2">
                  <span className="text-3xl font-semibold tabular-nums text-[var(--foreground)]">{kpi.value}</span>
                  {kpi.danger && <span className="status-chip status-chip--critical">Alert</span>}
                </div>
              </motion.div>
            ))}
          </div>
        )}
      </div>
    </PageTransition>
  );
}
