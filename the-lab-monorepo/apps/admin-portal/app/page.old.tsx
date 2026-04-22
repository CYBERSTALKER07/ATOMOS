"use client";

import { useEffect, useState, useCallback } from "react";
import { getAdminToken } from "@/lib/auth";
import { usePolling } from "@/lib/usePolling";

// ─── Types ─────────────────────────────────────────────────────────────────

type Order = {
  order_id: string;
  retailer_id: string;
  state: string;
  amount?: number;
  payment_gateway?: string;
  route_id?: string | null;
  // AI Empathy Engine fields
  order_source?: string | null;
  auto_confirm_at?: string | null;
  // Temporal Matrix
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

// ─── Helpers ────────────────────────────────────────────────────────────────

const statusStyles: Record<string, { bg: string; color: string; label?: string }> = {
  PENDING:        { bg: 'var(--surface)', color: 'var(--foreground)' },
  EN_ROUTE:       { bg: 'var(--accent)',                   color: 'var(--accent-foreground)' },
  IN_TRANSIT:     { bg: 'var(--accent-soft)',       color: 'var(--accent-soft-foreground)' },
  COMPLETED:      { bg: 'var(--default)',        color: 'var(--default-foreground)' },
  PENDING_REVIEW: { bg: 'var(--accent-soft)',         color: 'var(--accent-soft-foreground)', label: 'Review' },
};

const getStatusBadge = (status: string | undefined) => {
  const key = status?.toUpperCase() ?? '';
  const s = statusStyles[key] ?? { bg: 'var(--danger)', color: 'var(--danger-foreground)' };
  return (
    <span className="md-chip md-typescale-label-small" style={{ background: s.bg, color: s.color, borderColor: 'transparent', cursor: 'default', height: 26 }}>
      {s.label ?? status ?? 'Unknown'}
    </span>
  );
};

// ─── Temporal Urgency ───────────────────────────────────────────────────────

const getTemporalStatus = (deliverBefore: string | null | undefined) => {
  if (!deliverBefore) return { classes: "", label: null };

  const delta = new Date(deliverBefore).getTime() - Date.now();

  if (delta <= 0) {
    return { classes: "animate-pulse", style: { background: 'var(--danger)', borderLeft: '4px solid var(--danger)' }, label: "Overdue" };
  }
  if (delta <= 60 * 60 * 1000) {
    return { classes: "animate-pulse", style: { background: 'var(--danger)', borderLeft: '4px solid var(--danger)' }, label: "Critical" };
  }
  if (delta <= 3 * 60 * 60 * 1000) {
    return { classes: "", style: { background: 'var(--default)', borderLeft: '4px solid var(--muted)' }, label: "Urgent" };
  }
  return { classes: "", style: {}, label: null };
};

// ─── Main Component ──────────────────────────────────────────────────────────

export default function AdminDashboard() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [drivers, setDrivers] = useState<FleetDriver[]>([]);
  const [isApiOnline, setIsApiOnline] = useState(false);
  const [lastUpdated, setLastUpdated] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  // Dispatch state
  const [selectedOrders, setSelectedOrders] = useState<Set<string>>(new Set());
  const [targetRoute, setTargetRoute] = useState<string>("");
  const [isDispatching, setIsDispatching] = useState(false);
  const [dispatchMsg, setDispatchMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [manifests, setManifests] = useState<TruckManifest[]>([]);
  const [printManifest, setPrintManifest] = useState<TruckManifest | null>(null);

  // ── Polling ──────────────────────────────────────────────────────────────
  const fetchOrders = useCallback(async (signal?: AbortSignal) => {
    try {
      const token = await getAdminToken();
      const [ordersRes, driversRes] = await Promise.all([
        fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/orders`, {
          headers: { Authorization: `Bearer ${token}` }, signal,
        }),
        fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/fleet/drivers`, {
          headers: { Authorization: `Bearer ${token}` }, signal,
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
        const activeDrivers = (driverData ?? []).filter(d => d.is_active);
        setDrivers(activeDrivers);
        if (!targetRoute && activeDrivers.length > 0) {
          setTargetRoute(activeDrivers[0].driver_id);
        }
      }
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      setIsApiOnline(false);
    } finally {
      setIsLoading(false);
    }
  }, [targetRoute]);

  usePolling((signal) => fetchOrders(signal), 5000, [fetchOrders]);

  // ── Select All ───────────────────────────────────────────────────────────
  // Selectable = PENDING or PENDING_REVIEW (Admin override clears the AI timer)
  const pendingOrders = orders.filter(o => o.state === "PENDING" || o.state === "PENDING_REVIEW");
  const allSelected = pendingOrders.length > 0 && pendingOrders.every(o => selectedOrders.has(o.order_id));

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedOrders(new Set());
    } else {
      setSelectedOrders(new Set(pendingOrders.map(o => o.order_id)));
    }
  };

  const toggleOrder = (id: string) => {
    setSelectedOrders(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  // ── Dispatch Execution ───────────────────────────────────────────────────
  const executeDispatch = async () => {
    if (selectedOrders.size === 0 || isDispatching) return;
    setIsDispatching(true);
    setDispatchMsg(null);
    setManifests([]);
    try {
      const token = await getAdminToken();
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/fleet/dispatch`, {
        method: "POST",
        headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          order_ids: Array.from(selectedOrders),
          route_id: targetRoute,
        }),
      });
      if (res.ok) {
        const body = await res.json();
        setSelectedOrders(new Set());
        setDispatchMsg({ ok: true, text: `[ OK ] ${body.message}` });
        setManifests(body.manifests ?? []);
        fetchOrders(); // immediate refresh
      } else {
        const err = await res.text();
        setDispatchMsg({ ok: false, text: `[ ERR ] ${err}` });
      }
    } catch (e: unknown) {
      setDispatchMsg({ ok: false, text: `[ ERR ] ${e instanceof Error ? e.message : "Network failure"}` });
    } finally {
      setIsDispatching(false);
    }
  };

  // ── Supplier name from cookie ──
  const [supplierName, setSupplierName] = useState('');
  useEffect(() => {
    const match = document.cookie.match(/(?:admin_name|supplier_name)=([^;]+)/);
    if (match) setSupplierName(decodeURIComponent(match[1]));
  }, []);

  // ── Render ───────────────────────────────────────────────────────────────
  const greeting = (() => {
    const h = new Date().getHours();
    if (h < 12) return 'Good morning';
    if (h < 18) return 'Good afternoon';
    return 'Good evening';
  })();

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>

      {/* ── Greeting Hero ─────────────────────────────────────────────── */}
      <header className="mb-10 flex items-end justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-large tracking-tight" style={{ color: 'var(--foreground)' }}>
            {greeting}{supplierName ? `, ${supplierName}` : ''}
          </h1>
          <p className="md-typescale-body-large mt-1" style={{ color: 'var(--muted)' }}>
            Here&apos;s your operations overview for today.
          </p>
        </div>
        <div className="flex items-center gap-3 shrink-0">
          <div className="md-chip" style={{ cursor: 'default' }}>
            <div className={`w-2 h-2 rounded-full ${isApiOnline ? 'animate-pulse' : ''}`} style={{ background: isApiOnline ? 'var(--success)' : 'var(--danger)' }} />
            <span className="md-typescale-label-small">{isApiOnline ? 'Live' : 'Offline'}</span>
          </div>
          <span className="md-typescale-label-small font-mono" style={{ color: 'var(--border)' }}>
            {lastUpdated || '--:--:--'}
          </span>
        </div>
      </header>

      {/* ── M3 Metrics Grid — Elevated Cards ─────────────────────────── */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-10">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="md-card md-card-elevated p-6 h-32 flex flex-col justify-between">
              <div className="w-1/2 h-3 rounded animate-pulse" style={{ background: 'var(--surface)' }} />
              <div className="w-1/3 h-8 rounded mt-4 animate-pulse" style={{ background: 'var(--surface)' }} />
            </div>
          ))
        ) : (
          <>
            {(() => {
              const activeTrucks = new Set(orders.filter(o => o.route_id).map(o => o.route_id)).size;
              const completedToday = orders.filter(o => o.state === 'COMPLETED').length;
              const paymeRevenue = orders
                .filter(o => o.state === 'COMPLETED' && o.payment_gateway === 'payme')
                .reduce((sum, o) => sum + (o.amount ?? 0), 0);
              const clickRevenue = orders
                .filter(o => o.state === 'COMPLETED' && o.payment_gateway === 'click')
                .reduce((sum, o) => sum + (o.amount ?? 0), 0);
              return [
                { label: "Active Trucks", value: String(activeTrucks), sub: `${orders.length} total orders` },
                { label: "Completed Today", value: String(completedToday), sub: `of ${orders.length} orders` },
                { label: "Payme Revenue", value: `${paymeRevenue.toLocaleString()}`, sub: "Settled" },
                { label: "Click Revenue", value: `${clickRevenue.toLocaleString()}`, sub: "Real-time" },
              ];
            })().map(({ label, value, sub }, i) => (
              <div
                key={i}
                className="md-card md-card-outlined p-6 flex flex-col justify-between cursor-default md-animate-in"
                style={{ animationDelay: `${i * 50}ms` }}
              >
                <p className="md-typescale-label-medium mb-4" style={{ color: 'var(--muted)' }}>{label}</p>
                <div>
                  <p className="md-typescale-headline-small tracking-tight mb-1" style={{ color: 'var(--foreground)', fontVariantNumeric: 'tabular-nums' }}>{value}</p>
                  <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>{sub}</p>
                </div>
              </div>
            ))}
          </>
        )}
      </div>

      {/* ── Dispatch Section ──────────────────────────────────────────── */}
      <main>
        <section>
          {/* Header */}
          <div className="mb-4">
            <h2 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>System Ledger</h2>
          </div>

          {/* ── M3 Command Bar — Surface Container ─────────────────── */}
          <div
            className="md-card md-card-elevated mb-0 flex flex-col md:flex-row items-stretch md:items-center overflow-hidden p-0"
          >
            {/* Selection counter */}
            <div className="px-6 py-4 flex items-center gap-3 min-w-[200px]" style={{ borderRight: '1px solid var(--border)' }}>
              <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Selected</span>
              <span className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>{selectedOrders.size}</span>
              <span className="md-typescale-label-small" style={{ color: 'var(--border)' }}>/ {pendingOrders.length} pending</span>
            </div>

            {/* Truck selector */}
            <div className="px-6 py-4 flex items-center gap-3 flex-1" style={{ borderRight: '1px solid var(--border)' }}>
              <label htmlFor="truck-select" className="md-typescale-label-small whitespace-nowrap" style={{ color: 'var(--muted)' }}>
                Target Vehicle
              </label>
              <select
                id="truck-select"
                value={targetRoute}
                onChange={e => setTargetRoute(e.target.value)}
                className="md-input-outlined flex-1 cursor-pointer"
                style={{ height: 40, padding: '0 12px' }}
              >
                {drivers.length === 0 ? (
                  <option value="">No active drivers</option>
                ) : (
                  drivers.map(d => (
                    <option key={d.driver_id} value={d.driver_id}>
                      {d.name} — {d.vehicle_type} ({d.license_plate}) [{d.truck_status}]
                    </option>
                  ))
                )}
              </select>
            </div>

            {/* M3 Filled Button — Dispatch */}
            <button
              onClick={() => setShowConfirm(true)}
              disabled={selectedOrders.size === 0 || isDispatching || !targetRoute}
              className="md-btn md-btn-filled md-typescale-label-large"
              style={{ borderRadius: '0 12px 12px 0', height: 'auto', padding: '16px 32px' }}
            >
              {isDispatching ? "Dispatching..." : "Dispatch Fleet"}
            </button>
          </div>

          {/* Dispatch feedback — M3 Snackbar style */}
          {dispatchMsg && (
            <div
              className="md-shape-xs px-6 py-3 mt-2 md-typescale-body-small"
              style={{
                background: dispatchMsg.ok ? 'var(--foreground)' : 'var(--danger)',
                color: dispatchMsg.ok ? 'var(--background)' : 'var(--danger-foreground)',
              }}
            >
              {dispatchMsg.text}
            </div>
          )}

          {/* ── Loading Manifest Cards ─────────────────────────────── */}
          {manifests.length > 0 && (
            <div className="mt-4 flex flex-col gap-3">
              {manifests.map((mf) => (
                <div
                  key={mf.route_id}
                  className="md-card md-card-elevated p-4 flex items-center justify-between"
                  style={{ background: 'var(--surface)' }}
                >
                  <div>
                    <p className="md-typescale-label-large font-mono" style={{ color: 'var(--foreground)' }}>{mf.driver_name}</p>
                    <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                      {mf.orders.length} stop{mf.orders.length !== 1 ? 's' : ''} · Route {mf.route_id}
                    </p>
                  </div>
                  <button
                    onClick={() => setPrintManifest(mf)}
                    disabled={!mf.loading_manifest?.length}
                    className="md-btn md-btn-tonal md-typescale-label-medium"
                  >
                    Print Loading Manifest
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* ── M3 Data Table ────────────────────────────────────────── */}
          <div className="w-full overflow-x-auto md-card md-card-outlined mt-4 p-0">
            <table className="md-table">
              <thead>
                <tr>
                  <th className="w-12" style={{ padding: '12px 16px' }}>
                    <input
                      type="checkbox"
                      checked={allSelected}
                      onChange={toggleSelectAll}
                      disabled={pendingOrders.length === 0}
                      className="w-4 h-4 cursor-pointer disabled:opacity-30"
                      style={{ accentColor: 'var(--accent)' }}
                      aria-label="Select all pending orders"
                    />
                  </th>
                  <th>Order ID</th>
                  <th>Retailer</th>
                  <th>Amount (Amount)</th>
                  <th>Route</th>
                  <th className="text-right">Status</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={`skel-${i}`}>
                      <td><div className="w-4 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                      <td><div className="w-24 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                      <td><div className="w-32 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                      <td><div className="w-20 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                      <td><div className="w-28 h-4 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                      <td className="flex justify-end"><div className="w-16 h-6 rounded animate-pulse" style={{ background: 'var(--surface)' }} /></td>
                    </tr>
                  ))
                ) : orders.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="p-12 text-center md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
                      {isApiOnline ? "No active orders found" : "Awaiting connection..."}
                    </td>
                  </tr>
                ) : (
                  orders.map((order) => {
                    const isPending = order.state === "PENDING" || order.state === "PENDING_REVIEW";
                    const isSelected = selectedOrders.has(order.order_id);
                    const temporal = getTemporalStatus(order.deliver_before);
                    return (
                      <tr
                        key={order.order_id}
                        className={`transition-colors duration-100 ${temporal.classes}`}
                        style={{
                          ...(isSelected
                            ? { background: 'var(--accent-soft)' }
                            : temporal.style || {}),
                          cursor: isPending ? 'pointer' : 'default',
                        }}
                        onClick={() => isPending && toggleOrder(order.order_id)}
                      >
                        <td>
                          <input
                            type="checkbox"
                            checked={isSelected}
                            disabled={!isPending}
                            onChange={() => toggleOrder(order.order_id)}
                            onClick={e => e.stopPropagation()}
                            className="w-4 h-4 cursor-pointer disabled:opacity-20 disabled:cursor-not-allowed"
                            style={{ accentColor: 'var(--accent)' }}
                          />
                        </td>
                        <td className="font-mono md-typescale-body-small">{order.order_id}</td>
                        <td>
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="md-typescale-body-medium font-medium">{order.retailer_id}</span>
                            {order.order_source === "AI_GENERATED" && (
                              <span className="md-chip md-typescale-label-small md-chip-selected" style={{ height: 22, padding: '0 8px' }}>
                                AI Pre-Order
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="font-mono md-typescale-body-small">{order.amount?.toLocaleString() ?? "—"}</td>
                        <td className="font-mono md-typescale-body-small">
                          {order.route_id
                            ? <span className="font-medium">{order.route_id}</span>
                            : <span style={{ color: 'var(--border)' }}>—</span>
                          }
                        </td>
                        <td className="text-right">
                          {getStatusBadge(order.state)}
                          {order.state === "PENDING_REVIEW" && order.auto_confirm_at && (
                            <div className="mt-1 md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                              Auto-seals: {new Date(order.auto_confirm_at).toLocaleTimeString()}
                            </div>
                          )}
                          {order.deliver_before && (
                            <div className="mt-1 md-typescale-label-small font-mono" style={{
                              color: temporal.label === "Critical" || temporal.label === "Overdue"
                                ? 'var(--danger)'
                                : temporal.label === "Urgent"
                                  ? 'var(--muted)'
                                  : 'var(--border)'
                            }}>
                              Deadline: {new Date(order.deliver_before).toLocaleTimeString()}
                              {temporal.label && ` · ${temporal.label}`}
                            </div>
                          )}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </section>
      </main>

      {/* ── Print Loading Manifest Modal ──────────────────────────────── */}
      {printManifest && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.6)' }}>
          <div
            className="md-card md-card-elevated p-8 max-w-2xl w-full mx-4 flex flex-col gap-4 max-h-[90vh] overflow-y-auto"
            style={{ background: 'var(--surface)' }}
          >
            <div className="flex justify-between items-start">
              <div>
                <h3 className="md-typescale-title-large" style={{ color: 'var(--foreground)' }}>Loading Manifest</h3>
                <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                  {printManifest.driver_name} · Route {printManifest.route_id}
                </p>
              </div>
              <button onClick={() => setPrintManifest(null)} className="md-btn md-btn-outlined md-typescale-label-large">Close</button>
            </div>
            <div className="overflow-x-auto">
              <table className="md-table w-full">
                <thead>
                  <tr>
                    <th style={{ padding: '10px 16px' }}>Load #</th>
                    <th>Retailer</th>
                    <th>Order ID</th>
                    <th>Volume (VU)</th>
                    <th>Placement</th>
                  </tr>
                </thead>
                <tbody>
                  {printManifest.loading_manifest.map((entry) => (
                    <tr key={entry.load_sequence}>
                      <td className="font-mono md-typescale-body-medium font-bold" style={{ padding: '10px 16px' }}>{entry.load_sequence}</td>
                      <td className="md-typescale-body-medium">{entry.retailer_name}</td>
                      <td className="font-mono md-typescale-body-small">{entry.order_id}</td>
                      <td className="font-mono md-typescale-body-small">{entry.volume_vu.toFixed(2)}</td>
                      <td>
                        <span
                          className="md-chip md-typescale-label-small"
                          style={{
                            background: entry.instruction === 'By the Doors'
                              ? 'var(--accent-soft)'
                              : 'var(--surface)',
                            color: entry.instruction === 'By the Doors'
                              ? 'var(--accent-soft-foreground)'
                              : 'var(--foreground)',
                            borderColor: 'transparent',
                            cursor: 'default',
                          }}
                        >
                          {entry.instruction}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>
              Load #1 = first loaded (deepest in truck, last delivery). Highest # = last loaded (by doors, first delivery).
            </p>
          </div>
        </div>
      )}

      {/* ── Dispatch Confirmation Modal ────────────────────────────────── */}
      {showConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.6)' }}>
          <div className="md-card md-card-elevated p-8 max-w-md w-full mx-4" style={{ background: 'var(--surface)' }}>
            <h3 className="md-typescale-title-large mb-4" style={{ color: 'var(--foreground)' }}>Confirm Dispatch</h3>
            <p className="md-typescale-body-medium mb-2" style={{ color: 'var(--muted)' }}>
              You are about to dispatch <strong>{selectedOrders.size}</strong> order{selectedOrders.size !== 1 ? "s" : ""} to:
            </p>
            <p className="md-typescale-body-medium font-medium mb-6 font-mono" style={{ color: 'var(--foreground)' }}>
              {drivers.find(d => d.driver_id === targetRoute)?.name ?? targetRoute}
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowConfirm(false)}
                className="md-btn md-btn-outlined md-typescale-label-large"
              >
                Cancel
              </button>
              <button
                onClick={() => { setShowConfirm(false); executeDispatch(); }}
                className="md-btn md-btn-filled md-typescale-label-large"
              >
                Confirm Dispatch
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
