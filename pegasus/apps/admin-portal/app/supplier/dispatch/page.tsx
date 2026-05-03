'use client';

import { useState, useCallback, useMemo } from 'react';
import { useToken } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
import StatusBadge from '@/components/StatusBadge';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import { Button } from '@heroui/react';
import {
  buildSupplierAutoDispatchIdempotencyKey,
  buildSupplierManualDispatchIdempotencyKey,
} from '../_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/* ─── Truck Color Palette for Manual Dispatch ─────────────── */

const TRUCK_COLORS = [
  { bg: 'var(--color-md-primary)', text: 'var(--color-md-on-primary)', label: 'Blue' },
  { bg: 'var(--color-md-tertiary)', text: 'var(--color-md-on-tertiary)', label: 'Teal' },
  { bg: '#34A853', text: '#fff', label: 'Green' },
  { bg: '#FF6D01', text: '#fff', label: 'Orange' },
  { bg: '#7B1FA2', text: '#fff', label: 'Purple' },
  { bg: '#E91E63', text: '#fff', label: 'Pink' },
  { bg: '#795548', text: '#fff', label: 'Brown' },
  { bg: '#607D8B', text: '#fff', label: 'Slate' },
];

/* ─── Types ───────────────────────────────────────────────── */

interface ManifestOrder {
  order_id: string;
  retailer_id: string;
  retailer_name: string;
  amount: number;
  volume_vu: number;
  lat: number;
  lng: number;
  force_assigned?: boolean;
}

interface LoadStep {
  load_sequence: number;
  order_id: string;
  retailer_name: string;
  volume_vu: number;
  lat: number;
  lng: number;
  instruction: string;
}

interface TruckManifest {
  route_id: string;
  driver_id: string;
  driver_name: string;
  vehicle_type: string;
  vehicle_class: string;
  max_volume_vu: number;
  used_volume_vu: number;
  orders: ManifestOrder[];
  loading_manifest: LoadStep[];
  geo_zone: string;
  force_assigned_count?: number;
  navigation_url?: string;
  navigation_segments?: string[];
  segment_count?: number;
}

interface OrphanOrder {
  order_id: string;
  retailer_name: string;
  reason: string;
}

interface TruckRecommendation {
  driver_id: string;
  driver_name: string;
  vehicle_id: string;
  vehicle_class: string;
  license_plate: string;
  max_volume_vu: number;
  used_volume_vu: number;
  free_volume_vu: number;
  distance_km: number;
  order_count: number;
  truck_status: string;
  score: number;
  recommendation: string;
}

interface DispatchResult {
  snapshot_timestamp: string;
  manifests: TruckManifest[];
  orphans: OrphanOrder[];
}

interface WaitingRoomResponse {
  count: number;
  orders: { order_id: string; retailer_name: string; amount: number; created_at: string }[];
}

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

function shortId(id: string): string {
  return id.length > 12 ? id.slice(0, 12) + '…' : id;
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function DispatchPage() {
  const token = useToken();
  const { toast } = useToast();

  // Dispatch state
  const [dispatching, setDispatching] = useState(false);
  const [result, setResult] = useState<DispatchResult | null>(null);
  const [confirming, setConfirming] = useState(false);

  // Waiting room state
  const [waitingRoom, setWaitingRoom] = useState<WaitingRoomResponse | null>(null);
  const [waitingLoading, setWaitingLoading] = useState(true);

  // Expanded manifest details
  const [expandedManifest, setExpandedManifest] = useState<string | null>(null);

  // Excluded trucks
  const [excludedTrucks, setExcludedTrucks] = useState<Set<string>>(new Set());

  // Re-dispatch modal state
  const [reDispatchOrderId, setReDispatchOrderId] = useState<string | null>(null);
  const [reDispatchRecs, setReDispatchRecs] = useState<TruckRecommendation[]>([]);
  const [reDispatchRetailer, setReDispatchRetailer] = useState('');
  const [reDispatchVolume, setReDispatchVolume] = useState(0);
  const [recsLoading, setRecsLoading] = useState(false);
  const [reassigning, setReassigning] = useState(false);

  // Manual dispatch mode
  const [mode, setMode] = useState<'auto' | 'manual'>('auto');
  const [recommendations, setRecommendations] = useState<DispatchResult | null>(null);
  const [recommendLoading, setRecommendLoading] = useState(false);
  const [manualDispatching, setManualDispatching] = useState<string | null>(null);
  const [dispatchedDrivers, setDispatchedDrivers] = useState<Set<string>>(new Set());
  // Admin overrides: driver_id → order_ids
  const [orderAssignments, setOrderAssignments] = useState<Map<string, Set<string>>>(new Map());

  /* ─── Waiting Room Polling ──────────────────────────────── */

  usePolling(
    async (signal) => {
      if (!token) return;
      try {
        const ts = result?.snapshot_timestamp || '';
        const q = ts ? `?after=${encodeURIComponent(ts)}` : '';
        const res = await fetch(`${API}/v1/supplier/manifests/waiting-room${q}`, {
          headers: { Authorization: `Bearer ${token}` },
          signal,
        });
        if (res.ok) {
          setWaitingRoom(await res.json());
        }
      } catch {
        // ignore poll errors
      } finally {
        setWaitingLoading(false);
      }
    },
    15_000,
    [token, result?.snapshot_timestamp],
  );

  /* ─── Auto-Dispatch ─────────────────────────────────────── */

  const runAutoDispatch = useCallback(async () => {
    if (!token) return;
    setDispatching(true);
    try {
      const body: Record<string, unknown> = {};
      const excludedTruckIds = [...excludedTrucks];
      if (excludedTruckIds.length > 0) {
        body.excluded_truck_ids = excludedTruckIds;
      }
      const res = await fetch(`${API}/v1/supplier/manifests/auto-dispatch`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierAutoDispatchIdempotencyKey(excludedTruckIds),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Auto-dispatch failed');
      }
      const data: DispatchResult = await res.json();
      setResult(data);
      setExpandedManifest(null);
      toast(
        `Dispatch complete: ${data.manifests.length} truck(s), ${data.orphans.length} orphan(s)`,
        data.orphans.length > 0 ? 'warning' : 'success',
      );
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setDispatching(false);
    }
  }, [token, excludedTrucks, toast]);

  /* ─── Confirm & Fleet-Dispatch ──────────────────────────── */

  const confirmDispatch = useCallback(async () => {
    if (!token || !result) return;
    setConfirming(true);
    let ok = 0;
    let fail = 0;
    for (const m of result.manifests) {
      try {
        const res = await fetch(`${API}/v1/fleet/dispatch`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ order_ids: m.orders.map((o) => o.order_id), route_id: m.route_id }),
        });
        if (res.ok) ok++;
        else fail++;
      } catch {
        fail++;
      }
    }
    setConfirming(false);
    if (fail === 0) {
      toast(`Dispatched ${ok} truck(s) successfully`, 'success');
      setResult(null);
    } else {
      toast(`${ok} succeeded, ${fail} failed — check fleet status`, 'error');
    }
  }, [token, result, toast]);

  /* ─── Re-Dispatch Handlers ──────────────────────────────── */

  const openReDispatch = useCallback(async (orderId: string) => {
    if (!token) return;
    setReDispatchOrderId(orderId);
    setReDispatchRecs([]);
    setReDispatchRetailer('');
    setReDispatchVolume(0);
    setRecsLoading(true);
    try {
      const res = await fetch(`${API}/v1/payloader/recommend-reassign`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_id: orderId }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setReDispatchRecs(data.recommendations ?? []);
      setReDispatchRetailer(data.retailer_name ?? '');
      setReDispatchVolume(data.order_volume_vu ?? 0);
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setRecsLoading(false);
    }
  }, [token, toast]);

  const handleReassign = useCallback(async (newDriverId: string, _newVehicleId: string) => {
    if (!token || !reDispatchOrderId) return;
    setReassigning(true);
    try {
      // RouteId == DriverId in this codebase; vehicle is bound to the driver.
      const res = await fetch(`${API}/v1/fleet/reassign`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_ids: [reDispatchOrderId], new_route_id: newDriverId }),
      });
      if (!res.ok) throw new Error(await res.text() || `HTTP ${res.status}`);
      const data: { conflicts?: Array<{ order_id: string; reason: string }> } = await res.json().catch(() => ({}));
      if (data.conflicts && data.conflicts.length > 0) {
        toast(data.conflicts.map(c => `${c.order_id.slice(0, 8)}: ${c.reason}`).join('; '), 'error');
      } else {
        toast('Order reassigned successfully', 'success');
      }
      setReDispatchOrderId(null);
      runAutoDispatch();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setReassigning(false);
    }
  }, [token, reDispatchOrderId, toast, runAutoDispatch]);

  /* ─── Manual Dispatch: Get Recommendations ──────────────── */

  const getRecommendations = useCallback(async () => {
    if (!token) return;
    setRecommendLoading(true);
    try {
      const body: Record<string, unknown> = {};
      if (excludedTrucks.size > 0) body.excluded_truck_ids = [...excludedTrucks];
      const res = await fetch(`${API}/v1/supplier/manifests/dispatch-recommend`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text() || 'Failed to get recommendations');
      const data: DispatchResult = await res.json();
      setRecommendations(data);
      const assignments = new Map<string, Set<string>>();
      for (const m of data.manifests) {
        assignments.set(m.driver_id, new Set(m.orders.map((o) => o.order_id)));
      }
      setOrderAssignments(assignments);
      setDispatchedDrivers(new Set());
      toast(
        `${data.manifests.length} truck(s) recommended, ${data.orphans.length} orphan(s)`,
        data.orphans.length > 0 ? 'warning' : 'success',
      );
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setRecommendLoading(false);
    }
  }, [token, excludedTrucks, toast]);

  /* ─── Manual Dispatch: Dispatch One Truck Group ─────────── */

  const manualDispatchGroup = useCallback(async (driverId: string) => {
    if (!token) return;
    const orderIds = orderAssignments.get(driverId);
    if (!orderIds || orderIds.size === 0) {
      toast('No orders in this group', 'warning');
      return;
    }
    setManualDispatching(driverId);
    try {
      const res = await fetch(`${API}/v1/supplier/manifests/manual-dispatch`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierManualDispatchIdempotencyKey(driverId, [...orderIds]),
        },
        body: JSON.stringify({ driver_id: driverId, order_ids: [...orderIds] }),
      });
      if (!res.ok) throw new Error(await res.text() || 'Manual dispatch failed');
      const manifest: TruckManifest = await res.json();

      const confirmRes = await fetch(`${API}/v1/fleet/dispatch`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_ids: [...orderIds], route_id: manifest.route_id }),
      });
      if (!confirmRes.ok) throw new Error('Fleet dispatch confirmation failed');

      setDispatchedDrivers((prev) => new Set([...prev, driverId]));
      const driverName = recommendations?.manifests.find((m) => m.driver_id === driverId)?.driver_name ?? driverId;
      toast(`Dispatched ${orderIds.size} order(s) to ${driverName}`, 'success');
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setManualDispatching(null);
    }
  }, [token, orderAssignments, recommendations, toast]);

  /* ─── Manual Dispatch: Move Order Between Trucks ────────── */

  const moveOrder = useCallback((orderId: string, fromDriverId: string, toDriverId: string) => {
    setOrderAssignments((prev) => {
      const next = new Map(prev);
      const fromSet = new Set(next.get(fromDriverId) ?? []);
      fromSet.delete(orderId);
      next.set(fromDriverId, fromSet);
      const toSet = new Set(next.get(toDriverId) ?? []);
      toSet.add(orderId);
      next.set(toDriverId, toSet);
      return next;
    });
  }, []);

  /* ─── Manual Dispatch: Truck Color Map ──────────────────── */

  const truckColorMap = useMemo(() => {
    if (!recommendations) return new Map<string, number>();
    const map = new Map<string, number>();
    recommendations.manifests.forEach((m, i) => {
      map.set(m.driver_id, i % TRUCK_COLORS.length);
    });
    return map;
  }, [recommendations]);

  /* ─── Manual Dispatch: Flat order list with truck badges ── */

  const allOrdersFlat = useMemo(() => {
    if (!recommendations) return [];
    const orders: Array<ManifestOrder & { recommended_driver_id: string; recommended_driver_name: string }> = [];
    for (const m of recommendations.manifests) {
      for (const o of m.orders) {
        orders.push({ ...o, recommended_driver_id: m.driver_id, recommended_driver_name: m.driver_name });
      }
    }
    return orders;
  }, [recommendations]);

  /* ─── Utilization Bar ───────────────────────────────────── */

  const UtilBar = ({ used, max }: { used: number; max: number }) => {
    const pct = max > 0 ? Math.min((used / max) * 100, 100) : 0;
    const color = pct > 90 ? 'var(--color-md-error)' : pct > 70 ? 'var(--color-md-warning)' : 'var(--color-md-primary)';
    return (
      <div className="flex items-center gap-2 text-xs">
        <div className="flex-1 h-2 rounded-full" style={{ background: 'var(--color-md-surface-container)' }}>
          <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, background: color }} />
        </div>
        <span className="tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          {used.toFixed(1)}/{max.toFixed(0)} VU
        </span>
      </div>
    );
  };

  /* ─── Render ────────────────────────────────────────────── */

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Dispatch Control
          </h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            {mode === 'auto'
              ? 'Auto-assign pending orders to available trucks via spatial clustering and bin-packing.'
              : 'Get AI recommendations, review and edit truck assignments, then dispatch one truck at a time.'}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {/* Mode toggle */}
          <div
            className="flex rounded-lg overflow-hidden border"
            style={{ borderColor: 'var(--color-md-outline-variant)' }}
          >
            <button
              type="button"
              onClick={() => setMode('auto')}
              className="px-3 py-1.5 text-xs font-medium transition-colors"
              style={{
                background: mode === 'auto' ? 'var(--color-md-primary)' : 'var(--color-md-surface)',
                color: mode === 'auto' ? 'var(--color-md-on-primary)' : 'var(--color-md-on-surface-variant)',
              }}
            >
              Auto
            </button>
            <button
              type="button"
              onClick={() => setMode('manual')}
              className="px-3 py-1.5 text-xs font-medium transition-colors"
              style={{
                background: mode === 'manual' ? 'var(--color-md-primary)' : 'var(--color-md-surface)',
                color: mode === 'manual' ? 'var(--color-md-on-primary)' : 'var(--color-md-on-surface-variant)',
              }}
            >
              Manual
            </button>
          </div>

          {/* Waiting room badge */}
          {!waitingLoading && waitingRoom && waitingRoom.count > 0 && (
            <div
              className="md-shape-full px-3 py-1.5 flex items-center gap-2 text-xs font-medium"
              style={{ background: 'var(--color-md-tertiary-container)', color: 'var(--color-md-on-tertiary-container)' }}
            >
              <Icon name="orders" size={14} />
              {waitingRoom.count} new since last dispatch
            </div>
          )}

          {mode === 'auto' ? (
            <Button
              variant="primary"
              isDisabled={dispatching}
              onPress={runAutoDispatch}
              className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5"
            >
              <Icon name="dispatch" size={18} className="mr-1.5" />
              {dispatching ? 'Dispatching…' : 'Run Auto-Dispatch'}
            </Button>
          ) : (
            <Button
              variant="primary"
              isDisabled={recommendLoading}
              onPress={getRecommendations}
              className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5"
            >
              <Icon name="dispatch" size={18} className="mr-1.5" />
              {recommendLoading ? 'Analyzing…' : 'Get Recommendations'}
            </Button>
          )}
        </div>
      </div>

      {/* Waiting Room Section */}
      {waitingRoom && waitingRoom.count > 0 && (
        <div className="md-card md-elevation-0 md-shape-md p-4" style={{ background: 'var(--color-md-tertiary-container)', color: 'var(--color-md-on-tertiary-container)' }}>
          <div className="flex items-center gap-2 mb-3">
            <Icon name="warning" size={18} />
            <span className="md-typescale-title-small">
              Waiting Room — {waitingRoom.count} order(s) pending dispatch
            </span>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {waitingRoom.orders.slice(0, 9).map((o) => (
              <div
                key={o.order_id}
                className="flex items-center justify-between px-3 py-2 rounded-lg text-xs"
                style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)' }}
              >
                <div>
                  <span className="font-medium">{o.retailer_name}</span>
                  <span className="ml-2" style={{ color: 'var(--color-md-on-surface-variant)' }}>{shortId(o.order_id)}</span>
                </div>
                <span className="tabular-nums font-medium">{formatAmount(o.amount)}</span>
              </div>
            ))}
            {waitingRoom.count > 9 && (
              <div className="flex items-center justify-center text-xs font-medium px-3 py-2">
                +{waitingRoom.count - 9} more
              </div>
            )}
          </div>
        </div>
      )}

      {/* ══════════════ AUTO MODE ══════════════ */}

      {/* No result yet */}
      {mode === 'auto' && !result && !dispatching && (
        <EmptyState
          icon="dispatch"
          headline="No dispatch result"
          body="Run auto-dispatch to generate optimized truck manifests from pending orders."
        />
      )}

      {/* Dispatching skeleton */}
      {mode === 'auto' && dispatching && (
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-xl" />
          ))}
        </div>
      )}

      {/* Dispatch Result */}
      {mode === 'auto' && result && !dispatching && (
        <>
          {/* Summary bar */}
          <div
            className="md-card md-elevation-1 md-shape-md p-4 flex items-center justify-between"
            style={{ background: 'var(--color-md-surface-container)' }}
          >
            <div className="flex gap-8">
              <div>
                <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Trucks
                </span>
                <span className="md-typescale-headline-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                  {result.manifests.length}
                </span>
              </div>
              <div>
                <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Orders Assigned
                </span>
                <span className="md-typescale-headline-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                  {result.manifests.reduce((a, m) => a + m.orders.length, 0)}
                </span>
              </div>
              <div>
                <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Orphans
                </span>
                <span
                  className="md-typescale-headline-small tabular-nums"
                  style={{ color: result.orphans.length > 0 ? 'var(--color-md-error)' : 'var(--color-md-on-surface)' }}
                >
                  {result.orphans.length}
                </span>
              </div>
              <div>
                <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Snapshot
                </span>
                <span className="md-typescale-body-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                  {new Date(result.snapshot_timestamp).toLocaleTimeString()}
                </span>
              </div>
            </div>
            <Button
              variant="primary"
              isDisabled={confirming}
              onPress={confirmDispatch}
              className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5"
            >
              {confirming ? 'Dispatching…' : 'Confirm & Dispatch All'}
            </Button>
          </div>

          {/* Truck Manifests */}
          <div className="flex flex-col gap-4">
            {result.manifests.map((m) => {
              const expanded = expandedManifest === m.route_id;
              return (
                <div
                  key={m.route_id}
                  className="md-card md-elevation-1 md-shape-md overflow-hidden"
                  style={{ background: 'var(--color-md-surface-container-low)' }}
                >
                  {/* Truck header */}
                  <button
                    type="button"
                    onClick={() => setExpandedManifest(expanded ? null : m.route_id)}
                    className="w-full flex items-center justify-between p-4 text-left hover:opacity-80 transition-opacity"
                  >
                    <div className="flex items-center gap-4">
                      <div
                        className="w-10 h-10 rounded-full flex items-center justify-center"
                        style={{ background: 'var(--color-md-primary-container)', color: 'var(--color-md-on-primary-container)' }}
                      >
                        <Icon name="dispatch" size={20} />
                      </div>
                      <div>
                        <div className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
                          {m.driver_name}
                        </div>
                        <div className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          {m.vehicle_class} — {m.geo_zone || 'All zones'}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          Orders
                        </span>
                        <span className="md-typescale-title-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                          {m.orders.length}
                          {(m.force_assigned_count ?? 0) > 0 && (
                            <span
                              className="ml-1 md-typescale-label-small px-1.5 py-0.5"
                              style={{
                                borderRadius: '99px',
                                background: 'color-mix(in srgb, var(--color-md-tertiary) 15%, transparent)',
                                color: 'var(--color-md-tertiary)',
                              }}
                              title={`${m.force_assigned_count} order(s) force-assigned beyond normal radius`}
                            >
                              {m.force_assigned_count} forced
                            </span>
                          )}
                        </span>
                      </div>
                      <div className="w-40">
                        <UtilBar used={m.used_volume_vu} max={m.max_volume_vu} />
                      </div>
                      <Icon
                        name={expanded ? 'left' : 'right'}
                        size={18}
                        className="transition-transform"
                      />
                    </div>
                  </button>

                  {/* Expanded: Loading Manifest */}
                  {expanded && (
                    <div className="border-t px-4 pb-4" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                      <div className="mt-4 mb-2">
                        <span className="md-typescale-label-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          Loading Sequence
                        </span>
                      </div>
                      <table className="w-full text-xs">
                        <thead>
                          <tr style={{ color: 'var(--color-md-on-surface-variant)' }}>
                            <th className="text-left py-1.5 font-medium">#</th>
                            <th className="text-left py-1.5 font-medium">Order</th>
                            <th className="text-left py-1.5 font-medium">Retailer</th>
                            <th className="text-right py-1.5 font-medium">Volume</th>
                            <th className="text-left py-1.5 font-medium">Instruction</th>
                          </tr>
                        </thead>
                        <tbody>
                          {m.loading_manifest.map((step) => (
                            <tr
                              key={step.load_sequence}
                              className="border-t"
                              style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface)' }}
                            >
                              <td className="py-2 tabular-nums font-medium">{step.load_sequence}</td>
                              <td className="py-2 font-mono">{shortId(step.order_id)}</td>
                              <td className="py-2">{step.retailer_name}</td>
                              <td className="py-2 text-right tabular-nums">{step.volume_vu.toFixed(1)} VU</td>
                              <td className="py-2" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                                {step.instruction || '—'}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>

                      {/* Orders summary */}
                      <div className="mt-4 mb-2">
                        <span className="md-typescale-label-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          Assigned Orders
                        </span>
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                        {m.orders.map((o) => (
                          <div
                            key={o.order_id}
                            className="flex items-center justify-between px-3 py-2 rounded-lg"
                            style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)' }}
                          >
                            <div>
                              <span className="font-medium text-xs">{o.retailer_name}</span>
                              <span className="ml-2 text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                                {shortId(o.order_id)}
                              </span>
                              {o.force_assigned && (
                                <span className="ml-1 text-[10px] px-1 py-0.5 rounded" style={{ background: 'color-mix(in srgb, var(--color-md-tertiary) 15%, transparent)', color: 'var(--color-md-tertiary)' }}>
                                  forced
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="tabular-nums font-medium text-xs">{formatAmount(o.amount)}</span>
                              <button
                                onClick={(e) => { e.stopPropagation(); openReDispatch(o.order_id); }}
                                className="p-1 rounded hover:opacity-80"
                                style={{ color: 'var(--color-md-on-surface-variant)' }}
                                title="Re-dispatch to another truck"
                              >
                                <Icon name="dispatch" size={14} />
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>

                      {/* Navigation Segments */}
                      {m.navigation_segments && m.navigation_segments.length > 0 && (
                        <div className="mt-4">
                          <div className="flex items-center gap-2 mb-2">
                            <Icon name="gps" size={16} className="text-muted" />
                            <span className="md-typescale-label-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                              Navigation — {m.segment_count ?? m.navigation_segments.length} segment{(m.segment_count ?? m.navigation_segments.length) !== 1 ? 's' : ''}
                            </span>
                          </div>
                          <div className="flex flex-wrap gap-2">
                            {m.navigation_segments.map((url, idx) => {
                              const segStops = m.orders.length > 25 ? 20 : m.orders.length;
                              const from = idx * segStops + 1;
                              const to = Math.min((idx + 1) * segStops, m.orders.length);
                              return (
                                <a
                                  key={idx}
                                  href={url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors"
                                  style={{
                                    background: 'var(--color-md-primary-container)',
                                    color: 'var(--color-md-on-primary-container)',
                                    textDecoration: 'none',
                                  }}
                                >
                                  <Icon name="gps" size={12} />
                                  Segment {idx + 1} (stops {from}–{to})
                                </a>
                              );
                            })}
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {/* Orphan Orders */}
          {result.orphans.length > 0 && (
            <div className="md-card md-elevation-1 md-shape-md p-4" style={{ background: 'var(--color-md-error-container)' }}>
              <div className="flex items-center gap-2 mb-3">
                <Icon name="warning" size={18} className="text-inherit" />
                <span
                  className="md-typescale-title-small"
                  style={{ color: 'var(--color-md-on-error-container)' }}
                >
                  Orphan Orders — {result.orphans.length} unassigned
                </span>
              </div>
              <table className="w-full text-xs" style={{ color: 'var(--color-md-on-error-container)' }}>
                <thead>
                  <tr>
                    <th className="text-left py-1.5 font-medium">Order ID</th>
                    <th className="text-left py-1.5 font-medium">Retailer</th>
                    <th className="text-left py-1.5 font-medium">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {result.orphans.map((o) => (
                    <tr key={o.order_id} className="border-t" style={{ borderColor: 'var(--color-md-on-error-container)', opacity: 0.2 }}>
                      <td className="py-2 font-mono">{shortId(o.order_id)}</td>
                      <td className="py-2">{o.retailer_name}</td>
                      <td className="py-2">{o.reason.replace(/_/g, ' ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ══════════════ MANUAL MODE ══════════════ */}

      {/* No recommendations yet */}
      {mode === 'manual' && !recommendations && !recommendLoading && (
        <EmptyState
          icon="dispatch"
          headline="Manual Dispatch"
          body="Click 'Get Recommendations' to see AI-powered truck assignment suggestions. You can then review, edit, and dispatch one truck at a time."
        />
      )}

      {/* Loading skeleton */}
      {mode === 'manual' && recommendLoading && (
        <div className="flex flex-col gap-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-xl" />
          ))}
        </div>
      )}

      {/* Recommendations */}
      {mode === 'manual' && recommendations && !recommendLoading && (
        <>
          {/* Truck Legend Bar */}
          <div
            className="md-card md-elevation-0 md-shape-md p-3 flex flex-wrap items-center gap-3"
            style={{ background: 'var(--color-md-surface-container)' }}
          >
            <span className="md-typescale-label-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Truck Legend:
            </span>
            {recommendations.manifests
              .filter((m) => !dispatchedDrivers.has(m.driver_id))
              .map((m) => {
                const colorIdx = truckColorMap.get(m.driver_id) ?? 0;
                const color = TRUCK_COLORS[colorIdx];
                const assignedCount = orderAssignments.get(m.driver_id)?.size ?? 0;
                return (
                  <div key={m.driver_id} className="flex items-center gap-1.5">
                    <div
                      className="w-3 h-3 rounded-full shrink-0"
                      style={{ background: color.bg }}
                    />
                    <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface)' }}>
                      {m.driver_name}
                    </span>
                    <span className="md-typescale-label-small tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                      ({assignedCount})
                    </span>
                  </div>
                );
              })}
            {recommendations.orphans.length > 0 && (
              <div className="flex items-center gap-1.5">
                <div className="w-3 h-3 rounded-full shrink-0" style={{ background: 'var(--color-md-error)' }} />
                <span className="md-typescale-label-small" style={{ color: 'var(--color-md-error)' }}>
                  {recommendations.orphans.length} orphan(s)
                </span>
              </div>
            )}
            <div className="ml-auto">
              <Button
                variant="outline"
                size="sm"
                isDisabled={recommendLoading}
                onPress={getRecommendations}
                className="md-btn md-btn-outlined md-typescale-label-small px-3 py-1"
              >
                Re-recommend Remaining
              </Button>
            </div>
          </div>

          {/* Flat Order List with Truck Badges */}
          <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface-container-low)' }}>
            <div className="px-4 py-3 border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
              <span className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
                All Orders — {allOrdersFlat.length} total
              </span>
            </div>
            <table className="w-full text-xs">
              <thead>
                <tr style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  <th className="text-left px-4 py-2 font-medium">Retailer</th>
                  <th className="text-left px-4 py-2 font-medium">Order ID</th>
                  <th className="text-right px-4 py-2 font-medium">Amount</th>
                  <th className="text-right px-4 py-2 font-medium">Volume</th>
                  <th className="text-center px-4 py-2 font-medium">Truck</th>
                </tr>
              </thead>
              <tbody>
                {allOrdersFlat.map((o) => {
                  // Find the current assignment (might differ from recommendation if admin moved it)
                  let assignedDriverId = o.recommended_driver_id;
                  let assignedDriverName = o.recommended_driver_name;
                  for (const [driverId, orderSet] of orderAssignments) {
                    if (orderSet.has(o.order_id)) {
                      assignedDriverId = driverId;
                      const rec = recommendations.manifests.find((m) => m.driver_id === driverId);
                      assignedDriverName = rec?.driver_name ?? driverId;
                      break;
                    }
                  }
                  const colorIdx = truckColorMap.get(assignedDriverId) ?? 0;
                  const color = TRUCK_COLORS[colorIdx];
                  const isDispatched = dispatchedDrivers.has(assignedDriverId);

                  if (isDispatched) return null;

                  return (
                    <tr
                      key={o.order_id}
                      className="border-t hover:opacity-90 transition-opacity"
                      style={{
                        borderColor: 'var(--color-md-outline-variant)',
                        color: 'var(--color-md-on-surface)',
                      }}
                    >
                      <td className="px-4 py-2.5 font-medium">{o.retailer_name}</td>
                      <td className="px-4 py-2.5 font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {shortId(o.order_id)}
                      </td>
                      <td className="px-4 py-2.5 text-right tabular-nums font-medium">{formatAmount(o.amount)}</td>
                      <td className="px-4 py-2.5 text-right tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {o.volume_vu.toFixed(1)} VU
                      </td>
                      <td className="px-4 py-2.5 text-center">
                        {/* Truck badge — dropdown to reassign */}
                        <div className="relative inline-block group">
                          <div
                            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold cursor-pointer"
                            style={{ background: color.bg, color: color.text }}
                            title={`Assigned to ${assignedDriverName} — click to change`}
                          >
                            <Icon name="dispatch" size={10} />
                            {assignedDriverName.split(' ')[0]}
                          </div>
                          {/* Reassign dropdown */}
                          <div
                            className="absolute right-0 top-full mt-1 z-10 hidden group-hover:block md-card md-elevation-3 md-shape-sm py-1"
                            style={{ background: 'var(--color-md-surface)', minWidth: 160 }}
                          >
                            {recommendations.manifests
                              .filter((m) => m.driver_id !== assignedDriverId && !dispatchedDrivers.has(m.driver_id))
                              .map((m) => {
                                const ci = truckColorMap.get(m.driver_id) ?? 0;
                                const c = TRUCK_COLORS[ci];
                                return (
                                  <button
                                    key={m.driver_id}
                                    type="button"
                                    onClick={() => moveOrder(o.order_id, assignedDriverId, m.driver_id)}
                                    className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-left hover:opacity-80"
                                    style={{ color: 'var(--color-md-on-surface)' }}
                                  >
                                    <div className="w-2.5 h-2.5 rounded-full shrink-0" style={{ background: c.bg }} />
                                    {m.driver_name}
                                  </button>
                                );
                              })}
                          </div>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Per-Truck Dispatch Cards */}
          <div className="flex flex-col gap-4">
            {recommendations.manifests
              .filter((m) => !dispatchedDrivers.has(m.driver_id))
              .map((m) => {
                const colorIdx = truckColorMap.get(m.driver_id) ?? 0;
                const color = TRUCK_COLORS[colorIdx];
                const assignedOrders = orderAssignments.get(m.driver_id);
                const orderCount = assignedOrders?.size ?? 0;
                const assignedVol = m.orders
                  .filter((o) => assignedOrders?.has(o.order_id))
                  .reduce((sum, o) => sum + o.volume_vu, 0);
                const isDispatching = manualDispatching === m.driver_id;

                return (
                  <div
                    key={m.driver_id}
                    className="md-card md-elevation-1 md-shape-md overflow-hidden flex"
                    style={{ background: 'var(--color-md-surface-container-low)' }}
                  >
                    {/* Color stripe */}
                    <div className="w-1.5 shrink-0" style={{ background: color.bg }} />

                    {/* Content */}
                    <div className="flex-1 p-4 flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div
                          className="w-10 h-10 rounded-full flex items-center justify-center"
                          style={{ background: color.bg, color: color.text }}
                        >
                          <Icon name="dispatch" size={20} />
                        </div>
                        <div>
                          <div className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
                            {m.driver_name}
                          </div>
                          <div className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                            {m.vehicle_class} — {m.geo_zone || 'All zones'}
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center gap-6">
                        <div className="text-right">
                          <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                            Orders
                          </span>
                          <span className="md-typescale-title-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                            {orderCount}
                          </span>
                        </div>
                        <div className="w-40">
                          <UtilBar used={assignedVol} max={m.max_volume_vu} />
                        </div>
                        <Button
                          variant="primary"
                          size="sm"
                          isDisabled={isDispatching || orderCount === 0}
                          onPress={() => manualDispatchGroup(m.driver_id)}
                          className="md-btn md-btn-filled md-typescale-label-medium px-4 py-2"
                        >
                          {isDispatching ? 'Dispatching…' : 'Dispatch'}
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
          </div>

          {/* Dispatched groups summary */}
          {dispatchedDrivers.size > 0 && (
            <div
              className="md-card md-elevation-0 md-shape-md p-3 flex items-center gap-3"
              style={{ background: 'color-mix(in srgb, var(--color-md-success) 10%, transparent)' }}
            >
              <Icon name="orders" size={16} style={{ color: 'var(--color-md-success)' }} />
              <span className="md-typescale-body-small" style={{ color: 'var(--color-md-success)' }}>
                {dispatchedDrivers.size} truck(s) dispatched
              </span>
            </div>
          )}

          {/* Orphans */}
          {recommendations.orphans.length > 0 && (
            <div className="md-card md-elevation-1 md-shape-md p-4" style={{ background: 'var(--color-md-error-container)' }}>
              <div className="flex items-center gap-2 mb-3">
                <Icon name="warning" size={18} />
                <span className="md-typescale-title-small" style={{ color: 'var(--color-md-on-error-container)' }}>
                  Orphan Orders — {recommendations.orphans.length} unassigned
                </span>
              </div>
              <table className="w-full text-xs" style={{ color: 'var(--color-md-on-error-container)' }}>
                <thead>
                  <tr>
                    <th className="text-left py-1.5 font-medium">Order ID</th>
                    <th className="text-left py-1.5 font-medium">Retailer</th>
                    <th className="text-left py-1.5 font-medium">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {recommendations.orphans.map((o) => (
                    <tr key={o.order_id} className="border-t" style={{ borderColor: 'var(--color-md-on-error-container)', opacity: 0.2 }}>
                      <td className="py-2 font-mono">{shortId(o.order_id)}</td>
                      <td className="py-2">{o.retailer_name}</td>
                      <td className="py-2">{o.reason.replace(/_/g, ' ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* ── Re-Dispatch Modal ──────────────────────────────────────────── */}
      {reDispatchOrderId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div
            className="md-card md-elevation-3 md-shape-lg"
            style={{ background: 'var(--color-md-surface)', width: 540, maxHeight: '80vh', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}
          >
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
              <div>
                <h3 className="md-typescale-title-medium" style={{ color: 'var(--color-md-on-surface)' }}>
                  Re-Dispatch Order
                </h3>
                <p className="md-typescale-body-small mt-1 font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  {shortId(reDispatchOrderId)}
                  {reDispatchRetailer && ` · ${reDispatchRetailer}`}
                  {reDispatchVolume > 0 && ` · ${reDispatchVolume.toFixed(1)} VU`}
                </p>
              </div>
              <button
                onClick={() => setReDispatchOrderId(null)}
                className="p-2 rounded-full hover:opacity-70"
                style={{ color: 'var(--color-md-on-surface-variant)' }}
              >
                <Icon name="left" size={18} />
              </button>
            </div>

            {/* Recommendations */}
            <div className="flex-1 overflow-y-auto">
              {recsLoading ? (
                <div className="flex items-center justify-center py-12">
                  <span className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    Analyzing fleet positions…
                  </span>
                </div>
              ) : reDispatchRecs.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12">
                  <Icon name="warning" size={28} className="mb-2" />
                  <span className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    No available trucks found
                  </span>
                </div>
              ) : (
                reDispatchRecs.map((rec, idx) => {
                  const isBest = idx === 0;
                  const fits = rec.free_volume_vu >= reDispatchVolume;
                  const isMaintenance = rec.truck_status === 'MAINTENANCE';
                  return (
                    <button
                      key={rec.driver_id}
                      onClick={() => !isMaintenance && !reassigning && handleReassign(rec.driver_id, rec.vehicle_id)}
                      disabled={isMaintenance || reassigning}
                      className="w-full flex items-center gap-4 px-6 py-4 border-b text-left hover:opacity-90 transition-opacity disabled:opacity-40"
                      style={{
                        borderColor: 'var(--color-md-outline-variant)',
                        background: isBest ? 'color-mix(in srgb, var(--color-md-primary) 5%, transparent)' : 'transparent',
                      }}
                    >
                      {/* Rank */}
                      <div
                        className="w-7 h-7 rounded-full flex items-center justify-center shrink-0 text-xs font-bold"
                        style={{
                          background: isBest ? 'var(--color-md-primary)' : 'var(--color-md-surface-container)',
                          color: isBest ? 'var(--color-md-on-primary)' : 'var(--color-md-on-surface-variant)',
                        }}
                      >
                        {idx + 1}
                      </div>

                      {/* Info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="md-typescale-label-large" style={{ color: 'var(--color-md-on-surface)' }}>
                            {rec.driver_name}
                          </span>
                          {isBest && (
                            <span
                              className="md-typescale-label-small px-1.5 py-0.5 rounded"
                              style={{ background: 'var(--color-md-primary)', color: 'var(--color-md-on-primary)' }}
                            >
                              Best
                            </span>
                          )}
                          {isMaintenance && (
                            <span
                              className="md-typescale-label-small px-1.5 py-0.5 rounded"
                              style={{ background: 'var(--color-md-error)', color: 'var(--color-md-on-error)' }}
                            >
                              Maintenance
                            </span>
                          )}
                        </div>
                        <div className="md-typescale-body-small font-mono mt-0.5" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          {rec.license_plate} · {rec.vehicle_class}
                        </div>
                        <div className="md-typescale-body-small mt-0.5" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          {rec.recommendation}
                        </div>
                      </div>

                      {/* Metrics */}
                      <div className="text-right shrink-0">
                        {rec.distance_km >= 0 ? (
                          <div className="md-typescale-label-large tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                            {rec.distance_km < 1 ? `${(rec.distance_km * 1000).toFixed(0)}m` : `${rec.distance_km.toFixed(1)}km`}
                          </div>
                        ) : (
                          <div className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>No GPS</div>
                        )}
                        <div
                          className="md-typescale-body-small tabular-nums mt-0.5"
                          style={{ color: fits ? 'var(--color-md-success)' : 'var(--color-md-error)' }}
                        >
                          {rec.free_volume_vu.toFixed(1)} VU free
                        </div>
                        <div className="md-typescale-body-small tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          {rec.order_count} orders
                        </div>
                      </div>
                    </button>
                  );
                })
              )}
            </div>

            {/* Footer */}
            <div className="px-6 py-3 border-t text-center" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
              <span className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                Click a truck to reassign this order
              </span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
