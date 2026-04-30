'use client';

import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';

interface ForecastProduct {
  product_id: string;
  product_name: string;
  current_stock: number;
  recommended_qty: number;
  days_until_stockout: number;
  priority: string;
  unit: string;
  sources: {
    incoming_orders: number;
    ai_prediction: number;
    pre_orders: number;
    burn_rate: number;
  };
}

interface Forecast {
  warehouse_id: string;
  forecast_days: number;
  generated_at: string;
  products: ForecastProduct[];
}

export default function DemandForecastPage() {
  const { toast } = useToast();
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [loading, setLoading] = useState(true);
  const [days, setDays] = useState(7);

  async function loadForecast() {
    setLoading(true);
    try {
      const res = await apiFetch(`/v1/warehouse/demand/forecast?days=${days}`);
      if (res.ok) {
        setForecast(await res.json());
      } else {
        toast('Failed to load forecast', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadForecast(); }, [days]); // eslint-disable-line react-hooks/exhaustive-deps

  const products = forecast?.products || [];
  const critical = products.filter(p => p.priority === 'CRITICAL');
  const urgent = products.filter(p => p.priority === 'URGENT');
  const normal = products.filter(p => p.priority === 'NORMAL');

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Demand Forecast</h1>
          <p className="text-xs text-[var(--muted)] mt-0.5">
            AI-powered stock recommendations from 4 data sources
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={days}
            onChange={e => setDays(Number(e.target.value))}
            className="px-3 py-2 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--field-background)',
              color: 'var(--field-foreground)',
              borderColor: 'var(--field-border)',
            }}
          >
            <option value={7}>7 days</option>
            <option value={14}>14 days</option>
            <option value={30}>30 days</option>
          </select>
          <button
            onClick={loadForecast}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
          >
            <Icon name="refresh" size={16} />
            Refresh
          </button>
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Critical Items</div>
          <div className="text-2xl font-bold text-[var(--danger)]">{critical.length}</div>
          <div className="text-xs text-[var(--muted)]">&lt; 2 days to stockout</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Urgent Items</div>
          <div className="text-2xl font-bold text-[var(--warning)]">{urgent.length}</div>
          <div className="text-xs text-[var(--muted)]">&lt; 5 days to stockout</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Healthy Items</div>
          <div className="text-2xl font-bold text-[var(--success)]">{normal.length}</div>
          <div className="text-xs text-[var(--muted)]">5+ days of stock</div>
        </div>
      </div>

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="md-skeleton md-skeleton-row" />
          ))}
        </div>
      ) : products.length === 0 ? (
        <div className="text-center py-20 text-[var(--muted)]">
          <Icon name="forecast" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No products tracked yet</p>
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Product</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Stock</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Recommended</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Stockout</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Priority</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Incoming</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">AI Pred</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Pre-Orders</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Burn/Day</th>
              </tr>
            </thead>
            <tbody>
              {products.map(p => (
                <tr key={p.product_id} className="border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--surface)] transition-colors">
                  <td className="px-4 py-3">{p.product_name || p.product_id.slice(0, 8)}</td>
                  <td className="px-4 py-3 text-right font-mono">{p.current_stock}</td>
                  <td className="px-4 py-3 text-right font-mono font-semibold">{p.recommended_qty}</td>
                  <td className="px-4 py-3 text-right">
                    <span className={
                      p.days_until_stockout < 2 ? 'text-[var(--danger)] font-semibold' :
                      p.days_until_stockout < 5 ? 'text-[var(--warning)]' : ''
                    }>
                      {p.days_until_stockout.toFixed(1)}d
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-semibold ${
                      p.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
                      p.priority === 'URGENT' ? 'text-[var(--warning)]' : 'text-[var(--muted)]'
                    }`}>{p.priority}</span>
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.incoming_orders || 0}</td>
                  <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.ai_prediction || 0}</td>
                  <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.pre_orders || 0}</td>
                  <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.burn_rate?.toFixed(1) || '0.0'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {forecast && (
        <div className="text-xs text-[var(--muted)]">
          Generated at {new Date(forecast.generated_at).toLocaleString()} for {forecast.forecast_days}-day window
        </div>
      )}
    </div>
  );
}
