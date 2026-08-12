import { useCallback, useEffect, useRef, useState, type Dispatch, type SetStateAction } from 'react';
import { AppState } from 'react-native';
import * as Haptics from 'expo-haptics';
import * as SecureStore from 'expo-secure-store';

import { API_BASE, PayloadTerminalApi } from '../api';
import { authFetch } from '../authSession';
import type { NotifItem } from '../components/NotificationsSheet';
import { extractProblemMessage } from '../localization';
import { reconcilePayloadSession } from '../session-reconcile';
import { buildManifest, type LiveOrder, type ManifestItem } from '../utils/manifest';
import { reconnectDelayMs } from '../utils/reconnect';
import type { Locale } from '../../../packages/i18n/locales';
import type { LiveNotifFrame } from './useNotifications';
import type { ShowToast } from './useToast';

// ─── Manifest data domain ─────────────────────────────────────────────────────
// Trucks, orders, manifest entity, live-sync WebSocket and reconnect refetches.

export function useManifestData({
  token,
  locale,
  tx,
  showToast,
  authHeaders,
  fetchNotifications,
  flushOfflineQueue,
  setNotifications,
  setUnreadCount,
}: {
  token: string | null;
  locale: Locale;
  tx: (key: string) => string;
  showToast: ShowToast;
  authHeaders: Record<string, string | undefined>;
  fetchNotifications: () => Promise<void>;
  flushOfflineQueue: () => Promise<void>;
  setNotifications: Dispatch<SetStateAction<NotifItem[]>>;
  setUnreadCount: Dispatch<SetStateAction<number>>;
}) {
  // Truck selector
  const [trucks, setTrucks] = useState<{ id: string; label: string; license_plate: string; vehicle_class: string }[]>([]);
  const [isLoadingTrucks, setIsLoadingTrucks] = useState(false);
  const [activeTruck, setActiveTruck] = useState<string | null>(null);

  // Orders for the active truck
  const [orders, setOrders] = useState<LiveOrder[]>([]);
  const [manifest, setManifest] = useState<ManifestItem[]>([]);
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);
  const [sealedOrderIds, setSealedOrderIds] = useState<Set<string>>(new Set());
  const [sealedOrdersByTruck, setSealedOrdersByTruck] = useState<Record<string, string[]>>({});
  const [batchReadyManifestIds, setBatchReadyManifestIds] = useState<string[]>([]);

  // UI state
  const [isLoading, setIsLoading] = useState(false);
  const [allSealed, setAllSealed] = useState(false);
  const [dispatchCodes, setDispatchCodes] = useState<Record<string, string>>({});

  // Post-seal double-check countdown (Edge 33)
  const [postSealCountdown, setPostSealCountdown] = useState(0);
  const [postSealOrderId, setPostSealOrderId] = useState<string | null>(null);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // LEO: Manifest Loading Gate state
  const [manifestId, setManifestId] = useState<string | null>(null);
  const [manifestSource, setManifestSource] = useState<'payloader' | 'factory'>('payloader');
  const [manifestState, setManifestState] = useState<string>(''); // DRAFT | LOADING | SEALED
  const [manifestVolume, setManifestVolume] = useState(0);
  const [manifestMaxVolume, setManifestMaxVolume] = useState(0);
  const [manifestStopCount, setManifestStopCount] = useState(0);
  const [manifestRegionCode, setManifestRegionCode] = useState('');
  const [inboundDriverLat, setInboundDriverLat] = useState<number | null>(null);
  const [inboundDriverLng, setInboundDriverLng] = useState<number | null>(null);
  const [inboundLive, setInboundLive] = useState(false);
  const [deliveryLabelsByOrder, setDeliveryLabelsByOrder] = useState<Record<string, string>>({});
  const [isStartingLoad, setIsStartingLoad] = useState(false);
  const [isSealingManifest, setIsSealingManifest] = useState(false);
  const [isInjecting, setIsInjecting] = useState(false);

  const [isOnline, setIsOnline] = useState(true);
  const [liveSyncRevision, setLiveSyncRevision] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const wsHasConnectedOnceRef = useRef(false);

  const fetchTrucks = useCallback(async () => {
    if (!token) return;
    setIsLoadingTrucks(true);
    try {
      const res = await authFetch('/v1/payloader/trucks');
      if (!res.ok) return;
      const vehicles: { id: string; label: string; license_plate: string; vehicle_class: string }[] = await res.json();
      setTrucks(vehicles.map(v => ({
        id: v.id,
        label: v.label || v.license_plate || v.id.slice(0, 8),
        license_plate: v.license_plate,
        vehicle_class: v.vehicle_class,
      })));
    } catch {
    } finally {
      setIsLoadingTrucks(false);
    }
  }, [token]);

  // Fetch supplier's vehicles once authenticated
  useEffect(() => {
    fetchTrucks();
  }, [fetchTrucks]);

  // ── Fetch manifest for selected truck ────────────────────────────────────
  const fetchManifest = useCallback(async (truckId: string) => {
    setIsLoading(true);
    setOrders([]);
    setManifest([]);
    setSelectedOrderId(null);
    setSealedOrderIds(new Set());
    setAllSealed(false);
    setPostSealCountdown(0);
    setPostSealOrderId(null);
    if (countdownRef.current) { clearInterval(countdownRef.current); countdownRef.current = null; }

    const cacheKey = `manifest_${truckId}`;
    const CACHE_TTL_MS = 15 * 60 * 1000; // 15 minutes
    try {
      const res = await authFetch(
        `/v1/payloader/orders?vehicle_id=${encodeURIComponent(truckId)}&state=LOADED`,
      );
      if (!res.ok) throw new Error(await extractProblemMessage(res, locale));
      const data: LiveOrder[] = await res.json();
      // Cache for offline fallback with timestamp
      try {
        const cachePayload = { data, timestamp: Date.now() };
        await SecureStore.setItemAsync(cacheKey, JSON.stringify(cachePayload));
      } catch {}
      setOrders(data);
      const m = buildManifest(data);
      setManifest(m);
      if (data.length > 0) setSelectedOrderId(data[0].order_id);
    } catch (e: unknown) {
      // Attempt to load cached manifest (with TTL validation)
      try {
        const cached = await SecureStore.getItemAsync(cacheKey);
        if (cached) {
          const cachePayload = JSON.parse(cached);
          const { data, timestamp } = cachePayload;
          const age = Date.now() - (timestamp || 0);

          // Only use cache if fresher than 15 minutes
          if (age < CACHE_TTL_MS) {
            setOrders(data);
            setManifest(buildManifest(data));
            if (data.length > 0) setSelectedOrderId(data[0].order_id);
            showToast(tx('payload.alert.offline_cached_manifest_title'), tx('payload.alert.offline_cached_manifest_body'), 'info', 3200);
            return;
          } else {
            // Cache is stale; do not use
            showToast(tx('payload.alert.cache_stale_title'), tx('payload.alert.cache_stale_body'), 'warning', 3400);
          }
        }
      } catch {}
      showToast(tx('payload.alert.manifest_fetch_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoading(false);
    }
  }, [authHeaders, locale, showToast, tx]);

  const handleTruckSelect = (truckId: string) => {
    setActiveTruck(truckId);
    setManifestId(null);
    setManifestSource('payloader');
    setManifestState('');
    setManifestVolume(0);
    setManifestMaxVolume(0);
    setManifestStopCount(0);
    setManifestRegionCode('');
    fetchManifest(truckId);
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
  };

  // ── LEO: Fetch manifest entity for this truck ─────────────────────────
  const fetchTruckManifest = useCallback(async () => {
    if (!token || !activeTruck) return;
    try {
      const matchesTruck = (m: { truck_id?: string; vehicle_id?: string }) =>
        m.truck_id === activeTruck || m.vehicle_id === activeTruck;

      const fetchManifestForState = async (state: 'DRAFT' | 'LOADING') => {
        const data = await PayloadTerminalApi.listLoadingBayManifests(token, state);
        return (data.manifests || []).find(matchesTruck) ?? null;
      };

      const m = await fetchManifestForState('DRAFT') ?? await fetchManifestForState('LOADING');
      if (m) {
        setManifestId(m.manifest_id);
        setManifestSource(m.source);
        setManifestState(m.state || '');
        setManifestVolume(m.total_volume_vu || 0);
        setManifestMaxVolume(m.max_volume_vu || 0);
        setManifestStopCount(m.stop_count || 0);
        setManifestRegionCode(m.region_code || '');
        try {
          const detailPath = m.source === 'factory'
            ? `/v1/factory/manifests/${encodeURIComponent(m.manifest_id)}`
            : `/v1/payloader/manifests/${encodeURIComponent(m.manifest_id)}`;
          const detailRes = await authFetch(detailPath);
          if (detailRes.ok) {
            const detail = await detailRes.json();
            const labels: Record<string, string> = {};
            const orderRows = detail.orders ?? detail.transfers ?? [];
            for (const row of orderRows) {
              const orderId = row.order_id;
              if (!orderId) continue;
              const label = row.delivery_expectation?.target_label || row.delivery_expectation?.badge_label;
              if (label) labels[orderId] = label;
            }
            setDeliveryLabelsByOrder(labels);
            const lat = typeof detail.driver_lat === 'number' ? detail.driver_lat : null;
            const lng = typeof detail.driver_lng === 'number' ? detail.driver_lng : null;
            setInboundDriverLat(lat);
            setInboundDriverLng(lng);
            setInboundLive(Boolean(detail.live_location_available));
          }
        } catch {
          setDeliveryLabelsByOrder({});
          setInboundDriverLat(null);
          setInboundDriverLng(null);
          setInboundLive(false);
        }
      }
    } catch {}
  }, [token, activeTruck]);

  const refreshBatchReadyManifests = useCallback(async () => {
    if (!token || !activeTruck) return;
    try {
      const data = await PayloadTerminalApi.listLoadingBayManifests(token, 'LOADING');
      const loadingManifests = data.manifests || [];
      const sealedByTruck = { ...sealedOrdersByTruck };
      if (sealedOrderIds.size > 0) {
        sealedByTruck[activeTruck] = Array.from(sealedOrderIds);
      }
      const ready: string[] = [];
      for (const m of loadingManifests) {
        const truckId = m.truck_id || m.vehicle_id;
        if (!truckId) continue;
        let truckOrders: LiveOrder[];
        if (truckId === activeTruck) {
          truckOrders = orders;
        } else {
          const res = await authFetch(
            `/v1/payloader/orders?vehicle_id=${encodeURIComponent(truckId)}&state=LOADED`,
          );
          if (!res.ok) continue;
          truckOrders = await res.json();
        }
        const sealed = new Set(sealedByTruck[truckId] || []);
        if (truckOrders.length > 0 && truckOrders.every(o => sealed.has(o.order_id))) {
          ready.push(m.manifest_id);
        }
      }
      setBatchReadyManifestIds(Array.from(new Set(ready)));
    } catch {
      // Best-effort; batch seal is optional when manifests API is unavailable.
    }
  }, [token, activeTruck, sealedOrderIds, sealedOrdersByTruck, orders]);

  // ── Notifications: WebSocket live sync ─────────────────────────────────
  useEffect(() => {
    if (!token) return;
    fetchNotifications();
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let reconnectAttempt = 0;
    const connect = async () => {
      let wsTicket = '';
      try {
        const sessionRes = await authFetch('/v1/payload/ws-session');
        const sessionBody = (await sessionRes.json().catch(() => null)) as { token?: string } | null;
        if (!sessionRes.ok || typeof sessionBody?.token !== 'string' || !sessionBody.token) {
          throw new Error('ws-session failed');
        }
        wsTicket = sessionBody.token;
      } catch {
        reconnectAttempt += 1;
        reconnectTimer = setTimeout(() => {
          void connect();
        }, reconnectDelayMs(reconnectAttempt));
        return;
      }
      const wsUrl = `${API_BASE.replace(/^http/, 'ws')}/v1/ws?token=${encodeURIComponent(wsTicket)}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;
      ws.onopen = () => {
        reconnectAttempt = 0;
        setIsOnline(true);
        const wasReconnect = wsHasConnectedOnceRef.current;
        wsHasConnectedOnceRef.current = true;
        void (async () => {
          if (wasReconnect) {
            await reconcilePayloadSession();
            setIsStartingLoad(false);
            setIsSealingManifest(false);
            setIsInjecting(false);
            await fetchTrucks();
            await fetchNotifications();
            if (activeTruck) {
              await fetchManifest(activeTruck);
              await fetchTruckManifest();
            }
            await refreshBatchReadyManifests();
          }
          await flushOfflineQueue();
        })();
      };
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as LiveNotifFrame;
          if (msg.type === 'PAYLOAD_SYNC') {
            setLiveSyncRevision(prev => prev + 1);
            return;
          }
          if ((msg.title && msg.title.length > 0) || (msg.body && msg.body.length > 0)) {
            const n: NotifItem = {
              id: `live-${Date.now()}`,
              type: msg.type ?? '',
              title: msg.title ?? '',
              body: msg.body ?? '',
              read_at: null,
              created_at: new Date().toISOString(),
            };
            setNotifications(prev => [n, ...prev]);
            setUnreadCount(prev => prev + 1);
          }
        } catch {}
      };
      ws.onclose = () => {
        setIsOnline(false);
        const delay = reconnectDelayMs(reconnectAttempt);
        reconnectAttempt += 1;
        reconnectTimer = setTimeout(() => { void connect(); }, delay);
      };
      ws.onerror = () => { setIsOnline(false); ws.close(); };
    };
    void connect();
    return () => {
      clearTimeout(reconnectTimer);
      wsRef.current?.close();
      wsRef.current = null;
      wsHasConnectedOnceRef.current = false;
    };
  }, [token, fetchNotifications]);

  useEffect(() => { fetchTruckManifest(); }, [fetchTruckManifest]);

  useEffect(() => {
    if (!token || !activeTruck || liveSyncRevision === 0) return;
    fetchManifest(activeTruck);
    fetchTruckManifest();
  }, [token, activeTruck, liveSyncRevision, fetchManifest, fetchTruckManifest]);

  useEffect(() => {
    if (!token || !isOnline) return;
    fetchTrucks();
    fetchNotifications();
    if (activeTruck) {
      fetchManifest(activeTruck);
      fetchTruckManifest();
    }
  }, [token, isOnline, activeTruck, fetchTrucks, fetchNotifications, fetchManifest, fetchTruckManifest, refreshBatchReadyManifests]);

  useEffect(() => {
    if (!token) return;
    refreshBatchReadyManifests();
  }, [token, activeTruck, sealedOrderIds, orders, refreshBatchReadyManifests]);

  useEffect(() => {
    if (!token) return;
    const subscription = AppState.addEventListener('change', (nextState) => {
      if (nextState !== 'active') return;
      fetchTrucks();
      fetchNotifications();
      if (activeTruck) {
        fetchManifest(activeTruck);
        fetchTruckManifest();
      }
    });

    return () => {
      subscription.remove();
    };
  }, [token, activeTruck, fetchTrucks, fetchNotifications, fetchManifest, fetchTruckManifest]);

  // Clear the post-seal countdown timer on unmount
  useEffect(() => {
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
  }, []);

  // ── Derived selection ───────────────────────────────────────────────────────
  const selectedOrder = orders.find(o => o.order_id === selectedOrderId);
  const selectedManifest = manifest.filter(i => i.orderId === selectedOrderId);
  const allChecked = selectedManifest.length > 0 && selectedManifest.every(i => i.scanned);

  return {
    trucks,
    setTrucks,
    isLoadingTrucks,
    setIsLoadingTrucks,
    activeTruck,
    setActiveTruck,
    orders,
    setOrders,
    manifest,
    setManifest,
    selectedOrderId,
    setSelectedOrderId,
    sealedOrderIds,
    setSealedOrderIds,
    sealedOrdersByTruck,
    setSealedOrdersByTruck,
    batchReadyManifestIds,
    setBatchReadyManifestIds,
    isLoading,
    allSealed,
    setAllSealed,
    dispatchCodes,
    setDispatchCodes,
    postSealCountdown,
    setPostSealCountdown,
    postSealOrderId,
    setPostSealOrderId,
    countdownRef,
    manifestId,
    manifestSource,
    manifestState,
    setManifestState,
    manifestVolume,
    manifestMaxVolume,
    manifestStopCount,
    manifestRegionCode,
    inboundDriverLat,
    inboundDriverLng,
    inboundLive,
    deliveryLabelsByOrder,
    isStartingLoad,
    setIsStartingLoad,
    isSealingManifest,
    setIsSealingManifest,
    isInjecting,
    setIsInjecting,
    isOnline,
    liveSyncRevision,
    fetchTrucks,
    fetchManifest,
    fetchTruckManifest,
    refreshBatchReadyManifests,
    handleTruckSelect,
    selectedOrder,
    selectedManifest,
    allChecked,
  };
}
