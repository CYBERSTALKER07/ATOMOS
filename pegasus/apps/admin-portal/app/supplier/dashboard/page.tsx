'use client';

import VelocityChart from '@/components/VelocityChart';
import Link from 'next/link';
import { useState, useEffect } from 'react';
import { apiFetch } from '@/lib/auth';

interface SkuVelocity {
  sku_id: string;
  total_pallets: number;
  gross_volume: number;
}

interface DemandSummary {
  total_retailers: number;
  total_pallets: number;
  total_value: number;
  prediction_count: number;
  items: { sku_id: string; product_name: string; total_qty: number; retailer_count: number }[];
  generated_at: string;
}

export default function SupplierDashboard() {
  const [velocityData, setVelocityData] = useState<SkuVelocity[]>([]);
  const [demand, setDemand] = useState<DemandSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      apiFetch('/v1/supplier/analytics/velocity')
        .then(res => { if (!res.ok) throw new Error(`Analytics fetch failed: ${res.status}`); return res.json(); })
        .then(json => setVelocityData(json.data || [])),
      apiFetch('/v1/supplier/analytics/demand/today')
        .then(res => res.ok ? res.json() : null)
        .then(json => { if (json) setDemand(json); }),
    ])
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const totalVolume = velocityData.reduce((sum, item) => sum + item.gross_volume, 0);
  const totalPallets = velocityData.reduce((sum, item) => sum + item.total_pallets, 0);
  const topSku = velocityData.length > 0
    ? velocityData.reduce((top, cur) => cur.gross_volume > top.gross_volume ? cur : top)
    : null;
  const avgVelocity = velocityData.length > 0 ? Math.round(totalPallets / velocityData.length) : 0;

  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)' }}>
        <div className="mb-10 flex items-center justify-between">
          <div>
            <div className="w-48 h-7 rounded animate-pulse mb-2" style={{ background: 'var(--surface)' }} />
            <div className="w-72 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
          </div>
          <div className="w-44 h-10 rounded-full animate-pulse" style={{ background: 'var(--surface)' }} />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="md-card md-card-elevated p-6 h-32 flex flex-col justify-between">
              <div className="w-1/2 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
              <div className="w-2/3 h-8 rounded mt-4 animate-pulse" style={{ background: 'var(--surface)' }} />
            </div>
          ))}
        </div>
        <div className="md-card md-card-elevated p-6 h-64 animate-pulse" style={{ background: 'var(--surface)' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>{error}</div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="flex items-center justify-between mb-10">
        <div>
          <h1 className="md-typescale-headline-large tracking-tight">Analytics</h1>
          <p className="md-typescale-body-large mt-1" style={{ color: 'var(--muted)' }}>Financial overview and operational intelligence</p>
        </div>
        <Link
          href="/supplier/manifests"
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl bg-accent text-accent-foreground md-typescale-label-large transition-opacity hover:opacity-90"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M19 3h-4.18C14.4 1.84 13.3 1 12 1c-1.3 0-2.4.84-2.82 2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-7 0c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1 .45-1 1-1zm2 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"/></svg>
          Dispatch Control Room
        </Link>
      </header>

      {/* AI Future Demand Card — High Priority */}
      {demand && demand.prediction_count > 0 && (
        <div className="mb-8 md-animate-in">
          <div className="md-card md-card-filled p-6" style={{ background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)' }}>
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full flex items-center justify-center" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
                </div>
                <div>
                  <h2 className="md-typescale-title-medium font-semibold">AI Future Demand (Next 24H)</h2>
                  <p className="md-typescale-body-small" style={{ opacity: 0.8 }}>Empathy Engine predictions for your catalog</p>
                </div>
              </div>
              <span className="md-typescale-label-small px-2 py-1 rounded-full" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                {demand.prediction_count} predictions
              </span>
            </div>

            <div className="grid grid-cols-3 gap-4 mb-4">
              <div>
                <p className="md-typescale-label-small" style={{ opacity: 0.7 }}>Retailers</p>
                <p className="md-typescale-headline-small">{demand.total_retailers}</p>
              </div>
              <div>
                <p className="md-typescale-label-small" style={{ opacity: 0.7 }}>Total Pallets</p>
                <p className="md-typescale-headline-small">{new Intl.NumberFormat('en-US').format(demand.total_pallets)}</p>
              </div>
              <div>
                <p className="md-typescale-label-small" style={{ opacity: 0.7 }}>Forecast Value</p>
                <p className="md-typescale-headline-small">{new Intl.NumberFormat('uz-UZ').format(demand.total_value)} <span className="md-typescale-label-small"></span></p>
              </div>
            </div>

            {demand.items.length > 0 && (
              <div className="flex flex-wrap gap-2 mb-4">
                {demand.items.slice(0, 5).map((item) => (
                  <span key={item.sku_id} className="md-typescale-label-small px-3 py-1 rounded-full" style={{ background: 'var(--accent-soft-foreground)', color: 'var(--accent-soft)', opacity: 0.9 }}>
                    {item.total_qty}× {item.product_name || item.sku_id}
                  </span>
                ))}
                {demand.items.length > 5 && (
                  <span className="md-typescale-label-small px-3 py-1 rounded-full" style={{ background: 'var(--accent-soft-foreground)', color: 'var(--accent-soft)', opacity: 0.6 }}>
                    +{demand.items.length - 5} more
                  </span>
                )}
              </div>
            )}

            <Link
              href="/supplier/analytics/demand"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-full md-typescale-label-large transition-colors"
              style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
            >
              View Advanced Analytics
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14m-7-7 7 7-7 7"/></svg>
            </Link>
          </div>
        </div>
      )}

      {/* KPI Grid — M3 Filled Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
        {[
          { label: "Gross Volume (Amount)", value: new Intl.NumberFormat('uz-UZ').format(totalVolume) },
          { label: "Total Pallets Moved", value: new Intl.NumberFormat('en-US').format(totalPallets) },
          { label: "Avg Velocity / SKU", value: `${avgVelocity}`, sub: "pallets" },
          { label: "Top Performing SKU", value: topSku?.sku_id || '—', sub: topSku ? `${new Intl.NumberFormat('uz-UZ').format(topSku.gross_volume)}` : undefined },
        ].map(({ label, value, sub }, i) => (
          <div key={i} className="md-card md-card-elevated p-6 flex flex-col justify-between cursor-default md-animate-in" style={{ animationDelay: `${i * 50}ms` }}>
            <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
            <div>
              <p className="md-typescale-headline-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{value}</p>
              {sub && <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>{sub}</p>}
            </div>
          </div>
        ))}
      </div>

      {/* Velocity Chart */}
      <VelocityChart data={velocityData} />

      {/* SKU Breakdown Table — M3 Data Table */}
      {velocityData && velocityData.length > 0 && (
        <div className="mt-8 md-animate-in" style={{ animationDelay: '200ms' }}>
          <div className="md-card md-card-outlined p-0 w-full overflow-hidden">
            <table className="md-table">
              <thead>
                <tr>
                  <th>SKU ID</th>
                  <th className="text-right">Pallets</th>
                  <th className="text-right">Volume (Amount)</th>
                  <th className="text-right">Share</th>
                </tr>
              </thead>
              <tbody>
                {velocityData.map((item) => (
                  <tr key={item.sku_id} className="transition-colors">
                    <td className="font-mono md-typescale-body-small">{item.sku_id}</td>
                    <td className="text-right" style={{ color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>{item.total_pallets}</td>
                    <td className="text-right font-mono md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{new Intl.NumberFormat('uz-UZ').format(item.gross_volume)}</td>
                    <td className="text-right">
                      <div className="flex items-center justify-end gap-3">
                        <div className="w-20 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--surface)' }}>
                          <div
                            className="h-full rounded-full"
                            style={{ width: `${totalVolume > 0 ? (item.gross_volume / totalVolume * 100) : 0}%`, background: 'var(--accent)' }}
                          />
                        </div>
                        <span className="md-typescale-label-small w-10 text-right" style={{ color: 'var(--muted)' }}>
                          {totalVolume > 0 ? (item.gross_volume / totalVolume * 100).toFixed(1) : '0'}%
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
