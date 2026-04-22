"use client";

import React, { useState, useCallback } from 'react';
import { Button } from '@heroui/react';
import { getAdminToken } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
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
      const token = await getAdminToken();
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/treasury/cash-holdings`, {
        headers: { Authorization: `Bearer ${token}` }, signal,
      });
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

  usePolling((signal) => fetchHoldings(signal), 30_000);

  const filteredHoldings = data?.holdings.filter(h => {
    if (activeTab === 'PENDING') return h.custody_status === 'PENDING';
    if (activeTab === 'COLLECTED') return h.custody_status === 'COLLECTED';
    return true;
  }) ?? [];

  const fmt = (n: number) => n.toLocaleString('en-US');

  return (
    <div className="min-h-screen" style={{ background: 'var(--background)' }}>
      {/* Header */}
      <div className="px-6 pt-6 pb-4">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>
              Cash Holdings
            </h1>
            <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
              Cash custody pipeline — driver collections and pending handovers
            </p>
          </div>
          <div className="flex items-center gap-3">
            {/* Live indicator */}
            <span
              className="inline-flex items-center gap-1.5 px-3 py-1 md-shape-full md-typescale-label-small"
              style={{
                background: isLive ? 'var(--success)' : 'var(--danger)',
                color: isLive ? 'var(--success-foreground)' : 'var(--danger-foreground)',
              }}
            >
              <span className={`w-2 h-2 rounded-full ${isLive ? 'animate-pulse' : ''}`}
                    style={{ background: isLive ? 'var(--success)' : 'var(--danger)' }} />
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
            accent="var(--warning)"
            accentBg="var(--warning)"
          />
          <SummaryCard
            label="Collected (In Custody)"
            value={`${fmt(data.total_collected)}`}
            count={data.collected_count}
            accent="var(--success)"
            accentBg="var(--success)"
          />
          <SummaryCard
            label="Total Cash Volume"
            value={`${fmt(data.total_pending + data.total_collected)}`}
            count={data.pending_count + data.collected_count}
            accent="var(--accent)"
            accentBg="var(--accent-soft)"
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
              background: activeTab === tab ? 'var(--accent-soft)' : 'transparent',
              color: activeTab === tab ? 'var(--accent-soft-foreground)' : 'var(--muted)',
              border: activeTab === tab ? 'none' : '1px solid var(--border)',
            }}
          >
            {tab === 'ALL' ? 'All' : tab === 'PENDING' ? 'Pending' : 'Collected'}
          </button>
        ))}
      </div>

      {/* Divider */}
      <div className="mx-6 mb-4" style={{ height: 1, background: 'var(--border)' }} />

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
            className="md-card md-elevation-0 md-shape-md overflow-hidden"
            style={{ background: 'var(--background)' }}
          >
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--surface)' }}>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Order</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Driver</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Retailer</th>
                  <th className="text-right px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Amount</th>
                  <th className="text-center px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Status</th>
                  <th className="text-right px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Distance</th>
                  <th className="text-left px-4 py-3 md-typescale-label-small" style={{ color: 'var(--muted)' }}>Collected</th>
                </tr>
              </thead>
              <tbody>
                {filteredHoldings.map(h => (
                  <tr
                    key={h.invoice_id}
                    className="border-t transition-colors hover:opacity-90"
                    style={{ borderColor: 'var(--border)' }}
                  >
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--foreground)', fontFamily: 'var(--font-mono, monospace)' }}>
                      {h.order_id.slice(-8)}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                      {h.driver_id ? h.driver_id.slice(-6) : '—'}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                      {h.retailer_id ? h.retailer_id.slice(-6) : '—'}
                    </td>
                    <td className="px-4 py-3 text-right md-typescale-body-small font-semibold" style={{ color: 'var(--foreground)', fontVariantNumeric: 'tabular-nums' }}>
                      {fmt(h.amount)} <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}></span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <CustodyBadge status={h.custody_status} />
                    </td>
                    <td className="px-4 py-3 text-right md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                      {h.geofence_dist_m > 0 ? `${h.geofence_dist_m.toFixed(0)}m` : '—'}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
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
          <p className="mt-3 md-typescale-label-small text-right" style={{ color: 'var(--muted)' }}>
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
      className="md-card md-elevation-1 md-shape-md p-4 relative overflow-hidden"
      style={{ background: 'var(--surface)' }}
    >
      <div className="absolute top-0 left-0 w-1 h-full" style={{ background: accent }} />
      <p className="md-typescale-label-small mb-1" style={{ color: 'var(--muted)' }}>{label}</p>
      <p className="md-typescale-headline-small font-bold" style={{ color: 'var(--foreground)' }}>{value}</p>
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
  return (
    <span
      className="inline-flex items-center gap-1 px-2 py-0.5 md-shape-full md-typescale-label-small"
      style={{
        background: isPending ? 'var(--warning)' : 'var(--success)',
        color: isPending ? 'var(--warning-foreground)' : 'var(--success-foreground)',
      }}
    >
      {isPending && <span className="w-1.5 h-1.5 rounded-full animate-pulse" style={{ background: 'var(--warning)' }} />}
      {isPending ? 'Pending' : 'Collected'}
    </span>
  );
}
