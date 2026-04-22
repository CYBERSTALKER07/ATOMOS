'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';

interface ForecastItem {
  product_id: string;
  product_name: string;
  current_stock: number;
  recommended_qty: number;
  days_until_stockout: number;
  priority: string;
  unit: string;
}

export default function NewSupplyRequestPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [factoryId, setFactoryId] = useState('');
  const [deliveryDate, setDeliveryDate] = useState('');
  const [notes, setNotes] = useState('');
  const [useForecast, setUseForecast] = useState(true);
  const [forecast, setForecast] = useState<ForecastItem[]>([]);
  const [forecastLoading, setForecastLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // Manual items (when not using forecast)
  const [manualItems, setManualItems] = useState<{ product_id: string; quantity: number; unit: string }[]>([]);

  useEffect(() => {
    if (useForecast) {
      loadForecast();
    }
  }, [useForecast]); // eslint-disable-line react-hooks/exhaustive-deps

  async function loadForecast() {
    setForecastLoading(true);
    try {
      const res = await apiFetch('/v1/warehouse/demand/forecast?days=7');
      if (res.ok) {
        const data = await res.json();
        setForecast(data.products || []);
      }
    } catch {
      toast('Failed to load forecast', 'error');
    } finally {
      setForecastLoading(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!factoryId.trim()) {
      toast('Select a factory', 'warning');
      return;
    }

    setSubmitting(true);
    try {
      const body: Record<string, unknown> = {
        factory_id: factoryId,
        notes,
        use_demand_forecast: useForecast,
      };
      if (deliveryDate) {
        body.requested_delivery_date = new Date(deliveryDate).toISOString();
      }
      if (!useForecast && manualItems.length > 0) {
        body.items = manualItems;
      }

      const res = await apiFetch('/v1/warehouse/supply-requests', {
        method: 'POST',
        body: JSON.stringify(body),
      });

      if (res.ok) {
        const data = await res.json();
        toast('Supply request created', 'success');
        router.push(`/supply-requests/${data.request_id}`);
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to create request', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="p-6 space-y-6 md-animate-in max-w-3xl">
      <div className="flex items-center gap-3">
        <button onClick={() => router.back()} className="p-1 rounded-lg hover:bg-[var(--surface)]">
          <Icon name="left" size={20} />
        </button>
        <h1 className="text-xl font-bold tracking-tight">New Supply Request</h1>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Factory selector */}
        <div>
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Factory ID</label>
          <input
            type="text"
            value={factoryId}
            onChange={e => setFactoryId(e.target.value)}
            placeholder="Enter factory UUID"
            required
            className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--field-background)',
              color: 'var(--field-foreground)',
              borderColor: 'var(--field-border)',
            }}
          />
        </div>

        {/* Delivery date */}
        <div>
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Requested Delivery Date</label>
          <input
            type="date"
            value={deliveryDate}
            onChange={e => setDeliveryDate(e.target.value)}
            className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--field-background)',
              color: 'var(--field-foreground)',
              borderColor: 'var(--field-border)',
            }}
          />
        </div>

        {/* Use AI forecast toggle */}
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={useForecast}
              onChange={e => setUseForecast(e.target.checked)}
              className="w-4 h-4 rounded"
            />
            <span className="text-sm font-medium">Use AI demand forecast</span>
          </label>
          <span className="text-xs text-[var(--muted)]">Auto-fill items from demand engine</span>
        </div>

        {/* Forecast preview */}
        {useForecast && (
          <div className="border border-[var(--border)] rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between" style={{ background: 'var(--surface)' }}>
              <span className="text-sm font-semibold">Demand Forecast (7-day)</span>
              <button type="button" onClick={loadForecast} className="text-xs text-[var(--muted)] hover:text-[var(--foreground)]">
                <Icon name="refresh" size={14} />
              </button>
            </div>
            {forecastLoading ? (
              <div className="p-4 space-y-2">
                {Array.from({ length: 3 }).map((_, i) => (
                  <div key={i} className="md-skeleton md-skeleton-row" />
                ))}
              </div>
            ) : forecast.length === 0 ? (
              <div className="p-8 text-center text-[var(--muted)] text-sm">
                No forecast data available
              </div>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">Product</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">Stock</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">Recommended</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">Stockout In</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">Priority</th>
                  </tr>
                </thead>
                <tbody>
                  {forecast.map(item => (
                    <tr key={item.product_id} className="border-b border-[var(--border)] last:border-b-0">
                      <td className="px-4 py-2">{item.product_name || item.product_id.slice(0, 8)}</td>
                      <td className="px-4 py-2 font-mono">{item.current_stock}</td>
                      <td className="px-4 py-2 font-mono font-semibold">{item.recommended_qty}</td>
                      <td className="px-4 py-2">
                        <span className={item.days_until_stockout < 3 ? 'text-[var(--danger)] font-semibold' : ''}>
                          {item.days_until_stockout.toFixed(1)}d
                        </span>
                      </td>
                      <td className="px-4 py-2">
                        <span className={`text-xs font-semibold ${
                          item.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
                          item.priority === 'URGENT' ? 'text-[var(--warning)]' : 'text-[var(--muted)]'
                        }`}>{item.priority}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}

        {/* Manual items */}
        {!useForecast && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Manual Items</span>
              <button
                type="button"
                onClick={() => setManualItems(prev => [...prev, { product_id: '', quantity: 0, unit: 'units' }])}
                className="text-xs text-[var(--link)] hover:underline"
              >
                + Add item
              </button>
            </div>
            {manualItems.map((item, idx) => (
              <div key={idx} className="flex gap-2">
                <input
                  type="text"
                  placeholder="Product ID"
                  value={item.product_id}
                  onChange={e => {
                    const next = [...manualItems];
                    next[idx] = { ...next[idx], product_id: e.target.value };
                    setManualItems(next);
                  }}
                  className="flex-1 px-3 py-2 rounded-lg border text-sm outline-none"
                  style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                />
                <input
                  type="number"
                  placeholder="Qty"
                  min={1}
                  value={item.quantity || ''}
                  onChange={e => {
                    const next = [...manualItems];
                    next[idx] = { ...next[idx], quantity: parseInt(e.target.value) || 0 };
                    setManualItems(next);
                  }}
                  className="w-24 px-3 py-2 rounded-lg border text-sm outline-none"
                  style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                />
                <button
                  type="button"
                  onClick={() => setManualItems(prev => prev.filter((_, i) => i !== idx))}
                  className="px-2 text-[var(--danger)] hover:text-[var(--danger)]"
                >
                  <Icon name="cancel" size={16} />
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Notes */}
        <div>
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">Notes</label>
          <textarea
            value={notes}
            onChange={e => setNotes(e.target.value)}
            rows={3}
            placeholder="Optional notes for the factory..."
            className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none resize-y"
            style={{
              background: 'var(--field-background)',
              color: 'var(--field-foreground)',
              borderColor: 'var(--field-border)',
            }}
          />
        </div>

        <div className="flex gap-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
          >
            {submitting ? 'Creating...' : 'Create Supply Request'}
          </button>
          <button
            type="button"
            onClick={() => router.back()}
            className="px-4 py-2.5 rounded-lg text-sm button--secondary border border-[var(--border)]"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
