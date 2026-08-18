'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { createWarehouseApi } from '@/lib/api';
import { useRouter } from 'next/navigation';
import { warehouseCreateSupplyRequestKey, warehouseSupplyRequestTransitionKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
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
  const t = usePortalT();
  const router = useRouter();
  const { toast } = useToast();
  const [factoryId, setFactoryId] = useState('');
  const [factorySource, setFactorySource] = useState('');
  const [factoryLocked, setFactoryLocked] = useState(false);
  const [deliveryDate, setDeliveryDate] = useState('');
  const [notes, setNotes] = useState('');
  const [useForecast, setUseForecast] = useState(true);
  const [forecast, setForecast] = useState<ForecastItem[]>([]);
  const [forecastLoading, setForecastLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // Manual items (when not using forecast)
  const [manualItems, setManualItems] = useState<{ product_id: string; quantity: number; unit: string }[]>([]);

  const loadEngineFactory = useCallback(async () => {
    try {
      const supply = await createWarehouseApi().getWarehouseOpsSupplyFactory();
      if (supply.factory_id) {
        setFactoryId(supply.factory_id);
        setFactorySource(supply.source);
        setFactoryLocked(true);
      }
    } catch {
      setFactoryLocked(false);
    }
  }, []);

  useEffect(() => {
    void loadEngineFactory();
  }, [loadEngineFactory]);

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

      const warehouseId = warehouseHomeNodeId() || 'warehouse';
      const res = await apiFetch('/v1/warehouse/supply-requests', {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseCreateSupplyRequestKey(
            warehouseId,
            factoryId.trim(),
            useForecast ? 'FORECAST' : 'MANUAL',
            notes,
          ),
        },
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
    <PageTransition>
      <PageChrome
        icon="supplyRequests"
        title={t("warehouse_portal.supply_requests.new.text.new_supply_request")}
        description={t("warehouse_portal.residual.text.create_a_factory_replenishment_request_from_forecast_or_manual_l")}
        actions={
          <button type="button" onClick={() => router.back()} className="p-1 rounded-lg hover:bg-[var(--surface)]">
            <Icon name="left" size={20} />
          </button>
        }
      >
      <div className="max-w-3xl space-y-6">
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Factory selector */}
        <div>
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">{t("warehouse_portal.supply_requests.new.text.factory_id")}</label>
          <input
            type="text"
            value={factoryId}
            onChange={e => setFactoryId(e.target.value)}
            placeholder={t("warehouse_portal.supply_requests.new.text.enter_factory_uuid")}
            required
            readOnly={factoryLocked}
            className="w-full px-3 py-2.5 rounded-lg border text-sm outline-none"
            style={{
              background: 'var(--field-background)',
              color: 'var(--field-foreground)',
              borderColor: 'var(--field-border)',
            }}
          />
          {factoryLocked ? (
            <p className="mt-1 text-xs text-[var(--muted)]">Nearest factory from the engine{factorySource ? ` (${factorySource})` : ""}.</p>
          ) : null}
        </div>

        {/* Delivery date */}
        <div>
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">{t("warehouse_portal.supply_requests.new.text.requested_delivery_date")}</label>
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
            <span className="text-sm font-medium">{t("warehouse_portal.supply_requests.new.text.use_ai_demand_forecast")}</span>
          </label>
          <span className="text-xs text-[var(--muted)]">{t("warehouse_portal.supply_requests.new.text.auto_fill_items_from_demand_engine")}</span>
        </div>

        {/* Forecast preview */}
        {useForecast && (
          <div className="border border-[var(--border)] rounded-xl overflow-hidden">
            <div className="px-4 py-3 border-b border-[var(--border)] flex items-center justify-between" style={{ background: 'var(--surface)' }}>
              <span className="text-sm font-semibold">{t("warehouse_portal.supply_requests.new.text.demand_forecast_7_day")}</span>
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
              <table className="desk-table w-full text-sm">
                <thead>
                  <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">{t("portal.nav.stock")}</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.recommended")}</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">{t("warehouse_portal.supply_requests.new.text.stockout_in")}</th>
                    <th className="text-left px-4 py-2 text-xs text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.priority")}</th>
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
              <span className="text-sm font-medium">{t("warehouse_portal.supply_requests.new.text.manual_items")}</span>
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
                  placeholder={t("warehouse_portal.supply_requests.new.text.product_id")}
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
                  placeholder={t("warehouse_portal.pick_waves.text.qty")}
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
          <label className="block text-xs font-medium mb-1.5 text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.notes")}</label>
          <textarea
            value={notes}
            onChange={e => setNotes(e.target.value)}
            rows={3}
            placeholder={t("warehouse_portal.supply_requests.new.text.optional_notes_for_the_factory")}
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
      </PageChrome>
    </PageTransition>
  );
}
