"use client";

import { useState, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import { useSyncHub } from '@/lib/useSyncHub';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';

interface CashHolding {
  order_id: string;
  invoice_id: string;
  driver_id: string;
  retailer_id: string;
  amount: number;
  custody_status: string;
  collected_at: string;
  geofence_dist_m: number;
}

interface CashHoldingsData {
  total_pending: number;
  total_collected: number;
  pending_count: number;
  collected_count: number;
  holdings: CashHolding[];
}

type FilterTab = 'ALL' | 'PENDING' | 'COLLECTED';

export default function CashHoldingsPage() {
  const [data, setData] = useState<CashHoldingsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [isLive, setIsLive] = useState(false);
  const [activeTab, setActiveTab] = useState<FilterTab>('ALL');
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null);

  const fetchHoldings = useCallback(async (signal?: AbortSignal) => {
    try {
      const res = await apiFetch('/v1/treasury/cash-holdings', { signal });
      if (!res.ok) throw new Error('Failed to fetch cash holdings');
      const json: CashHoldingsData = await res.json();
      setData(json);
      setIsLive(true);
      setLastRefreshed(new Date());
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      console.error('[CASH_HOLDINGS]', err);
      setIsLive(false);
    } finally {
      setLoading(false);
    }
  }, []);

  useSyncHub("POLL", "default", (signal) => fetchHoldings(signal), 30_000);

  const filteredHoldings = data?.holdings.filter(h => {
    if (activeTab === 'PENDING') return h.custody_status === 'PENDING';
    if (activeTab === 'COLLECTED') return h.custody_status === 'COLLECTED';
    return true;
  }) ?? [];

  const fmt = (n: number) => n.toLocaleString('en-US');

  return (
    <div className="min-h-full" style={{ background: 'var(--desk-bg)' }}>
      {/* Header */}
      <div className="px-6 pt-6 pb-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="md-typescale-headline-small" style={{ color: 'var(--desk-text-primary)' }}>
              Cash Holdings
            </h1>
            <p className="md-typescale-body-small mt-1" style={{ color: 'var(--desk-text-secondary)' }}>
              Cash custody pipeline — driver collections and pending handovers
            </p>
          </div>
          <div className="flex items-center gap-3">
            {/* Live indicator */}
            <span className="md-chip" style={{ cursor: 'default' }}>
              <span className={isLive ? 'desk-status-dot desk-status-dot--success' : 'desk-status-dot desk-status-dot--danger'} />
              {isLive ? 'Live' : 'Offline'}
            </span>
            <Button
              variant="secondary"
              isIconOnly
              onPress={() => fetchHoldings()}
            >
              <Icon name="refresh" size={16} />
            </Button>
          </div>
        </div>
      </div>

      {/* Summary cards */}
      {!loading && data && (
        <div className="px-6 pb-4 grid grid-cols-1 md:grid-cols-3 gap-4">
          <SummaryCard
            label="Pending Collection"
            value={`${fmt(data.total_pending)}`}
            count={data.pending_count}
            accent="var(--desk-warning)"
            accentBg="var(--desk-warning-soft)"
          />
          <SummaryCard
            label="Collected (In Custody)"
            value={`${fmt(data.total_collected)}`}
            count={data.collected_count}
            accent="var(--desk-success)"
            accentBg="var(--desk-success-soft)"
          />
          <SummaryCard
            label="Total Cash Volume"
            value={`${fmt(data.total_pending + data.total_collected)}`}
            count={data.pending_count + data.collected_count}
            accent="var(--desk-accent-strong)"
            accentBg="var(--desk-accent-soft)"
          />
        </div>
      )}

      {/* Tabs */}
      <div className="px-6 pb-2 flex gap-2">
        {(['ALL', 'PENDING', 'COLLECTED'] as FilterTab[]).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className="md-typescale-label-medium px-4 py-1.5 md-shape-full transition-colors"
            style={{
              background: activeTab === tab ? 'var(--desk-accent-soft)' : 'transparent',
              color: activeTab === tab ? 'var(--desk-accent-strong)' : 'var(--desk-text-secondary)',
              border: activeTab === tab ? 'none' : '1px solid var(--desk-border)',
            }}
          >
            {tab === 'ALL' ? 'All' : tab === 'PENDING' ? 'Pending' : 'Collected'}
          </button>
        ))}
      </div>

      {/* Divider */}
      <div className="mx-6 mb-4" style={{ height: 1, background: 'var(--desk-border)' }} />

      {/* Content */}
      <div className="px-6 pb-8">
        {loading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full md-shape-sm" />
            ))}
          </div>
        ) : filteredHoldings.length === 0 ? (
          <EmptyState
            icon="treasury"
            headline="No cash holdings"
            body={activeTab === 'ALL'
              ? 'Cash collection records will appear here as orders are fulfilled.'
              : `No ${activeTab.toLowerCase()} cash holdings found.`}
            action="Refresh"
            onAction={fetchHoldings}
          />
        ) : (
          <div
            className="desk-card overflow-hidden"
            style={{ background: 'var(--desk-surface)' }}
          >
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--desk-surface-alt)' }}>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Order</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Driver</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Retailer</th>
                  <th className="text-right px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Amount</th>
                  <th className="text-center px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Status</th>
                  <th className="text-right px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Distance</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--desk-text-secondary)' }}>Collected</th>
                </tr>
              </thead>
              <tbody>
                {filteredHoldings.map(h => (
                  <tr
                    key={h.invoice_id}
                    className="border-t transition-colors hover:opacity-90"
                    style={{ borderColor: 'var(--desk-border)' }}
                  >
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--desk-text-primary)', fontFamily: 'var(--font-mono, monospace)' }}>
                      {h.order_id.slice(-8)}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>
                      {h.driver_id ? h.driver_id.slice(-6) : '—'}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>
                      {h.retailer_id ? h.retailer_id.slice(-6) : '—'}
                    </td>
                    <td className="px-4 py-3 text-right md-typescale-body-small font-semibold" style={{ color: 'var(--desk-text-primary)', fontVariantNumeric: 'tabular-nums' }}>
                      {fmt(h.amount)}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <CustodyBadge status={h.custody_status} />
                    </td>
                    <td className="px-4 py-3 text-right md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>
                      {h.geofence_dist_m > 0 ? `${h.geofence_dist_m.toFixed(0)}m` : '—'}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--desk-text-secondary)' }}>
                      {h.collected_at ? new Date(h.collected_at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Last refreshed */}
        {lastRefreshed && (
          <p className="mt-3 md-typescale-label-small text-right" style={{ color: 'var(--desk-text-secondary)' }}>
            Last updated: {lastRefreshed.toLocaleTimeString()}
          </p>
        )}
      </div>
    </div>
  );
}

function SummaryCard({ label, value, count, accent, accentBg }: {
  label: string;
  value: string;
  count: number;
  accent: string;
  accentBg: string;
}) {
  return (
    <div
      className="desk-card p-4 relative overflow-hidden"
      style={{ background: 'var(--desk-surface)' }}
    >
      <div className="absolute top-0 left-0 w-1 h-full" style={{ background: accent }} />
      <p className="md-typescale-label-small mb-1" style={{ color: 'var(--desk-text-secondary)' }}>{label}</p>
      <p className="md-typescale-headline-small font-bold" style={{ color: 'var(--desk-text-primary)' }}>{value}</p>
      <span
        className="inline-flex items-center mt-2 px-2 py-0.5 md-shape-full md-typescale-label-small"
        style={{ background: accentBg, color: accent }}
      >
        {count} {count === 1 ? 'order' : 'orders'}
      </span>
    </div>
  );
}

function CustodyBadge({ status }: { status: string }) {
  const isPending = status === 'PENDING';
  const background = isPending ? 'var(--desk-warning-soft)' : 'var(--desk-success-soft)';
  const color = isPending ? 'var(--desk-warning)' : 'var(--desk-success)';
  return (
    <span
      className="inline-flex items-center gap-1 px-2 py-0.5 md-shape-full md-typescale-label-small"
      style={{
        background,
        color,
      }}
    >
      {isPending && <span className="desk-status-dot desk-status-dot--warning" />}
      {isPending ? 'Pending' : 'Collected'}
    </span>
  );
}
