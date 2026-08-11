"use client";

import { useCallback, useEffect, useState } from "react";
import { api, type Tenant } from "@/lib/api";

const STATUSES = ["", "PENDING", "APPROVED", "SUSPENDED", "OFFBOARDED"] as const;
const TRANSITIONS: Record<string, string[]> = {
  PENDING: ["APPROVED", "OFFBOARDED"],
  APPROVED: ["SUSPENDED", "OFFBOARDED"],
  SUSPENDED: ["APPROVED", "OFFBOARDED"],
  OFFBOARDED: [],
};

export default function TenantsPanel({ token }: { token: string }) {
  const [status, setStatus] = useState("");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const r = await api.listTenants(token, status);
      setTenants(r.tenants || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "load failed");
    } finally {
      setLoading(false);
    }
  }, [token, status]);

  useEffect(() => {
    void load();
  }, [load]);

  const transition = async (t: Tenant, to: string) => {
    const key = `${t.TenantType}/${t.TenantID}`;
    const notes = to === "APPROVED" ? window.prompt("KYB notes (optional):") ?? "" : "";
    setActing(key);
    setError("");
    try {
      await api.transitionTenant(token, t.TenantType, t.TenantID, to, notes);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "transition failed");
    } finally {
      setActing(null);
    }
  };

  return (
    <section>
      <div className="mb-4 flex items-center gap-3">
        <label className="text-sm text-gray-600">Status</label>
        <select value={status} onChange={(e) => setStatus(e.target.value)} className="rounded border px-2 py-1 text-sm">
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {s || "All"}
            </option>
          ))}
        </select>
        <button onClick={() => void load()} className="rounded border px-3 py-1 text-sm hover:bg-gray-100">
          Refresh
        </button>
      </div>
      {error && <p className="mb-3 rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
      {loading ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : tenants.length === 0 ? (
        <p className="text-sm text-gray-500">No tenants.</p>
      ) : (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b text-left text-gray-500">
              <th className="py-2 pr-4 font-medium">Type</th>
              <th className="py-2 pr-4 font-medium">ID</th>
              <th className="py-2 pr-4 font-medium">Name</th>
              <th className="py-2 pr-4 font-medium">Status</th>
              <th className="py-2 pr-4 font-medium">Updated</th>
              <th className="py-2 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {tenants.map((t) => {
              const key = `${t.TenantType}/${t.TenantID}`;
              return (
                <tr key={key} className="border-b last:border-0">
                  <td className="py-2 pr-4">{t.TenantType}</td>
                  <td className="py-2 pr-4 font-mono text-xs">{t.TenantID}</td>
                  <td className="py-2 pr-4">{t.DisplayName || "—"}</td>
                  <td className="py-2 pr-4">
                    <StatusBadge status={t.Status} />
                  </td>
                  <td className="py-2 pr-4 text-gray-500">{t.UpdatedAt ? new Date(t.UpdatedAt).toLocaleDateString() : "—"}</td>
                  <td className="py-2">
                    <div className="flex gap-2">
                      {(TRANSITIONS[t.Status] || []).map((to) => (
                        <button
                          key={to}
                          disabled={acting === key}
                          onClick={() => void transition(t, to)}
                          className="rounded border px-2 py-1 text-xs hover:bg-gray-100 disabled:opacity-50"
                        >
                          {to}
                        </button>
                      ))}
                      {(TRANSITIONS[t.Status] || []).length === 0 && <span className="text-xs text-gray-400">—</span>}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </section>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    PENDING: "bg-amber-100 text-amber-800",
    APPROVED: "bg-green-100 text-green-800",
    SUSPENDED: "bg-red-100 text-red-800",
    OFFBOARDED: "bg-gray-200 text-gray-700",
  };
  return <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors[status] || "bg-gray-100 text-gray-700"}`}>{status}</span>;
}
