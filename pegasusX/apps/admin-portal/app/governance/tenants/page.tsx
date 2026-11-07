"use client";

import React, { useEffect, useState, useCallback } from "react";
import { api, type Tenant, type MatchQueueItem } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

const STATUSES = ["", "PENDING", "APPROVED", "SUSPENDED", "OFFBOARDED"] as const;
const TRANSITIONS: Record<string, string[]> = {
  PENDING: ["APPROVED", "OFFBOARDED"],
  APPROVED: ["SUSPENDED", "OFFBOARDED"],
  SUSPENDED: ["APPROVED", "OFFBOARDED"],
  OFFBOARDED: [],
};

export default function TenantsCatalogGovernancePage() {
  const { token } = useAdminToken();

  const [activeTab, setActiveTab] = useState<"tenants" | "match">("tenants");

  // Tenants State
  const [tenantStatusFilter, setTenantStatusFilter] = useState("");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loadingTenants, setLoadingTenants] = useState(false);

  // Match Queue State
  const [matchStatusFilter, setMatchStatusFilter] = useState("PENDING");
  const [matchItems, setMatchItems] = useState<MatchQueueItem[]>([]);
  const [loadingMatches, setLoadingMatches] = useState(false);

  const [acting, setActing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const loadTenants = useCallback(async () => {
    if (!token) return;
    setLoadingTenants(true);
    setError(null);
    try {
      const res = await api.listTenants(token, tenantStatusFilter);
      setTenants(res.tenants || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load tenants");
    } finally {
      setLoadingTenants(false);
    }
  }, [token, tenantStatusFilter]);

  const loadMatches = useCallback(async () => {
    if (!token) return;
    setLoadingMatches(false);
    setError(null);
    try {
      const res = await api.listMatchQueue(token, matchStatusFilter);
      setMatchItems(res.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load product match queue");
    } finally {
      setLoadingMatches(false);
    }
  }, [token, matchStatusFilter]);

  useEffect(() => {
    if (activeTab === "tenants") {
      void loadTenants();
    } else {
      void loadMatches();
    }
  }, [activeTab, loadTenants, loadMatches]);

  const handleTransitionTenant = async (t: Tenant, toStatus: string) => {
    if (!token) return;
    const key = `${t.TenantType}/${t.TenantID}`;
    const notes = toStatus === "APPROVED" ? window.prompt("KYB verification notes (optional):") ?? "" : "";
    setActing(key);
    setError(null);
    setSuccessMsg(null);

    try {
      await api.transitionTenant(token, t.TenantType, t.TenantID, toStatus, notes);
      setSuccessMsg(`Tenant ${t.TenantID} transitioned to ${toStatus}.`);
      await loadTenants();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Tenant transition failed");
    } finally {
      setActing(null);
    }
  };

  const handleResolveMatch = async (id: string, decision: "ACCEPT" | "REJECT") => {
    if (!token) return;
    setActing(id);
    setError(null);
    setSuccessMsg(null);

    try {
      await api.resolveMatch(token, id, decision);
      setSuccessMsg(`Match item resolved as ${decision}.`);
      await loadMatches();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Match resolution failed");
    } finally {
      setActing(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Tenant KYB Pipeline & Catalog Taxonomy Engine
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Role Platform Governance: Multi-tenant KYB lifecycle transitions and automated GTIN product match deduplication (UA-3).
          </p>
        </div>

        {/* Tab Switcher */}
        <div className="flex rounded-lg border bg-gray-100 p-0.5 text-xs font-semibold">
          <button
            type="button"
            onClick={() => setActiveTab("tenants")}
            className={`rounded-md px-3 py-1 ${
              activeTab === "tenants" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
            }`}
          >
            Tenants KYB Pipeline
          </button>
          <button
            type="button"
            onClick={() => setActiveTab("match")}
            className={`rounded-md px-3 py-1 ${
              activeTab === "match" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
            }`}
          >
            Product Match Queue
          </button>
        </div>
      </div>

      {/* Notifications */}
      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
          <span className="text-base text-rose-600 font-bold">⚠</span>
          <div>{error}</div>
        </div>
      )}
      {successMsg && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800 flex items-start gap-2">
          <span className="text-base text-emerald-600 font-bold">✓</span>
          <div>{successMsg}</div>
        </div>
      )}

      {/* Tab 1: Tenants KYB Pipeline */}
      {activeTab === "tenants" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <label className="text-xs font-bold text-gray-700">Filter Status:</label>
              <select
                value={tenantStatusFilter}
                onChange={(e) => setTenantStatusFilter(e.target.value)}
                className="rounded border px-2.5 py-1 text-xs font-medium"
              >
                {STATUSES.map((s) => (
                  <option key={s || "all"} value={s}>
                    {s || "All Statuses"}
                  </option>
                ))}
              </select>
            </div>

            <button
              type="button"
              onClick={loadTenants}
              disabled={loadingTenants}
              className="rounded bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
            >
              {loadingTenants ? "Refreshing..." : "Refresh"}
            </button>
          </div>

          <div className="overflow-hidden rounded-xl border bg-white shadow-sm">
            <table className="w-full border-collapse text-left text-xs">
              <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-4 py-3">Tenant Type</th>
                  <th className="px-4 py-3">Tenant ID</th>
                  <th className="px-4 py-3">Name / Entity</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Home Cell</th>
                  <th className="px-4 py-3">Created</th>
                  <th className="px-4 py-3 text-right">KYB Transition</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {tenants.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                      {loadingTenants ? "Loading tenants..." : "No tenants found."}
                    </td>
                  </tr>
                ) : (
                  tenants.map((t) => {
                    const key = `${t.TenantType}/${t.TenantID}`;
                    const isBusy = acting === key;
                    const nextStatuses = TRANSITIONS[t.Status] || [];
                    return (
                      <tr key={key} className="hover:bg-gray-50/70 transition-colors">
                        <td className="whitespace-nowrap px-4 py-2.5 font-semibold text-gray-800">
                          {t.TenantType}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 font-mono">
                          <CopyableId id={t.TenantID} label="Tenant" />
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 font-medium text-gray-900">
                          {t.DisplayName || "-"}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5">
                          <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold ${
                              t.Status === "APPROVED"
                                ? "bg-emerald-100 text-emerald-800"
                                : t.Status === "PENDING"
                                ? "bg-amber-100 text-amber-800"
                                : "bg-rose-100 text-rose-800"
                            }`}
                          >
                            {t.Status}
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 text-gray-600 font-mono text-[11px]">
                          {t.home_cell || "cell-uz"}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 text-gray-500">
                          {new Date(t.CreatedAt).toLocaleDateString()}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 text-right space-x-1.5">
                          {nextStatuses.map((next) => (
                            <button
                              key={next}
                              type="button"
                              onClick={() => handleTransitionTenant(t, next)}
                              disabled={isBusy}
                              className={`rounded px-2 py-1 text-[10px] font-bold shadow-xs disabled:opacity-50 ${
                                next === "APPROVED"
                                  ? "bg-emerald-600 text-white hover:bg-emerald-700"
                                  : next === "SUSPENDED"
                                  ? "bg-amber-600 text-white hover:bg-amber-700"
                                  : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                              }`}
                            >
                              {next}
                            </button>
                          ))}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Tab 2: Catalog GTIN Product Match Queue */}
      {activeTab === "match" && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <label className="text-xs font-bold text-gray-700">Filter Status:</label>
              <select
                value={matchStatusFilter}
                onChange={(e) => setMatchStatusFilter(e.target.value)}
                className="rounded border px-2.5 py-1 text-xs font-medium"
              >
                {["PENDING", "RESOLVED", "REJECTED", ""].map((s) => (
                  <option key={s || "all"} value={s}>
                    {s || "All"}
                  </option>
                ))}
              </select>
            </div>

            <button
              type="button"
              onClick={loadMatches}
              disabled={loadingMatches}
              className="rounded bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
            >
              {loadingMatches ? "Refreshing..." : "Refresh"}
            </button>
          </div>

          <div className="overflow-hidden rounded-xl border bg-white shadow-sm">
            <table className="w-full border-collapse text-left text-xs">
              <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
                <tr>
                  <th className="px-4 py-3">Queue ID</th>
                  <th className="px-4 py-3">Supplier ID</th>
                  <th className="px-4 py-3">Product ID</th>
                  <th className="px-4 py-3">Candidate Global ID</th>
                  <th className="px-4 py-3">Score</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3 text-right">Decision</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {matchItems.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                      {loadingMatches ? "Loading product match items..." : "No product match items in queue."}
                    </td>
                  </tr>
                ) : (
                  matchItems.map((item) => {
                    const isBusy = acting === item.queue_id;
                    return (
                      <tr key={item.queue_id} className="hover:bg-gray-50/70 transition-colors">
                        <td className="whitespace-nowrap px-4 py-2.5 font-mono">
                          <CopyableId id={item.queue_id} label="Queue" />
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 font-mono text-gray-700">
                          {item.supplier_id}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 font-semibold text-gray-900">
                          {item.product_id}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 font-mono text-gray-600">
                          {item.candidate_global_product_id || "-"}
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5">
                          <span className="rounded bg-blue-50 px-1.5 py-0.5 font-bold text-blue-700">
                            {((item.score || 0) * 100).toFixed(0)}%
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5">
                          <span
                            className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                              item.status === "RESOLVED"
                                ? "bg-emerald-100 text-emerald-800"
                                : item.status === "PENDING"
                                ? "bg-amber-100 text-amber-800"
                                : "bg-gray-100 text-gray-700"
                            }`}
                          >
                            {item.status}
                          </span>
                        </td>
                        <td className="whitespace-nowrap px-4 py-2.5 text-right space-x-1.5">
                          {item.status === "PENDING" && (
                            <>
                              <button
                                type="button"
                                onClick={() => handleResolveMatch(item.queue_id, "ACCEPT")}
                                disabled={isBusy}
                                className="rounded bg-emerald-600 px-2.5 py-1 text-[10px] font-bold text-white shadow hover:bg-emerald-700 disabled:opacity-50"
                              >
                                Accept Match
                              </button>
                              <button
                                type="button"
                                onClick={() => handleResolveMatch(item.queue_id, "REJECT")}
                                disabled={isBusy}
                                className="rounded bg-rose-600 px-2.5 py-1 text-[10px] font-bold text-white shadow hover:bg-rose-700 disabled:opacity-50"
                              >
                                Reject
                              </button>
                            </>
                          )}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
