"use client";

import { useCallback, useEffect, useState } from "react";
import { api, type AuditRow } from "@/lib/api";

export default function AuditPanel({ token, refreshKey = 0 }: { token: string; refreshKey?: number }) {
  const [rows, setRows] = useState<AuditRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const r = await api.listAudit(token, 200);
      setRows(r.audit || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "load failed");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  return (
    <section>
      <div className="mb-4">
        <button onClick={() => void load()} className="rounded border px-3 py-1 text-sm hover:bg-gray-100">
          Refresh
        </button>
      </div>
      {error && <p className="mb-3 rounded bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>}
      {loading ? (
        <p className="text-sm text-gray-500">Loading…</p>
      ) : rows.length === 0 ? (
        <p className="text-sm text-gray-500">No audit records.</p>
      ) : (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b text-left text-gray-500">
              <th className="py-2 pr-4 font-medium">Time</th>
              <th className="py-2 pr-4 font-medium">Actor</th>
              <th className="py-2 pr-4 font-medium">Action</th>
              <th className="py-2 pr-4 font-medium">Tenant</th>
              <th className="py-2 font-medium">Detail</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((a) => (
              <tr key={a.AuditID} className="border-b last:border-0">
                <td className="py-2 pr-4 text-gray-500">{new Date(a.CreatedAt).toLocaleString()}</td>
                <td className="py-2 pr-4 font-mono text-xs">{a.ActorSubject}</td>
                <td className="py-2 pr-4">{a.Action}</td>
                <td className="py-2 pr-4 font-mono text-xs">
                  {a.TenantType}/{a.TenantID}
                </td>
                <td className="py-2 font-mono text-xs text-gray-600">{a.DetailJSON}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
