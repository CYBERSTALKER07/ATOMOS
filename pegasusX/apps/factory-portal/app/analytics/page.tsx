'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';

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
          </>
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
