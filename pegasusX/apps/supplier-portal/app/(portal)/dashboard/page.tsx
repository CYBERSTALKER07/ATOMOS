"use client";

import Link from "next/link";
import { BentoCard, BentoGrid } from "@/components/BentoGrid";
import FleetLiveMapPanel from "@/components/FleetLiveMapPanel";
import NetworkPulsePanel from "@/components/NetworkPulsePanel";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";
import { FormAlert } from "@/components/portal";
import { useDashboardData } from "./use-dashboard-data";

export default function DashboardPage() {
  const { metrics, recentManifests, recentEvents, isPaymentConfigured, loading, error } = useDashboardData();

  const formatCurrency = (minor: number) =>
    new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "UZS",
      maximumFractionDigits: 0,
    }).format(minor / 100);

  const liveOrderStats = [
    { label: "Pending", count: metrics.ordersByStatus.PENDING ?? 0, color: "var(--desk-text-secondary)" },
    { label: "Loaded", count: metrics.ordersByStatus.LOADED ?? 0, color: "var(--desk-warning)" },
    { label: "In transit", count: metrics.ordersByStatus.IN_TRANSIT ?? 0, color: "var(--desk-info)" },
    { label: "Arrived", count: metrics.ordersByStatus.ARRIVED ?? 0, color: "var(--desk-success)" },
  ];
  const maxLive = Math.max(1, ...liveOrderStats.map((s) => s.count));
  const driverPct = metrics.totalDrivers > 0 ? metrics.activeDrivers / metrics.totalDrivers : 0;
  const vuPct = metrics.fleetVuTotal > 0 ? (metrics.fleetVuUsed / metrics.fleetVuTotal) * 100 : 0;

  return (
    <PageChrome
      icon="overview"
      title="Overview"
      description="Live operational command center."
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

        <BentoGrid theme="apple">
          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Revenue today
            </h2>
            <div>
              <div className="md-kpi-value">{formatCurrency(metrics.revenueToday)}</div>
              <div
                className="md-typescale-label-medium mt-1"
                style={{
                  color: metrics.revenueChangePct >= 0 ? "var(--desk-success)" : "var(--desk-danger)",
                }}
              >
                {metrics.revenueChangePct >= 0 ? "+" : ""}
                {metrics.revenueChangePct}% vs yesterday
              </div>
            </div>
          </BentoCard>

          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Completion rate
            </h2>
            <div className="md-kpi-value">{metrics.deliveryCompletionRate}%</div>
            <p className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Successful deliveries today
            </p>
          </BentoCard>

          <BentoCard size="stat" className="p-5 flex flex-col justify-between">
            <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Retailer activity
            </h2>
            <div className="flex items-baseline gap-2">
              <span className="md-kpi-value">{metrics.retailersOrderedToday}</span>
              <span className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
                / {metrics.totalRetailers}
              </span>
            </div>
            <p className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
              Ordered today
            </p>
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
                <h2 className="md-typescale-title-medium">Live fleet map</h2>
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
            <div className="flex-1 grid grid-cols-4 gap-3 items-end">
              {liveOrderStats.map((stat) => (
                <div key={stat.label} className="flex flex-col items-center h-full justify-end">
                  <div className="md-typescale-title-medium mb-2">{stat.count}</div>
                  <div
                    className="w-full rounded-t-sm"
                    style={{
                      height: `${Math.max(10, (stat.count / maxLive) * 100)}%`,
                      backgroundColor: stat.color,
                      opacity: 0.85,
                      minHeight: 8,
                    }}
                  />
                  <div
                    className="md-typescale-label-small mt-2 text-center truncate w-full"
                    style={{ color: "var(--desk-text-secondary)" }}
                  >
                    {stat.label}
                  </div>
                </div>
              ))}
            </div>
          </BentoCard>

          <BentoCard size="list" className="p-0 flex flex-col overflow-hidden max-h-[320px]">
            <div
              className="p-4 border-b flex justify-between items-center gap-3"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <h2 className="md-typescale-title-medium">Dispatch queue</h2>
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
            </div>
          </BentoCard>

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
              <div className="h-full transition-all duration-500" style={{ width: `${vuPct}%`, background: "var(--desk-accent)" }} />
            </div>
          </BentoCard>

          <BentoCard size="full" className="p-5">
            <NetworkPulsePanel />
          </BentoCard>

          <BentoCard size="full" className="p-0 flex flex-col max-h-[300px]">
            <div
              className="p-4 border-b sticky top-0"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              <h2 className="md-typescale-title-medium">Live event stream</h2>
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
