'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import type { ForecastConfidence, WarehouseReplenishmentInsight } from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { ForecastChartPanel } from '@/components/forecast/ForecastChartPanel';
import { ForecastSkuTable } from '@/components/forecast/ForecastSkuTable';

export interface ForecastProduct {
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
  confidence?: ForecastConfidence;
}

interface Forecast {
  warehouse_id: string;
  forecast_days: number;
  generated_at: string;
  products: ForecastProduct[];
}

export default function DemandForecastPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [forecast, setForecast] = useState<Forecast | null>(null);
  const [loading, setLoading] = useState(true);
  const [days, setDays] = useState(7);

  async function loadForecast() {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehouseDemandForecast({ days });
      const portalForecast = data as unknown as Forecast & {
        products?: ForecastProduct[];
        generated_at?: string;
        forecast_days?: number;
      };
      if (portalForecast.products?.length) {
        setForecast({
          warehouse_id: portalForecast.warehouse_id || '',
          forecast_days: portalForecast.forecast_days ?? days,
          generated_at: portalForecast.generated_at || new Date().toISOString(),
          products: portalForecast.products.map((product) => ({
            ...product,
            confidence:
              product.confidence
              ?? parseForecastConfidence(
                (product as ForecastProduct & { demand_breakdown?: Record<string, unknown> }).demand_breakdown,
              )
              ?? undefined,
          })),
        });
        return;
      }

      const insightsData = await warehouseApi.getWarehouseReplenishmentInsights().catch(() => null);
      const insightRows: WarehouseReplenishmentInsight[] = insightsData?.insights || insightsData?.data || [];
      const products: ForecastProduct[] = insightRows.map((insight) => {
        const urgency = insight.urgency.toUpperCase();
        const priority =
          urgency === 'CRITICAL' || urgency === 'HIGH'
            ? 'CRITICAL'
            : urgency === 'URGENT' || urgency === 'MEDIUM'
              ? 'URGENT'
              : 'NORMAL';
        return {
          product_id: insight.product_id,
          product_name: insight.product_name,
          current_stock: insight.current_stock,
          recommended_qty: insight.reorder_quantity,
          days_until_stockout: insight.days_until_stockout,
          priority,
          unit: 'VU',
          sources: {
            incoming_orders: 0,
            ai_prediction: 0,
            pre_orders: 0,
            burn_rate: insight.avg_daily_velocity,
          },
          confidence: parseForecastConfidence(insight.demand_breakdown) ?? undefined,
        };
      });
      setForecast({
        warehouse_id: insightRows[0]?.warehouse_id || portalForecast.warehouse_id || '',
        forecast_days: days,
        generated_at: insightRows[0]?.created_at || new Date().toISOString(),
        products,
      });
    } catch {
      toast('Failed to load forecast', 'error');
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
    <PageTransition>
      <PageChrome
        icon="forecast"
        title={t("portal.nav.demand_forecast")}
        description={t("warehouse_portal.residual.text.ai_powered_stock_recommendations_from_4_data_sources")}
        loading={loading}
        actions={
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
              type="button"
              onClick={loadForecast}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
            >
              <Icon name="refresh" size={16} />
              Refresh
            </button>
          </div>
        }
      >
      <div className="space-y-6">
      <ForecastChartPanel products={products} />

      {!loading && products.length === 0 ? (
        <div className="text-center py-20 text-[var(--muted)]">
          <Icon name="forecast" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">{t("warehouse_portal.demand_forecast.text.no_products_tracked_yet")}</p>
        </div>
      ) : !loading ? (
        <ForecastSkuTable products={products} />
      ) : null}

      {forecast && !loading && (
        <div className="text-xs text-[var(--muted)]">
          Generated at {new Date(forecast.generated_at).toLocaleString()} for {forecast.forecast_days}-day window
        </div>
      )}
      </div>
      </PageChrome>
    </PageTransition>
  );
}
