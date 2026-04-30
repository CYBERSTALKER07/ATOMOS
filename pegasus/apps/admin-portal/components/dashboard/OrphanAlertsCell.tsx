'use client';

import { useEffect, useState, useCallback } from 'react';
import { getAdminToken } from '@/lib/auth';
import { AlertTriangle, CheckCircle2, Radio } from 'lucide-react';

// ── Types ───────────────────────────────────────────────────────────────────

type AuditEntry = {
  audit_id: string;
  retailer_id: string;
  retailer_cell: string;
  audit_type: 'ORPHAN_DETECTED' | 'COVERAGE_RESTORED' | 'COVERAGE_GAP';
  warehouse_id?: string;
  distance_km?: number;
  resolved_at?: string;
  created_at: string;
};

// ── Orphan Alerts Cell — The List (1×2) ─────────────────────────────────────
// Shows critical dispatch alerts from the DispatchAudit table: retailers whose
// H3 cell has no warehouse coverage (orphans). Tall cell for scrollable list.

export default function OrphanAlertsCell() {
  const [alerts, setAlerts] = useState<AuditEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAlerts = useCallback(async (signal?: AbortSignal) => {
    try {
      const token = await getAdminToken();
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/dispatch-audits?type=ORPHAN_DETECTED&unresolved=true&limit=50`,
        { headers: { Authorization: `Bearer ${token}` }, signal },
      );
      if (res.ok) {
        const data = await res.json();
        setAlerts(Array.isArray(data) ? data : data?.audits ?? []);
        setError(null);
      } else if (res.status === 404) {
        // Endpoint not wired yet — show empty
        setAlerts([]);
        setError(null);
      } else {
        setError('Failed to load');
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        setError('Offline');
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    fetchAlerts(controller.signal);
    const id = setInterval(() => fetchAlerts(), 30_000);
    return () => {
      controller.abort();
      clearInterval(id);
    };
  }, [fetchAlerts]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="bento-card-header">
        <span className="bento-card-title">Coverage Alerts</span>
        <div className="flex items-center gap-1.5">
          {alerts.length > 0 ? (
            <span
              className="md-typescale-label-small tabular-nums font-bold"
              style={{ color: 'var(--danger)' }}
            >
              {alerts.length}
            </span>
          ) : (
            <CheckCircle2 size={14} style={{ color: 'var(--success)' }} />
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto min-h-0 -mx-1 px-1">
        {isLoading ? (
          <div className="flex flex-col gap-3">
            {[...Array(4)].map((_, i) => (
              <div
                key={i}
                className="skeleton h-14 rounded"
                style={{ animationDelay: `${i * 100}ms` }}
              />
            ))}
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full gap-2">
            <AlertTriangle size={20} style={{ color: 'var(--danger)' }} />
            <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
              {error}
            </span>
          </div>
        ) : alerts.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-2">
            <Radio size={20} style={{ color: 'var(--success)' }} />
            <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
              All retailers covered
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-1">
            {alerts.map((alert) => (
              <div
                key={alert.audit_id}
                className="flex items-start gap-3 px-3 py-2.5 rounded transition-colors"
                style={{
                  borderBottom: '1px solid var(--border)',
                }}
              >
                <AlertTriangle
                  size={14}
                  className="mt-0.5 shrink-0"
                  style={{ color: 'var(--danger)' }}
                />
                <div className="flex-1 min-w-0">
                  <p
                    className="md-typescale-label-medium truncate"
                    style={{ color: 'var(--foreground)' }}
                  >
                    {alert.retailer_id}
                  </p>
                  <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                    Cell {alert.retailer_cell}
                    {alert.distance_km != null && (
                      <> · {alert.distance_km.toFixed(1)} km to nearest</>
                    )}
                  </p>
                </div>
                <span
                  className="md-typescale-label-small tabular-nums shrink-0"
                  style={{ color: 'var(--muted)' }}
                >
                  {formatRelativeTime(alert.created_at)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      {alerts.length > 0 && (
        <div
          className="pt-2 mt-auto"
          style={{ borderTop: '1px solid var(--border)' }}
        >
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
            {alerts.length} orphaned retailer{alerts.length !== 1 ? 's' : ''} — no warehouse H3 coverage
          </p>
        </div>
      )}
    </div>
  );
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function formatRelativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return 'now';
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}
