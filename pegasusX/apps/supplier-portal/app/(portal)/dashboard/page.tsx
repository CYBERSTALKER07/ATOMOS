"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { BentoCard, BentoGrid } from "@/components/BentoGrid";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import PlanningOutcomesPanel from "@/components/PlanningOutcomesPanel";
import NetworkPulsePanel from "@/components/NetworkPulsePanel";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
import { FormAlert } from "@/components/portal";
import { usePortalT } from "@/lib/i18n";
import { useDashboardData } from "./use-dashboard-data";
import { useDashboardHistory } from "./use-dashboard-history";
import { formatPackMoney, readCachedAuthSession } from "@pegasusx/api-client";
import { MANIFEST_STATES, ORDER_STATUS_FUNNEL, guardHistorySeries } from "@pegasusx/types";
import { HealthStrip, KpiStat, RangeToggle, SourceChip, StatusStack } from "@pegasusx/ui-kit/portal";

export default function DashboardPage() {
  const t = usePortalT();
  const router = useRouter();
  const { metrics, recentManifests, recentEvents, isPaymentConfigured, loading, error } = useDashboardData();
  const history = useDashboardHistory();

  const formatCurrency = (minor: number) =>
    formatPackMoney(minor, readCachedAuthSession()?.pack);

  const openStatus = (status: string) => {
    router.push(`/orders?status=${encodeURIComponent(status)}`);
  };

  const driverPct = metrics.totalDrivers > 0 ? metrics.activeDrivers / metrics.totalDrivers : 0;
  const vuPct = metrics.fleetVuAvailable && metrics.fleetVuTotal > 0
    ? (metrics.fleetVuUsed / metrics.fleetVuTotal) * 100
    : 0;
  const revenueSpark = guardHistorySeries(history.revenue);
  const velocitySpark = guardHistorySeries(history.velocity);
  const historyLive = Boolean(revenueSpark || velocitySpark);
  const deltaLabel =
    history.revenueDeltaPct == null
      ? null
      : `${history.revenueDeltaPct >= 0 ? "+" : ""}${history.revenueDeltaPct.toFixed(0)}% vs yesterday`;

  return (
    <PageChrome
      icon="overview"
      title={t("portal.page.dashboard.supplier.title")}
      description={t("portal.page.dashboard.supplier.description")}
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
    >
      <div className="flex flex-col gap-6">
        {isPaymentConfigured === false ? (
          <FormAlert variant="error">
            <span className="flex flex-wrap items-center justify-between gap-3 w-full">
              <span>
                Payment gateway required — configure gateways to receive retailer payments.
              </span>
              <Link href="/setup/billing" className="portal-btn portal-btn--primary shrink-0">
                Configure now
              </Link>
            </span>
          </FormAlert>
        ) : null}

        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            {metrics.asOf ? `as-of ${metrics.asOf}` : "as-of —"}
          </p>
          <RangeToggle value={history.range} onChange={history.setRange} />
        </div>

        <BentoGrid theme="apple">
          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <KpiStat
              label="Revenue today"
              value={formatCurrency(metrics.revenueToday)}
              delta={deltaLabel}
              spark={history.revenue}
              source="live"
            />
          </BentoCard>

          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <KpiStat
              label="Completion rate"
              value={`${metrics.deliveryCompletionRate}%`}
              delta={`${metrics.completedToday} / ${metrics.attemptedToday} today`}
              source="live"
            />
          </BentoCard>

          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <KpiStat
              label="Retailer activity"
              value={`${metrics.retailersOrderedToday} / ${metrics.totalRetailers}`}
              delta="Ordered today"
              source="live"
            />
          </BentoCard>

          <BentoCard size="stat" className="p-5 flex flex-col items-center justify-between text-center">
            <h2 className="md-typescale-title-medium self-start" style={{ color: "var(--desk-text-secondary)" }}>
              Active drivers
            </h2>
            <div className="relative w-24 h-24 flex items-center justify-center my-2">
              <svg className="absolute inset-0 w-full h-full -rotate-90" aria-hidden>
                <circle cx="48" cy="48" r="40" fill="none" stroke="var(--desk-surface-raised)" strokeWidth="8" />
                <circle
                  cx="48"
                  cy="48"
                  r="40"
                  fill="none"
                  stroke="var(--desk-accent)"
                  strokeWidth="8"
                  strokeDasharray={`${driverPct * 251} 251`}
                />
              </svg>
              <div className="md-typescale-title-large font-light">{metrics.activeDrivers}</div>
            </div>
            <p className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              of {metrics.totalDrivers} online
            </p>
          </BentoCard>

          <BentoCard size="anchor" className="p-0 flex flex-col overflow-hidden min-h-[480px]">
            <div
              className="p-4 border-b flex justify-between items-center gap-3"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <div>
                <h2 className="md-typescale-title-medium">{t("supplier_portal.dashboard.text.live_fleet_map")}</h2>
                <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  Sealed manifest polylines and driver GPS.
                </p>
              </div>
              <Link href="/fleet" className="portal-btn portal-btn--ghost text-sm h-8 px-2 shrink-0">
                Fleet
              </Link>
            </div>
            <FleetLiveMapPanel className="flex-1 min-h-[360px]" />
          </BentoCard>

          <BentoCard size="list" className="p-5 flex flex-col">
            <h2 className="md-typescale-title-medium mb-4" style={{ color: "var(--desk-text-secondary)" }}>
              Live orders
            </h2>
            <StatusStack
              dictionary={ORDER_STATUS_FUNNEL}
              counts={metrics.ordersByStatus as Record<string, number>}
              source="live"
              onSelect={openStatus}
            />
          </BentoCard>

          <BentoCard size="list" className="p-5 flex flex-col">
            <h2 className="md-typescale-title-medium mb-4" style={{ color: "var(--desk-text-secondary)" }}>
              Health
            </h2>
            <HealthStrip
              items={[
                {
                  key: "FISCAL_FAILED",
                  label: "Fiscal failed",
                  count: metrics.ordersByStatus.FISCAL_FAILED ?? 0,
                },
                {
                  key: "ARRIVED_SHOP_CLOSED",
                  label: "Shop closed",
                  count: metrics.ordersByStatus.ARRIVED_SHOP_CLOSED ?? 0,
                },
                {
                  key: "RECONCILIATION_REQUIRED",
                  label: "Reconciliation",
                  count: metrics.ordersByStatus.RECONCILIATION_REQUIRED ?? 0,
                },
              ]}
              onSelect={openStatus}
            />
          </BentoCard>

          <BentoCard size="list" className="p-5 flex flex-col">
            <h2 className="md-typescale-title-medium mb-4" style={{ color: "var(--desk-text-secondary)" }}>
              Manifests
            </h2>
            <StatusStack
              dictionary={MANIFEST_STATES}
              counts={metrics.manifestsByState}
              source="live"
            />
          </BentoCard>

          <BentoCard size="list" className="p-5 flex flex-col">
            <h2 className="md-typescale-title-medium mb-2" style={{ color: "var(--desk-text-secondary)" }}>
              Truck duty
            </h2>
            <SourceChip source="unavailable" />
            <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
              Duty rollup unavailable
            </p>
          </BentoCard>

          <BentoCard size="list" className="p-0 flex flex-col overflow-hidden max-h-[320px]">
            <div
              className="p-4 border-b flex justify-between items-center gap-3"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <h2 className="md-typescale-title-medium">{t("supplier_portal.dashboard.text.dispatch_queue")}</h2>
              <Link href="/dispatch" className="portal-btn portal-btn--ghost text-sm h-8 px-2 shrink-0">
                View all
              </Link>
            </div>
            <div className="overflow-y-auto flex-1 p-4 space-y-2">
              {recentManifests.length === 0 ? (
                <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  No active manifests in queue.
                </p>
              ) : (
                recentManifests.map((manifest) => (
                  <div
                    key={manifest.id}
                    className="flex items-center justify-between gap-3 py-2 border-b last:border-b-0"
                    style={{ borderColor: "var(--desk-border)" }}
                  >
                    <div className="min-w-0">
                      <div className="font-mono text-sm truncate" style={{ color: "var(--desk-accent-strong)" }}>
                        {manifest.id.substring(0, 12)}
                      </div>
                      <div className="md-typescale-body-small truncate" style={{ color: "var(--desk-text-secondary)" }}>
                        {manifest.driverName}
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="md-typescale-label-medium tabular-nums">{manifest.ordersCount}</span>
                      <StatusBadge state={manifest.status} />
                    </div>
                  </div>
                ))
              )}
            </div>
          </BentoCard>

          <BentoCard size="control" className="p-5 flex flex-col justify-center gap-3">
            <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Quick actions
            </h2>
            <div className="flex flex-wrap gap-2">
              <Link href="/dispatch" className="portal-btn portal-btn--primary">
                Auto-dispatch preview
              </Link>
              <Link href="/orders" className="portal-btn portal-btn--outline">
                Review orders
              </Link>
              <Link href="/exceptions" className="portal-btn portal-btn--outline">
                Exception queue
              </Link>
              <Link href="/planning" className="portal-btn portal-btn--outline">
                Open Plan
              </Link>
            </div>
          </BentoCard>

          {metrics.fleetVuAvailable ? (
          <BentoCard size="wide" className="p-5 flex flex-col justify-center">
            <div className="flex justify-between items-end mb-2">
              <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
                Fleet volume utilization
              </h2>
              <div className="md-typescale-title-medium">
                {metrics.fleetVuUsed.toLocaleString()}{" "}
                <span className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  / {metrics.fleetVuTotal.toLocaleString()} VU
                </span>
              </div>
            </div>
            <div className="h-4 w-full rounded-full overflow-hidden" style={{ background: "var(--desk-surface-raised)" }}>
              <div className="h-full" style={{ width: `${vuPct}%`, background: "var(--desk-accent)" }} />
            </div>
          </BentoCard>
          ) : (
          <BentoCard size="wide" className="p-5 flex flex-col justify-center">
            <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Fleet volume utilization
            </h2>
            <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
              unavailable
            </p>
          </BentoCard>
          )}

          <BentoCard size="full" className="p-5">
            <div className="flex items-center justify-between gap-3 mb-3">
              <h2 className="md-typescale-title-medium">History</h2>
              <SourceChip source={historyLive ? "live" : history.velocity.source} />
            </div>
            {historyLive ? (
              <div className="grid gap-4 md:grid-cols-2">
                <KpiStat
                  label="Completed"
                  value={velocitySpark ? String(velocitySpark.points[velocitySpark.points.length - 1]) : "—"}
                  spark={history.velocity}
                  source={history.velocity.source}
                />
                <KpiStat
                  label="Revenue"
                  value={revenueSpark ? formatCurrency(revenueSpark.points[revenueSpark.points.length - 1]) : "—"}
                  spark={history.revenue}
                  source={history.revenue.source}
                />
              </div>
            ) : (
              <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                {history.loading ? "Loading history" : "History unavailable"}
              </p>
            )}
          </BentoCard>

          <BentoCard size="full" className="p-0 overflow-hidden min-h-[200px]">
            <PlanningOutcomesPanel />
          </BentoCard>

          <BentoCard size="full" className="p-5">
            <NetworkPulsePanel />
          </BentoCard>

          <BentoCard size="full" className="p-0 flex flex-col max-h-[300px]">
            <div
              className="p-4 border-b sticky top-0"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <h2 className="md-typescale-title-medium">{t("supplier_portal.dashboard.text.live_event_stream")}</h2>
            </div>
            <div className="overflow-y-auto p-4 space-y-3 flex-1">
              {recentEvents.length === 0 ? (
                <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  No recent activity events.
                </p>
              ) : (
                recentEvents.map((event) => (
                  <div key={event.id} className="flex gap-4 items-start text-sm">
                    <div className="w-24 shrink-0 tabular-nums" style={{ color: "var(--desk-text-secondary)" }}>
                      {new Date(event.timestamp).toLocaleTimeString([], {
                        hour12: false,
                        hour: "2-digit",
                        minute: "2-digit",
                        second: "2-digit",
                      })}
                    </div>
                    <span className="md-chip h-6 text-[10px] px-2 shrink-0">{event.type}</span>
                    <div className="truncate">{event.description}</div>
                  </div>
                ))
              )}
            </div>
          </BentoCard>
        </BentoGrid>
      </div>
    </PageChrome>
  );
}
