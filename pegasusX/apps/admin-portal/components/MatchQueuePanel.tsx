"use client";

import { useCallback, useEffect, useState } from "react";
import { api, type MatchQueueItem } from "@/lib/api";

export default function MatchQueuePanel({ token }: { token: string }) {
  const [status, setStatus] = useState("PENDING");
  const [items, setItems] = useState<MatchQueueItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const r = await api.listMatchQueue(token, status);
      setItems(r.items || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "load failed");
    } finally {
      setLoading(false);
    }
  }, [token, status]);

  useEffect(() => {
    void load();
  }, [load]);

  const resolve = async (id: string, decision: "ACCEPT" | "REJECT") => {
    setActing(id);
    setError("");
    try {
      await api.resolveMatch(token, id, decision);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "resolve failed");
    } finally {
      setActing(null);
    }
  };

  return (
    <section>
      <div className="mb-4 flex items-center gap-3">
        <label className="text-sm text-gray-600">Status</label>
        <select value={status} onChange={(e) => setStatus(e.target.value)} className="rounded border px-2 py-1 text-sm">
          {["PENDING", "RESOLVED", "REJECTED", ""].map((s) => (
            <option key={s || "all"} value={s}>
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
      ) : items.length === 0 ? (
        <p className="text-sm text-gray-500">No match-queue items.</p>
      ) : (
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="border-b text-left text-gray-500">
              <th className="py-2 pr-4 font-medium">Queue</th>
              <th className="py-2 pr-4 font-medium">Supplier</th>
              <th className="py-2 pr-4 font-medium">Product</th>
              <th className="py-2 pr-4 font-medium">Score</th>
              <th className="py-2 pr-4 font-medium">Status</th>
              <th className="py-2 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.queue_id} className="border-b">
                <td className="py-2 pr-4 font-mono text-xs">{it.queue_id.slice(0, 10)}</td>
                <td className="py-2 pr-4 font-mono text-xs">{it.supplier_id.slice(0, 10)}</td>
                <td className="py-2 pr-4 font-mono text-xs">{it.product_id.slice(0, 10)}</td>
                <td className="py-2 pr-4">{it.score.toFixed(2)}</td>
                <td className="py-2 pr-4">{it.status}</td>
                <td className="py-2">
                  {it.status === "PENDING" && (
                    <span className="flex gap-2">
                      <button
                        disabled={acting === it.queue_id}
                        onClick={() => void resolve(it.queue_id, "ACCEPT")}
                        className="rounded border px-2 py-0.5 text-xs hover:bg-green-50"
                      >
                        Accept
                      </button>
                      <button
                        disabled={acting === it.queue_id}
                        onClick={() => void resolve(it.queue_id, "REJECT")}
                        className="rounded border px-2 py-0.5 text-xs hover:bg-red-50"
                      >
                        Reject
                      </button>
                    </span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
