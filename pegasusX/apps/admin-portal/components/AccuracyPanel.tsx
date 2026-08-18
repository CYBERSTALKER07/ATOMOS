"use client";

import { useState } from "react";
import { api, type AccuracyRow } from "@/lib/api";

export default function AccuracyPanel({ token }: { token: string }) {
  const [supplierId, setSupplierId] = useState("");
  const [rows, setRows] = useState<AccuracyRow[] | null>(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [demote, setDemote] = useState<boolean | null>(null);

  const load = async () => {
    const id = supplierId.trim();
    if (!id) {
      setErr("supplier_id required — no invented series");
      setRows(null);
      return;
    }
    setLoading(true);
    setErr("");
    try {
      const r = await api.listPlanningAccuracy(token, id);
      setRows(r.items || []);
      setDemote(r.demote_enabled);
    } catch (e) {
      setRows(null);
      setErr(e instanceof Error ? e.message : "accuracy_load_failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="space-y-4" data-testid="gs-u-admin-accuracy">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm">
          <span className="mb-1 block text-gray-600">Supplier ID</span>
          <input
            value={supplierId}
            onChange={(e) => setSupplierId(e.target.value)}
            placeholder="supplier_…"
            className="w-64 rounded border px-2 py-1 text-sm font-mono"
          />
        </label>
        <button onClick={() => void load()} className="rounded border px-3 py-1 text-sm hover:bg-gray-100">
          Load mape28
        </button>
      </div>
      {demote != null && (
        <p className="text-xs text-gray-500">demote_enabled={String(demote)} — live flag, not a chart.</p>
      )}
      {err && <p className="text-sm text-red-700">{err}</p>}
      {loading && <p className="text-sm text-gray-600">Loading…</p>}
      {rows && rows.length === 0 && <p className="text-sm text-gray-600">empty</p>}
      {rows && rows.length > 0 && (
        <table className="w-full text-left text-xs">
          <thead>
            <tr className="border-b text-gray-500">
              <th className="py-1 pr-2">Date</th>
              <th className="py-1 pr-2">SKU</th>
              <th className="py-1 pr-2">mape28</th>
              <th className="py-1 pr-2">wape28</th>
              <th className="py-1">demoted</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr key={`${row.product_id}-${row.forecast_date}-${i}`} className="border-b border-gray-100">
                <td className="py-1 pr-2">{String(row.forecast_date)}</td>
                <td className="py-1 pr-2 font-mono">{row.product_id}</td>
                <td className="py-1 pr-2">{row.mape28}</td>
                <td className="py-1 pr-2">{row.wape28}</td>
                <td className="py-1">{row.demoted ? "yes" : "no"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {!rows && !loading && !err && (
        <p className="text-sm text-gray-600">unavailable until supplier_id — no invented mape28 line</p>
      )}
    </section>
  );
}
