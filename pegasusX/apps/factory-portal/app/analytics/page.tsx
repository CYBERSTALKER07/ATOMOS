'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import ProductionForecastChart from './ProductionForecastChart';

interface AnalyticsOverview {
  daily_activity: unknown[];
  transfers_total: number;
  manifests_active: number;
  exception_queue: number;
  avg_lead_time_mins: number;
  product_drill_down: {
    product_id: string;
    requested: number;
    shipped: number;
    fulfillment_ratio: number;
  }[];
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

  return (
    <PageTransition>
      <PageChrome
        icon="analytics"
        title="Analytics overview"
        description="Factory throughput, manifest pressure, and exception queue from the analytics endpoint."
        loading={loading}
        skeletonVariant="dashboard"
        error={error && !overview ? error : null}
        actions={
          <button
            type="button"
            onClick={() => {
              setLoading(true);
              void load();
            }}
            className="portal-btn portal-btn--ghost flex items-center gap-1.5"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        {overview ? (
          <>
            <KpiStatGrid columns={4}>
              <KpiStatCard label="Transfers total" value={overview.transfers_total} sub="All-time factory transfers" />
              <KpiStatCard label="Active manifests" value={overview.manifests_active} sub="In loading gate pipeline" />
              <KpiStatCard
                label="Exception queue"
                value={overview.exception_queue}
                sub={overview.exception_queue > 0 ? 'Review gate exceptions' : 'No open exceptions'}
              />
              <KpiStatCard
                label="Avg lead time"
                value={`${overview.avg_lead_time_mins} min`}
                sub="Transfer approval to dispatch"
              />
            </KpiStatGrid>

            {overview.exception_queue > 0 && (
              <div className="mt-6">
                <Link
                  href="/manifest-exceptions"
                  className="portal-btn portal-btn--ghost inline-flex items-center gap-2"
                >
                  <Icon name="warning" size={16} />
                  Open gate exceptions ({overview.exception_queue})
                </Link>
              </div>
            )}

            {overview.product_drill_down && overview.product_drill_down.length > 0 && (
              <PageSection title="Product drill-down" description="Aggregate product demand and fulfillment ratios across active requests." className="mt-8">
                <div className="overflow-x-auto rounded-lg border" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <table className="desk-table w-full text-sm">
                    <thead>
                      <tr style={{ background: 'var(--color-md-surface-container)' }}>
                        <th className="text-left px-4 py-3 font-medium">Product ID</th>
                        <th className="text-left px-4 py-3 font-medium">Requested Qty</th>
                        <th className="text-left px-4 py-3 font-medium">Shipped Qty</th>
                        <th className="text-left px-4 py-3 font-medium">Fulfillment %</th>
                      </tr>
                    </thead>
                    <tbody>
                      {overview.product_drill_down.map((row, i) => (
                        <tr key={row.product_id} className={i !== overview.product_drill_down.length - 1 ? 'border-b' : ''} style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                          <td className="px-4 py-3 font-mono text-xs">{row.product_id}</td>
                          <td className="px-4 py-3 tabular-nums">{row.requested.toLocaleString()}</td>
                          <td className="px-4 py-3 tabular-nums">{row.shipped.toLocaleString()}</td>
                          <td className="px-4 py-3 tabular-nums">
                            <span className={`px-2 py-0.5 rounded text-xs font-medium ${row.fulfillment_ratio < 1 ? 'bg-[var(--color-md-error-container)] text-[var(--color-md-on-error-container)]' : 'bg-[var(--color-md-success-container)] text-[var(--color-md-on-success-container)]'}`}>
                              {(row.fulfillment_ratio * 100).toFixed(1)}%
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </PageSection>
            )}

            <PageSection title="Production Forecast" description="Expected raw material burn rate versus actual stock over the next 7 days." className="mt-8">
              <div className="h-80 w-full p-4 border rounded-lg" style={{ borderColor: 'var(--color-md-outline-variant)', background: 'var(--color-md-surface-container)' }}>
                <ProductionForecastChart className="w-full h-full" />
              </div>
            </PageSection>
          </>
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
