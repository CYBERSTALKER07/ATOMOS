'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { apiFetch } from '@/lib/auth';
import VelocityChart from '@/components/VelocityChart';
import Icon from '@/components/Icon';

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

export default function AnalyticsHub() {
  const [velocity, setVelocity] = useState<SkuVelocity[]>([]);
  const [demand, setDemand] = useState<DemandSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      apiFetch('/v1/supplier/analytics/velocity')
        .then(r => r.ok ? r.json() : Promise.reject(new Error(`${r.status}`)))
        .then(j => setVelocity(j.data || [])),
      apiFetch('/v1/supplier/analytics/demand/today')
        .then(r => r.ok ? r.json() : null)
        .then(j => { if (j) setDemand(j); }),
    ])
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const totalVolume = velocity.reduce((s, i) => s + i.gross_volume, 0);
  const totalPallets = velocity.reduce((s, i) => s + i.total_pallets, 0);
  const topSku = velocity.length > 0
    ? velocity.reduce((top, c) => c.gross_volume > top.gross_volume ? c : top)
    : null;
  const avgVelocity = velocity.length > 0 ? Math.round(totalPallets / velocity.length) : 0;

  if (loading) {
    return (
      <div className="p-6 md:p-10 space-y-8" style={{ background: 'var(--background)' }}>
        <div className="w-48 h-7 rounded skeleton mb-2" />
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="md-card md-card-elevated p-6 h-32 skeleton" />
          ))}
        </div>
        <div className="md-card md-card-elevated h-64 skeleton" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6 text-center space-y-3">
          <Icon name="error" className="w-10 h-10 mx-auto text-[var(--danger)]" />
          <p className="md-typescale-body-large" style={{ color: 'var(--danger)' }}>{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="flex items-center justify-between mb-10">
        <div>
          <h1 className="md-typescale-headline-large tracking-tight">Analytics</h1>
          <p className="md-typescale-body-large mt-1" style={{ color: 'var(--muted)' }}>
            Financial overview and operational intelligence
          </p>
        </div>
        <div className="flex gap-3">
          <Link href="/supplier/analytics/demand" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent-soft text-accent-soft-foreground hover:opacity-80 transition-opacity">
            <Icon name="analytics" className="w-5 h-5" /> Demand Forecast
          </Link>
          <Link href="/" className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-accent text-accent-foreground hover:opacity-90 transition-opacity">
            <Icon name="orders" className="w-5 h-5" /> Dispatch Room
          </Link>
        </div>
      </header>

      {/* AI Future Demand Card */}
      {demand && demand.prediction_count > 0 && (
        <div className="mb-8">
          <div className="md-card p-6" style={{ background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)' }}>
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full flex items-center justify-center" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                  <Icon name="analytics" className="w-5 h-5" />
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
                {demand.items.slice(0, 5).map(item => (
                  <span key={item.sku_id} className="md-typescale-label-small px-3 py-1 rounded-full" style={{ background: 'var(--accent-soft-foreground)', color: 'var(--accent-soft)', opacity: 0.9 }}>
                    {item.total_qty}&times; {item.product_name || item.sku_id}
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
              className="inline-flex items-center gap-2 px-4 py-2 rounded-full md-typescale-label-large"
              style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
            >
              View Advanced Analytics
              <Icon name="arrow_forward" className="w-4 h-4" />
            </Link>
          </div>
        </div>
      )}

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
        {[
          { label: 'Gross Volume (Amount)', value: new Intl.NumberFormat('uz-UZ').format(totalVolume) },
          { label: 'Total Pallets Moved', value: new Intl.NumberFormat('en-US').format(totalPallets) },
          { label: 'Avg Velocity / SKU', value: `${avgVelocity}`, sub: 'pallets' },
          { label: 'Top SKU', value: topSku?.sku_id || '\u2014', sub: topSku ? `${new Intl.NumberFormat('uz-UZ').format(topSku.gross_volume)}` : undefined },
        ].map(({ label, value, sub }, i) => (
          <div key={i} className="md-card md-card-elevated p-6 flex flex-col justify-between">
            <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
            <div>
              <p className="md-typescale-headline-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{value}</p>
              {sub && <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>{sub}</p>}
            </div>
          </div>
        ))}
      </div>

      {/* Velocity Chart */}
      <VelocityChart data={velocity} />

      {/* SKU Breakdown Table */}
      {velocity.length > 0 && (
        <div className="mt-8">
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
                {velocity.map(item => (
                  <tr key={item.sku_id}>
                    <td className="font-mono md-typescale-body-small">{item.sku_id}</td>
                    <td className="text-right" style={{ fontVariantNumeric: 'tabular-nums', color: 'var(--muted)' }}>{item.total_pallets}</td>
                    <td className="text-right font-mono md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{new Intl.NumberFormat('uz-UZ').format(item.gross_volume)}</td>
                    <td className="text-right">
                      <div className="flex items-center justify-end gap-3">
                        <div className="w-20 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--surface)' }}>
                          <div className="h-full rounded-full" style={{ width: `${totalVolume > 0 ? (item.gross_volume / totalVolume * 100) : 0}%`, background: 'var(--accent)' }} />
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
