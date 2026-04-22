'use client';

import { useAuth } from '@/hooks/useAuth';
import { useAdvancedAnalytics } from '@/hooks/useAdvancedAnalytics';
import { BentoGrid, BentoCard, BentoSkeleton } from '@/components/BentoGrid';
import DateRangePicker from '@/components/analytics/advanced/DateRangePicker';
import KPIStatsRow from '@/components/analytics/advanced/KPIStatsRow';
import RevenueChart from '@/components/analytics/advanced/RevenueChart';
import GatewayPieChart from '@/components/analytics/advanced/GatewayPieChart';
import TopRetailersTable from '@/components/analytics/advanced/TopRetailersTable';
import SLAStackedBar from '@/components/analytics/advanced/SLAStackedBar';
import FleetLoadBars from '@/components/analytics/advanced/FleetLoadBars';
import NodeEfficiencyGrid from '@/components/analytics/advanced/NodeEfficiencyGrid';
import ThroughputLine from '@/components/analytics/advanced/ThroughputLine';
import FactoryActivityChart from '@/components/analytics/advanced/FactoryActivityChart';
import FactoryTransfersPie from '@/components/analytics/advanced/FactoryTransfersPie';
import CircularProgress from '@/components/analytics/advanced/CircularProgress';

export default function AdvancedAnalyticsPage() {
  const auth = useAuth();
  const analytics = useAdvancedAnalytics();

  const isFactory = auth.isFactoryStaff;

  // ── Loading State ─────────────────────────────────────────────────────────
  if (analytics.loading) {
    return (
      <div className="p-6 flex flex-col gap-6">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Advanced Analytics
          </h1>
        </div>
        <BentoGrid>
          <BentoSkeleton size="wide" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="anchor" />
          <BentoSkeleton size="list" />
          <BentoSkeleton size="wide" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="full" />
        </BentoGrid>
      </div>
    );
  }

  // ── Error State ───────────────────────────────────────────────────────────
  if (analytics.error) {
    return (
      <div className="p-6 flex flex-col gap-6">
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
          Advanced Analytics
        </h1>
        <div
          className="md-card md-elevation-1 md-shape-md p-6 flex items-center justify-center h-48"
          style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}
        >
          <span className="md-typescale-body-medium">Analytics fault: {analytics.error}</span>
        </div>
      </div>
    );
  }

  // ── Factory View ──────────────────────────────────────────────────────────
  if (isFactory) {
    const fo = analytics.factoryOverview;
    return (
      <div className="p-6 flex flex-col gap-6">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Factory Analytics
          </h1>
          <DateRangePicker
            dateRange={analytics.dateRange}
            onPreset={analytics.setPreset}
            onCustom={analytics.setDateRange}
          />
        </div>

        <BentoGrid>
          {/* KPI Stats */}
          <BentoCard size="wide" delay={0}>
            <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
              <div className="flex flex-col gap-2 h-full justify-center">
                <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>Factory Summary</h3>
                <div className="grid grid-cols-2 gap-4">
                  <div className="flex flex-col">
                    <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>Total Transfers</span>
                    <span className="md-typescale-headline-small font-bold" style={{ color: 'var(--color-md-primary)' }}>
                      {fo?.total_transfers ?? 0}
                    </span>
                  </div>
                  <div className="flex flex-col">
                    <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>Avg Lead Time</span>
                    <span className="md-typescale-headline-small font-bold" style={{ color: 'var(--color-md-secondary)' }}>
                      {Math.round(fo?.avg_lead_time_mins ?? 0)} min
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </BentoCard>

          {/* Factory Activity Chart */}
          <BentoCard size="anchor" delay={60}>
            <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
              <FactoryActivityChart data={fo?.daily_activity ?? []} />
            </div>
          </BentoCard>

          {/* Transfers Pie */}
          <BentoCard size="stat" delay={120}>
            <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
              <FactoryTransfersPie data={fo?.transfers_by_state ?? []} />
            </div>
          </BentoCard>
        </BentoGrid>
      </div>
    );
  }

  // ── Supplier View (Global Admin / Node Admin) ─────────────────────────────
  // Compute aggregate KPIs
  const totalRevenue = analytics.revenue?.time_series.reduce((s, d) => s + d.total, 0) ?? 0;
  const slaTotal = analytics.slaHealth.reduce((s, d) => s + d.total_orders, 0);
  const slaOnTime = analytics.slaHealth.reduce((s, d) => s + d.on_time, 0);
  const slaRate = slaTotal > 0 ? (slaOnTime / slaTotal) * 100 : 0;

  return (
    <div className="p-6 flex flex-col gap-6">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Advanced Analytics
          </h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            {analytics.dateRange.from} → {analytics.dateRange.to}
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={analytics.refresh}
            className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1.5"
          >
            Refresh
          </button>
          <DateRangePicker
            dateRange={analytics.dateRange}
            onPreset={analytics.setPreset}
            onCustom={analytics.setDateRange}
          />
        </div>
      </div>

      <BentoGrid>
        {/* Row 1: KPI Stats (wide) + Revenue Stat + SLA Gauge */}
        <BentoCard size="wide" delay={0}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <KPIStatsRow data={analytics.throughput} />
          </div>
        </BentoCard>

        <BentoCard size="stat" delay={40}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full flex flex-col items-center justify-center" style={{ background: 'var(--color-md-surface-container)' }}>
            <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>Total Revenue</span>
            <span className="md-typescale-headline-medium font-bold" style={{ color: 'var(--color-md-primary)' }}>
              {totalRevenue >= 1_000_000
                ? `${(totalRevenue / 1_000_000).toFixed(1)}M`
                : totalRevenue >= 1_000
                  ? `${(totalRevenue / 1_000).toFixed(0)}K`
                  : totalRevenue.toLocaleString()
              }
            </span>
          </div>
        </BentoCard>

        <BentoCard size="stat" delay={80}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full flex items-center justify-center" style={{ background: 'var(--color-md-surface-container)' }}>
            <CircularProgress
              value={slaRate}
              size={100}
              strokeWidth={8}
              label="SLA Rate"
              color={slaRate >= 80 ? 'var(--color-md-success)' : slaRate >= 50 ? 'var(--color-md-warning)' : 'var(--color-md-error)'}
            />
          </div>
        </BentoCard>

        {/* Row 2: Revenue Chart (anchor) + Gateway Pie */}
        <BentoCard size="anchor" delay={120}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <RevenueChart data={analytics.revenue?.time_series ?? []} />
          </div>
        </BentoCard>

        <BentoCard size="list" delay={160}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <GatewayPieChart data={analytics.revenue?.gateway_breakdown ?? []} />
          </div>
        </BentoCard>

        {/* Row 3: Throughput Line (wide) + Fleet Load */}
        <BentoCard size="wide" delay={200}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <ThroughputLine data={analytics.throughput} />
          </div>
        </BentoCard>

        <BentoCard size="list" delay={240}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <FleetLoadBars data={analytics.loadDistribution} />
          </div>
        </BentoCard>

        {/* Row 4: SLA Stacked Bar (wide) + Top Retailers (list) */}
        <BentoCard size="wide" delay={280}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <SLAStackedBar data={analytics.slaHealth} />
          </div>
        </BentoCard>

        <BentoCard size="list" delay={320}>
          <div className="md-card md-elevation-1 md-shape-md p-5 h-full" style={{ background: 'var(--color-md-surface-container)' }}>
            <TopRetailersTable data={analytics.topRetailers} />
          </div>
        </BentoCard>

        {/* Row 5: Node Efficiency (full) */}
        <BentoCard size="full" delay={360}>
          <div className="md-card md-elevation-1 md-shape-md p-5" style={{ background: 'var(--color-md-surface-container)' }}>
            <NodeEfficiencyGrid data={analytics.nodeEfficiency} />
          </div>
        </BentoCard>
      </BentoGrid>
    </div>
  );
}
