"use client";

import { useEffect, useRef, useState, useCallback, useMemo, lazy, Suspense } from "react";
import { apiFetch } from "@/lib/auth";
import { usePolling } from "@/lib/usePolling";
import { useTelemetry } from "@/hooks/useTelemetry";
import type { TelemetryMessage } from "@/hooks/useTelemetry";
import { isTauri } from "@/lib/bridge";
import { buildSupplierFleetDispatchIdempotencyKey } from "@/app/supplier/_shared/idempotency";
import { Card, Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard, BentoSkeleton } from "@/components/BentoGrid";
import CountUp from "@/components/CountUp";
import MiniSparkline from "@/components/MiniSparkline";
import {
  Truck, CheckCircle, CreditCard,
  ArrowUpRight, Activity, Send, Search,
} from "lucide-react";
import {
  PieChart, Pie, Cell, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, Tooltip,
} from "recharts";

// Lazy-load heavy cells
const FleetMapCell = lazy(() => import("@/components/dashboard/FleetMapCell"));
const OrphanAlertsCell = lazy(() => import("@/components/dashboard/OrphanAlertsCell"));
const QuickActionsCell = lazy(() => import("@/components/dashboard/QuickActionsCell"));

// ─── Types ─────────────────────────────────────────────────────────────────

type Order = {
  order_id: string;
  retailer_id: string;
  state: string;
  amount?: number;
  payment_gateway?: string;
  route_id?: string | null;
  order_source?: string | null;
  auto_confirm_at?: string | null;
  deliver_before?: string | null;
};

type FleetDriver = {
  driver_id: string;
  name: string;
  vehicle_type: string;
  license_plate: string;
  truck_status: string;
  is_active: boolean;
};

type LoadingManifestEntry = {
  load_sequence: number;
  order_id: string;
  retailer_name: string;
  volume_vu: number;
  lat: number;
  lng: number;
  instruction: string;
};

type TruckManifest = {
  route_id: string;
  driver_name: string;
  orders: { order_id: string }[];
  loading_manifest: LoadingManifestEntry[];
};

type OrderViewFilter = "ALL" | "PENDING" | "ACTIVE" | "COMPLETED" | "REVIEW";

// ─── Status Chip ────────────────────────────────────────────────────────────

const chipConfig: Record<
  string,
  {
    color: "default" | "accent" | "success" | "warning" | "danger";
    variant?: "primary" | "secondary" | "soft";
    label?: string;
  }
> = {
  PENDING: { color: "default", variant: "soft" },
  EN_ROUTE: { color: "accent", variant: "primary" },
  IN_TRANSIT: { color: "warning", variant: "soft" },
  DISPATCHED: { color: "accent", variant: "soft", label: "Dispatched" },
  COMPLETED: { color: "success", variant: "soft" },
  PENDING_REVIEW: { color: "accent", variant: "soft", label: "Review" },
};

const StatusChip = ({ status }: { status: string | undefined }) => {
  const key = status?.toUpperCase() ?? "";
  const cfg = chipConfig[key] ?? { color: "danger" as const, variant: "soft" as const };
  return (
    <Chip color={cfg.color} variant={cfg.variant ?? "soft"} size="sm">
      {cfg.label ?? status ?? "Unknown"}
    </Chip>
  );
};

// ─── Temporal Urgency ───────────────────────────────────────────────────────

const getTemporalStatus = (deliverBefore: string | null | undefined) => {
  if (!deliverBefore) return { isUrgent: false, label: null as string | null };
  const delta = new Date(deliverBefore).getTime() - Date.now();
  if (delta <= 0) return { isUrgent: true, label: "Overdue" };
  if (delta <= 60 * 60 * 1000) return { isUrgent: true, label: "Critical" };
  if (delta <= 3 * 60 * 60 * 1000) return { isUrgent: false, label: "Urgent" };
  return { isUrgent: false, label: null as string | null };
};

// ─── Chart Colors (monochrome) ──────────────────────────────────────────────

const MONO_SHADES = [
  'var(--foreground)',
  'var(--muted)',
  'var(--border)',
  'var(--border)',
  'var(--surface)',
];

// ─── Main Component ─────────────────────────────────────────────────────────

export default function AdminDashboard() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [drivers, setDrivers] = useState<FleetDriver[]>([]);
  const [isApiOnline, setIsApiOnline] = useState(false);
  const [lastUpdated, setLastUpdated] = useState("");
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const [selectedOrders, setSelectedOrders] = useState<Set<string>>(new Set());
  const [targetRoute, setTargetRoute] = useState("");
  const [isDispatching, setIsDispatching] = useState(false);
  const [dispatchMsg, setDispatchMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [manifests, setManifests] = useState<TruckManifest[]>([]);
  const [printManifest, setPrintManifest] = useState<TruckManifest | null>(null);
  const [orderView, setOrderView] = useState<OrderViewFilter>("ALL");
  const [searchTerm, setSearchTerm] = useState("");
  const [showUrgentOnly, setShowUrgentOnly] = useState(false);

  // ── Polling ──────────────────────────────────────────────────────────────

  const fetchOrders = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const [ordersRes, driversRes] = await Promise.all([
          apiFetch("/v1/orders", { signal }),
          apiFetch("/v1/supplier/fleet/drivers", { signal }),
        ]);
        if (ordersRes.ok) {
          const data = await ordersRes.json();
          setOrders(data ?? []);
          setIsApiOnline(true);
          setLastUpdated(new Date().toLocaleTimeString());
          setLastUpdatedAt(Date.now());
        } else {
          setIsApiOnline(false);
        }
        if (driversRes.ok) {
          const driverData: FleetDriver[] = await driversRes.json();
          const active = (driverData ?? []).filter((d) => d.is_active);
          setDrivers(active);
          if (!targetRoute && active.length > 0) setTargetRoute(active[0].driver_id);
        }
      } catch (err) {
        if ((err as Error).name === "AbortError") return;
        setIsApiOnline(false);
      } finally {
        setIsLoading(false);
      }
    },
    [targetRoute],
  );

  usePolling((signal) => fetchOrders(signal), 5000, [fetchOrders]);

  // ── WebSocket: instant order state change notifications ──────────────────
  const fetchOrdersRef = useRef(fetchOrders);
  fetchOrdersRef.current = fetchOrders;
  useTelemetry(
    useCallback((data: TelemetryMessage) => {
      if (data.type === "ORDER_STATE_CHANGED") {
        fetchOrdersRef.current();
      }
    }, []),
    { enabled: !isTauri() },
  );

  // ── Selection ────────────────────────────────────────────────────────────

  const pendingOrders = useMemo(
    () => orders.filter((o) => (o.state === "PENDING" || o.state === "PENDING_REVIEW") && !o.route_id),
    [orders],
  );

  useEffect(() => {
    setSelectedOrders((prev) => {
      if (prev.size === 0) return prev;
      const selectable = new Set(pendingOrders.map((o) => o.order_id));
      const next = new Set([...prev].filter((id) => selectable.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [pendingOrders]);

  const filteredOrders = useMemo(() => {
    const query = searchTerm.trim().toLowerCase();
    const matchesOrderView = (order: Order) => {
      switch (orderView) {
        case "PENDING":
          return order.state === "PENDING";
        case "ACTIVE":
          return order.state === "IN_TRANSIT" || order.state === "EN_ROUTE" || order.state === "DISPATCHED";
        case "COMPLETED":
          return order.state === "COMPLETED";
        case "REVIEW":
          return order.state === "PENDING_REVIEW";
        default:
          return true;
      }
    };

    const urgencyWeight = (label: string | null) => {
      if (label === "Overdue") return 3;
      if (label === "Critical") return 2;
      if (label === "Urgent") return 1;
      return 0;
    };

    return orders
      .filter((order) => {
        if (!matchesOrderView(order)) return false;
        const temporal = getTemporalStatus(order.deliver_before);
        if (showUrgentOnly && !temporal.label) return false;
        if (!query) return true;
        const haystack = `${order.order_id} ${order.retailer_id} ${order.route_id ?? ""}`.toLowerCase();
        return haystack.includes(query);
      })
      .sort((a, b) => {
        const urgencyDelta = urgencyWeight(getTemporalStatus(b.deliver_before).label) - urgencyWeight(getTemporalStatus(a.deliver_before).label);
        if (urgencyDelta !== 0) return urgencyDelta;
        return a.order_id.localeCompare(b.order_id);
      });
  }, [orders, orderView, searchTerm, showUrgentOnly]);

  const allSelected =
    pendingOrders.length > 0 &&
    pendingOrders.every((o) => selectedOrders.has(o.order_id));

  const toggleSelectAll = useCallback(() => {
    setSelectedOrders((prev) => {
      const allIds = new Set(pendingOrders.map((o) => o.order_id));
      return prev.size === allIds.size && [...allIds].every((id) => prev.has(id))
        ? new Set()
        : allIds;
    });
  }, [pendingOrders]);

  const toggleOrder = useCallback((id: string) => {
    setSelectedOrders((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  // ── Dispatch ─────────────────────────────────────────────────────────────

  const executeDispatch = useCallback(async () => {
    if (selectedOrders.size === 0 || isDispatching) return;
    setIsDispatching(true);
    setDispatchMsg(null);
    setManifests([]);
    try {
      const payload = { order_ids: Array.from(selectedOrders), route_id: targetRoute };
      const res = await apiFetch("/v1/fleet/dispatch", {
        method: "POST",
        headers: {
          "Idempotency-Key": buildSupplierFleetDispatchIdempotencyKey(targetRoute, payload.order_ids),
        },
        body: JSON.stringify(payload),
      });
      const body = await res.json().catch(() => ({} as {
        queued?: boolean;
        message?: string;
        error?: string;
        manifests?: TruckManifest[];
      }));
      if (body.queued) {
        setSelectedOrders(new Set());
        setDispatchMsg({ ok: true, text: "Dispatch queued — will replay when back online" });
      } else if (res.ok) {
        setSelectedOrders(new Set());
        setDispatchMsg({ ok: true, text: body.message || "Dispatch submitted" });
        setManifests(body.manifests ?? []);
        fetchOrders();
      } else {
        setDispatchMsg({ ok: false, text: body.error || body.message || `Error ${res.status}` });
      }
    } catch (e: unknown) {
      setDispatchMsg({
        ok: false,
        text: `${e instanceof Error ? e.message : "Network failure"}`,
      });
    } finally {
      setIsDispatching(false);
    }
  }, [selectedOrders, isDispatching, targetRoute, fetchOrders]);

  // ── Supplier name cookie ─────────────────────────────────────────────────

  const [supplierName, setSupplierName] = useState("");
  useEffect(() => {
    const m = document.cookie.match(/(?:admin_name|supplier_name)=([^;]+)/);
    if (m) setSupplierName(decodeURIComponent(m[1]));
  }, []);

  // ── Computed values ──────────────────────────────────────────────────────

  const greeting = useMemo(() => {
    const h = new Date().getHours();
    return h < 12 ? "Good morning" : h < 18 ? "Good afternoon" : "Good evening";
  }, []);

  const isDataStale = useMemo(() => {
    if (lastUpdatedAt == null) return false;
    return Date.now() - lastUpdatedAt > 20_000;
  }, [lastUpdatedAt]);

  const filterCounts = useMemo(() => {
    const pending = orders.filter((o) => o.state === "PENDING").length;
    const active = orders.filter((o) => o.state === "IN_TRANSIT" || o.state === "EN_ROUTE" || o.state === "DISPATCHED").length;
    const completed = orders.filter((o) => o.state === "COMPLETED").length;
    const review = orders.filter((o) => o.state === "PENDING_REVIEW").length;
    return { all: orders.length, pending, active, completed, review };
  }, [orders]);

  const kpi = useMemo(() => {
    const activeTrucks = new Set(
      orders.filter((o) => o.route_id).map((o) => o.route_id),
    ).size;
    const completed = orders.filter((o) => o.state === "COMPLETED").length;
    const inTransit = orders.filter((o) => o.state === "IN_TRANSIT" || o.state === "EN_ROUTE" || o.state === "DISPATCHED").length;
    const pending = orders.filter((o) => o.state === "PENDING" || o.state === "PENDING_REVIEW").length;
    const totalRev = orders
      .filter((o) => o.state === "COMPLETED")
      .reduce((s, o) => s + (o.amount ?? 0), 0);
    const globalPayRev = orders
      .filter((o) => o.state === "COMPLETED" && o.payment_gateway?.toUpperCase() === "GLOBAL_PAY")
      .reduce((s, o) => s + (o.amount ?? 0), 0);
    const cashRev = orders
      .filter((o) => o.state === "COMPLETED" && o.payment_gateway?.toUpperCase() === "CASH")
      .reduce((s, o) => s + (o.amount ?? 0), 0);
    return { activeTrucks, completed, inTransit, pending, totalRev, globalPayRev, cashRev, total: orders.length };
  }, [orders]);

  // Fake sparkline data (deterministic from order counts)
  const truckSparkline = useMemo(() =>
    Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.activeTrucks + Math.sin(i * 0.8) * 2)),
    [kpi.activeTrucks]
  );
  const completedSparkline = useMemo(() =>
    Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.completed * 0.3 + i * (kpi.completed * 0.06))),
    [kpi.completed]
  );

  // Pipeline data for donut chart
  const pipelineData = useMemo(() => [
    { name: 'Completed', value: kpi.completed, sparkline: completedSparkline },
    { name: 'In Transit', value: kpi.inTransit },
    { name: 'Pending', value: kpi.pending },
    { name: 'Other', value: Math.max(0, kpi.total - kpi.completed - kpi.inTransit - kpi.pending) },
  ].filter(d => d.value > 0), [kpi, completedSparkline]);

  const revData = useMemo(() => {
    const revenueByGateway = orders
      .filter((order) => order.state === "COMPLETED" && (order.amount ?? 0) > 0)
      .reduce<Record<string, number>>((acc, order) => {
        const gateway = (order.payment_gateway ?? "UNSPECIFIED").toUpperCase();
        acc[gateway] = (acc[gateway] ?? 0) + (order.amount ?? 0);
        return acc;
      }, {});

    return Object.entries(revenueByGateway)
      .map(([gateway, amount]) => ({
        gateway: gateway.replace(/_/g, " "),
        amount,
      }))
      .sort((left, right) => right.amount - left.amount);
  }, [orders]);

  // ── Render ───────────────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div className="p-6 md:p-8 space-y-6">
        <div className="flex items-end justify-between">
          <div className="space-y-2">
            <Skeleton className="h-8 w-64 rounded-lg" />
            <Skeleton className="h-4 w-40 rounded" />
          </div>
        </div>
        <BentoGrid>
          <BentoSkeleton size="anchor" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="stat" />
          <BentoSkeleton size="control" />
          <BentoSkeleton size="list" />
          <BentoSkeleton span={2} />
          <BentoSkeleton span={2} />
        </BentoGrid>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8">
      {/* ── Header ──────────────────────────────────────────────────────── */}
      <header className="mb-6 flex items-end justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-large">
            {greeting}{supplierName ? `, ${supplierName}` : ""}
          </h1>
          <div className="flex items-center gap-3 mt-1">
            <div className="flex items-center gap-1.5">
              <div className={`w-1.5 h-1.5 rounded-full ${isApiOnline ? 'animate-pulse' : ''}`}
                style={{ background: isApiOnline ? 'var(--success)' : 'var(--danger)' }} />
              <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                {isApiOnline ? 'Live' : 'Offline'}
              </span>
            </div>
            <span className="md-typescale-label-small font-mono tabular-nums" style={{ color: 'var(--muted)' }}>
              {lastUpdated || "--:--:--"}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isDataStale && (
            <Chip color="warning" variant="soft" size="sm">
              Data stale
            </Chip>
          )}
          <Button variant="outline" size="sm" onPress={() => void fetchOrders()}>
            Refresh
          </Button>
        </div>
      </header>

      {(!isApiOnline || isDataStale) && (
        <div
          className="mb-6 rounded-xl px-4 py-3 flex flex-col md:flex-row md:items-center md:justify-between gap-2"
          style={{
            border: '1px solid var(--border)',
            background: 'var(--surface)',
          }}
        >
          <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
            {!isApiOnline
              ? "Realtime feed is disconnected. Dashboard is showing the most recent successful snapshot."
              : "Data is older than 20 seconds. Verify network stability or refresh manually."}
          </span>
          <Button variant="secondary" size="sm" onPress={() => void fetchOrders()}>
            Retry sync
          </Button>
        </div>
      )}

      {/* ══════════════════════════════════════════════════════════════════ */}
      {/* BENTO GRID — Modular cells. Size = Priority.                     */}
      {/* Anchor (2×2) → Statistics (1×1) → List (1×2) → Control (2×1)    */}
      {/* ══════════════════════════════════════════════════════════════════ */}

      <BentoGrid className="mb-8">

        {/* ── ANCHOR (2×2): Real-Time Fleet GPS Map ───────────────────── */}
        <BentoCard size="anchor" delay={0}>
          <Suspense fallback={<div className="skeleton w-full h-full rounded" />}>
            <FleetMapCell />
          </Suspense>
        </BentoCard>

        {/* ── STATISTICS (1×1): Active Trucks ─────────────────────────── */}
        <BentoCard size="stat" delay={60}>
          <div className="flex flex-col justify-between h-full p-5 active:scale-[0.98] transition-transform cursor-default">
            <div className="flex items-center justify-between mb-2">
              <div className="w-10 h-10 rounded-2xl bg-surface-container flex items-center justify-center">
                <Truck size={20} strokeWidth={1.5} className="text-muted" />
              </div>
              <MiniSparkline data={truckSparkline} width={64} height={24} />
            </div>
            <div>
              <p className="md-typescale-label-medium text-muted mb-1">Active Trucks</p>
              <div className="flex items-baseline gap-2">
                <CountUp end={kpi.activeTrucks} className="md-typescale-display-small font-bold tabular-nums tracking-tighter" />
                <span className="md-typescale-label-small text-muted">/ {kpi.total} orders</span>
              </div>
            </div>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): Completed ─────────────────────────────── */}
        <BentoCard size="stat" delay={120}>
          <div className="flex flex-col justify-between h-full p-5 active:scale-[0.98] transition-transform cursor-default">
            <div className="flex items-center justify-between mb-2">
              <div className="w-10 h-10 rounded-2xl bg-success-container/10 flex items-center justify-center">
                <CheckCircle size={20} strokeWidth={1.5} className="text-success" />
              </div>
              {kpi.total > 0 && (
                <div className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-success-container/10 text-success md-typescale-label-small">
                  <ArrowUpRight size={12} strokeWidth={2.5} />
                  {((kpi.completed / kpi.total) * 100).toFixed(0)}%
                </div>
              )}
            </div>
            <div>
              <p className="md-typescale-label-medium text-muted mb-1">Completed</p>
              <CountUp end={kpi.completed} className="md-typescale-display-small font-bold tabular-nums tracking-tighter" />
            </div>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): In Transit ────────────────────────────── */}
        <BentoCard size="stat" delay={180}>
          <div className="flex flex-col justify-between h-full p-5 active:scale-[0.98] transition-transform cursor-default">
            <div className="flex items-center justify-between mb-2">
              <div className="w-10 h-10 rounded-2xl bg-surface-container flex items-center justify-center">
                <Activity size={20} strokeWidth={1.5} className="text-muted" />
              </div>
            </div>
            <div>
              <p className="md-typescale-label-medium text-muted mb-1">In Transit</p>
              <div className="flex items-baseline gap-2">
                <CountUp end={kpi.inTransit} className="md-typescale-display-small font-bold tabular-nums tracking-tighter" />
                <span className="md-typescale-label-small text-muted">{kpi.pending} pending</span>
              </div>
            </div>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): Revenue ───────────────────────────────── */}
        <BentoCard size="stat" delay={240}>
          <div className="flex flex-col justify-between h-full p-5 active:scale-[0.98] transition-transform cursor-default">
            <div className="flex items-center justify-between mb-2">
              <div className="w-10 h-10 rounded-2xl bg-accent text-accent-foreground flex items-center justify-center">
                <CreditCard size={20} strokeWidth={1.5} />
              </div>
            </div>
            <div>
              <p className="md-typescale-label-medium text-muted mb-1">Revenue</p>
              <div className="flex items-baseline gap-1">
                <CountUp end={kpi.totalRev} className="md-typescale-display-small font-bold tabular-nums tracking-tighter" />
                <span className="md-typescale-label-small text-muted">Total</span>
              </div>
            </div>
          </div>
        </BentoCard>

        {/* ── CONTROL (2×1): Quick Actions ────────────────────────────── */}
        <BentoCard size="control" delay={300}>
          <Suspense fallback={<div className="skeleton w-full h-full rounded" />}>
            <QuickActionsCell />
          </Suspense>
        </BentoCard>

        {/* ── LIST (1×2): Orphaned Retailer Alerts ────────────────────── */}
        <BentoCard size="list" delay={360}>
          <Suspense fallback={<div className="skeleton w-full h-full rounded" />}>
            <OrphanAlertsCell />
          </Suspense>
        </BentoCard>

        {/* ── Pipeline Donut (2×1) ────────────────────────────────────── */}
        <BentoCard span={2} delay={420}>
          <div className="flex flex-col h-full">
            <div className="bento-card-header">
              <span className="bento-card-title">Order Pipeline</span>
            </div>
            {pipelineData.length > 0 ? (
              <div className="flex items-center gap-6 flex-1 min-h-0">
                <ResponsiveContainer width="50%" height="100%">
                  <PieChart>
                    <Pie
                      data={pipelineData}
                      innerRadius={55}
                      outerRadius={80}
                      paddingAngle={3}
                      dataKey="value"
                      strokeWidth={0}
                    >
                      {pipelineData.map((_, i) => (
                        <Cell key={i} fill={MONO_SHADES[i % MONO_SHADES.length]} />
                      ))}
                    </Pie>
                    <Tooltip
                      contentStyle={{
                        background: 'var(--surface)',
                        border: '1px solid var(--border)',
                        borderRadius: 0,
                        fontSize: 13,
                        color: 'var(--foreground)',
                      }}
                    />
                  </PieChart>
                </ResponsiveContainer>
                <div className="flex flex-col gap-3 flex-1">
                  {pipelineData.map((d, i) => (
                    <div key={d.name} className="flex items-center gap-3">
                      <div className="w-3 h-3 shrink-0" style={{ background: MONO_SHADES[i % MONO_SHADES.length] }} />
                      <span className="md-typescale-body-small flex-1" style={{ color: 'var(--muted)' }}>
                        {d.name}
                      </span>
                      <span className="md-typescale-label-large tabular-nums">
                        {d.value}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="flex-1 flex items-center justify-center">
                <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>No orders</span>
              </div>
            )}
          </div>
        </BentoCard>

        {/* ── Revenue Split (2×1) ─────────────────────────────────────── */}
        <BentoCard span={2} delay={480}>
          <div className="flex flex-col h-full">
            <div className="bento-card-header">
              <span className="bento-card-title">Revenue Split</span>
            </div>
            {kpi.totalRev > 0 ? (
              <ResponsiveContainer width="100%" height="100%" minHeight={140}>
                <BarChart data={revData} barSize={40}>
                  <XAxis
                    dataKey="gateway"
                    tick={{ fill: 'var(--muted)', fontSize: 12 }}
                    axisLine={false}
                    tickLine={false}
                  />
                  <YAxis
                    tick={{ fill: 'var(--muted)', fontSize: 11 }}
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(v: number) => v >= 1000000 ? `${(v / 1000000).toFixed(1)}M` : v >= 1000 ? `${(v / 1000).toFixed(0)}K` : String(v)}
                  />
                  <Tooltip
                    contentStyle={{
                      background: 'var(--surface)',
                      border: '1px solid var(--border)',
                      borderRadius: 0,
                      fontSize: 13,
                      color: 'var(--foreground)',
                    }}
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    formatter={(value: any) => [`${Number(value ?? 0).toLocaleString()}`, 'Revenue']}
                  />
                  <Bar dataKey="amount" radius={[0, 0, 0, 0]}>
                    {revData.map((_, i) => (
                      <Cell key={i} fill={MONO_SHADES[i]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex-1 flex items-center justify-center min-h-35">
                <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>No revenue data</span>
              </div>
            )}
          </div>
        </BentoCard>
      </BentoGrid>

      {/* ── Dispatch Command ────────────────────────────────────────────── */}
      <section className="mb-8">
        <Card className="p-0 overflow-hidden">
          <div className="flex flex-col md:flex-row items-stretch md:items-center">
            <div className="px-5 py-4 flex items-center gap-3 min-w-48 border-b md:border-b-0 md:border-r" style={{ borderColor: 'var(--border)' }}>
              <Send size={16} strokeWidth={1.75} style={{ color: 'var(--muted)' }} />
              <span className="md-typescale-label-large tabular-nums">{selectedOrders.size}</span>
              <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                / {pendingOrders.length}
              </span>
            </div>
            <div className="px-5 py-4 flex items-center gap-3 flex-1 border-b md:border-b-0 md:border-r" style={{ borderColor: 'var(--border)' }}>
              <select
                value={targetRoute}
                onChange={(e) => setTargetRoute(e.target.value)}
                className="md-select flex-1"
              >
                {drivers.length === 0 ? (
                  <option value="">No active drivers</option>
                ) : (
                  drivers.map((d) => (
                    <option key={d.driver_id} value={d.driver_id}>
                      {d.name} — {d.license_plate} [{d.truck_status}]
                    </option>
                  ))
                )}
              </select>
            </div>
            <div className="p-3">
              <Button
                onPress={() => setShowConfirm(true)}
                isDisabled={selectedOrders.size === 0 || isDispatching || !targetRoute}
                size="lg"
                className="w-full md:w-auto"
              >
                {isDispatching ? "Dispatching..." : "Dispatch"}
              </Button>
            </div>
          </div>
        </Card>

        {dispatchMsg && (
          <div
            className="rounded-xl px-5 py-3 mt-3 md-typescale-body-small"
            style={{
              background: dispatchMsg.ok ? 'var(--foreground)' : 'var(--danger)',
              color: dispatchMsg.ok ? 'var(--background)' : 'var(--danger-foreground)',
            }}
          >
            {dispatchMsg.text}
          </div>
        )}

        {manifests.length > 0 && (
          <div className="mt-3 flex flex-col gap-2">
            {manifests.map((mf) => (
              <Card key={mf.route_id} variant="secondary" className="flex-row items-center justify-between">
                <div>
                  <p className="md-typescale-label-large font-mono">{mf.driver_name}</p>
                  <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                    {mf.orders.length} stop{mf.orders.length !== 1 ? "s" : ""}
                  </p>
                </div>
                <Button variant="secondary" size="sm" isDisabled={!mf.loading_manifest?.length} onPress={() => setPrintManifest(mf)}>
                  Manifest
                </Button>
              </Card>
            ))}
          </div>
        )}
      </section>

      {/* ── Order View Controls ────────────────────────────────────────── */}
      <section className="mb-4">
        <div className="flex flex-col xl:flex-row gap-4 xl:items-center">
          <div className="flex flex-wrap p-1 gap-1 bg-surface-container rounded-[20px] shadow-sm">
            {[
              { id: "ALL", label: "All", count: filterCounts.all },
              { id: "PENDING", label: "Pending", count: filterCounts.pending },
              { id: "ACTIVE", label: "Active", count: filterCounts.active },
              { id: "COMPLETED", label: "Completed", count: filterCounts.completed },
              { id: "REVIEW", label: "Review", count: filterCounts.review },
            ].map((v) => (
              <button
                key={v.id}
                onClick={() => setOrderView(v.id as "ALL" | "PENDING" | "ACTIVE" | "COMPLETED" | "REVIEW")}
                className={`px-4 py-2 rounded-4xl md-typescale-label-medium transition-all duration-200 flex items-center gap-2 ${
                  orderView === v.id
                    ? "bg-accent text-accent-foreground shadow-md scale-105"
                    : "text-muted hover:bg-surface-container-high"
                }`}
              >
                {v.label}
                <span className={`px-1.5 py-0.5 rounded-full text-[10px] tabular-nums ${
                  orderView === v.id ? "bg-accent-foreground/20" : "bg-surface-container-highest"
                }`}>
                  {v.count}
                </span>
              </button>
            ))}
          </div>

          <div className="flex flex-col sm:flex-row gap-3 xl:ml-auto xl:min-w-140">
            <div className="relative flex-1 group">
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search orders..."
                className="w-full md-input-outlined rounded-[20px]! pl-10 h-10 transition-all focus:ring-2 focus:ring-accent/20"
                aria-label="Search orders"
              />
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 text-muted group-focus-within:text-accent transition-colors" size={16} />
            </div>
            
            <button
              onClick={() => setShowUrgentOnly(!showUrgentOnly)}
              className={`flex items-center gap-2 px-4 rounded-[20px] border transition-all h-10 ${
                showUrgentOnly 
                ? "border-danger bg-danger/5 text-danger" 
                : "border-border bg-surface text-muted hover:bg-surface-container"
              }`}
            >
              <div className={`w-1.5 h-1.5 rounded-full ${showUrgentOnly ? "bg-danger animate-pulse" : "bg-muted"}`} />
              <span className="md-typescale-label-small">Urgency</span>
            </button>

            <Button
              size="sm"
              variant="outline"
              className="rounded-[20px]! px-4"
              onPress={() => {
                setOrderView("ALL");
                setSearchTerm("");
                setShowUrgentOnly(false);
              }}
            >
              Clear
            </Button>
          </div>
        </div>
      </section>

      {/* ── Orders Table ────────────────────────────────────────────────── */}
      {filteredOrders.length === 0 ? (
        <div className="bento-card flex flex-col items-center justify-center py-24 rounded-[32px]!">
          <div className="w-16 h-16 rounded-full bg-surface-container flex items-center justify-center mb-4">
            <Search size={24} className="text-muted" />
          </div>
          <span className="md-typescale-body-medium text-muted">
            {orders.length === 0
              ? (isApiOnline ? "No active orders" : "Awaiting connection...")
              : "No orders match the current filters"}
          </span>
        </div>
      ) : (
        <div className="bento-card p-0 overflow-hidden rounded-[32px]! border-none! shadow-xl bg-surface">
          <div
            className="px-6 py-4 flex items-center justify-between bg-surface-container/30"
          >
            <div className="flex items-center gap-2">
              <span className="md-typescale-label-medium">Manifest Feed</span>
              <span className="md-typescale-label-small text-muted px-2 py-0.5 rounded-full bg-surface-container">
                {filteredOrders.length} records
              </span>
            </div>
            {selectedOrders.size > 0 && (
              <div className="flex items-center gap-2 px-3 py-1 rounded-full bg-accent text-accent-foreground md-typescale-label-small animate-in fade-in zoom-in duration-300">
                <Activity size={12} strokeWidth={3} />
                {selectedOrders.size} selected for dispatch
              </div>
            )}
          </div>
          <div className="overflow-x-auto">
            <table className="md-table border-none!">
              <thead className="bg-surface-container/30">
                <tr>
                  <th className="w-14 px-6 py-4">
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={toggleSelectAll}
                      disabled={pendingOrders.length === 0}
                      className="h-4 w-4 cursor-pointer rounded-xl transition-all accent-accent"
                      aria-label="Select all pending orders"
                    />
                  </th>
                  <th className="px-4 py-4 text-left md-typescale-label-medium text-muted uppercase tracking-wider">Order ID</th>
                  <th className="px-4 py-4 text-left md-typescale-label-medium text-muted uppercase tracking-wider">Retailer</th>
                  <th className="px-4 py-4 text-left md-typescale-label-medium text-muted uppercase tracking-wider">Amount</th>
                  <th className="px-4 py-4 text-left md-typescale-label-medium text-muted uppercase tracking-wider">Fleet Assignment</th>
                  <th className="px-4 py-4 text-right md-typescale-label-medium text-muted uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/20">
                {filteredOrders.map((order) => {
                  const isPending = order.state === "PENDING" || order.state === "PENDING_REVIEW";
                  const isSelected = selectedOrders.has(order.order_id);
                  const temporal = getTemporalStatus(order.deliver_before);

                  return (
                    <tr
                      key={order.order_id}
                      className={`group transition-all duration-200 ${isPending ? "cursor-pointer" : ""} ${isSelected ? "bg-accent/3" : "hover:bg-surface-container/20"}`}
                      onClick={() => isPending && toggleOrder(order.order_id)}
                    >
                      <td className="px-6 py-4">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          disabled={!isPending}
                          onChange={() => toggleOrder(order.order_id)}
                          onClick={(e) => e.stopPropagation()}
                          className={`h-4 w-4 cursor-pointer rounded-xl transition-all accent-accent ${isSelected ? "scale-110" : "scale-100 opacity-40 group-hover:opacity-100"}`}
                        />
                      </td>
                      <td className="px-4 py-4">
                        <span className="md-typescale-label-medium font-mono text-muted group-hover:text-foreground transition-colors">
                          #{order.order_id.slice(-8).toUpperCase()}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex flex-col">
                          <span className="md-typescale-body-medium font-medium group-hover:text-accent transition-colors">{order.retailer_id}</span>
                          <span className="md-typescale-label-small text-muted">Buston, UZ</span>
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <span className="md-typescale-label-large tabular-nums font-mono">
                          {order.amount?.toLocaleString() ?? "—"}<span className="text-[10px] ml-0.5 text-muted">UZS</span>
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 rounded-full ${order.route_id ? "bg-success" : "bg-muted/30"}`} />
                          <span className="md-typescale-label-medium font-mono text-muted">
                            {order.route_id ?? "NOT_ASSIGNED"}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex flex-col items-end gap-1.5">
                          <StatusChip status={order.state} />
                          {temporal.label && (
                            <div className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-danger/10 text-danger md-typescale-label-small animate-pulse font-medium">
                              <Activity size={10} />
                              {temporal.label.toUpperCase()}
                            </div>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ── Print Loading Manifest Modal ─────────────────────────────────── */}
      {printManifest && (
        <div className="md-dialog-scrim" onClick={() => setPrintManifest(null)}>
          <div className="md-dialog max-w-2xl" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-start justify-between mb-4">
              <div>
                <h3 className="md-dialog-title mb-0">Loading Manifest</h3>
                <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                  {printManifest.driver_name} · Route {printManifest.route_id}
                </p>
              </div>
              <button className="md-icon-btn" onClick={() => setPrintManifest(null)}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
              </button>
            </div>
            <div className="overflow-x-auto -mx-6">
              <table className="md-table">
                <thead>
                  <tr>
                    <th>Load #</th>
                    <th>Retailer</th>
                    <th>Order</th>
                    <th>Volume</th>
                    <th>Placement</th>
                  </tr>
                </thead>
                <tbody>
                  {printManifest.loading_manifest.map((entry) => (
                    <tr key={entry.load_sequence}>
                      <td className="font-mono font-bold">{entry.load_sequence}</td>
                      <td>{entry.retailer_name}</td>
                      <td className="font-mono text-xs">{entry.order_id}</td>
                      <td className="font-mono text-xs">{entry.volume_vu.toFixed(2)}</td>
                      <td>
                        <Chip color={entry.instruction === "By the Doors" ? "accent" : "default"} variant="soft" size="sm">
                          {entry.instruction}
                        </Chip>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="md-typescale-body-small mt-4" style={{ color: 'var(--muted)' }}>
              Load #1 = first loaded (deepest). Highest # = last loaded (by doors, first delivery).
            </p>
          </div>
        </div>
      )}

      {/* ── Dispatch Confirm Modal ───────────────────────────────────────── */}
      {showConfirm && (
        <div className="md-dialog-scrim" onClick={() => setShowConfirm(false)}>
          <div className="md-dialog max-w-md" onClick={(e) => e.stopPropagation()}>
            <h3 className="md-dialog-title">Confirm Dispatch</h3>
            <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
              Dispatch <strong>{selectedOrders.size}</strong> order{selectedOrders.size !== 1 ? "s" : ""} to:
            </p>
            <p className="font-mono font-medium mt-2">
              {drivers.find((d) => d.driver_id === targetRoute)?.name ?? targetRoute}
            </p>
            <div className="md-dialog-actions">
              <Button variant="outline" onPress={() => setShowConfirm(false)}>Cancel</Button>
              <Button variant="primary" onPress={() => { setShowConfirm(false); executeDispatch(); }}>Confirm</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
