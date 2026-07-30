"use client";

import { useCallback, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { supplierFetch } from "@/lib/auth";
import {
  RefreshCw,
  ShieldAlert,
  Download,
  AlertTriangle,
  Lock,
  FileWarning,
  Activity,
  AlertOctagon,
  Clock,
  XCircle,
} from "lucide-react";
import { MetricCard } from "@/components/compliance/MetricCard";

type DashboardStats = {
  fiscalizing: number;
  fiscalFailed: number;
  forceCompleted: number;
  buyerAcceptancePending: number;
  buyerAcceptanceRejected: number;
  claimMismatches: number;
  creditFrozen: number;
};

type ProblemOrder = {
  orderId: string;
  status: string;
  fiscalStatus: string;
  ehfId?: string;
  buyerAcceptanceStatus: string;
  forceCompletedAt?: string;
  forceReason?: string;
  claimId?: string;
  claimedAmountMinor?: number;
  createdAt: string;
};

type ComplianceDashboardResponse = {
  stats: DashboardStats;
  problems: ProblemOrder[];
};

export default function ComplianceDashboardPage() {
  // Default to last 30 days for compliance dashboard bounds
  const [dateRange] = useState(() => {
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - 30);
    return {
      from: from.toISOString(),
      to: to.toISOString(),
    };
  });

  const { data, loading, error, isRefreshing, mutate } =
    useLiveData<ComplianceDashboardResponse>(
      `/v1/compliance/dashboard?from=${encodeURIComponent(
        dateRange.from
      )}&to=${encodeURIComponent(dateRange.to)}`
    );

  const [exporting, setExporting] = useState(false);

  const refreshAll = useCallback(() => {
    void mutate();
  }, [mutate]);

  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await supplierFetch(
        `/v1/compliance/export?from=${encodeURIComponent(
          dateRange.from
        )}&to=${encodeURIComponent(dateRange.to)}`
      );
      if (!res.ok) throw new Error("Export failed");
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `compliance_audit_${
        new Date().toISOString().split("T")[0]
      }.csv`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(url);
    } catch (err: any) {
      console.error("Export error", err);
      alert("Failed to export compliance data");
    } finally {
      setExporting(false);
    }
  };

  const formatMinor = (val?: number, currency = "UZS") => {
    if (val === undefined || val === null) return "-";
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency,
    }).format(val / 100);
  };

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="shield"
        title="Compliance & Audit"
        description="Monitor fiscal irregularities, credit freezes, and claim mismatches across your network."
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={exporting || loading}
              onClick={handleExport}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <Download
                size={16}
                className={`mr-2 ${exporting ? "animate-bounce" : ""}`}
              />
              Export CSV
            </button>
            <button
              type="button"
              disabled={loading || isRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
              />
              {isRefreshing ? "Syncing" : "Sync"}
            </button>
          </div>
        }
      >
        {error && (
          <div className="mb-6 p-4 rounded-xl border bg-[var(--desk-danger)]/10 text-[var(--desk-danger)] border-[var(--desk-danger)]/30">
            {error.message || "Failed to load compliance dashboard."}
          </div>
        )}

        {data && (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
              <MetricCard
                title="Fiscalizing"
                value={data.stats.fiscalizing}
                icon={<Activity size={20} />}
                isAlert={data.stats.fiscalizing > 10}
                tooltipText="Orders currently stuck in fiscalization (e.g., Soliq API downtime)"
              />
              <MetricCard
                title="Fiscal Failed"
                value={data.stats.fiscalFailed}
                icon={<AlertOctagon size={20} />}
                isAlert={data.stats.fiscalFailed > 0}
                tooltipText="Orders that failed fiscalization completely"
              />
              <MetricCard
                title="Force Completed"
                value={data.stats.forceCompleted}
                icon={<AlertTriangle size={20} />}
                isAlert={data.stats.forceCompleted > 0}
                tooltipText="Orders forcefully completed bypassing fiscal gates"
              />
              <MetricCard
                title="Buyer Acceptance Pend."
                value={data.stats.buyerAcceptancePending}
                icon={<Clock size={20} />}
                isAlert={false}
                tooltipText="Orders waiting for buyer acceptance via Soliq API"
              />
              <MetricCard
                title="Buyer Acceptance Rej."
                value={data.stats.buyerAcceptanceRejected}
                icon={<XCircle size={20} />}
                isAlert={data.stats.buyerAcceptanceRejected > 0}
                tooltipText="Orders rejected by buyer on Soliq"
              />
              <MetricCard
                title="Claim Mismatches"
                value={data.stats.claimMismatches}
                icon={<FileWarning size={20} />}
                isAlert={data.stats.claimMismatches > 0}
                tooltipText="Claims exceeding order total or open on terminal orders"
              />
              <MetricCard
                title="Credit Freezes"
                value={data.stats.creditFrozen}
                icon={<Lock size={20} />}
                isAlert={data.stats.creditFrozen > 0}
                tooltipText="Retailers currently blacklisted or frozen due to credit limits"
              />
            </div>

            <div className="grid grid-cols-1 gap-8">
              <PageSection
                title="Problematic Orders"
                description="Orders requiring compliance review or manual intervention"
              >
                <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)] overflow-hidden">
                  {data.problems.length === 0 ? (
                    <div className="p-12 text-center text-[var(--desk-text-tertiary)] flex flex-col items-center">
                      <ShieldAlert size={48} className="opacity-20 mb-4" />
                      <p className="md-typescale-body-large">
                        No problematic orders found.
                      </p>
                    </div>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left">
                        <thead className="bg-[var(--desk-surface-hover)] border-b border-[var(--desk-border)]">
                          <tr>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)]">
                              Order ID
                            </th>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)]">
                              Status
                            </th>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)]">
                              Fiscal Status
                            </th>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)]">
                              Buyer Acceptance
                            </th>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)]">
                              Force Reason
                            </th>
                            <th className="px-4 py-3 text-sm font-medium text-[var(--desk-text-secondary)] text-right">
                              Claim Amount
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-[var(--desk-border)]">
                          {data.problems.map((row: ProblemOrder) => (
                            <tr
                              key={row.orderId}
                              className="hover:bg-[var(--desk-surface-hover)] transition-colors"
                            >
                              <td className="px-4 py-3">
                                <div className="font-medium text-[var(--desk-text-primary)]">
                                  {row.orderId}
                                </div>
                                {row.ehfId && (
                                  <div className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                                    EHF: {row.ehfId}
                                  </div>
                                )}
                              </td>
                              <td className="px-4 py-3 text-sm">
                                {row.status}
                              </td>
                              <td className="px-4 py-3 text-sm">
                                <span
                                  className={
                                    row.fiscalStatus.includes("FAILED")
                                      ? "text-[var(--desk-danger)] font-medium"
                                      : "text-[var(--desk-text-secondary)]"
                                  }
                                >
                                  {row.fiscalStatus}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-sm">
                                {row.buyerAcceptanceStatus}
                              </td>
                              <td className="px-4 py-3 text-sm text-[var(--desk-text-secondary)]">
                                {row.forceReason || "-"}
                                {row.forceCompletedAt && (
                                  <div className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                                    {new Date(
                                      row.forceCompletedAt
                                    ).toLocaleDateString()}
                                  </div>
                                )}
                              </td>
                              <td className="px-4 py-3 text-sm text-right">
                                {row.claimId ? (
                                  <>
                                    <div className="font-medium">
                                      {formatMinor(row.claimedAmountMinor)}
                                    </div>
                                    <div className="text-xs text-[var(--desk-text-tertiary)] mt-1 truncate max-w-[120px]">
                                      {row.claimId}
                                    </div>
                                  </>
                                ) : (
                                  "-"
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </PageSection>
            </div>
          </>
        )}
      </PageChrome>
    </div>
  );
}
