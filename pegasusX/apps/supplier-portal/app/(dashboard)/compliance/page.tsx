"use client";

import React, { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import type { Route } from "next";
import { supplierFetch } from "@/lib/auth";
import { sessionSupplierId } from "@/lib/supplier-scope";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";

interface DashboardStats {
  fiscalizing: number;
  fiscalFailed: number;
  forceCompleted: number;
  buyerAcceptancePending: number;
  buyerAcceptanceRejected: number;
  claimMismatches: number;
  creditFrozen: number;
  openCreditNotes: number;
  openReverseLogisticsTasks: number;
  openCashDiscrepancies: number;
}

interface ProblemOrder {
  orderId: string;
  status: string;
  fiscalStatus: string;
  ehfId?: string;
  buyerAcceptanceStatus?: string;
  forceCompletedAt?: string;
  forceReason?: string;
  claimId?: string;
  claimedAmountMinor?: number;
  createdAt: string;
}

function MetricCard({
  title,
  value,
  alert,
  href,
}: {
  title: string;
  value: number;
  alert?: boolean;
  href?: Route;
}) {
  const card = (
    <div className={`p-6 rounded-xl border ${alert ? "border-red-300 bg-red-50" : "border-gray-200 bg-white"}`}>
      <h3 className={`text-sm font-medium ${alert ? "text-red-800" : "text-gray-500"}`}>{title}</h3>
      <p className={`text-3xl font-bold mt-2 ${alert ? "text-red-600" : "text-gray-900"}`}>{value}</p>
      {href ? (
        <p className="text-xs text-gray-500 mt-2 underline">View in Exception Centre</p>
      ) : null}
    </div>
  );
  if (href) {
    return (
      <Link href={href} className="block hover:opacity-90 transition-opacity">
        {card}
      </Link>
    );
  }
  return card;
}

export default function ComplianceDashboard() {
  const supplierId = sessionSupplierId();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [orders, setOrders] = useState<ProblemOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const fetchDashboardData = useCallback(async () => {
    if (!supplierId) {
      setLoading(false);
      setStats(null);
      setOrders([]);
      setError("No supplier session. Sign in again to load compliance for your tenant.");
      return;
    }
    try {
      setLoading(true);
      setError(null);
      // Supplier scope comes from the session JWT on the backend; do not trust query tenant ids.
      const res = await supplierFetch("/v1/compliance/dashboard");
      if (!res.ok) {
        const body = await res.text().catch(() => "");
        throw new Error(body || `Failed to load compliance stats (${res.status})`);
      }
      const data = await res.json();
      setStats(data.stats);
      setOrders(data.orders || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load compliance stats");
      setStats(null);
      setOrders([]);
    } finally {
      setLoading(false);
    }
  }, [supplierId]);

  useSupplierSessionReconcile(fetchDashboardData);

  useEffect(() => {
    void fetchDashboardData();
  }, [fetchDashboardData]);

  const handleExport = async () => {
    if (!supplierId || exporting) return;
    try {
      setExporting(true);
      setError(null);
      const res = await supplierFetch("/v1/compliance/export");
      if (!res.ok) {
        throw new Error(`Export failed (${res.status})`);
      }
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `compliance_export_${new Date().toISOString().slice(0, 10)}.csv`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Export failed");
    } finally {
      setExporting(false);
    }
  };

  if (!supplierId) {
    return (
      <div className="p-8 max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold tracking-tight">Compliance & Fiscal Audit</h1>
        <p className="text-gray-500 mt-2">
          No supplier session. Sign in again to load compliance for your tenant.
        </p>
      </div>
    );
  }

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Compliance & Fiscal Audit</h1>
          <p className="text-gray-500 mt-2">
            Track open fiscal states, claim mismatches, and ecosystem integrity for {supplierId}.
          </p>
        </div>
        <button
          onClick={() => void handleExport()}
          disabled={exporting}
          className="bg-black hover:bg-gray-800 disabled:opacity-50 text-white px-4 py-2 rounded-lg font-medium transition-colors"
        >
          {exporting ? "Exporting…" : "Export Soliq Audit (CSV)"}
        </button>
      </div>

      {error ? <div className="text-red-600 text-sm">{error}</div> : null}

      {loading ? (
        <div className="text-gray-500 text-center py-12">Loading statistics...</div>
      ) : (
        <>
          {stats && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mt-8">
              <MetricCard title="Fiscalizing" value={stats.fiscalizing} />
              <MetricCard title="Fiscal Failed" value={stats.fiscalFailed} alert={stats.fiscalFailed > 0} />
              <MetricCard title="Force Completed" value={stats.forceCompleted} alert={stats.forceCompleted > 0} />
              <MetricCard title="BA Pending" value={stats.buyerAcceptancePending} />
              <MetricCard title="BA Rejected" value={stats.buyerAcceptanceRejected} alert={stats.buyerAcceptanceRejected > 0} />
              <MetricCard title="Claim Mismatches" value={stats.claimMismatches} alert={stats.claimMismatches > 0} />
              <MetricCard title="Credit Frozen" value={stats.creditFrozen} alert={stats.creditFrozen > 0} />
              <MetricCard
                title="Open Credit Notes"
                value={stats.openCreditNotes}
                alert={stats.openCreditNotes > 0}
                href={"/finance/credit-notes" as Route}
              />
              <MetricCard
                title="Open Reverse Logistics"
                value={stats.openReverseLogisticsTasks}
                alert={stats.openReverseLogisticsTasks > 0}
                href={"/exceptions" as Route}
              />
              <MetricCard
                title="Open Cash Discrepancies"
                value={stats.openCashDiscrepancies}
                alert={stats.openCashDiscrepancies > 0}
                href={"/treasury/cash-reconciliations" as Route}
              />
            </div>
          )}

          {!stats && !error ? (
            <div className="text-gray-500 text-center py-12">No compliance stats for this supplier yet.</div>
          ) : null}

          {orders && orders.length > 0 && (
            <div className="mt-12">
              <h2 className="text-xl font-bold mb-4">Problematic Orders</h2>
              <div className="overflow-x-auto border border-gray-200 rounded-lg">
                <table className="w-full text-left text-sm text-gray-500">
                  <thead className="text-xs text-gray-700 uppercase bg-gray-50">
                    <tr>
                      <th className="px-4 py-3">Order ID</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Fiscal Status</th>
                      <th className="px-4 py-3">EHF ID</th>
                      <th className="px-4 py-3">BA Status</th>
                      <th className="px-4 py-3">Created At</th>
                    </tr>
                  </thead>
                  <tbody>
                    {orders.map((order) => (
                      <tr key={order.orderId} className="border-b">
                        <td className="px-4 py-3 font-medium text-gray-900">{order.orderId}</td>
                        <td className="px-4 py-3">{order.status}</td>
                        <td className="px-4 py-3">{order.fiscalStatus}</td>
                        <td className="px-4 py-3">{order.ehfId || "-"}</td>
                        <td className="px-4 py-3">{order.buyerAcceptanceStatus || "-"}</td>
                        <td className="px-4 py-3">{new Date(order.createdAt).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
