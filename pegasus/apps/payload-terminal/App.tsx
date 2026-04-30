import { useState, useEffect, useCallback, useRef } from 'react';
import { Text, View, TouchableOpacity, Alert, ScrollView, TextInput, Modal, FlatList } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import * as ScreenOrientation from 'expo-screen-orientation';
import * as SecureStore from 'expo-secure-store';
import "./global.css";
import { useT, isIOS } from './theme';
import { buildManifest, type LiveOrder, type ManifestItem } from './utils/manifest';

// ─── API ──────────────────────────────────────────────────────────────────────
// Resolution order for the backend base URL:
//   1. EXPO_PUBLIC_API_URL env var (set in .env or via `npx expo start --dev-client`)
//      — required for physical devices so they can reach the Mac's LAN IP.
//   2. __DEV__ fallback = http://localhost:8080 (simulator only).
//   3. Production = https://api.thelab.uz.
const API_BASE = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8080' : 'https://api.thelab.uz');

// ─── Main Component ───────────────────────────────────────────────────────────

export default function App() {
  const T = useT();

  type BackendNotifItem = {
    notification_id: string;
    type: string;
    title: string;
    body: string;
    read_at: string | null;
    created_at: string;
  };

  type LiveNotifFrame = {
    type?: string;
    title?: string;
    body?: string;
    channel?: string;
  };

  const normalizeNotification = (item: BackendNotifItem): NotifItem => ({
    id: item.notification_id,
    type: item.type,
    title: item.title,
    body: item.body,
    read_at: item.read_at,
    created_at: item.created_at,
  });

  // Auth state
  const [token, setToken] = useState<string | null>(null);
  const [workerName, setWorkerName] = useState('');
  const [phoneInput, setPhoneInput] = useState('');
  const [pinInput, setPinInput] = useState('');
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const [authLoading, setAuthLoading] = useState(true);

  // Supplier context
  const [supplierId, setSupplierId] = useState<string | null>(null);

  // Truck selector
  const [trucks, setTrucks] = useState<{ id: string; label: string; license_plate: string; vehicle_class: string }[]>([]);
  const [activeTruck, setActiveTruck] = useState<string | null>(null);

  // Orders for the active truck
  const [orders, setOrders] = useState<LiveOrder[]>([]);
  const [manifest, setManifest] = useState<ManifestItem[]>([]);
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);
  const [sealedOrderIds, setSealedOrderIds] = useState<Set<string>>(new Set());

  // UI state
  const [isLoading, setIsLoading] = useState(false);
  const [isSealing, setIsSealing] = useState(false);
  const [allSealed, setAllSealed] = useState(false);
  const [dispatchCodes, setDispatchCodes] = useState<Record<string, string>>({});

  // Post-seal double-check countdown (Edge 33)
  const [postSealCountdown, setPostSealCountdown] = useState(0);
  const [postSealOrderId, setPostSealOrderId] = useState<string | null>(null);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // LEO: Manifest Loading Gate state
  const [manifestId, setManifestId] = useState<string | null>(null);
  const [manifestState, setManifestState] = useState<string>(''); // DRAFT | LOADING | SEALED
  const [manifestVolume, setManifestVolume] = useState(0);
  const [manifestMaxVolume, setManifestMaxVolume] = useState(0);
  const [isStartingLoad, setIsStartingLoad] = useState(false);
  const [isSealingManifest, setIsSealingManifest] = useState(false);
  const [exceptionLoading, setExceptionLoading] = useState<string | null>(null); // orderId being excepted
  const [showInjectOrder, setShowInjectOrder] = useState(false);
  const [injectOrderId, setInjectOrderId] = useState('');
  const [isInjecting, setIsInjecting] = useState(false);

  // Offline action queue — persisted in SecureStore, flushed on reconnect
  type QueuedAction = { id: string; endpoint: string; method: string; body: string; createdAt: number };
  const [offlineQueue, setOfflineQueue] = useState<QueuedAction[]>([]);
  const [isOnline, setIsOnline] = useState(true);

  // Notification state
  type NotifItem = { id: string; type: string; title: string; body: string; read_at: string | null; created_at: string };
  const [notifications, setNotifications] = useState<NotifItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [showNotifPanel, setShowNotifPanel] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  // Re-dispatch state
  type TruckRecommendation = {
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
  };
  const [showReDispatch, setShowReDispatch] = useState(false);
  const [reDispatchOrderId, setReDispatchOrderId] = useState<string | null>(null);
  const [reDispatchRetailer, setReDispatchRetailer] = useState('');
  const [reDispatchVolume, setReDispatchVolume] = useState(0);
  const [recommendations, setRecommendations] = useState<TruckRecommendation[]>([]);
  const [isLoadingRecs, setIsLoadingRecs] = useState(false);
  const [isReassigning, setIsReassigning] = useState(false);

  // Lock tablet to landscape + restore session on mount
  useEffect(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.LANDSCAPE_LEFT);
    (async () => {
      try {
        const saved = await SecureStore.getItemAsync('payloader_token');
        const name = await SecureStore.getItemAsync('payloader_name');
        const sid = await SecureStore.getItemAsync('payloader_supplier_id');
        const wid = await SecureStore.getItemAsync('payloader_warehouse_id');
        const wname = await SecureStore.getItemAsync('payloader_warehouse_name');
        if (saved) {
          setToken(saved);
          setWorkerName(name || 'Payloader');
          if (sid) setSupplierId(sid);
        }
        // Restore offline queue
        const queueStr = await SecureStore.getItemAsync('offline_queue');
        if (queueStr) {
          try { setOfflineQueue(JSON.parse(queueStr)); } catch {}
        }
      } catch {} finally {
        setAuthLoading(false);
      }
    })();
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
  }, []);

  // Fetch supplier's vehicles once authenticated
  useEffect(() => {
    if (!token) return;
    (async () => {
      try {
        const res = await fetch(`${API_BASE}/v1/payloader/trucks`, {
          headers: { 'Authorization': `Bearer ${token}` },
        });
        if (!res.ok) return;
        const vehicles: { id: string; label: string; license_plate: string; vehicle_class: string }[] = await res.json();
        setTrucks(vehicles.map(v => ({
          id: v.id,
          label: v.label || v.license_plate || v.id.slice(0, 8),
          license_plate: v.license_plate,
          vehicle_class: v.vehicle_class,
        })));
      } catch {}
    })();
  }, [token]);

  // ── Payloader Login ──────────────────────────────────────────────────────
  const handleLogin = async () => {
    if (!phoneInput || !pinInput) return;
    setIsLoggingIn(true);
    try {
      const res = await fetch(`${API_BASE}/v1/auth/payloader/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone: phoneInput, pin: pinInput }),
      });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || `HTTP ${res.status}`);
      }
      const data = await res.json();
      await SecureStore.setItemAsync('payloader_token', data.token);
      await SecureStore.setItemAsync('payloader_name', data.name || 'Payloader');
      if (data.supplier_id) await SecureStore.setItemAsync('payloader_supplier_id', data.supplier_id);
      if (data.warehouse_id) await SecureStore.setItemAsync('payloader_warehouse_id', data.warehouse_id);
      if (data.warehouse_name) await SecureStore.setItemAsync('payloader_warehouse_name', data.warehouse_name);
      // Store Firebase custom token for future SDK integration (graceful — not exchanged yet)
      if (data.firebase_token) {
        await SecureStore.setItemAsync('payloader_firebase_token', data.firebase_token);
      }
      setToken(data.token);
      setWorkerName(data.name || 'Payloader');
      if (data.supplier_id) setSupplierId(data.supplier_id);
    } catch (e: unknown) {
      Alert.alert('LOGIN FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleLogout = async () => {
    await SecureStore.deleteItemAsync('payloader_token');
    await SecureStore.deleteItemAsync('payloader_name');
    await SecureStore.deleteItemAsync('payloader_supplier_id');
    await SecureStore.deleteItemAsync('payloader_firebase_token');
    await SecureStore.deleteItemAsync('payloader_warehouse_id');
    await SecureStore.deleteItemAsync('payloader_warehouse_name');
    setToken(null);
    setWorkerName('');
    setSupplierId(null);
    setActiveTruck(null);
    setTrucks([]);
  };

  // ── Notifications: WebSocket + fetch ───────────────────────────────────
  const fetchNotifications = useCallback(async () => {
    if (!token) return;
    try {
      const res = await fetch(`${API_BASE}/v1/user/notifications?limit=50`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (!res.ok) return;
      const data = await res.json();
      const items = Array.isArray(data.notifications)
        ? data.notifications.map((item: BackendNotifItem) => normalizeNotification(item))
        : [];
      setNotifications(items);
      setUnreadCount(data.unread_count ?? 0);
    } catch {}
  }, [token]);

  useEffect(() => {
    if (!token) return;
    fetchNotifications();
    const wsUrl = `${API_BASE.replace(/^http/, 'ws')}/v1/ws/payloader?token=${encodeURIComponent(token)}`;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    const connect = () => {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;
      ws.onopen = () => {
        setIsOnline(true);
        // Flush offline queue on reconnect
        flushOfflineQueue();
      };
      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as LiveNotifFrame;
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
      ws.onclose = () => { setIsOnline(false); reconnectTimer = setTimeout(connect, 3000); };
      ws.onerror = () => { setIsOnline(false); ws.close(); };
    };
    connect();
    return () => {
      clearTimeout(reconnectTimer);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [token, fetchNotifications]);

  const markNotifRead = useCallback(async (id: string) => {
    if (!token) return;
    setNotifications(prev => prev.map(n => n.id === id ? { ...n, read_at: new Date().toISOString() } : n));
    setUnreadCount(prev => Math.max(0, prev - 1));
    try {
      await fetch(`${API_BASE}/v1/user/notifications/read`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ notification_ids: [id] }),
      });
    } catch {}
  }, [token]);

  const markAllNotifsRead = useCallback(async () => {
    if (!token) return;
    setNotifications(prev => prev.map(n => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
    setUnreadCount(0);
    try {
      await fetch(`${API_BASE}/v1/user/notifications/read`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ mark_all: true }),
      });
    } catch {}
  }, [token]);

  const getAuthHeaders = () => {
    const traceId = crypto.randomUUID();
    return token
      ? { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json', 'X-Trace-Id': traceId }
      : { 'Content-Type': 'application/json', 'X-Trace-Id': traceId };
  };
  const authHeaders = getAuthHeaders();

  // ── Re-dispatch: fetch recommendations ──────────────────────────────────
  const openReDispatch = useCallback(async (orderId: string) => {
    setReDispatchOrderId(orderId);
    setShowReDispatch(true);
    setRecommendations([]);
    setReDispatchRetailer('');
    setReDispatchVolume(0);
    setIsLoadingRecs(true);
    try {
      const res = await fetch(`${API_BASE}/v1/payloader/recommend-reassign`, {
        method: 'POST',
        headers: authHeaders as HeadersInit,
        body: JSON.stringify({ order_id: orderId }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRecommendations(data.recommendations ?? []);
      setReDispatchRetailer(data.retailer_name ?? '');
      setReDispatchVolume(data.order_volume_vu ?? 0);
    } catch (e: unknown) {
      Alert.alert('RECOMMENDATION FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsLoadingRecs(false);
    }
  }, [token]);

  const handleReassign = useCallback(async (newDriverId: string, _newVehicleId: string) => {
    if (!reDispatchOrderId || !token) return;
    setIsReassigning(true);
    try {
      // RouteId == DriverId in this codebase; vehicle is bound to the driver.
      const res = await fetch(`${API_BASE}/v1/fleet/reassign`, {
        method: 'POST',
        headers: authHeaders as HeadersInit,
        body: JSON.stringify({
          order_ids: [reDispatchOrderId],
          new_route_id: newDriverId,
        }),
      });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || `HTTP ${res.status}`);
      }
      const data: { conflicts?: Array<{ order_id: string; reason: string }>; reassigned?: number } = await res.json().catch(() => ({}));
      if (data.conflicts && data.conflicts.length > 0) {
        throw new Error(data.conflicts.map(c => `${c.order_id.slice(0, 8)}: ${c.reason}`).join('; '));
      }
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      // Remove the reassigned order from local state
      setOrders(prev => prev.filter(o => o.order_id !== reDispatchOrderId));
      setManifest(prev => prev.filter(m => m.orderId !== reDispatchOrderId));
      if (selectedOrderId === reDispatchOrderId) {
        const remaining = orders.filter(o => o.order_id !== reDispatchOrderId && !sealedOrderIds.has(o.order_id));
        setSelectedOrderId(remaining.length > 0 ? remaining[0].order_id : null);
      }
      setShowReDispatch(false);
      setReDispatchOrderId(null);
    } catch (e: unknown) {
      Alert.alert('REASSIGN FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsReassigning(false);
    }
  }, [token, reDispatchOrderId, orders, selectedOrderId, sealedOrderIds]);

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
      const res = await fetch(
        `${API_BASE}/v1/payloader/orders?vehicle_id=${encodeURIComponent(truckId)}&state=LOADED`,
        { headers: authHeaders as HeadersInit }
      );
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
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
            Alert.alert('OFFLINE MODE', 'Loaded cached manifest.');
            return;
          } else {
            // Cache is stale; do not use
            Alert.alert('CACHE STALE', 'Cached manifest is older than 15 minutes. Cannot proceed offline.');
          }
        }
      } catch {}
      Alert.alert('MANIFEST FETCH ERROR', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const handleTruckSelect = (truckId: string) => {
    setActiveTruck(truckId);
    setManifestId(null);
    setManifestState('');
    setManifestVolume(0);
    setManifestMaxVolume(0);
    fetchManifest(truckId);
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
  };

  // ── LEO: Fetch manifest entity for this truck ─────────────────────────
  const fetchTruckManifest = useCallback(async () => {
    if (!token || !activeTruck) return;
    try {
      const res = await fetch(`${API_BASE}/v1/supplier/manifests?state=DRAFT`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (!res.ok) return;
      const data = await res.json();
      // Find manifest for this truck (most recent DRAFT or LOADING)
      const m = (data.manifests || []).find((m: any) => m.truck_id === activeTruck || m.state === 'LOADING');
      if (m) {
        setManifestId(m.manifest_id);
        setManifestState(m.state);
        setManifestVolume(m.total_volume_vu || 0);
        setManifestMaxVolume(m.max_volume_vu || 0);
      }
    } catch {}
  }, [token, activeTruck]);

  useEffect(() => { fetchTruckManifest(); }, [fetchTruckManifest]);

  // ── LEO: Start Loading (DRAFT → LOADING) ─────────────────────────────
  const handleStartLoading = async () => {
    if (!manifestId || !token) return;
    setIsStartingLoad(true);
    try {
      const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/start-loading`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setManifestState('LOADING');
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch (e: unknown) {
      Alert.alert('START LOADING FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsStartingLoad(false);
    }
  };

  // ── LEO: Exception — remove order from manifest ──────────────────────
  const handleException = async (orderId: string, reason: 'OVERFLOW' | 'DAMAGED' | 'MANUAL') => {
    if (!manifestId || !token) return;
    setExceptionLoading(orderId);
    try {
      const res = await fetch(`${API_BASE}/v1/payload/manifest-exception`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ manifest_id: manifestId, order_id: orderId, reason }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
      // Remove order from local state
      setOrders(prev => prev.filter(o => o.order_id !== orderId));
      setManifest(prev => prev.filter(i => i.orderId !== orderId));
      if (selectedOrderId === orderId) {
        const remaining = orders.filter(o => o.order_id !== orderId);
        setSelectedOrderId(remaining.length > 0 ? remaining[0].order_id : null);
      }
      if (data.escalated) {
        Alert.alert('DLQ ESCALATION', `Order ${orderId.slice(0, 8)} has been escalated to admin after ${data.overflow_count} overflow attempts.`);
      }
    } catch (e: unknown) {
      Alert.alert('EXCEPTION FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setExceptionLoading(null);
    }
  };

  // ── LEO: Manifest-level Seal (LOADING → SEALED) ──────────────────────
  const handleManifestSeal = async () => {
    if (!manifestId || !token) return;
    setIsSealingManifest(true);
    try {
      const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/seal`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error || `HTTP ${res.status}`);
      }
      const data = await res.json();
      setManifestState('SEALED');
      setAllSealed(true);
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      Alert.alert('MANIFEST SEALED', `${data.stop_count} stops sealed. ${data.volume_vu?.toFixed(1)}/${data.max_vu?.toFixed(1)} VU. Route finalized.`);
    } catch (e: unknown) {
      Alert.alert('SEAL FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsSealingManifest(false);
    }
  };

  // ── Phase A: Mid-Load Order Injection ─────────────────────────────────
  const handleInjectOrder = async () => {
    if (!manifestId || !token || !injectOrderId.trim()) return;
    setIsInjecting(true);
    try {
      const res = await fetch(`${API_BASE}/v1/supplier/manifests/${manifestId}/inject-order`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_id: injectOrderId.trim() }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error || `HTTP ${res.status}`);
      }
      const data = await res.json();
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      Alert.alert('ORDER INJECTED', `Order ${injectOrderId.slice(0, 8)} added. ${data.stop_count} stops, ${data.total_volume_vu?.toFixed(1)} VU.`);
      setInjectOrderId('');
      setShowInjectOrder(false);
      // Refresh manifest to pick up new order
      if (activeTruck) fetchManifest(activeTruck);
    } catch (e: unknown) {
      if (!isOnline) {
        // Offline: queue the action
        const action: QueuedAction = {
          id: Date.now().toString(),
          endpoint: `/v1/supplier/manifests/${manifestId}/inject-order`,
          method: 'POST',
          body: JSON.stringify({ order_id: injectOrderId.trim() }),
          createdAt: Date.now(),
        };
        const updated = [...offlineQueue, action];
        setOfflineQueue(updated);
        await SecureStore.setItemAsync('offline_queue', JSON.stringify(updated));
        Alert.alert('QUEUED OFFLINE', 'Inject order queued. Will sync when connection restores.');
        setInjectOrderId('');
        setShowInjectOrder(false);
      } else {
        Alert.alert('INJECT FAILED', e instanceof Error ? e.message : 'Unknown error');
      }
    } finally {
      setIsInjecting(false);
    }
  };

  // ── Phase C: Offline Queue Flush ──────────────────────────────────────
  const flushOfflineQueue = async () => {
    if (offlineQueue.length === 0 || !token) return;
    const remaining: QueuedAction[] = [];
    for (const action of offlineQueue) {
      try {
        const res = await fetch(`${API_BASE}${action.endpoint}`, {
          method: action.method,
          headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: action.body,
        });
        if (!res.ok) {
          remaining.push(action); // keep for retry
        }
      } catch {
        remaining.push(action);
      }
    }
    setOfflineQueue(remaining);
    await SecureStore.setItemAsync('offline_queue', JSON.stringify(remaining));
    if (remaining.length === 0 && offlineQueue.length > 0) {
      Alert.alert('SYNC COMPLETE', `${offlineQueue.length} queued actions synced.`);
    }
  };

  // ── Checkbox toggle ───────────────────────────────────────────────────────
  const toggleCheck = (itemId: string) => {
    setManifest(prev =>
      prev.map(item =>
        item.id === itemId ? { ...item, scanned: !item.scanned } : item
      )
    );
    Haptics.selectionAsync();
  };

  // ── Seal & dispatch ───────────────────────────────────────────────────────
  const selectedOrder = orders.find(o => o.order_id === selectedOrderId);
  const selectedManifest = manifest.filter(i => i.orderId === selectedOrderId);
  const allChecked = selectedManifest.length > 0 && selectedManifest.every(i => i.scanned);

  const handleSeal = async () => {
    if (!selectedOrderId || !allChecked) return;
    setIsSealing(true);
    try {
      const res = await fetch(`${API_BASE}/v1/payload/seal`, {
        method: 'POST',
        headers: authHeaders as HeadersInit,
        body: JSON.stringify({
          order_id: selectedOrderId,
          terminal_id: activeTruck || 'WH-UNKNOWN',
          manifest_cleared: true,
        }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const sealData = await res.json();
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      const next = new Set(sealedOrderIds).add(selectedOrderId);
      setSealedOrderIds(next);
      if (sealData.dispatch_code) {
        setDispatchCodes(prev => ({ ...prev, [selectedOrderId]: sealData.dispatch_code }));
      }
      // Enter 60-second double-check countdown (Edge 33)
      setPostSealOrderId(selectedOrderId);
      setPostSealCountdown(60);
      if (countdownRef.current) clearInterval(countdownRef.current);
      countdownRef.current = setInterval(() => {
        setPostSealCountdown(prev => {
          if (prev <= 1) {
            if (countdownRef.current) clearInterval(countdownRef.current);
            countdownRef.current = null;
            // Advance to next order or allSealed after countdown
            const remainingOrders = orders.filter(o => !next.has(o.order_id));
            if (remainingOrders.length > 0) {
              setSelectedOrderId(remainingOrders[0].order_id);
            } else {
              setAllSealed(true);
            }
            setPostSealOrderId(null);
            return 0;
          }
          return prev - 1;
        });
      }, 1000);
    } catch (e: unknown) {
      Alert.alert('SEAL FAILED', e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setIsSealing(false);
    }
  };

  // ── Render: AUTH LOADING ────────────────────────────────────────────────
  if (authLoading) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background, alignItems: 'center', justifyContent: 'center' }}>
        <Text style={{ color: T.colors.tertiaryLabel, fontSize: 13, letterSpacing: 0.3 }}>
          {isIOS ? 'Restoring session...' : 'RESTORING SESSION...'}
        </Text>
      </View>
    );
  }

  // ── Render: POST-SEAL DOUBLE-CHECK COUNTDOWN (Edge 33) ────────────────
  if (postSealOrderId && postSealCountdown > 0) {
    return (
      <View style={{ flex: 1, backgroundColor: (T.colors as any)?.warning ?? '#F59E0B', alignItems: 'center', justifyContent: 'center', padding: 48 }}>
        <MaterialIcons name="verified-user" size={64} color="rgba(255,255,255,0.9)" style={{ marginBottom: 24 }} />
        <Text style={{ fontSize: 28, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.4 : 0.5, marginBottom: 8 }}>
          {isIOS ? 'Double-Check' : 'DOUBLE-CHECK'}
        </Text>
        <Text style={{ fontSize: 14, color: 'rgba(255,255,255,0.85)', textAlign: 'center', maxWidth: 400, marginBottom: 32 }}>
          {isIOS
            ? 'Verify the sealed order before it dispatches. Report missing items now if anything was forgotten.'
            : 'VERIFY THE SEALED ORDER BEFORE IT DISPATCHES. REPORT MISSING ITEMS NOW IF ANYTHING WAS FORGOTTEN.'}
        </Text>
        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 72, fontWeight: '800', color: '#FFFFFF', marginBottom: 32 }}>
          {postSealCountdown}
        </Text>
        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 12, color: 'rgba(255,255,255,0.7)', letterSpacing: 0.5, marginBottom: 32 }}>
          {postSealOrderId}
        </Text>
        <TouchableOpacity
          onPress={() => {
            Alert.alert(
              isIOS ? 'Report Missing Items?' : 'REPORT MISSING ITEMS?',
              isIOS
                ? 'This will flag the sealed order for review. The driver will be notified to verify the load.'
                : 'THIS WILL FLAG THE SEALED ORDER FOR REVIEW. THE DRIVER WILL BE NOTIFIED TO VERIFY THE LOAD.',
              [
                { text: isIOS ? 'Cancel' : 'CANCEL', style: 'cancel' },
                {
                  text: isIOS ? 'Report' : 'REPORT',
                  style: 'destructive',
                  onPress: async () => {
                    try {
                      await fetch(`${API_BASE}/v1/delivery/missing-items`, {
                        method: 'POST',
                        headers: authHeaders as HeadersInit,
                        body: JSON.stringify({ order_id: postSealOrderId, items: [], source: 'PAYLOAD_TERMINAL' }),
                      });
                      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
                      Alert.alert(
                        isIOS ? 'Reported' : 'REPORTED',
                        isIOS ? 'Missing items flagged for review.' : 'MISSING ITEMS FLAGGED FOR REVIEW.'
                      );
                    } catch (e: unknown) {
                      Alert.alert('ERROR', e instanceof Error ? e.message : 'Failed to report');
                    }
                  },
                },
              ]
            );
          }}
          style={{
            borderWidth: 2,
            borderColor: 'rgba(255,255,255,0.6)',
            paddingHorizontal: 32,
            paddingVertical: 14,
            borderRadius: T.radius.button,
            flexDirection: 'row',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <MaterialIcons name="report-problem" size={20} color="#FFFFFF" />
          <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
            {isIOS ? 'Report Missing Items' : 'REPORT MISSING ITEMS'}
          </Text>
        </TouchableOpacity>
      </View>
    );
  }

  // ── Render: ALL SEALED ────────────────────────────────────────────────────
  if (allSealed) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.success, alignItems: 'center', justifyContent: 'center', padding: 48 }}>
        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: 'rgba(255,255,255,0.7)', letterSpacing: 0.5, marginBottom: 24 }}>
          {activeTruck}
        </Text>
        <Text style={{ fontSize: 32, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.6 : 0.5, marginBottom: 8 }}>
          {isIOS ? 'Manifest Secured.' : 'MANIFEST SECURED.'}
        </Text>
        <Text style={{ fontSize: 32, fontWeight: '700', color: '#FFFFFF', textAlign: 'center', letterSpacing: isIOS ? -0.6 : 0.5, marginBottom: 24 }}>
          {isIOS ? 'Fleet Dispatched.' : 'FLEET DISPATCHED.'}
        </Text>

        {/* Dispatch codes (JIT QR substitute) */}
        {Object.keys(dispatchCodes).length > 0 && (
          <View style={{ backgroundColor: 'rgba(255,255,255,0.15)', borderRadius: 16, padding: 20, marginBottom: 32, minWidth: 280 }}>
            <Text style={{ fontSize: 11, fontWeight: '700', color: 'rgba(255,255,255,0.7)', letterSpacing: 1, textAlign: 'center', marginBottom: 12 }}>
              {isIOS ? 'Dispatch Codes' : 'DISPATCH CODES'}
            </Text>
            {Object.entries(dispatchCodes).map(([orderId, code]) => (
              <View key={orderId} style={{ flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', paddingVertical: 8, borderBottomWidth: 0.5, borderBottomColor: 'rgba(255,255,255,0.2)' }}>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 12, color: 'rgba(255,255,255,0.8)' }}>{orderId}</Text>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 16, fontWeight: '700', color: '#FFFFFF', letterSpacing: 2 }}>{code}</Text>
              </View>
            ))}
          </View>
        )}
        <TouchableOpacity
          onPress={() => { setActiveTruck(null); setAllSealed(false); setDispatchCodes({}); }}
          style={{
            borderWidth: 1,
            borderColor: 'rgba(255,255,255,0.5)',
            paddingHorizontal: 32,
            paddingVertical: 14,
            borderRadius: T.radius.button,
          }}
        >
          <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
            {isIOS ? 'New Manifest' : 'NEW MANIFEST'}
          </Text>
        </TouchableOpacity>
      </View>
    );
  }

  // ── Render: AWAITING TRUCK SELECTION ─────────────────────────────────────
  if (!token) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background, alignItems: 'center', justifyContent: 'center', padding: 48 }}>
        <Text style={{ fontWeight: '700', fontSize: 14, color: T.colors.tertiaryLabel, letterSpacing: 0.5, marginBottom: 32 }}>
          {isIOS ? 'Pegasus · Payload Terminal' : 'PEGASUS · PAYLOAD TERMINAL'}
        </Text>
        <Text style={{ fontSize: 22, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0.5, marginBottom: 32 }}>
          {isIOS ? 'Payloader Login' : 'PAYLOADER LOGIN'}
        </Text>
        <TextInput
          placeholder={isIOS ? 'Phone number' : 'PHONE NUMBER'}
          placeholderTextColor={T.colors.tertiaryLabel}
          value={phoneInput}
          onChangeText={setPhoneInput}
          keyboardType="phone-pad"
          autoCapitalize="none"
          style={{
            width: 320,
            borderWidth: isIOS ? 0.33 : 1,
            borderColor: T.colors.separator,
            backgroundColor: T.colors.cardBackground,
            borderRadius: T.radius.card,
            paddingHorizontal: 16,
            paddingVertical: 14,
            fontSize: 15,
            color: T.colors.label,
            marginBottom: 12,
          }}
        />
        <TextInput
          placeholder={isIOS ? '6-digit PIN' : '6-DIGIT PIN'}
          placeholderTextColor={T.colors.tertiaryLabel}
          value={pinInput}
          onChangeText={setPinInput}
          keyboardType="number-pad"
          maxLength={6}
          secureTextEntry
          style={{
            width: 320,
            borderWidth: isIOS ? 0.33 : 1,
            borderColor: T.colors.separator,
            backgroundColor: T.colors.cardBackground,
            borderRadius: T.radius.card,
            paddingHorizontal: 16,
            paddingVertical: 14,
            fontSize: 15,
            color: T.colors.label,
            marginBottom: 24,
            letterSpacing: 8,
            textAlign: 'center',
          }}
        />
        <TouchableOpacity
          onPress={handleLogin}
          disabled={isLoggingIn || !phoneInput || pinInput.length < 6}
          style={{
            width: 320,
            paddingVertical: 16,
            alignItems: 'center',
            backgroundColor: !isLoggingIn && phoneInput && pinInput.length >= 6 ? T.colors.accent : T.colors.fillSecondary,
            borderRadius: T.radius.button,
          }}
        >
          <Text style={{
            fontWeight: '600',
            fontSize: 14,
            letterSpacing: isIOS ? 0.3 : 1,
            color: !isLoggingIn && phoneInput && pinInput.length >= 6 ? '#FFFFFF' : T.colors.tertiaryLabel,
          }}>
            {isLoggingIn ? (isIOS ? 'Authenticating...' : 'AUTHENTICATING...') : (isIOS ? 'Sign In' : 'SIGN IN')}
          </Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (!activeTruck) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background }}>
        {/* Header */}
        <View style={{ backgroundColor: T.colors.sidebarBackground, paddingHorizontal: 32, paddingVertical: 16, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
          <View>
            <Text style={{ color: T.colors.sidebarLabel, fontWeight: '700', fontSize: 14, letterSpacing: 0.3 }}>
              {isIOS ? 'Pegasus · Payload Terminal' : 'PEGASUS · PAYLOAD TERMINAL'}
            </Text>
            <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 11, marginTop: 2 }}>
              {workerName}
            </Text>
          </View>
          <TouchableOpacity onPress={handleLogout} style={{ paddingHorizontal: 16, paddingVertical: 8 }}>
            <Text style={{ color: T.colors.sidebarSecondary, fontSize: 12, fontWeight: '600', letterSpacing: 0.3 }}>
              {isIOS ? 'Sign Out' : 'SIGN OUT'}
            </Text>
          </TouchableOpacity>
        </View>

        {/* Truck selector */}
        <View className="flex-1 items-center justify-center p-12">
          <Text style={{ fontSize: 13, fontWeight: '500', color: T.colors.tertiaryLabel, marginBottom: 32, letterSpacing: 0.3 }}>
            {isIOS ? 'Select Target Vehicle' : 'SELECT TARGET VEHICLE'}
          </Text>
          <View className="flex-row gap-4">
            {trucks.length === 0 ? (
              <Text style={{ color: T.colors.tertiaryLabel, fontFamily: T.typography.mono.fontFamily, fontSize: 12 }}>
                {isIOS ? 'No vehicles available' : 'NO VEHICLES AVAILABLE'}
              </Text>
            ) : (
              trucks.map(truck => (
                <TouchableOpacity
                  key={truck.id}
                  onPress={() => handleTruckSelect(truck.id)}
                  style={{
                    borderWidth: isIOS ? 0.33 : 1,
                    borderColor: T.colors.separator,
                    backgroundColor: T.colors.cardBackground,
                    paddingHorizontal: 40,
                    paddingVertical: 32,
                    alignItems: 'center',
                    borderRadius: T.radius.card,
                    ...T.shadow.card,
                  }}
                >
                  <Text style={{ fontSize: 22, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 1 }}>
                    {truck.label}
                  </Text>
                  {truck.license_plate ? (
                    <Text style={{ fontSize: 11, fontFamily: T.typography.mono.fontFamily, color: T.colors.tertiaryLabel, marginTop: 6, letterSpacing: 0.5 }}>
                      {truck.license_plate}
                    </Text>
                  ) : null}
                  <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.3 }}>
                    {truck.vehicle_class}
                  </Text>
                </TouchableOpacity>
              ))
            )}
          </View>
          <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 40, letterSpacing: 0.3 }}>
            {trucks.length === 0
              ? (isIOS ? 'Loading vehicles...' : 'LOADING VEHICLES...')
              : (isIOS ? 'Select target vehicle' : 'SELECT TARGET VEHICLE')}
          </Text>
        </View>
      </View>
    );
  }

  // ── Render: MANIFEST VIEW ─────────────────────────────────────────────────
  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background, flexDirection: 'row' }}>

      {/* ── Left pane: Shop list ─────────────────────────────────────────── */}
      <View style={{ width: 288, backgroundColor: T.colors.sidebarBackground, flexDirection: 'column' }}>
        {/* Header */}
        <View style={{ paddingHorizontal: 24, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator, flexDirection: 'row', alignItems: 'center' }}>
          <View style={{ flex: 1 }}>
            <Text style={{ color: T.colors.sidebarLabel, fontWeight: '700', fontSize: 13, letterSpacing: 0.3, marginBottom: 2 }}>
              {isIOS ? 'Payload Terminal' : 'PAYLOAD TERMINAL'}
            </Text>
            <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 11 }}>
              {activeTruck}
            </Text>
          </View>
          <TouchableOpacity onPress={() => setShowNotifPanel(true)} style={{ padding: 6 }}>
            <MaterialIcons name="notifications" size={20} color={T.colors.sidebarLabel} />
            {unreadCount > 0 && (
              <View style={{ position: 'absolute', top: 2, right: 2, backgroundColor: '#EF4444', borderRadius: 8, minWidth: 16, height: 16, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#FFF', fontSize: 9, fontWeight: '700' }}>{unreadCount > 99 ? '99+' : unreadCount}</Text>
              </View>
            )}
          </TouchableOpacity>
        </View>

        {/* Truck toggle bar */}
        <View style={{ flexDirection: 'row', borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
          {trucks.map(truck => (
            <TouchableOpacity
              key={truck.id}
              onPress={() => handleTruckSelect(truck.id)}
              style={{
                flex: 1,
                paddingVertical: 10,
                alignItems: 'center',
                backgroundColor: activeTruck === truck.id ? T.colors.sidebarActive : 'transparent',
                borderRadius: activeTruck === truck.id ? 8 : 0,
                margin: activeTruck === truck.id ? 4 : 0,
              }}
            >
              <Text style={{
                fontWeight: '700',
                fontSize: 11,
                letterSpacing: 0.5,
                color: activeTruck === truck.id ? T.colors.sidebarActiveText : T.colors.sidebarSecondary,
              }}>
                {truck.label}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {/* LEO: Volume Progress Bar + Manifest State */}
        {manifestId && (
          <View style={{ paddingHorizontal: 16, paddingVertical: 10, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
            <View style={{ flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 }}>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 10, color: T.colors.sidebarSecondary, letterSpacing: 0.3 }}>
                {manifestState || 'DRAFT'}
              </Text>
              <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 10, color: T.colors.sidebarSecondary }}>
                {manifestVolume.toFixed(1)}/{manifestMaxVolume.toFixed(1)} VU
              </Text>
            </View>
            <View style={{ height: 6, backgroundColor: T.colors.fillTertiary, borderRadius: 3, overflow: 'hidden' }}>
              <View style={{
                height: 6,
                borderRadius: 3,
                width: manifestMaxVolume > 0 ? `${Math.min((manifestVolume / manifestMaxVolume) * 100, 100)}%` : '0%',
                backgroundColor: manifestVolume / manifestMaxVolume > 0.95 ? '#EF4444' : manifestVolume / manifestMaxVolume > 0.8 ? '#F59E0B' : T.colors.accent,
              } as any} />
            </View>
            {manifestState === 'DRAFT' && (
              <TouchableOpacity
                onPress={handleStartLoading}
                disabled={isStartingLoad}
                style={{
                  marginTop: 8,
                  paddingVertical: 10,
                  alignItems: 'center',
                  backgroundColor: isStartingLoad ? T.colors.fillSecondary : T.colors.accent,
                  borderRadius: T.radius.button,
                }}
              >
                <Text style={{ fontWeight: '600', fontSize: 12, letterSpacing: 0.5, color: isStartingLoad ? T.colors.tertiaryLabel : '#FFFFFF' }}>
                  {isStartingLoad ? (isIOS ? 'Starting...' : 'STARTING...') : (isIOS ? 'Start Loading' : 'START LOADING')}
                </Text>
              </TouchableOpacity>
            )}
          </View>
        )}

        {/* Order list */}
        <ScrollView>
          {isLoading ? (
            <View className="p-6">
              <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 12, textAlign: 'center', letterSpacing: 0.3 }}>
                Fetching manifest...
              </Text>
            </View>
          ) : orders.length === 0 ? (
            <View className="p-6">
              <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 12, textAlign: 'center' }}>
                No pending orders
              </Text>
            </View>
          ) : (
            orders.map(order => {
              const isSealed = sealedOrderIds.has(order.order_id);
              const isActive = order.order_id === selectedOrderId;
              return (
                <TouchableOpacity
                  key={order.order_id}
                  onPress={() => !isSealed && setSelectedOrderId(order.order_id)}
                  onLongPress={() => {
                    if (!isSealed) {
                      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
                      openReDispatch(order.order_id);
                    }
                  }}
                  delayLongPress={500}
                  style={{
                    paddingHorizontal: 24,
                    paddingVertical: 14,
                    borderBottomWidth: 0.5,
                    borderBottomColor: T.colors.sidebarSeparator,
                    flexDirection: 'row',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    backgroundColor: isActive ? T.colors.sidebarActive : 'transparent',
                    borderRadius: isActive ? (isIOS ? 10 : 16) : 0,
                    marginHorizontal: isActive ? 8 : 0,
                    marginVertical: isActive ? 2 : 0,
                  }}
                >
                  <View>
                    <Text style={{ fontWeight: '600', fontSize: 13, color: isActive ? T.colors.sidebarActiveText : isSealed ? T.colors.sidebarSecondary : T.colors.sidebarLabel }}>
                      {order.order_id}
                    </Text>
                    <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, marginTop: 2, color: isActive ? (isIOS ? 'rgba(0,0,0,0.5)' : T.colors.sidebarActiveText) : T.colors.sidebarSecondary }}>
                      {order.retailer_id}
                    </Text>
                  </View>
                  {isSealed && (
                    <Text style={{ fontWeight: '600', fontSize: 11, color: T.colors.sidebarSecondary }}>
                      {isIOS ? 'Cleared' : 'CLEARED'}
                    </Text>
                  )}
                </TouchableOpacity>
              );
            })
          )}
        </ScrollView>
      </View>

      {/* ── Right pane: Manifest detail ──────────────────────────────────── */}
      <View className="flex-1 flex-col">
        {/* Order header */}
        {selectedOrder ? (
          <>
            <View
              className="px-8 py-5 flex-row items-center justify-between"
              style={{ borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}
            >
              <View>
                <Text style={{ fontSize: 18, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0 }}>
                  {selectedOrder.order_id}
                </Text>
                <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.3 }}>
                  {selectedOrder.retailer_id} · {selectedOrder.payment_gateway} · {selectedOrder.amount?.toLocaleString()}
                </Text>
              </View>
              <View style={{
                borderWidth: isIOS ? 0.33 : 1,
                borderColor: T.colors.separator,
                borderRadius: T.radius.checkbox,
                paddingHorizontal: 12,
                paddingVertical: 6,
                backgroundColor: T.colors.fillTertiary,
              }}>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '600', fontSize: 11, color: T.colors.secondaryLabel, letterSpacing: 0.5 }}>
                  {activeTruck}
                </Text>
              </View>
              <TouchableOpacity
                onPress={() => selectedOrderId && openReDispatch(selectedOrderId)}
                style={{
                  marginLeft: 10,
                  flexDirection: 'row',
                  alignItems: 'center',
                  borderWidth: isIOS ? 0.33 : 1,
                  borderColor: T.colors.separator,
                  borderRadius: T.radius.checkbox,
                  paddingHorizontal: 12,
                  paddingVertical: 6,
                  backgroundColor: T.colors.fillTertiary,
                }}
              >
                <MaterialIcons name="swap-horiz" size={14} color={T.colors.secondaryLabel} style={{ marginRight: 4 }} />
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontWeight: '600', fontSize: 11, color: T.colors.secondaryLabel, letterSpacing: 0.5 }}>
                  {isIOS ? 'Re-Dispatch' : 'RE-DISPATCH'}
                </Text>
              </TouchableOpacity>

              {/* LEO: Exception buttons — remove order from manifest */}
              {manifestState === 'LOADING' && selectedOrderId && (
                <View style={{ flexDirection: 'row', marginLeft: 8, gap: 4 }}>
                  {(['OVERFLOW', 'DAMAGED', 'MANUAL'] as const).map(reason => (
                    <TouchableOpacity
                      key={reason}
                      onPress={() => {
                        Alert.alert(
                          `Remove Order (${reason})`,
                          `Remove ${selectedOrderId.slice(0, 8)} from manifest? It will be re-injected with priority.`,
                          [
                            { text: 'Cancel', style: 'cancel' },
                            { text: 'Remove', style: 'destructive', onPress: () => handleException(selectedOrderId, reason) },
                          ]
                        );
                      }}
                      disabled={exceptionLoading === selectedOrderId}
                      style={{
                        paddingHorizontal: 8,
                        paddingVertical: 6,
                        borderRadius: T.radius.checkbox,
                        borderWidth: isIOS ? 0.33 : 1,
                        borderColor: reason === 'DAMAGED' ? '#EF4444' : reason === 'OVERFLOW' ? '#F59E0B' : T.colors.separator,
                        backgroundColor: T.colors.fillTertiary,
                      }}
                    >
                      <Text style={{
                        fontFamily: T.typography.mono.fontFamily,
                        fontWeight: '600',
                        fontSize: 9,
                        letterSpacing: 0.5,
                        color: reason === 'DAMAGED' ? '#EF4444' : reason === 'OVERFLOW' ? '#F59E0B' : T.colors.secondaryLabel,
                      }}>
                        {reason}
                      </Text>
                    </TouchableOpacity>
                  ))}
                </View>
              )}
            </View>

            {/* Manifest checklist */}
            <ScrollView className="flex-1 px-8 py-4">
              {selectedManifest.map(item => (
                <TouchableOpacity
                  key={item.id}
                  onPress={() => toggleCheck(item.id)}
                  style={{
                    flexDirection: 'row',
                    alignItems: 'center',
                    paddingVertical: 16,
                    borderBottomWidth: isIOS ? 0.33 : 1,
                    borderBottomColor: T.colors.separator,
                    opacity: item.scanned ? 0.4 : 1,
                  }}
                >
                  {/* Checkbox */}
                  <View style={{
                    width: 22,
                    height: 22,
                    borderRadius: T.radius.checkbox,
                    borderWidth: item.scanned ? 0 : (isIOS ? 1.5 : 2),
                    borderColor: item.scanned ? 'transparent' : T.colors.tertiaryLabel,
                    backgroundColor: item.scanned ? T.colors.accent : 'transparent',
                    marginRight: 16,
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}>
                    {item.scanned && (
                      <Text style={{ color: '#FFFFFF', fontWeight: '700', fontSize: 12 }}>✓</Text>
                    )}
                  </View>
                  <View>
                    <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, letterSpacing: 0.5 }}>
                      {item.brand}
                    </Text>
                    <Text style={{ fontWeight: '600', fontSize: 15, color: T.colors.label, marginTop: 2 }}>
                      {item.label}
                    </Text>
                  </View>
                </TouchableOpacity>
              ))}
            </ScrollView>

            {/* Seal button — per-order (legacy) + manifest-level (LEO) */}
            <View style={{ paddingHorizontal: 32, paddingVertical: 20, borderTopWidth: isIOS ? 0.33 : 1, borderTopColor: T.colors.separator }}>
              {/* Per-order seal (legacy — when no manifest entity exists) */}
              {!manifestId && (
                <TouchableOpacity
                  onPress={handleSeal}
                  disabled={!allChecked || isSealing}
                  style={{
                    paddingVertical: 16,
                    alignItems: 'center',
                    backgroundColor: allChecked && !isSealing ? T.colors.accent : T.colors.fillSecondary,
                    borderRadius: T.radius.button,
                  }}
                >
                  <Text style={{
                    fontWeight: '600',
                    fontSize: 14,
                    letterSpacing: isIOS ? 0.3 : 1,
                    color: allChecked && !isSealing ? '#FFFFFF' : T.colors.tertiaryLabel,
                  }}>
                    {isSealing ? (isIOS ? 'Sealing...' : 'SEALING...') : (isIOS ? 'Mark as Loaded' : 'MARK AS LOADED')}
                  </Text>
                </TouchableOpacity>
              )}
              {/* Manifest-level seal (LEO — slide to seal entire manifest) */}
              {manifestId && manifestState === 'LOADING' && (
                <View style={{ gap: 10 }}>
                  {/* Inject order button */}
                  <TouchableOpacity
                    onPress={() => setShowInjectOrder(true)}
                    style={{
                      paddingVertical: 14,
                      alignItems: 'center',
                      backgroundColor: T.colors.fillSecondary,
                      borderRadius: T.radius.button,
                      borderWidth: 1,
                      borderColor: T.colors.accent,
                      flexDirection: 'row',
                      justifyContent: 'center',
                      gap: 8,
                    }}
                  >
                    <MaterialIcons name="add-circle-outline" size={18} color={T.colors.accent} />
                    <Text style={{
                      fontWeight: '600',
                      fontSize: 13,
                      letterSpacing: isIOS ? 0.3 : 1,
                      color: T.colors.accent,
                    }}>
                      {isIOS ? 'Add Order' : 'ADD ORDER'}
                    </Text>
                  </TouchableOpacity>
                  {/* Seal manifest button */}
                  <TouchableOpacity
                    onPress={handleManifestSeal}
                    disabled={isSealingManifest}
                    style={{
                      paddingVertical: 18,
                      alignItems: 'center',
                      backgroundColor: isSealingManifest ? T.colors.fillSecondary : '#16A34A',
                      borderRadius: T.radius.button,
                      flexDirection: 'row',
                      justifyContent: 'center',
                      gap: 8,
                    }}
                  >
                    <MaterialIcons name="verified" size={18} color="#FFFFFF" />
                    <Text style={{
                      fontWeight: '700',
                      fontSize: 14,
                      letterSpacing: isIOS ? 0.3 : 1.2,
                      color: '#FFFFFF',
                    }}>
                      {isSealingManifest ? (isIOS ? 'Sealing Manifest...' : 'SEALING MANIFEST...') : (isIOS ? 'Seal Manifest' : 'SEAL MANIFEST')}
                    </Text>
                  </TouchableOpacity>
                </View>
              )}
              {manifestId && manifestState === 'SEALED' && (
                <View style={{ paddingVertical: 14, alignItems: 'center', backgroundColor: T.colors.fillTertiary, borderRadius: T.radius.button }}>
                  <Text style={{ fontWeight: '600', fontSize: 13, color: '#16A34A', letterSpacing: 0.5 }}>
                    {isIOS ? 'Manifest Sealed — Route Finalized' : 'MANIFEST SEALED — ROUTE FINALIZED'}
                  </Text>
                </View>
              )}
            </View>
          </>
        ) : (
          <View className="flex-1 items-center justify-center">
            <Text style={{ fontSize: 13, color: T.colors.tertiaryLabel, letterSpacing: 0.3 }}>
              {isLoading ? (isIOS ? 'Fetching...' : 'FETCHING...') : (isIOS ? 'Select order from manifest' : 'SELECT ORDER FROM MANIFEST')}
            </Text>
          </View>
        )}
      </View>

      {/* ── Inject Order Modal ────────────────────────────────────────── */}
      <Modal visible={showInjectOrder} transparent animationType="fade" onRequestClose={() => setShowInjectOrder(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 400, backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden' }}>
            <View style={{ paddingHorizontal: 24, paddingVertical: 16, borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}>
              <Text style={{ fontWeight: '700', fontSize: 17, color: T.colors.label }}>
                {isIOS ? 'Add Order to Manifest' : 'ADD ORDER TO MANIFEST'}
              </Text>
              <Text style={{ fontSize: 12, color: T.colors.secondaryLabel, marginTop: 4 }}>
                Enter the Order ID to inject into the active loading session.
              </Text>
            </View>
            <View style={{ padding: 24, gap: 16 }}>
              <TextInput
                value={injectOrderId}
                onChangeText={setInjectOrderId}
                placeholder="Order ID (UUID)"
                placeholderTextColor={T.colors.tertiaryLabel}
                autoCapitalize="none"
                autoCorrect={false}
                style={{
                  fontFamily: T.typography.mono.fontFamily,
                  fontSize: 14,
                  color: T.colors.label,
                  backgroundColor: T.colors.fillTertiary,
                  borderRadius: (T.radius as any).input || 8,
                  paddingHorizontal: 16,
                  paddingVertical: 12,
                  borderWidth: 1,
                  borderColor: T.colors.separator,
                }}
              />
              {!isOnline && (
                <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6, paddingVertical: 4 }}>
                  <MaterialIcons name="cloud-off" size={14} color="#F59E0B" />
                  <Text style={{ fontSize: 11, color: '#F59E0B' }}>Offline — action will be queued</Text>
                </View>
              )}
              <View style={{ flexDirection: 'row', gap: 12 }}>
                <TouchableOpacity
                  onPress={() => { setShowInjectOrder(false); setInjectOrderId(''); }}
                  style={{ flex: 1, paddingVertical: 14, alignItems: 'center', backgroundColor: T.colors.fillSecondary, borderRadius: T.radius.button }}
                >
                  <Text style={{ fontWeight: '600', fontSize: 14, color: T.colors.secondaryLabel }}>Cancel</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  onPress={handleInjectOrder}
                  disabled={!injectOrderId.trim() || isInjecting}
                  style={{
                    flex: 1,
                    paddingVertical: 14,
                    alignItems: 'center',
                    backgroundColor: injectOrderId.trim() && !isInjecting ? T.colors.accent : T.colors.fillSecondary,
                    borderRadius: T.radius.button,
                  }}
                >
                  <Text style={{ fontWeight: '700', fontSize: 14, color: injectOrderId.trim() && !isInjecting ? '#FFFFFF' : T.colors.tertiaryLabel }}>
                    {isInjecting ? 'Adding...' : 'Add Order'}
                  </Text>
                </TouchableOpacity>
              </View>
            </View>
          </View>
        </View>
      </Modal>

      {/* ── Offline Queue Indicator ────────────────────────────────────── */}
      {offlineQueue.length > 0 && (
        <View style={{
          position: 'absolute', bottom: 12, left: 12,
          flexDirection: 'row', alignItems: 'center', gap: 6,
          backgroundColor: 'rgba(245, 158, 11, 0.95)', paddingHorizontal: 12, paddingVertical: 6,
          borderRadius: 8,
        }}>
          <MaterialIcons name="cloud-queue" size={14} color="#FFFFFF" />
          <Text style={{ fontSize: 11, fontWeight: '600', color: '#FFFFFF' }}>
            {offlineQueue.length} queued action{offlineQueue.length > 1 ? 's' : ''} pending sync
          </Text>
        </View>
      )}

      {/* ── Re-Dispatch Modal ────────────────────────────────────────── */}
      <Modal visible={showReDispatch} transparent animationType="fade" onRequestClose={() => { setShowReDispatch(false); setReDispatchOrderId(null); }}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 520, maxHeight: '85%', backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden' }}>
            {/* Header */}
            <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 24, paddingVertical: 16, borderBottomWidth: isIOS ? 0.33 : 1, borderBottomColor: T.colors.separator }}>
              <View style={{ flex: 1 }}>
                <Text style={{ fontWeight: '700', fontSize: 17, color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0 }}>
                  {isIOS ? 'Re-Dispatch Order' : 'RE-DISPATCH ORDER'}
                </Text>
                <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, marginTop: 4, letterSpacing: 0.5 }}>
                  {reDispatchOrderId}
                </Text>
                {reDispatchRetailer ? (
                  <Text style={{ fontSize: 12, color: T.colors.secondaryLabel, marginTop: 2 }}>
                    {reDispatchRetailer} · {reDispatchVolume.toFixed(1)} VU
                  </Text>
                ) : null}
              </View>
              <TouchableOpacity onPress={() => { setShowReDispatch(false); setReDispatchOrderId(null); }} style={{ padding: 8 }}>
                <MaterialIcons name="close" size={22} color={T.colors.tertiaryLabel} />
              </TouchableOpacity>
            </View>

            {/* Recommendation list */}
            {isLoadingRecs ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <Text style={{ color: T.colors.tertiaryLabel, fontFamily: T.typography.mono.fontFamily, fontSize: 12, letterSpacing: 0.3 }}>
                  {isIOS ? 'Analyzing fleet positions...' : 'ANALYZING FLEET POSITIONS...'}
                </Text>
              </View>
            ) : recommendations.length === 0 ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <MaterialIcons name="warning" size={32} color={T.colors.tertiaryLabel} />
                <Text style={{ color: T.colors.tertiaryLabel, fontSize: 13, marginTop: 12, textAlign: 'center' }}>
                  {isIOS ? 'No available trucks found' : 'NO AVAILABLE TRUCKS FOUND'}
                </Text>
              </View>
            ) : (
              <FlatList
                data={recommendations}
                keyExtractor={item => item.driver_id}
                style={{ maxHeight: 400 }}
                renderItem={({ item, index }) => {
                  const isBest = index === 0;
                  const fits = item.free_volume_vu >= reDispatchVolume;
                  const isMaintenance = item.truck_status === 'MAINTENANCE';
                  return (
                    <TouchableOpacity
                      onPress={() => { if (!isMaintenance && !isReassigning) handleReassign(item.driver_id, item.vehicle_id); }}
                      disabled={isMaintenance || isReassigning}
                      style={{
                        flexDirection: 'row',
                        alignItems: 'center',
                        paddingHorizontal: 24,
                        paddingVertical: 16,
                        borderBottomWidth: isIOS ? 0.33 : 1,
                        borderBottomColor: T.colors.separator,
                        opacity: isMaintenance ? 0.4 : 1,
                        backgroundColor: isBest ? `${T.colors.accent}08` : 'transparent',
                      }}
                    >
                      {/* Rank badge */}
                      <View style={{
                        width: 28,
                        height: 28,
                        borderRadius: 14,
                        backgroundColor: isBest ? T.colors.accent : T.colors.fillTertiary,
                        alignItems: 'center',
                        justifyContent: 'center',
                        marginRight: 16,
                      }}>
                        <Text style={{ fontWeight: '700', fontSize: 12, color: isBest ? '#FFFFFF' : T.colors.secondaryLabel }}>
                          {index + 1}
                        </Text>
                      </View>

                      {/* Truck info */}
                      <View style={{ flex: 1 }}>
                        <View style={{ flexDirection: 'row', alignItems: 'center' }}>
                          <Text style={{ fontWeight: '600', fontSize: 14, color: T.colors.label }}>
                            {item.driver_name}
                          </Text>
                          {isBest && (
                            <View style={{ marginLeft: 8, backgroundColor: T.colors.accent, borderRadius: 4, paddingHorizontal: 6, paddingVertical: 2 }}>
                              <Text style={{ fontSize: 9, fontWeight: '700', color: '#FFFFFF', letterSpacing: 0.5 }}>
                                {isIOS ? 'Best' : 'BEST'}
                              </Text>
                            </View>
                          )}
                          {isMaintenance && (
                            <View style={{ marginLeft: 8, backgroundColor: T.colors.destructive, borderRadius: 4, paddingHorizontal: 6, paddingVertical: 2 }}>
                              <Text style={{ fontSize: 9, fontWeight: '700', color: '#FFFFFF', letterSpacing: 0.5 }}>
                                {isIOS ? 'Maintenance' : 'MAINTENANCE'}
                              </Text>
                            </View>
                          )}
                        </View>
                        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel, marginTop: 3, letterSpacing: 0.3 }}>
                          {item.license_plate} · {item.vehicle_class}
                        </Text>
                        <Text style={{ fontSize: 11, color: T.colors.secondaryLabel, marginTop: 2 }}>
                          {item.recommendation}
                        </Text>
                      </View>

                      {/* Metrics */}
                      <View style={{ alignItems: 'flex-end', marginLeft: 12 }}>
                        {item.distance_km >= 0 ? (
                          <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 13, fontWeight: '600', color: T.colors.label }}>
                            {item.distance_km < 1 ? `${(item.distance_km * 1000).toFixed(0)}m` : `${item.distance_km.toFixed(1)}km`}
                          </Text>
                        ) : (
                          <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 11, color: T.colors.tertiaryLabel }}>
                            {isIOS ? 'No GPS' : 'NO GPS'}
                          </Text>
                        )}
                        <Text style={{
                          fontFamily: T.typography.mono.fontFamily,
                          fontSize: 11,
                          marginTop: 2,
                          color: fits ? T.colors.success : T.colors.destructive,
                        }}>
                          {item.free_volume_vu.toFixed(1)} VU free
                        </Text>
                        <Text style={{ fontFamily: T.typography.mono.fontFamily, fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 1 }}>
                          {item.order_count} orders
                        </Text>
                      </View>
                    </TouchableOpacity>
                  );
                }}
              />
            )}

            {/* Footer hint */}
            <View style={{ paddingHorizontal: 24, paddingVertical: 12, borderTopWidth: isIOS ? 0.33 : 1, borderTopColor: T.colors.separator }}>
              <Text style={{ fontSize: 11, color: T.colors.tertiaryLabel, textAlign: 'center', letterSpacing: 0.2 }}>
                {isIOS ? 'Tap a truck to reassign this order' : 'TAP A TRUCK TO REASSIGN THIS ORDER'}
              </Text>
            </View>
          </View>
        </View>
      </Modal>

      {/* ── Notification Panel Modal ─────────────────────────────────────── */}
      <Modal visible={showNotifPanel} transparent animationType="fade" onRequestClose={() => setShowNotifPanel(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.4)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 420, maxHeight: '80%', backgroundColor: T.colors.sidebarBackground, borderRadius: 12, overflow: 'hidden' }}>
            {/* Modal header */}
            <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
              <Text style={{ flex: 1, fontWeight: '700', fontSize: 15, color: T.colors.sidebarLabel, letterSpacing: 0.3 }}>
                {isIOS ? 'Notifications' : 'NOTIFICATIONS'}
              </Text>
              {unreadCount > 0 && (
                <TouchableOpacity onPress={markAllNotifsRead} style={{ marginRight: 12 }}>
                  <Text style={{ fontSize: 12, color: T.colors.accent, fontWeight: '600' }}>Mark all read</Text>
                </TouchableOpacity>
              )}
              <TouchableOpacity onPress={() => setShowNotifPanel(false)}>
                <MaterialIcons name="close" size={20} color={T.colors.sidebarSecondary} />
              </TouchableOpacity>
            </View>
            {/* Notification list */}
            <FlatList
              data={notifications}
              keyExtractor={item => item.id}
              ListEmptyComponent={
                <View style={{ padding: 40, alignItems: 'center' }}>
                  <Text style={{ color: T.colors.sidebarSecondary, fontSize: 13 }}>No notifications</Text>
                </View>
              }
              renderItem={({ item }) => {
                const isUnread = !item.read_at;
                const iconName: keyof typeof MaterialIcons.glyphMap =
                  item.type === 'PAYLOAD_READY_TO_SEAL' ? 'inventory' :
                  item.type === 'PAYLOAD_SEALED' ? 'verified' :
                  item.type === 'ORDER_DISPATCHED' ? 'local-shipping' :
                  item.type === 'ORDER_COMPLETED' ? 'check-circle' :
                  item.type === 'PAYMENT_SETTLED' ? 'payments' :
                  item.type === 'PAYMENT_FAILED' ? 'error' :
                  'notifications';
                return (
                  <TouchableOpacity
                    onPress={() => { if (isUnread) markNotifRead(item.id); }}
                    style={{
                      flexDirection: 'row',
                      paddingHorizontal: 20,
                      paddingVertical: 12,
                      borderBottomWidth: 0.5,
                      borderBottomColor: T.colors.sidebarSeparator,
                      backgroundColor: isUnread ? `${T.colors.accent}10` : 'transparent',
                    }}
                  >
                    <MaterialIcons name={iconName} size={18} color={isUnread ? T.colors.accent : T.colors.sidebarSecondary} style={{ marginRight: 12, marginTop: 2 }} />
                    <View style={{ flex: 1 }}>
                      <Text style={{ fontWeight: isUnread ? '700' : '500', fontSize: 13, color: T.colors.sidebarLabel, marginBottom: 2 }}>{item.title}</Text>
                      <Text style={{ fontSize: 12, color: T.colors.sidebarSecondary }} numberOfLines={2}>{item.body}</Text>
                      <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4 }}>{new Date(item.created_at).toLocaleString()}</Text>
                    </View>
                    {isUnread && <View style={{ width: 8, height: 8, borderRadius: 4, backgroundColor: T.colors.accent, alignSelf: 'center', marginLeft: 8 }} />}
                  </TouchableOpacity>
                );
              }}
            />
          </View>
        </View>
      </Modal>
    </View>
  );
}
