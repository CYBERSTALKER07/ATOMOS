"use client";

import React, { useEffect, useState, useMemo } from "react";
import { api, type AuditRow } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

export default function AuditLedgerPage() {
  const { token } = useAdminToken();

  const [rows, setRows] = useState<AuditRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedAudit, setSelectedAudit] = useState<AuditRow | null>(null);

  const fetchAudit = async () => {
    if (!token) return;
    setError(null);
    setLoading(true);
    try {
      const res = await api.listAudit(token, 250);
      setRows(res.audit || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load audit records");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAudit();
  }, [token]);

  const filteredRows = useMemo(() => {
    if (!searchTerm.trim()) return rows;
    const q = searchTerm.toLowerCase();
    return rows.filter(
      (r) =>
        r.AuditID?.toLowerCase().includes(q) ||
        r.ActorSubject?.toLowerCase().includes(q) ||
        r.Action?.toLowerCase().includes(q) ||
        r.TenantID?.toLowerCase().includes(q) ||
        r.DetailJSON?.toLowerCase().includes(q)
    );
  }, [rows, searchTerm]);

  const exportCSV = () => {
    if (filteredRows.length === 0) return;
    const headers = "AuditID,CreatedAt,Actor,Action,TenantType,TenantID,DetailJSON\n";
    const body = filteredRows
      .map(
        (r) =>
          `"${r.AuditID}","${r.CreatedAt}","${r.ActorSubject}","${r.Action}","${r.TenantType}","${r.TenantID}","${(r.DetailJSON || "").replace(/"/g, '""')}"`
      )
      .join("\n");
    const blob = new Blob([headers + body], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit_ledger_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Platform Tamper-Evident Audit Ledger
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Immutable Spanner mutation log recording actor attribution, tenant scope, and payload diffs (CG-2 & UA-12).
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={exportCSV}
            disabled={filteredRows.length === 0}
            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50"
          >
            Export CSV
          </button>
          <button
            type="button"
            onClick={fetchAudit}
            disabled={loading}
            className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
          >
            {loading ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Search & Filter Bar */}
      <div className="flex items-center justify-between gap-4">
        <div className="relative flex-1 max-w-md">
          <input
            type="text"
            placeholder="Search by Actor, Action, Tenant ID, or payload content..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-xs focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
          />
        </div>
        <span className="text-xs text-gray-500 font-medium">
          Showing {filteredRows.length} of {rows.length} records
        </span>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border bg-white shadow-sm">
        <table className="w-full border-collapse text-left text-xs">
          <thead className="border-b bg-gray-50 font-bold uppercase tracking-wider text-gray-500">
            <tr>
              <th className="px-4 py-3">Timestamp</th>
              <th className="px-4 py-3">Actor</th>
              <th className="px-4 py-3">Action</th>
              <th className="px-4 py-3">Tenant</th>
              <th className="px-4 py-3">Audit ID</th>
              <th className="px-4 py-3 text-right">Details</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {filteredRows.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                  {loading ? "Loading audit records from Spanner..." : "No audit entries matched the search criteria."}
                </td>
              </tr>
            ) : (
              filteredRows.map((row) => (
                <tr key={row.AuditID} className="hover:bg-gray-50/70 transition-colors">
                  <td className="whitespace-nowrap px-4 py-2.5 text-gray-500">
                    {new Date(row.CreatedAt).toLocaleString()}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 font-mono text-gray-800">
                    {row.ActorSubject}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 font-semibold text-gray-900">
                    <span className="rounded bg-indigo-50 px-1.5 py-0.5 text-[11px] text-indigo-700">
                      {row.Action}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 font-mono text-gray-600">
                    {row.TenantType ? `${row.TenantType}:${row.TenantID || "*"}` : "-"}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 font-mono text-gray-500">
                    <CopyableId id={row.AuditID} label="Audit" />
                  </td>
                  <td className="whitespace-nowrap px-4 py-2.5 text-right">
                    <button
                      type="button"
                      onClick={() => setSelectedAudit(row)}
                      className="rounded bg-gray-100 px-2 py-1 text-[11px] font-semibold text-gray-700 hover:bg-gray-200"
                    >
                      Inspect Diff
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Slide-Over Mutation Diff Drawer */}
      {selectedAudit && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-xs">
          <div className="w-full max-w-lg bg-white p-6 shadow-2xl flex flex-col justify-between overflow-y-auto">
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="text-sm font-bold text-gray-900">
                  Audit Mutation Inspection
                </h3>
                <button
                  type="button"
                  onClick={() => setSelectedAudit(null)}
                  className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Audit ID:</span>
                  <CopyableId id={selectedAudit.AuditID} label="Audit ID" />
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Actor Subject:</span>
                  <span className="font-mono font-semibold">{selectedAudit.ActorSubject}</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Action:</span>
                  <span className="font-bold text-indigo-700">{selectedAudit.Action}</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Recorded At:</span>
                  <span>{new Date(selectedAudit.CreatedAt).toLocaleString()}</span>
                </div>
                <div className="flex justify-between py-1 border-b">
                  <span className="text-gray-500">Tenant Target:</span>
                  <span className="font-mono">{selectedAudit.TenantType}:{selectedAudit.TenantID || "GLOBAL"}</span>
                </div>
              </div>

              <div>
                <h4 className="text-xs font-bold text-gray-700 uppercase tracking-wider mb-1">
                  Detail / Mutation JSON
                </h4>
                <pre className="rounded-lg bg-gray-900 p-3 text-xs text-emerald-400 font-mono overflow-x-auto max-h-96">
                  {(() => {
                    try {
                      return JSON.stringify(JSON.parse(selectedAudit.DetailJSON || "{}"), null, 2);
                    } catch {
                      return selectedAudit.DetailJSON || "{}";
                    }
                  })()}
                </pre>
              </div>
            </div>

            <div className="pt-4 border-t">
              <button
                type="button"
                onClick={() => setSelectedAudit(null)}
                className="w-full rounded-lg border border-gray-300 py-2 text-xs font-bold text-gray-700 hover:bg-gray-50"
              >
                Close Drawer
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
