"use client";

import { useEffect, useRef, useState, useCallback, useMemo, lazy, Suspense } from "react";
import { getAdminToken } from "@/lib/auth";
import { usePolling } from "@/lib/usePolling";
import { isTauri } from "@/lib/bridge";
import { readTokenFromCookie } from "@/lib/auth";
import { Card, Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard, BentoSkeleton } from "@/components/BentoGrid";
import CountUp from "@/components/CountUp";
import MiniSparkline from "@/components/MiniSparkline";
import {
  Truck, CheckCircle, CreditCard,
  ArrowUpRight, Activity, Send, Warehouse,
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
  const [isLoading, setIsLoading] = useState(true);

  const [selectedOrders, setSelectedOrders] = useState<Set<string>>(new Set());
  const [targetRoute, setTargetRoute] = useState("");
  const [isDispatching, setIsDispatching] = useState(false);
  const [dispatchMsg, setDispatchMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [manifests, setManifests] = useState<TruckManifest[]>([]);
  const [printManifest, setPrintManifest] = useState<TruckManifest | null>(null);

  // ── Polling ──────────────────────────────────────────────────────────────

  const fetchOrders = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const token = await getAdminToken();
        const [ordersRes, driversRes] = await Promise.all([
          fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/orders`, {
            headers: { Authorization: `Bearer ${token}` },
            signal,
          }),
          fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/fleet/drivers`, {
            headers: { Authorization: `Bearer ${token}` },
            signal,
          }),
        ]);
        if (ordersRes.ok) {
          const data = await ordersRes.json();
          setOrders(data ?? []);
          setIsApiOnline(true);
          setLastUpdated(new Date().toLocaleTimeString());
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
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    // Skip WS on desktop — fleet page already handles it via Tauri bridge
    if (isTauri()) return;

    let disposed = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let backoff = 1000;

    const connect = async () => {
      if (disposed) return;
      const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
      const wsBase = apiBase.replace(/^http/, "ws");
      let token = readTokenFromCookie();
      if (!token) {
        try { token = await getAdminToken(); } catch { return; }
      }
      if (disposed) return;

      const url = new URL("/ws/telemetry", wsBase);
      if (token) url.searchParams.set("token", token);

      const ws = new WebSocket(url.toString());
      wsRef.current = ws;

      ws.onopen = () => { backoff = 1000; };
      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const data = JSON.parse(event.data);
          if (data.type === "ORDER_STATE_CHANGED") {
            fetchOrdersRef.current();
          }
        } catch { /* ignore GPS pings and parse errors */ }
      };
      ws.onclose = () => {
        if (disposed) return;
        reconnectTimer = setTimeout(() => connect(), backoff);
        backoff = Math.min(backoff * 2, 30_000);
      };
      ws.onerror = () => { /* onclose will fire */ };
    };

    void connect();
    return () => {
      disposed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, []);

  // ── Selection ────────────────────────────────────────────────────────────

  const pendingOrders = useMemo(
    () => orders.filter((o) => (o.state === "PENDING" || o.state === "PENDING_REVIEW") && !o.route_id),
    [orders],
  );

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
      const token = await getAdminToken();
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/fleet/dispatch`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({ order_ids: Array.from(selectedOrders), route_id: targetRoute }),
      });
      if (res.ok) {
        const body = await res.json();
        setSelectedOrders(new Set());
        setDispatchMsg({ ok: true, text: `${body.message}` });
        setManifests(body.manifests ?? []);
        fetchOrders();
      } else {
        setDispatchMsg({ ok: false, text: `${await res.text()}` });
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
    const paymeRev = orders
      .filter((o) => o.state === "COMPLETED" && o.payment_gateway === "payme")
      .reduce((s, o) => s + (o.amount ?? 0), 0);
    const clickRev = orders
      .filter((o) => o.state === "COMPLETED" && o.payment_gateway === "click")
      .reduce((s, o) => s + (o.amount ?? 0), 0);
    return { activeTrucks, completed, inTransit, pending, totalRev, paymeRev, clickRev, total: orders.length };
  }, [orders]);

  // Pipeline data for donut chart
  const pipelineData = useMemo(() => [
    { name: 'Completed', value: kpi.completed },
    { name: 'In Transit', value: kpi.inTransit },
    { name: 'Pending', value: kpi.pending },
    { name: 'Other', value: Math.max(0, kpi.total - kpi.completed - kpi.inTransit - kpi.pending) },
  ].filter(d => d.value > 0), [kpi]);

  // Revenue split for bar chart
  const revData = useMemo(() => [
    { gateway: 'Payme', amount: kpi.paymeRev },
    { gateway: 'Click', amount: kpi.clickRev },
  ], [kpi]);

  // Fake sparkline data (deterministic from order counts)
  const truckSparkline = useMemo(() =>
    Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.activeTrucks + Math.sin(i * 0.8) * 2)),
    [kpi.activeTrucks]
  );
  const completedSparkline = useMemo(() =>
    Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.completed * 0.3 + i * (kpi.completed * 0.06))),
    [kpi.completed]
  );

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
      </header>

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
          <div className="md-kpi-card h-full">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Active Trucks</span>
              <Truck size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={kpi.activeTrucks} className="md-kpi-value" />
              <MiniSparkline data={truckSparkline} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">{kpi.total} total orders</span>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): Completed ─────────────────────────────── */}
        <BentoCard size="stat" delay={120}>
          <div className="md-kpi-card h-full">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Completed</span>
              <CheckCircle size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={kpi.completed} className="md-kpi-value" />
              <MiniSparkline data={completedSparkline} width={72} height={28} />
            </div>
            <div className="flex items-center gap-1.5">
              {kpi.total > 0 ? (
                <>
                  <ArrowUpRight size={14} strokeWidth={2} style={{ color: 'var(--success)' }} />
                  <span className="md-kpi-sub" style={{ color: 'var(--success)' }}>
                    {((kpi.completed / kpi.total) * 100).toFixed(0)}%
                  </span>
                </>
              ) : (
                <span className="md-kpi-sub">—</span>
              )}
            </div>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): In Transit ────────────────────────────── */}
        <BentoCard size="stat" delay={180}>
          <div className="md-kpi-card h-full">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">In Transit</span>
              <Activity size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <CountUp end={kpi.inTransit} className="md-kpi-value" />
            <span className="md-kpi-sub">{kpi.pending} pending</span>
          </div>
        </BentoCard>

        {/* ── STATISTICS (1×1): Revenue ───────────────────────────────── */}
        <BentoCard size="stat" delay={240}>
          <div className="md-kpi-card h-full">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Revenue</span>
              <CreditCard size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <CountUp end={kpi.totalRev} className="md-kpi-value" prefix="" suffix=" " />
            <span className="md-kpi-sub">Settled today</span>
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

      {/* ── Orders Table ────────────────────────────────────────────────── */}
      {orders.length === 0 ? (
        <div className="bento-card flex items-center justify-center py-16">
          <span className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
            {isApiOnline ? "No active orders" : "Awaiting connection..."}
          </span>
        </div>
      ) : (
        <div className="bento-card p-0 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="md-table">
              <thead>
                <tr>
                  <th className="w-12 px-4 py-3">
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={toggleSelectAll}
                      disabled={pendingOrders.length === 0}
                      className="h-4 w-4 cursor-pointer rounded disabled:opacity-30"
                      style={{ accentColor: 'var(--accent)' }}
                      aria-label="Select all pending orders"
                    />
                  </th>
                  <th>Order</th>
                  <th>Retailer</th>
                  <th>Amount</th>
                  <th>Route</th>
                  <th className="text-right">Status</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => {
                  const isPending = order.state === "PENDING" || order.state === "PENDING_REVIEW";
                  const isSelected = selectedOrders.has(order.order_id);
                  const temporal = getTemporalStatus(order.deliver_before);

                  return (
                    <tr
                      key={order.order_id}
                      className={`transition-colors duration-100 ${isPending ? "cursor-pointer" : ""}`}
                      style={{
                        background: isSelected
                          ? "var(--accent-soft)"
                          : temporal.label === "Overdue" || temporal.label === "Critical"
                          ? "var(--danger)"
                          : temporal.label === "Urgent"
                          ? "var(--default)"
                          : undefined,
                        borderLeft:
                          temporal.label === "Overdue" || temporal.label === "Critical"
                            ? "3px solid var(--danger)"
                            : temporal.label === "Urgent"
                            ? "3px solid var(--muted)"
                            : undefined,
                      }}
                      onClick={() => isPending && toggleOrder(order.order_id)}
                    >
                      <td className="px-4 py-3">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          disabled={!isPending}
                          onChange={() => toggleOrder(order.order_id)}
                          onClick={(e) => e.stopPropagation()}
                          className="h-4 w-4 cursor-pointer rounded disabled:opacity-20 disabled:cursor-not-allowed"
                          style={{ accentColor: 'var(--accent)' }}
                        />
                      </td>
                      <td className="font-mono text-xs">{order.order_id}</td>
                      <td>
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="md-typescale-body-small font-medium">{order.retailer_id}</span>
                          {order.order_source === "AI_GENERATED" && (
                            <Chip color="accent" variant="soft" size="sm">AI</Chip>
                          )}
                        </div>
                      </td>
                      <td className="font-mono text-xs tabular-nums">
                        {order.amount?.toLocaleString() ?? "—"}
                      </td>
                      <td className="font-mono text-xs">
                        {order.route_id ?? <span style={{ color: 'var(--muted)' }}>—</span>}
                      </td>
                      <td className="text-right">
                        <StatusChip status={order.state} />
                        {temporal.label && (
                          <div className="mt-1">
                            <Chip
                              color={temporal.label === "Overdue" || temporal.label === "Critical" ? "danger" : "warning"}
                              variant="soft"
                              size="sm"
                            >
                              {temporal.label}
                            </Chip>
                          </div>
                        )}
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
