"use client";

import { useState } from "react";
import { getAdminToken } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';

type ReconciliationRecord = {
  order_id: string;
  retailer_id: string;
  spanner_amount: number;
  gateway_amount: number;
  gateway_provider: string;
  status: string;
  timestamp: string;
};

export default function ReconciliationPage() {
  const [records, setRecords] = useState<ReconciliationRecord[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [hasMore, setHasMore] = useState(false);
  const offset = (page - 1) * pageSize;
  const canPrev = page > 1;
  const canNext = hasMore;

  usePolling(async (signal) => {
    const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    try {
      const token = await getAdminToken();
      const res = await fetch(`${API}/v1/admin/reconciliation?limit=${pageSize + 1}&offset=${offset}`, {
        headers: { Authorization: `Bearer ${token}` }, signal,
      });
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json();
      const rows: ReconciliationRecord[] = data.data || [];
      setHasMore(rows.length > pageSize);
      setRecords(rows.slice(0, pageSize));
      setLastRefreshed(new Date());
      setIsLoading(false);
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      console.error("Reconciliation fetch error:", err);
      setRecords([]);
      setHasMore(false);
      setIsLoading(false);
    }
  }, 5000, [pageSize, offset]);

  const deltaCount = records.filter(r => r.status === "DELTA").length;
  const orphanedCount = records.filter(r => r.status === "ORPHANED").length;
  const totalExposure = records.reduce((sum, r) => sum + Math.abs(r.spanner_amount - r.gateway_amount), 0);

  const getStatusBadge = (status: string) => {
    const map: Record<string, { bg: string; color: string; label: string }> = {
      DELTA:    { bg: 'var(--default)', color: 'var(--default-foreground)', label: 'Delta' },
      ORPHANED: { bg: 'var(--danger)',    color: 'var(--danger-foreground)', label: 'Orphaned' },
      MATCH:    { bg: 'var(--accent-soft)',  color: 'var(--accent-soft-foreground)', label: 'Match' },
    };
    const s = map[status] ?? { bg: 'var(--surface)', color: 'var(--muted)', label: status };
    return <span className="md-chip md-typescale-label-small" style={{ background: s.bg, color: s.color, borderColor: 'transparent', cursor: 'default', height: 26 }}>{s.label}</span>;
  };

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <header className="mb-10">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="md-typescale-headline-medium" style={{ color: 'var(--foreground)' }}>Reconciliation</h1>
            <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>Spanner ↔ Gateway Settlement Anomaly Scanner</p>
          </div>
          {lastRefreshed && (
            <span className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>
              Last refreshed {lastRefreshed.toLocaleTimeString()}
            </span>
          )}
        </div>
      </header>

      {/* KPI Cards — M3 Filled Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="md-card md-card-elevated p-6 h-32 flex flex-col justify-between">
              <div className="w-1/2 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
              <div className="w-2/3 h-8 rounded mt-4 animate-pulse" style={{ background: 'var(--surface)' }} />
            </div>
          ))
        ) : (
          <>
            {[
              { label: "Total Anomalies", value: records.length, color: 'var(--foreground)' },
              { label: "Delta Mismatches", value: deltaCount, color: 'var(--muted)' },
              { label: "Orphaned Records", value: orphanedCount, color: 'var(--danger)' },
              { label: "Total Exposure (Amount)", value: totalExposure.toLocaleString(), color: 'var(--foreground)' },
            ].map(({ label, value, color }, i) => (
              <div key={i} className="md-card md-card-elevated p-6 flex flex-col justify-between cursor-default md-animate-in" style={{ animationDelay: `${i * 50}ms` }}>
                <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
                <p className="md-typescale-headline-small tracking-tight" style={{ color, fontVariantNumeric: 'tabular-nums' }}>{value}</p>
              </div>
            ))}
          </>
        )}
      </div>

      {/* Anomaly Table — M3 Data Table */}
      <main className="md-animate-in" style={{ animationDelay: '200ms' }}>
        <div className="w-full overflow-hidden md-card md-card-outlined p-0">
          <table className="md-table">
            <thead>
              <tr>
                <th>Order ID</th>
                <th>Retailer</th>
                <th className="text-right">Spanner (Amount)</th>
                <th className="text-right">Gateway (Amount)</th>
                <th className="text-right">Delta</th>
                <th>Provider</th>
                <th>Detected</th>
                <th className="text-right">Status</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <tr key={`skel-${i}`}>
                    <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-20 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-20 h-4 rounded animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-20 h-4 rounded animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-16 h-4 rounded animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-16 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    <td><div className="w-20 h-4 rounded animate-pulse ml-auto" style={{ background: 'var(--surface)' }} /></td>
                  </tr>
                ))
              ) : records.length === 0 ? (
                <tr>
                  <td colSpan={8} className="p-16 text-center">
                    <svg width="40" height="40" viewBox="0 0 24 24" fill="var(--success)" className="mx-auto mb-4"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>
                    <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>No anomalies detected</p>
                    <p className="md-typescale-body-small mt-1" style={{ color: 'var(--border)' }}>Spanner and Gateway ledgers are in sync</p>
                  </td>
                </tr>
              ) : (
                records.map((rec, i) => {
                  const delta = rec.spanner_amount - rec.gateway_amount;
                  const isNegative = delta < 0;
                  return (
                    <tr key={rec.order_id || `idx-${i}`} className="transition-colors cursor-pointer">
                      <td className="font-mono md-typescale-body-small font-medium">{rec.order_id}</td>
                      <td className="md-typescale-body-medium font-medium">{rec.retailer_id}</td>
                      <td className="text-right font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{rec.spanner_amount.toLocaleString()}</td>
                      <td className="text-right font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{rec.gateway_amount.toLocaleString()}</td>
                      <td className="text-right font-mono font-medium" style={{ color: delta === 0 ? 'var(--border)' : isNegative ? 'var(--danger)' : 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>
                        {delta === 0 ? "—" : `${isNegative ? "" : "+"}${delta.toLocaleString()}`}
                      </td>
                      <td className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>{rec.gateway_provider}</td>
                      <td className="md-typescale-body-small whitespace-nowrap" style={{ color: 'var(--muted)' }}>
                        {rec.timestamp ? new Date(rec.timestamp).toLocaleString("en-US", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "—"}
                      </td>
                      <td className="text-right">{getStatusBadge(rec.status)}</td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
          {records.length > 0 && (
            <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--border)' }}>
              <div className="flex items-center gap-2">
                <label className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Rows</label>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value));
                    setPage(1);
                  }}
                  className="md-typescale-label-small px-2 py-1 rounded-md"
                  style={{ border: '1px solid var(--border)', background: 'var(--surface)', color: 'var(--foreground)' }}
                >
                  {[10, 25, 50, 100].map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
              <div className="flex items-center gap-3">
                <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                  Page {page}
                </span>
                <div className="flex gap-1">
                  <button className="md-btn md-btn-tonal" onClick={() => setPage(1)} disabled={!canPrev}>First</button>
                  <button className="md-btn md-btn-tonal" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={!canPrev}>Prev</button>
                  <button className="md-btn md-btn-tonal" onClick={() => setPage((p) => p + 1)} disabled={!canNext}>Next</button>
                </div>
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
