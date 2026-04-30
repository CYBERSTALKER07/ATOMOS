'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { apiFetch } from '@/lib/auth';
import { useAuth } from '@/hooks/useAuth';

// ── Response Types ──────────────────────────────────────────────────────────

export type TransitPoint = {
  lat: number; lng: number; count: number; state: string;
};
export type ThroughputBucket = {
  date: string; order_count: number; completed_count: number; cancelled_count: number;
};
export type LoadBucket = {
  vehicle_class: string; vehicle_count: number; avg_load_pct: number; max_load_pct: number;
};
export type NodeMetric = {
  warehouse_id: string; warehouse_name: string; order_count: number; avg_cycle_min: number; on_time_rate: number;
};
export type SLAEntry = {
  date: string; on_time: number; late: number; breached: number; total_orders: number;
};
export type RevenueDayBucket = {
  date: string; total: number; global_pay: number; card: number; cash: number;
};
export type GatewayBreakdown = {
  gateway: string; total: number; order_count: number;
};
export type TopRetailer = {
  retailer_id: string; shop_name: string; order_count: number; total_revenue: number; avg_order_value: number; last_order_at: string;
};
export type FactoryDayBucket = {
  date: string; transfers_created: number; transfers_shipped: number; units_produced: number;
};
export type FactoryStatusSummary = {
  state: string; count: number;
};

export interface RevenueData {
  time_series: RevenueDayBucket[];
  gateway_breakdown: GatewayBreakdown[];
}

export interface FactoryOverviewData {
  daily_activity: FactoryDayBucket[];
  transfers_by_state: FactoryStatusSummary[];
  total_transfers: number;
  avg_lead_time_mins: number;
}

// ── Date Range ──────────────────────────────────────────────────────────────

export type DateRangePreset = '7d' | '14d' | '30d' | '90d' | '180d' | '365d' | 'custom';

function presetToDays(preset: DateRangePreset): number {
  const map: Record<string, number> = { '7d': 7, '14d': 14, '30d': 30, '90d': 90, '180d': 180, '365d': 365 };
  return map[preset] ?? 30;
}

function dateStr(d: Date): string {
  return d.toISOString().split('T')[0];
}

export interface DateRange {
  from: string;
  to: string;
  preset: DateRangePreset;
}

function defaultRange(): DateRange {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  return { from: dateStr(from), to: dateStr(to), preset: '30d' };
}

// ── Hook ────────────────────────────────────────────────────────────────────

export interface AdvancedAnalyticsData {
  // Supplier intelligence
  transitHeatmap: TransitPoint[];
  throughput: ThroughputBucket[];
  loadDistribution: LoadBucket[];
  nodeEfficiency: NodeMetric[];
  slaHealth: SLAEntry[];
  revenue: RevenueData | null;
  topRetailers: TopRetailer[];
  // Factory
  factoryOverview: FactoryOverviewData | null;
  // State
  loading: boolean;
  error: string | null;
  dateRange: DateRange;
  setDateRange: (dr: DateRange) => void;
  setPreset: (p: DateRangePreset) => void;
  refresh: () => void;
}

export function useAdvancedAnalytics(warehouseId?: string): AdvancedAnalyticsData {
  const auth = useAuth();
  const [dateRange, setDateRange] = useState<DateRange>(defaultRange);

  const [transitHeatmap, setTransitHeatmap] = useState<TransitPoint[]>([]);
  const [throughput, setThroughput] = useState<ThroughputBucket[]>([]);
  const [loadDistribution, setLoadDistribution] = useState<LoadBucket[]>([]);
  const [nodeEfficiency, setNodeEfficiency] = useState<NodeMetric[]>([]);
  const [slaHealth, setSlaHealth] = useState<SLAEntry[]>([]);
  const [revenue, setRevenue] = useState<RevenueData | null>(null);
  const [topRetailers, setTopRetailers] = useState<TopRetailer[]>([]);
  const [factoryOverview, setFactoryOverview] = useState<FactoryOverviewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const abortRef = useRef<AbortController | null>(null);

  const setPreset = useCallback((p: DateRangePreset) => {
    const to = new Date();
    const from = new Date();
    from.setDate(from.getDate() - presetToDays(p));
    setDateRange({ from: dateStr(from), to: dateStr(to), preset: p });
  }, []);

  const fetchAll = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    const { signal } = controller;

    setLoading(true);
    setError(null);

    try {
      const rangeQs = `from=${dateRange.from}&to=${dateRange.to}`;
      const whQs = warehouseId ? `&warehouse_id=${warehouseId}` : '';
      const qs = `?${rangeQs}${whQs}`;

      const isFactory = auth.isFactoryStaff;

      // Supplier analytics (skip for factory-only users)
      const supplierEndpoints = isFactory ? [] : [
        `/v1/supplier/analytics/transit-heatmap${qs}`,
        `/v1/supplier/analytics/throughput${qs}`,
        `/v1/supplier/analytics/load-distribution${qs}`,
        `/v1/supplier/analytics/node-efficiency${qs}`,
        `/v1/supplier/analytics/sla-health${qs}`,
        `/v1/supplier/analytics/revenue${qs}`,
        `/v1/supplier/analytics/top-retailers${qs}`,
      ];

      // Factory analytics
      const factoryEndpoints = isFactory ? [
        `/v1/factory/analytics/overview?${rangeQs}`,
      ] : [];

      const allEndpoints = [...supplierEndpoints, ...factoryEndpoints];

      const responses = await Promise.all(
        allEndpoints.map((ep) => apiFetch(ep, { signal }).catch(() => null)),
      );

      const results = await Promise.all(
        responses.map(async (res) => {
          if (!res || !res.ok) return null;
          return res.json();
        }),
      );

      if (signal.aborted) return;

      if (!isFactory) {
        setTransitHeatmap(results[0]?.data ?? []);
        setThroughput(results[1]?.data ?? []);
        setLoadDistribution(results[2]?.data ?? []);
        setNodeEfficiency(results[3]?.data ?? []);
        setSlaHealth(results[4]?.data ?? []);
        setRevenue(results[5] ?? null);
        setTopRetailers(Array.isArray(results[6]) ? results[6] : results[6]?.data ?? []);
      }

      if (isFactory && results[0]) {
        setFactoryOverview(results[0]);
      }

      setError(null);
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, [dateRange, warehouseId, auth.isFactoryStaff]);

  useEffect(() => {
    fetchAll();
    return () => abortRef.current?.abort();
  }, [fetchAll]);

  return {
    transitHeatmap, throughput, loadDistribution, nodeEfficiency,
    slaHealth, revenue, topRetailers, factoryOverview,
    loading, error, dateRange, setDateRange, setPreset, refresh: fetchAll,
  };
}
