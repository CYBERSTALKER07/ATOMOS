"use client";

import { useState } from "react";
import { apiFetch } from '@/lib/auth';
import { useSyncHub } from '@/lib/useSyncHub';
import EmptyState from '@/components/EmptyState';
import StatusChip from '@/components/StatusChip';
import { Skeleton } from '@/components/Skeleton';

type ReconciliationRecord = {
  order_id: string;
  retailer_id: string;
  spanner_amount: number;
  gateway_amount: number;
  currency?: string;
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

  useSyncHub("POLL", "default", async (signal) => {
    try {
      const res = await apiFetch(`/v1/admin/reconciliation?limit=${pageSize + 1}&offset=${offset}`, { signal });
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
  const exposureByCurrency = records.reduce((acc, r) => {
    const code = (r.currency || 'UZS').toUpperCase();
    const exposure = Math.abs(r.spanner_amount - r.gateway_amount);
    acc[code] = (acc[code] || 0) + exposure;
    return acc;
  }, {} as Record<string, number>);
  const totalExposure = Object.entries(exposureByCurrency)
    .map(([code, amount]) => `${amount.toLocaleString()} ${code}`)
    .join(' · ') || '0 UZS';

  const getStatusBadge = (status: string) => {
    const normalized = status.toUpperCase();
    if (normalized === 'DELTA') {
      return <StatusChip status="PENDING_REVIEW" label="Delta" size="sm" />;
    }
    if (normalized === 'ORPHANED') {
      return <StatusChip status="FAILED" label="Orphaned" size="sm" />;
    }
    if (normalized === 'MATCH') {
      return <StatusChip status="MATCHED" label="Match" size="sm" />;
    }
    return <StatusChip status={normalized} size="sm" />;
  };

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--desk-canvas)', color: 'var(--desk-text-primary)' }}>
      {/* Header */}
      <header className="mb-10">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="md-typescale-headline-medium" style={{ color: 'var(--desk-text-primary)' }}>Reconciliation</h1>
            <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--desk-text-secondary)' }}>Spanner ↔ Gateway Settlement Anomaly Scanner</p>
          </div>
          {lastRefreshed && (
            <span className="md-typescale-label-small mt-1" style={{ color: 'var(--desk-text-tertiary)' }}>
              Last refreshed {lastRefreshed.toLocaleTimeString()}
            </span>
          )}
        </div>
      </header>

      {/* KPI Cards — M3 Filled Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="desk-card p-6 h-32 flex flex-col justify-between">
              <Skeleton className="w-1/2 h-3" style={{ background: 'var(--desk-surface-subtle)' }} />
              <Skeleton className="w-2/3 h-8 mt-4" style={{ background: 'var(--desk-surface-subtle)' }} />
            </div>
          ))
        ) : (
          <>
            {[
              { label: "Total Anomalies", value: records.length, color: 'var(--desk-text-primary)' },
              { label: "Delta Mismatches", value: deltaCount, color: 'var(--desk-text-secondary)' },
              { label: "Orphaned Records", value: orphanedCount, color: 'var(--desk-danger)' },
              { label: "Total Exposure (Amount)", value: totalExposure.toLocaleString(), color: 'var(--desk-text-primary)' },
            ].map(({ label, value, color }, i) => (
              <div key={i} className="desk-card p-6 flex flex-col justify-between cursor-default">
                <p className="md-typescale-label-small mb-4" style={{ color: 'var(--desk-text-secondary)' }}>{label}</p>
                <p className="md-typescale-headline-small tracking-tight" style={{ color, fontVariantNumeric: 'tabular-nums' }}>{value}</p>
              </div>
            ))}
          </>
        )}
      </div>

      {/* Anomaly Table — M3 Data Table */}
      <main>
        <div className="w-full overflow-hidden desk-card p-0">
          <table className="md-table">
            <thead>
              <tr>
                <th>Order ID</th>
                <th>Retailer</th>
                <th className="text-right">Spanner (Amount)</th>
                <th className="text-right">Gateway (Amount)</th>
                <th>Currency</th>
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
                    <td><Skeleton className="w-24 h-4" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-20 h-4" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-20 h-4 ml-auto" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-20 h-4 ml-auto" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-12 h-4" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-16 h-4 ml-auto" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-16 h-4" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-24 h-4" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                    <td><Skeleton className="w-20 h-4 ml-auto" style={{ background: 'var(--desk-surface-subtle)' }} /></td>
                  </tr>
                ))
              ) : records.length === 0 ? (
                <tr>
                  <td colSpan={9} className="p-16 text-center">
                    <EmptyState
                      icon="reconcile"
                      headline="No anomalies detected"
                      body="Spanner and gateway ledgers are in sync."
                    />
                  </td>
                </tr>
              ) : (
                records.map((rec, i) => {
                  const delta = rec.spanner_amount - rec.gateway_amount;
                  const isNegative = delta < 0;
                  return (
                    <tr key={rec.order_id || `idx-${i}`} className="transition-colors">
                      <td className="font-mono md-typescale-body-small font-medium">{rec.order_id}</td>
                      <td className="md-typescale-body-medium font-medium">{rec.retailer_id}</td>
                      <td className="text-right font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{rec.spanner_amount.toLocaleString()}</td>
                      <td className="text-right font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{rec.gateway_amount.toLocaleString()}</td>
                      <td className="md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>{(rec.currency || 'UZS').toUpperCase()}</td>
                      <td className="text-right font-mono font-medium" style={{ color: delta === 0 ? 'var(--desk-text-tertiary)' : isNegative ? 'var(--desk-danger)' : 'var(--desk-text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                        {delta === 0 ? "—" : `${isNegative ? "" : "+"}${delta.toLocaleString()}`}
                      </td>
                      <td className="md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>{rec.gateway_provider}</td>
                      <td className="md-typescale-body-small whitespace-nowrap" style={{ color: 'var(--desk-text-secondary)' }}>
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
            <div className="flex items-center justify-between px-4 py-3" style={{ borderTop: '1px solid var(--desk-border)' }}>
              <div className="flex items-center gap-2">
                <label className="md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Rows</label>
                <select
                  value={pageSize}
                  onChange={(e) => {
                    setPageSize(Number(e.target.value));
                    setPage(1);
                  }}
                  className="md-typescale-label-small px-2 py-1 rounded-md"
                  style={{ border: '1px solid var(--desk-border)', background: 'var(--desk-surface)', color: 'var(--desk-text-primary)' }}
                >
                  {[10, 25, 50, 100].map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
              <div className="flex items-center gap-3">
                <span className="md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>
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
