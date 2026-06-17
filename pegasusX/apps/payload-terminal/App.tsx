import { useState, useEffect, useCallback, useMemo, useRef, type ComponentProps } from 'react';
import { Text, View, Pressable as RNPressable, AppState, Alert, ScrollView, TextInput, Modal, FlatList, Animated, PanResponder, useWindowDimensions } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import * as ScreenOrientation from 'expo-screen-orientation';
import * as SecureStore from 'expo-secure-store';
import { CameraView } from 'expo-camera';
import "./global.css";
import ConnectionStrip from './components/ConnectionStrip';
import ManifestKpiGrid from './components/ManifestKpiGrid';
import PayloadStatePanel from './components/PayloadStatePanel';
import { SkeletonList } from './components/SkeletonPulse';
import StatusBadge, { exceptionReasonTone } from './components/StatusBadge';
import WorkflowSectionHeader from './components/WorkflowSectionHeader';
import { useT, isIOS } from './theme';
import { extractProblemMessage, getPayloadTranslator, resolvePayloadLocale } from './localization';
import * as Updates from 'expo-updates';
import { buildManifest, type LiveOrder, type ManifestItem } from './utils/manifest';
import { PayloadTerminalApi } from './api';
import { authFetch, clearPayloaderSession, savePayloaderSession, setTokenRefreshListener } from './authSession';
import {
  payloadApplyReassignKey,
  payloadManifestExceptionKey,
  payloadMissingItemsKey,
  payloadOrderSealKey,
  payloadRecommendReassignKey,
  payloadSealCompletedKey,
  payloadSupplierInjectKey,
  payloadSupplierStartLoadingKey,
} from './utils/idempotency';
import { InboundReturnsPanel } from './inboundReturns';
import { resetPhoneOtpFlow, sendPhoneOtp, verifyPhoneOtp } from './firebaseAuth';
import { reconcilePayloadSession } from './session-reconcile';
import { registerPayloadPushTokens } from './pushRegistration';
import { defaultLocale, type Locale } from '../../packages/i18n/locales';

// ─── API ──────────────────────────────────────────────────────────────────────
// Resolution order for the backend base URL:
//   1. EXPO_PUBLIC_API_URL env var (set in .env or via `npx expo start --dev-client`)
//      — required for physical devices so they can reach the Mac's LAN IP.
//   2. __DEV__ fallback = http://localhost:8080 (simulator only).
//   3. Production = https://api.pegasus.uz (legacy override still supported via EXPO_PUBLIC_API_URL).
const API_BASE = (process.env.EXPO_PUBLIC_API_URL?.trim() || '') ||
  (__DEV__ ? 'http://localhost:8180' : 'https://api.pegasus.uz');

function reconnectDelayMs(attempt: number, baseMs = 3_000, maxMs = 60_000): number {
  const capped = Math.min(Math.max(attempt, 0), 10);
  const exp = Math.min(baseMs * 2 ** capped, maxMs);
  return exp + Math.floor(Math.random() * (exp / 2 + 1));
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));
const defaultPressFeedback = { opacity: 0.82, transform: [{ scale: 0.97 }] } as const;

function Pressable(props: ComponentProps<typeof RNPressable>) {
  const { style, ...rest } = props;
  return (
    <RNPressable
      {...rest}
      style={(state) => {
        if (typeof style === 'function') return style(state);
        return [style, state.pressed ? defaultPressFeedback : null];
      }}
    />
  );
}

// ─── Main Component ───────────────────────────────────────────────────────────

export default function App() {
  const T = useT();
  const { width, height } = useWindowDimensions();
  const isTabletLayout = Math.min(width, height) >= 768;

  // ─── OTA Updates ──────────────────────────────────────────────────────────────
  useEffect(() => {
    async function onFetchUpdateAsync() {
      try {
        const update = await Updates.checkForUpdateAsync();
        if (update.isAvailable) {
          await Updates.fetchUpdateAsync();
          Alert.alert(
            'Update Available',
            'A new version has been downloaded. Restart the app to apply the update?',
            [
              { text: 'Cancel', style: 'cancel' },
              { text: 'Restart', onPress: () => Updates.reloadAsync() }
            ]
          );
        }
      } catch (error) {
        // Silent fail on OTA check (network error, etc.)
        console.log(`Error fetching latest Expo update: ${error}`);
      }
    }
    if (!__DEV__) {
      onFetchUpdateAsync();
    }
  }, []);

  const toastMotionProfile = useMemo(
    () => isTabletLayout
      ? {
          startOffsetY: 18,
          hiddenOffsetY: 12,
          initialScale: 0.975,
          enterDuration: 250,
          exitDuration: 170,
          swipeDismissDuration: 190,
          swipeSnapDuration: 170,
          swipeStartThreshold: 8,
          swipeDismissDistance: 124,
          swipeDismissVelocity: 1.2,
          maxDragDistance: 260,
          swipeDismissTravel: 520,
          swipeSnapFriction: 8,
          swipeSnapTension: 110,
          dragOpacityDistance: 300,
          defaultDurationMs: 3200,
        }
      : {
          startOffsetY: 10,
          hiddenOffsetY: 7,
          initialScale: 0.97,
          enterDuration: 165,
          exitDuration: 115,
          swipeDismissDuration: 125,
          swipeSnapDuration: 110,
          swipeStartThreshold: 4,
          swipeDismissDistance: 76,
          swipeDismissVelocity: 0.88,
          maxDragDistance: 200,
          swipeDismissTravel: 340,
          swipeSnapFriction: 6,
          swipeSnapTension: 158,
          dragOpacityDistance: 220,
          defaultDurationMs: 2300,
        },
    [isTabletLayout]
  );

  const [locale, setLocale] = useState<Locale>(defaultLocale);
  const tx = useMemo(() => getPayloadTranslator(locale), [locale]);

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
    manifest_id?: string;
    warehouse_id?: string;
    reason?: string;
    timestamp?: string;
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
  const [otpInput, setOtpInput] = useState('');
  const [loginMode, setLoginMode] = useState<'otp' | 'pin'>('otp');
  const [otpSent, setOtpSent] = useState(false);
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const [authLoading, setAuthLoading] = useState(true);

  // Supplier context
  const [supplierId, setSupplierId] = useState<string | null>(null);

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
  const [batchSealing, setBatchSealing] = useState(false);

  // UI state
  const [workspaceMode, setWorkspaceMode] = useState<'outbound' | 'inbound'>('outbound');
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
  const [manifestStopCount, setManifestStopCount] = useState(0);
  const [manifestRegionCode, setManifestRegionCode] = useState('');
  const [isStartingLoad, setIsStartingLoad] = useState(false);
  const [isSealingManifest, setIsSealingManifest] = useState(false);
  const [exceptionLoading, setExceptionLoading] = useState<string | null>(null); // orderId being excepted
  const [showInjectOrder, setShowInjectOrder] = useState(false);
  const [showProductScanner, setShowProductScanner] = useState(false);
  const [showInjectScanner, setShowInjectScanner] = useState(false);
  const [injectOrderId, setInjectOrderId] = useState('');
  const [isInjecting, setIsInjecting] = useState(false);

  // Offline action queue — persisted in SecureStore, flushed on reconnect
  type QueuedAction = { id: string; endpoint: string; method: string; body: string; createdAt: number };
  const [offlineQueue, setOfflineQueue] = useState<QueuedAction[]>([]);
  const [isOnline, setIsOnline] = useState(true);

  // Lightweight in-app toast for non-blocking feedback.
  type UiToastTone = 'info' | 'success' | 'warning' | 'error';
  type UiToast = { id: number; title: string; message?: string; tone: UiToastTone };
  const [uiToast, setUiToast] = useState<UiToast | null>(null);
  const toastTranslateX = useRef(new Animated.Value(0)).current;
  const toastTranslateY = useRef(new Animated.Value(toastMotionProfile.startOffsetY)).current;
  const toastOpacity = useRef(new Animated.Value(0)).current;
  const toastScale = useRef(new Animated.Value(toastMotionProfile.initialScale)).current;
  const toastTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const toastIdRef = useRef(0);
  const activeToastIdRef = useRef<number | null>(null);

  // Notification state
  type NotifItem = { id: string; type: string; title: string; body: string; read_at: string | null; created_at: string };
  const [notifications, setNotifications] = useState<NotifItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [showNotifPanel, setShowNotifPanel] = useState(false);
  type ManifestExceptionItem = {
    exception_id: string;
    manifest_id: string;
    order_id: string;
    reason: string;
    attempt_count: number;
    escalated: boolean;
    created_at: string;
  };
  const [showExceptionsPanel, setShowExceptionsPanel] = useState(false);
  const [manifestExceptions, setManifestExceptions] = useState<ManifestExceptionItem[]>([]);
  const [loadingExceptions, setLoadingExceptions] = useState(false);
  const [liveSyncRevision, setLiveSyncRevision] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const wsHasConnectedOnceRef = useRef(false);
  const [clientPolicyMessage, setClientPolicyMessage] = useState<string | null>(null);

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

  const fetchClientPolicy = useCallback(async () => {
    try {
      const policy = await PayloadTerminalApi.getClientPolicy('expo', '1.0.0');
      if (policy.outdated || policy.force_update) {
        let message = policy.force_update ? 'Update required' : 'Update available';
        if (policy.minimum_version) {
          message += ` — minimum version ${policy.minimum_version}`;
        }
        if (policy.defer_reason) {
          message += `. ${policy.defer_reason}`;
        }
        setClientPolicyMessage(message);
      } else {
        setClientPolicyMessage(null);
      }
    } catch {
      // Policy fetch is optional on local/dev stacks.
    }
  }, []);

  const renderClientPolicyBanner = () => {
    if (!clientPolicyMessage) return null;
    return (
      <View style={{
        backgroundColor: 'rgba(245, 158, 11, 0.14)',
        borderBottomWidth: 1,
        borderBottomColor: 'rgba(245, 158, 11, 0.4)',
        paddingHorizontal: 16,
        paddingVertical: 10,
        flexDirection: 'row',
        alignItems: 'center',
        gap: 8,
      }}>
        <MaterialIcons name="warning-amber" size={18} color="#B45309" />
        <Text style={{ flex: 1, color: '#92400E', fontSize: 13, fontWeight: '600' }}>{clientPolicyMessage}</Text>
      </View>
    );
  };

  // Lock tablet to landscape + restore session on mount
  useEffect(() => {
    setTokenRefreshListener(setToken);
    return () => setTokenRefreshListener(null);
  }, []);

  useEffect(() => {
    ScreenOrientation.lockAsync(ScreenOrientation.OrientationLock.LANDSCAPE_LEFT);
    (async () => {
      try {
        setLocale(await resolvePayloadLocale());
        const saved = await SecureStore.getItemAsync('payloader_token');
        const name = await SecureStore.getItemAsync('payloader_name');
        const sid = await SecureStore.getItemAsync('payloader_supplier_id');
        const wid = await SecureStore.getItemAsync('payloader_warehouse_id');
        const wname = await SecureStore.getItemAsync('payloader_warehouse_name');
        if (saved) {
          setToken(saved);
          setWorkerName(name || 'Payloader');
          if (sid) setSupplierId(sid);
          void registerPayloadPushTokens();
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

  // ── Payloader Login ──────────────────────────────────────────────────────
  const completeLogin = async (data: Record<string, unknown>) => {
    await savePayloaderSession(data as Parameters<typeof savePayloaderSession>[0]);
    setToken(String(data.token ?? ''));
    setWorkerName(String(data.name ?? 'Payloader'));
    if (data.supplier_id) setSupplierId(String(data.supplier_id));
    void registerPayloadPushTokens();
  };

  const handleLoginPin = async () => {
    if (!phoneInput || !pinInput) return;
    setIsLoggingIn(true);
    try {
      const res = await fetch(`${API_BASE}/v1/auth/payloader/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone: phoneInput, pin: pinInput }),
      });
      if (!res.ok) {
        throw new Error(await extractProblemMessage(res, locale));
      }
      const data = await res.json();
      await completeLogin(data);
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleSendOtp = async () => {
    if (!phoneInput.trim()) return;
    setIsLoggingIn(true);
    try {
      await sendPhoneOtp(phoneInput.trim());
      setOtpSent(true);
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleVerifyOtp = async () => {
    if (otpInput.trim().length < 6) return;
    setIsLoggingIn(true);
    try {
      const idToken = await verifyPhoneOtp(otpInput.trim());
      const res = await fetch(`${API_BASE}/v1/auth/payloader/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id_token: idToken }),
      });
      if (!res.ok) {
        throw new Error(await extractProblemMessage(res, locale));
      }
      const data = await res.json();
      await completeLogin(data);
      setOtpSent(false);
      setOtpInput('');
      resetPhoneOtpFlow();
    } catch (e: unknown) {
      showToast(tx('auth.error.login_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleLogout = async () => {
    await clearPayloaderSession();
    setToken(null);
    setWorkerName('');
    setSupplierId(null);
    setActiveTruck(null);
    setTrucks([]);
    setIsLoadingTrucks(false);
  };

  // ── Notifications: WebSocket + fetch ───────────────────────────────────
  const fetchNotifications = useCallback(async () => {
    if (!token) return;
    try {
      const pageSize = 100;
      let offset = 0;
      const items: ReturnType<typeof normalizeNotification>[] = [];
      let unreadCount = 0;
      let hasMore = true;
      while (hasMore && offset < 2500) {
        const res = await authFetch(`/v1/user/notifications?limit=${pageSize}&offset=${offset}`);
        if (!res.ok) return;
        const data = await res.json();
        const page = Array.isArray(data.notifications)
          ? data.notifications.map((item: BackendNotifItem) => normalizeNotification(item))
          : [];
        items.push(...page);
        unreadCount = data.unread_count ?? unreadCount;
        hasMore = Boolean(data.has_more);
        offset += pageSize;
      }
      setNotifications(items);
      setUnreadCount(unreadCount);
    } catch {}
  }, [token]);

  useEffect(() => {
    fetchClientPolicy();
  }, [fetchClientPolicy]);

  useEffect(() => {
    if (!token) return;
    fetchClientPolicy();
    void registerPayloadPushTokens();
  }, [token, fetchClientPolicy]);

  useEffect(() => {
    if (!token) return;
    fetchNotifications();
    const wsUrl = `${API_BASE.replace(/^http/, 'ws')}/v1/ws?token=${encodeURIComponent(token)}`;
    let reconnectTimer: ReturnType<typeof setTimeout>;
    let reconnectAttempt = 0;
    const connect = () => {
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
        reconnectTimer = setTimeout(connect, delay);
      };
      ws.onerror = () => { setIsOnline(false); ws.close(); };
    };
    connect();
    return () => {
      clearTimeout(reconnectTimer);
      wsRef.current?.close();
      wsRef.current = null;
      wsHasConnectedOnceRef.current = false;
    };
  }, [token, fetchNotifications]);

  const markNotifRead = useCallback(async (id: string) => {
    if (!token) return;
    setNotifications(prev => prev.map(n => n.id === id ? { ...n, read_at: new Date().toISOString() } : n));
    setUnreadCount(prev => Math.max(0, prev - 1));
    try {
      await authFetch('/v1/user/notifications/read', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notification_ids: [id] }),
      });
    } catch {}
  }, [token]);

  const markAllNotifsRead = useCallback(async () => {
    if (!token) return;
    setNotifications(prev => prev.map(n => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
    setUnreadCount(0);
    try {
      await authFetch('/v1/user/notifications/read', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
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

  const clearToastTimer = useCallback(() => {
    if (toastTimerRef.current) {
      clearTimeout(toastTimerRef.current);
      toastTimerRef.current = null;
    }
  }, []);

  const clearToastImmediate = useCallback(() => {
    activeToastIdRef.current = null;
    setUiToast(null);
    toastTranslateX.setValue(0);
    toastTranslateY.setValue(toastMotionProfile.startOffsetY);
    toastOpacity.setValue(0);
    toastScale.setValue(toastMotionProfile.initialScale);
  }, [toastMotionProfile.initialScale, toastMotionProfile.startOffsetY, toastOpacity, toastScale, toastTranslateX, toastTranslateY]);

  const dismissToast = useCallback((immediate = false) => {
    clearToastTimer();
    if (immediate) {
      clearToastImmediate();
      return;
    }

    Animated.parallel([
      Animated.timing(toastOpacity, { toValue: 0, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
      Animated.timing(toastTranslateY, { toValue: toastMotionProfile.hiddenOffsetY, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
      Animated.timing(toastScale, { toValue: 0.985, duration: toastMotionProfile.exitDuration, useNativeDriver: true }),
    ]).start(() => clearToastImmediate());
  }, [clearToastImmediate, clearToastTimer, toastMotionProfile.exitDuration, toastMotionProfile.hiddenOffsetY, toastOpacity, toastScale, toastTranslateY]);

  const showToast = useCallback((title: string, message?: string, tone: UiToastTone = 'info', durationMs = toastMotionProfile.defaultDurationMs) => {
    clearToastTimer();

    const id = ++toastIdRef.current;
    activeToastIdRef.current = id;
    setUiToast({ id, title, message, tone });

    toastTranslateX.setValue(0);
    toastTranslateY.setValue(toastMotionProfile.startOffsetY);
    toastOpacity.setValue(0);
    toastScale.setValue(toastMotionProfile.initialScale);

    Animated.parallel([
      Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
      Animated.timing(toastTranslateY, { toValue: 0, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
      Animated.timing(toastScale, { toValue: 1, duration: toastMotionProfile.enterDuration, useNativeDriver: true }),
    ]).start();

    toastTimerRef.current = setTimeout(() => {
      if (activeToastIdRef.current === id) {
        dismissToast();
      }
    }, durationMs);
  }, [clearToastTimer, dismissToast, toastMotionProfile.defaultDurationMs, toastMotionProfile.enterDuration, toastMotionProfile.initialScale, toastMotionProfile.startOffsetY, toastOpacity, toastScale, toastTranslateX, toastTranslateY]);

  const loadManifestExceptions = useCallback(async () => {
    if (!token) return;
    setLoadingExceptions(true);
    try {
      const data = await PayloadTerminalApi.getManifestExceptions(token);
      setManifestExceptions(Array.isArray(data.exceptions) ? data.exceptions : []);
    } catch (e: unknown) {
      showToast('ERROR', e instanceof Error ? e.message : 'Failed to load exceptions', 'error');
    } finally {
      setLoadingExceptions(false);
    }
  }, [token, showToast]);

  useEffect(() => {
    if (!token) return;
    void loadManifestExceptions();
  }, [token, liveSyncRevision, loadManifestExceptions]);

  const toastPanResponder = useMemo(
    () => PanResponder.create({
      onMoveShouldSetPanResponder: (_, gesture) => {
        if (!uiToast) return false;
        return Math.abs(gesture.dx) > toastMotionProfile.swipeStartThreshold && Math.abs(gesture.dx) > Math.abs(gesture.dy);
      },
      onPanResponderMove: (_, gesture) => {
        toastTranslateX.setValue(clamp(gesture.dx, -toastMotionProfile.maxDragDistance, toastMotionProfile.maxDragDistance));
        toastOpacity.setValue(Math.max(0.35, 1 - Math.abs(gesture.dx) / toastMotionProfile.dragOpacityDistance));
      },
      onPanResponderRelease: (_, gesture) => {
        const shouldDismiss =
          Math.abs(gesture.dx) > toastMotionProfile.swipeDismissDistance ||
          Math.abs(gesture.vx) > toastMotionProfile.swipeDismissVelocity;
        if (shouldDismiss) {
          clearToastTimer();
          Animated.parallel([
            Animated.timing(toastTranslateX, {
              toValue: gesture.dx >= 0 ? toastMotionProfile.swipeDismissTravel : -toastMotionProfile.swipeDismissTravel,
              duration: toastMotionProfile.swipeDismissDuration,
              useNativeDriver: true,
            }),
            Animated.timing(toastOpacity, { toValue: 0, duration: toastMotionProfile.swipeDismissDuration, useNativeDriver: true }),
          ]).start(() => dismissToast(true));
          return;
        }

        Animated.parallel([
          Animated.spring(toastTranslateX, {
            toValue: 0,
            friction: toastMotionProfile.swipeSnapFriction,
            tension: toastMotionProfile.swipeSnapTension,
            useNativeDriver: true,
          }),
          Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.swipeSnapDuration, useNativeDriver: true }),
        ]).start();
      },
      onPanResponderTerminate: () => {
        Animated.parallel([
          Animated.spring(toastTranslateX, {
            toValue: 0,
            friction: toastMotionProfile.swipeSnapFriction,
            tension: toastMotionProfile.swipeSnapTension,
            useNativeDriver: true,
          }),
          Animated.timing(toastOpacity, { toValue: 1, duration: toastMotionProfile.swipeSnapDuration, useNativeDriver: true }),
        ]).start();
      },
    }),
    [
      clearToastTimer,
      dismissToast,
      toastMotionProfile.dragOpacityDistance,
      toastMotionProfile.maxDragDistance,
      toastMotionProfile.swipeDismissDistance,
      toastMotionProfile.swipeDismissDuration,
      toastMotionProfile.swipeDismissTravel,
      toastMotionProfile.swipeDismissVelocity,
      toastMotionProfile.swipeSnapDuration,
      toastMotionProfile.swipeSnapFriction,
      toastMotionProfile.swipeSnapTension,
      toastMotionProfile.swipeStartThreshold,
      toastOpacity,
      toastTranslateX,
      uiToast,
    ]
  );

  const renderUiToast = () => {
    if (!uiToast) return null;

    const toneStyles: Record<UiToastTone, {
      bg: string;
      border: string;
      title: string;
      message: string;
      icon: keyof typeof MaterialIcons.glyphMap;
    }> = {
      info: {
        bg: T.colors.cardBackground,
        border: T.colors.separator,
        title: T.colors.label,
        message: T.colors.secondaryLabel,
        icon: 'info-outline',
      },
      success: {
        bg: 'rgba(22, 163, 74, 0.12)',
        border: 'rgba(22, 163, 74, 0.35)',
        title: '#166534',
        message: '#15803D',
        icon: 'check-circle',
      },
      warning: {
        bg: 'rgba(245, 158, 11, 0.14)',
        border: 'rgba(245, 158, 11, 0.4)',
        title: '#92400E',
        message: '#B45309',
        icon: 'warning-amber',
      },
      error: {
        bg: 'rgba(239, 68, 68, 0.14)',
        border: 'rgba(239, 68, 68, 0.45)',
        title: '#991B1B',
        message: '#B91C1C',
        icon: 'error-outline',
      },
    };

    const tone = toneStyles[uiToast.tone];

    return (
      <View pointerEvents="box-none" style={{ position: 'absolute', left: 0, right: 0, bottom: 14, alignItems: 'center', zIndex: 1200, elevation: 1200 }}>
        <Animated.View
          {...toastPanResponder.panHandlers}
          style={{
            width: '86%',
            maxWidth: 560,
            minHeight: 62,
            borderRadius: isIOS ? 18 : 14,
            borderWidth: 1,
            borderColor: tone.border,
            backgroundColor: tone.bg,
            paddingHorizontal: 14,
            paddingVertical: 12,
            flexDirection: 'row',
            alignItems: 'flex-start',
            opacity: toastOpacity,
            transform: [
              { translateX: toastTranslateX },
              { translateY: toastTranslateY },
              { scale: toastScale },
            ],
            ...T.shadow.card,
          }}
        >
          <MaterialIcons name={tone.icon} size={18} color={tone.title} style={{ marginTop: 1, marginRight: 10 }} />
          <View style={{ flex: 1 }}>
            <Text style={{ color: tone.title, fontWeight: '700', fontSize: 13, letterSpacing: 0.2 }} numberOfLines={2}>
              {uiToast.title}
            </Text>
            {uiToast.message ? (
              <Text style={{ color: tone.message, fontSize: 12, marginTop: 3, lineHeight: 17 }} numberOfLines={2}>
                {uiToast.message}
              </Text>
            ) : null}
          </View>
          <Pressable onPress={() => dismissToast()} style={({ pressed }) => ({ marginLeft: 10, padding: 2, opacity: pressed ? 0.7 : 1 })}>
            <MaterialIcons name="close" size={16} color={tone.message} />
          </Pressable>
        </Animated.View>
      </View>
    );
  };

  useEffect(() => {
    return () => {
      clearToastTimer();
    };
  }, [clearToastTimer]);

  // ── Re-dispatch: fetch recommendations ──────────────────────────────────
  const openReDispatch = useCallback(async (orderId: string) => {
    setReDispatchOrderId(orderId);
    setShowReDispatch(true);
    setRecommendations([]);
    setReDispatchRetailer('');
    setReDispatchVolume(0);
    setIsLoadingRecs(true);
    try {
      const res = await authFetch('/v1/payloader/recommend-reassign', {
        method: 'POST',
        headers: {
          'Idempotency-Key': payloadRecommendReassignKey(orderId),
        },
        body: JSON.stringify({ order_id: orderId }),
      });
      if (!res.ok) throw new Error(await extractProblemMessage(res, locale));
      const data = await res.json();
      setRecommendations(data.recommendations ?? []);
      setReDispatchRetailer(data.retailer_name ?? '');
      setReDispatchVolume(data.order_volume_vu ?? 0);
    } catch (e: unknown) {
      showToast(tx('payload.alert.recommendation_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsLoadingRecs(false);
    }
  }, [authHeaders, locale, showToast, tx]);

  const handleReassign = useCallback(async (newDriverId: string, _newVehicleId: string) => {
    if (!reDispatchOrderId || !token) return;
    setIsReassigning(true);
    try {
      const res = await authFetch('/v1/payloader/reassign-order', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': payloadApplyReassignKey(reDispatchOrderId, newDriverId),
        },
        body: JSON.stringify({
          order_id: reDispatchOrderId,
          to_driver_id: newDriverId,
          reason: 'payload-redispatch',
        }),
      });
      if (!res.ok) {
        throw new Error(await extractProblemMessage(res, locale));
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
      showToast(tx('payload.alert.reassign_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsReassigning(false);
    }
  }, [authHeaders, locale, orders, reDispatchOrderId, sealedOrderIds, selectedOrderId, showToast, token, tx]);

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
      const fetchManifestForState = async (state: 'DRAFT' | 'LOADING') => {
        const data = await PayloadTerminalApi.getSupplierManifests(token, state);
        return (data.manifests || []).find((m: any) => m.truck_id === activeTruck) ?? null;
      };

      const m = await fetchManifestForState('DRAFT') ?? await fetchManifestForState('LOADING');
      if (m) {
        setManifestId(m.manifest_id);
        setManifestState(m.state);
        setManifestVolume(m.total_volume_vu || 0);
        setManifestMaxVolume(m.max_volume_vu || 0);
        setManifestStopCount(m.stop_count || 0);
        setManifestRegionCode(m.region_code || '');
      }
    } catch {}
  }, [token, activeTruck]);

  const refreshBatchReadyManifests = useCallback(async () => {
    if (!token || !activeTruck) return;
    try {
      const data = await PayloadTerminalApi.getSupplierManifests(token, 'LOADING');
      const loadingManifests: Array<{ manifest_id: string; truck_id?: string }> = data.manifests || [];
      const sealedByTruck = { ...sealedOrdersByTruck };
      if (sealedOrderIds.size > 0) {
        sealedByTruck[activeTruck] = Array.from(sealedOrderIds);
      }
      const ready: string[] = [];
      for (const m of loadingManifests) {
        const truckId = m.truck_id;
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

  // ── LEO: Start Loading (DRAFT → LOADING) ─────────────────────────────
  const handleStartLoading = async () => {
    if (!manifestId || !token) return;
    setIsStartingLoad(true);
    try {
      await PayloadTerminalApi.supplierStartLoading(
        token,
        manifestId,
        payloadSupplierStartLoadingKey(manifestId),
      );
      setManifestState('LOADING');
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch (e: unknown) {
      showToast(tx('payload.alert.start_loading_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsStartingLoad(false);
    }
  };

  // ── LEO: Exception — remove order from manifest ──────────────────────
  const handleException = async (orderId: string, reason: 'OVERFLOW' | 'DAMAGED' | 'MANUAL') => {
    if (!manifestId || !token) return;
    setExceptionLoading(orderId);
    try {
      const res = await authFetch('/v1/payload/manifest-exception', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': payloadManifestExceptionKey(manifestId, orderId),
        },
        body: JSON.stringify({ manifest_id: manifestId, order_id: orderId, reason }),
      });
      if (!res.ok) throw new Error(await extractProblemMessage(res, locale));
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
        showToast('DLQ ESCALATION', `Order ${orderId.slice(0, 8)} escalated after ${data.overflow_count} overflow attempts.`, 'warning', 3800);
      }
      void fetchTruckManifest();
    } catch (e: unknown) {
      showToast(tx('payload.alert.exception_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setExceptionLoading(null);
    }
  };

  // ── LEO: Manifest-level Seal (LOADING → SEALED) ──────────────────────
  const handleManifestSeal = async () => {
    if (!manifestId || !token) return;
    setIsSealingManifest(true);
    try {
      const data = await PayloadTerminalApi.sealCompletedManifests(
        token,
        [manifestId],
        payloadSealCompletedKey([manifestId]),
      );
      const sealed = Array.isArray(data.results)
        ? data.results.find((row: { manifest_id?: string; status?: string }) => row.manifest_id === manifestId && row.status === 'sealed')
        : null;
      setManifestState('SEALED');
      setAllSealed(true);
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      showToast(
        tx('payload.alert.manifest_sealed_title'),
        sealed
          ? `Manifest sealed (${data.sealed_count ?? 1} truck(s)). Route finalized.`
          : `${data.sealed_count ?? 0} truck(s) sealed. Route finalized.`,
        'success',
        3600
      );
    } catch (e: unknown) {
      showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsSealingManifest(false);
    }
  };

  // ── Phase A: Mid-Load Order Injection ─────────────────────────────────
  const handleInjectOrder = async () => {
    if (!manifestId || !token || !injectOrderId.trim()) return;
    setIsInjecting(true);
    try {
      const trimmed = injectOrderId.trim();
      const data = await PayloadTerminalApi.supplierInjectOrder(
        token,
        manifestId,
        trimmed,
        payloadSupplierInjectKey(manifestId, trimmed),
      );
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      showToast(
        tx('payload.alert.order_injected_title'),
        `Order ${injectOrderId.slice(0, 8)} added. ${data.stop_count} stops, ${data.total_volume_vu?.toFixed(1)} VU.`,
        'success',
        3400
      );
      setInjectOrderId('');
      setShowInjectOrder(false);
      // Refresh manifest to pick up new order
      if (activeTruck) {
        fetchManifest(activeTruck);
        void fetchTruckManifest();
      }
    } catch (e: unknown) {
      if (!isOnline) {
        // Offline: queue the action
        const action: QueuedAction = {
          id: payloadSupplierInjectKey(manifestId, injectOrderId.trim()),
          endpoint: `/v1/supplier/manifests/${manifestId}/inject-order`,
          method: 'POST',
          body: JSON.stringify({ order_id: injectOrderId.trim() }),
          createdAt: Date.now(),
        };
        const updated = [...offlineQueue, action];
        setOfflineQueue(updated);
        await SecureStore.setItemAsync('offline_queue', JSON.stringify(updated));
        showToast(tx('payload.alert.queued_offline_title'), tx('payload.alert.queued_offline_body'), 'warning', 3600);
        setInjectOrderId('');
        setShowInjectOrder(false);
      } else {
        showToast(tx('payload.alert.inject_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
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
        const res = await authFetch(action.endpoint, {
          method: action.method,
          headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': action.id,
          },
          body: action.body,
        });
        if (res.ok || res.status === 409) continue;
        if (res.status === 408 || res.status === 429 || res.status >= 500) {
          remaining.push(action); // retryable
        }
        // Non-retryable 4xx are dropped so poison-pill entries cannot block queue drain.
      } catch {
        remaining.push(action);
      }
    }
    setOfflineQueue(remaining);
    await SecureStore.setItemAsync('offline_queue', JSON.stringify(remaining));
    if (remaining.length === 0 && offlineQueue.length > 0) {
      showToast(tx('payload.alert.sync_complete_title'), `${offlineQueue.length} queued actions synced.`, 'success', 3200);
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

  const handleProductBarcodeScan = async (ean: string) => {
    const trimmed = ean.trim();
    if (!trimmed) return;
    if (!selectedOrderId) {
      showToast(isIOS ? 'Select an order first' : 'SELECT AN ORDER FIRST', '', 'warning', 2400);
      return;
    }
    try {
      const product = await PayloadTerminalApi.lookupBarcode(trimmed);
      const sku = product.sku_id ?? product.product_id ?? '';
      const match = selectedManifest.find((item) => item.brand === sku);
      if (!match) {
        showToast(isIOS ? 'SKU not on this order' : 'SKU NOT ON THIS ORDER', '', 'warning', 2400);
        return;
      }
      if (!match.scanned) toggleCheck(match.id);
      showToast(isIOS ? 'Checked item' : 'CHECKED ITEM', product.name ?? sku, 'success', 2200);
      setShowProductScanner(false);
    } catch (e: unknown) {
      showToast(
        isIOS ? 'Barcode lookup failed' : 'BARCODE LOOKUP FAILED',
        e instanceof Error ? e.message : tx('common.error.unknown'),
        'error',
      );
    }
  };

  // ── Seal & dispatch ───────────────────────────────────────────────────────
  const selectedOrder = orders.find(o => o.order_id === selectedOrderId);
  const selectedManifest = manifest.filter(i => i.orderId === selectedOrderId);
  const allChecked = selectedManifest.length > 0 && selectedManifest.every(i => i.scanned);

  const handleSeal = async () => {
    if (!selectedOrderId || !allChecked) return;
    setIsSealing(true);
    try {
      const res = await authFetch('/v1/payload/seal', {
        method: 'POST',
        headers: {
          'Idempotency-Key': payloadOrderSealKey(selectedOrderId),
        },
        body: JSON.stringify({
          order_id: selectedOrderId,
          terminal_id: activeTruck || 'WH-UNKNOWN',
          manifest_cleared: true,
        }),
      });
      if (!res.ok) throw new Error(await extractProblemMessage(res, locale));
      const sealData = await res.json();
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      const next = new Set(sealedOrderIds).add(selectedOrderId);
      setSealedOrderIds(next);
      if (activeTruck) {
        setSealedOrdersByTruck(prev => ({
          ...prev,
          [activeTruck]: Array.from(next),
        }));
      }
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
      showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setIsSealing(false);
    }
  };

  const handleFinalizeBatchSeal = async () => {
    if (batchReadyManifestIds.length < 2 || batchSealing || !token) return;
    setBatchSealing(true);
    const manifestCount = batchReadyManifestIds.length;
    try {
      const data = await PayloadTerminalApi.sealCompletedManifests(
        token,
        batchReadyManifestIds,
        payloadSealCompletedKey(batchReadyManifestIds),
      );
      setBatchReadyManifestIds([]);
      setAllSealed(true);
      setManifestState('SEALED');
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      showToast(
        tx('payload.alert.manifest_sealed_title'),
        `${data.sealed_count ?? manifestCount} truck(s) sealed. Route finalized.`,
        'success',
        3600,
      );
    } catch (e: unknown) {
      showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
    } finally {
      setBatchSealing(false);
    }
  };

  // ── Render: AUTH LOADING ────────────────────────────────────────────────
  if (authLoading) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background, alignItems: 'center', justifyContent: 'center' }}>
        <PayloadStatePanel
          theme={T}
          variant="sync"
          title={tx('common.status.restoring_session')}
          message={isIOS ? 'Rehydrating the saved operator session and pending queue.' : 'REHYDRATING THE SAVED OPERATOR SESSION AND PENDING QUEUE.'}
        />
        {renderUiToast()}
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
        <Pressable
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
                      await authFetch('/v1/delivery/missing-items', {
                        method: 'POST',
                        headers: {
                          'Idempotency-Key': payloadMissingItemsKey(postSealOrderId ?? 'unknown-order'),
                        },
                        body: JSON.stringify({ order_id: postSealOrderId, items: [], source: 'PAYLOAD_TERMINAL' }),
                      });
                      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
                      showToast(
                        isIOS ? 'Reported' : 'REPORTED',
                        isIOS ? 'Missing items flagged for review.' : 'MISSING ITEMS FLAGGED FOR REVIEW.',
                        'warning'
                      );
                    } catch (e: unknown) {
                      showToast('ERROR', e instanceof Error ? e.message : 'Failed to report', 'error');
                    }
                  },
                },
              ]
            );
          }}
          style={({ pressed }) => ({
            borderWidth: 2,
            borderColor: 'rgba(255,255,255,0.6)',
            paddingHorizontal: 32,
            paddingVertical: 14,
            borderRadius: T.radius.button,
            flexDirection: 'row' as const,
            alignItems: 'center' as const,
            gap: 8,
            opacity: pressed ? 0.82 : 1,
            transform: [{ scale: pressed ? 0.97 : 1 }],
          })}
        >
          <MaterialIcons name="report-problem" size={20} color="#FFFFFF" />
          <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
            {isIOS ? 'Report Missing Items' : 'REPORT MISSING ITEMS'}
          </Text>
        </Pressable>
        {renderUiToast()}
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
        <Pressable
          onPress={() => { setActiveTruck(null); setAllSealed(false); setDispatchCodes({}); }}
          style={({ pressed }) => ({
            borderWidth: 1,
            borderColor: 'rgba(255,255,255,0.5)',
            paddingHorizontal: 32,
            paddingVertical: 14,
            borderRadius: T.radius.button,
            opacity: pressed ? 0.82 : 1,
            transform: [{ scale: pressed ? 0.97 : 1 }],
          })}
        >
          <Text style={{ color: '#FFFFFF', fontWeight: '600', fontSize: 14, letterSpacing: 0.3 }}>
            {isIOS ? 'New Manifest' : 'NEW MANIFEST'}
          </Text>
        </Pressable>
        {renderUiToast()}
      </View>
    );
  }

  // ── Render: AWAITING TRUCK SELECTION ─────────────────────────────────────
  if (!token) {
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background }}>
        {renderClientPolicyBanner()}
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', padding: 48 }}>
        <Text style={{ fontWeight: '700', fontSize: 14, color: T.colors.tertiaryLabel, letterSpacing: 0.5, marginBottom: 32 }}>
          {tx('auth.login.payload_terminal')}
        </Text>
        <Text style={{ fontSize: 22, fontWeight: '700', color: T.colors.label, letterSpacing: isIOS ? -0.4 : 0.5, marginBottom: 32 }}>
          {tx('auth.login.payloader_login')}
        </Text>
        <Text style={{ fontSize: 13, color: T.colors.secondaryLabel, marginBottom: 16, textAlign: 'center' }}>
          {loginMode === 'otp'
            ? (isIOS ? 'Sign in with warehouse phone OTP.' : 'SIGN IN WITH WAREHOUSE PHONE OTP.')
            : (isIOS ? 'Dev login with phone and PIN.' : 'DEV LOGIN WITH PHONE AND PIN.')}
        </Text>
        <TextInput
          placeholder={tx('common.field.phone')}
          placeholderTextColor={T.colors.tertiaryLabel}
          value={phoneInput}
          onChangeText={setPhoneInput}
          keyboardType="phone-pad"
          autoCapitalize="none"
          editable={!isLoggingIn && (loginMode === 'pin' || !otpSent)}
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
        {loginMode === 'otp' && otpSent ? (
          <TextInput
            placeholder={isIOS ? 'Verification code' : 'VERIFICATION CODE'}
            placeholderTextColor={T.colors.tertiaryLabel}
            value={otpInput}
            onChangeText={setOtpInput}
            keyboardType="number-pad"
            maxLength={6}
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
        ) : null}
        {loginMode === 'pin' ? (
        <TextInput
          placeholder={tx('auth.login.pin_label')}
          placeholderTextColor={T.colors.tertiaryLabel}
          value={pinInput}
          onChangeText={setPinInput}
          keyboardType="number-pad"
          maxLength={8}
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
        ) : null}
        {loginMode === 'otp' ? (
          !otpSent ? (
            <Pressable
              onPress={handleSendOtp}
              disabled={isLoggingIn || !phoneInput.trim()}
              style={({ pressed }) => ({
                width: 320,
                paddingVertical: 16,
                alignItems: 'center' as const,
                backgroundColor: !isLoggingIn && phoneInput.trim() ? T.colors.accent : T.colors.fillSecondary,
                borderRadius: T.radius.button,
                opacity: pressed ? 0.82 : 1,
                transform: [{ scale: pressed ? 0.97 : 1 }],
                marginBottom: 12,
              })}
            >
              <Text style={{
                fontWeight: '600',
                fontSize: 14,
                letterSpacing: isIOS ? 0.3 : 1,
                color: !isLoggingIn && phoneInput.trim() ? '#FFFFFF' : T.colors.tertiaryLabel,
              }}>
                {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Send Code' : 'SEND CODE')}
              </Text>
            </Pressable>
          ) : (
            <>
              <Pressable
                onPress={handleVerifyOtp}
                disabled={isLoggingIn || otpInput.trim().length < 6}
                style={({ pressed }) => ({
                  width: 320,
                  paddingVertical: 16,
                  alignItems: 'center' as const,
                  backgroundColor: !isLoggingIn && otpInput.trim().length >= 6 ? T.colors.accent : T.colors.fillSecondary,
                  borderRadius: T.radius.button,
                  opacity: pressed ? 0.82 : 1,
                  transform: [{ scale: pressed ? 0.97 : 1 }],
                  marginBottom: 12,
                })}
              >
                <Text style={{
                  fontWeight: '600',
                  fontSize: 14,
                  letterSpacing: isIOS ? 0.3 : 1,
                  color: !isLoggingIn && otpInput.trim().length >= 6 ? '#FFFFFF' : T.colors.tertiaryLabel,
                }}>
                  {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Verify & Sign In' : 'VERIFY & SIGN IN')}
                </Text>
              </Pressable>
              <Pressable onPress={handleSendOtp} disabled={isLoggingIn} style={{ marginBottom: 12 }}>
                <Text style={{ color: T.colors.secondaryLabel, fontSize: 13 }}>
                  {isIOS ? 'Resend code' : 'RESEND CODE'}
                </Text>
              </Pressable>
            </>
          )
        ) : (
        <Pressable
          onPress={handleLoginPin}
          disabled={isLoggingIn || !phoneInput || pinInput.length < 6}
          style={({ pressed }) => ({
            width: 320,
            paddingVertical: 16,
            alignItems: 'center' as const,
            backgroundColor: !isLoggingIn && phoneInput && pinInput.length >= 6 ? T.colors.accent : T.colors.fillSecondary,
            borderRadius: T.radius.button,
            opacity: pressed ? 0.82 : 1,
            transform: [{ scale: pressed ? 0.97 : 1 }],
            marginBottom: 12,
          })}
        >
          <Text style={{
            fontWeight: '600',
            fontSize: 14,
            letterSpacing: isIOS ? 0.3 : 1,
            color: !isLoggingIn && phoneInput && pinInput.length >= 6 ? '#FFFFFF' : T.colors.tertiaryLabel,
          }}>
            {isLoggingIn ? tx('auth.login.authenticating') : (isIOS ? 'Sign In with PIN' : 'SIGN IN WITH PIN')}
          </Text>
        </Pressable>
        )}
        <Pressable
          onPress={() => {
            setLoginMode(loginMode === 'otp' ? 'pin' : 'otp');
            setOtpSent(false);
            setOtpInput('');
            resetPhoneOtpFlow();
          }}
          disabled={isLoggingIn}
        >
          <Text style={{ color: T.colors.secondaryLabel, fontSize: 13 }}>
            {loginMode === 'otp' ? (isIOS ? 'Use PIN (dev)' : 'USE PIN (DEV)') : (isIOS ? 'Use phone OTP' : 'USE PHONE OTP')}
          </Text>
        </Pressable>
        {renderUiToast()}
        </View>
      </View>
    );
  }

  if (!activeTruck) {
    if (workspaceMode === 'inbound') {
      return (
        <InboundReturnsPanel
          theme={T}
          isIOS={isIOS}
          isOnline={isOnline}
          onBack={() => setWorkspaceMode('outbound')}
          showToast={(title, message, kind) => showToast(title, message, kind === 'success' ? 'success' : kind === 'error' ? 'error' : 'info')}
        />
      );
    }
    return (
      <View style={{ flex: 1, backgroundColor: T.colors.background }}>
        {renderClientPolicyBanner()}
        {/* Header */}
        <View style={{ backgroundColor: T.colors.sidebarBackground, paddingHorizontal: 32, paddingVertical: 16, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
          <View>
            <Text style={{ color: T.colors.sidebarLabel, fontWeight: '700', fontSize: 14, letterSpacing: 0.3 }}>
              {tx('auth.login.payload_terminal')}
            </Text>
            <Text style={{ color: T.colors.sidebarSecondary, fontFamily: T.typography.mono.fontFamily, fontSize: 11, marginTop: 2 }}>
              {workerName}
            </Text>
          </View>
          <View style={{ flexDirection: 'row', alignItems: 'center' }}>
            <Pressable onPress={() => setWorkspaceMode('inbound')} style={{ paddingHorizontal: 16, paddingVertical: 8, marginRight: 8 }}>
              <Text style={{ color: T.colors.sidebarLabel, fontSize: 12, fontWeight: '700', letterSpacing: 0.3 }}>
                {isIOS ? 'Inbound Returns' : 'INBOUND RETURNS'}
              </Text>
            </Pressable>
            <Pressable onPress={handleLogout} style={{ paddingHorizontal: 16, paddingVertical: 8 }}>
              <Text style={{ color: T.colors.sidebarSecondary, fontSize: 12, fontWeight: '600', letterSpacing: 0.3 }}>
                {tx('common.action.sign_out')}
              </Text>
            </Pressable>
          </View>
        </View>

        {/* Truck selector */}
        <View className="flex-1 items-center justify-center p-12">
          {isLoadingTrucks ? (
            <PayloadStatePanel
              theme={T}
              variant="truck"
              title={isIOS ? 'Loading vehicles...' : 'LOADING VEHICLES...'}
              message={isIOS ? 'Refreshing supplier fleet availability for this shift.' : 'REFRESHING SUPPLIER FLEET AVAILABILITY FOR THIS SHIFT.'}
            />
          ) : trucks.length === 0 ? (
            <PayloadStatePanel
              theme={T}
              variant="truck"
              title={tx('payload.vehicle.none_available')}
              message={isIOS ? 'No payload vehicle is currently ready for assignment.' : 'NO PAYLOAD VEHICLE IS CURRENTLY READY FOR ASSIGNMENT.'}
              tone="warning"
            />
          ) : (
            <>
              <Text style={{ fontSize: 13, fontWeight: '500', color: T.colors.tertiaryLabel, marginBottom: 32, letterSpacing: 0.3 }}>
                {tx('payload.vehicle.select_target')}
              </Text>
              <View className="flex-row gap-4">
                {trucks.map(truck => (
                  <Pressable
                    key={truck.id}
                    onPress={() => handleTruckSelect(truck.id)}
                    style={({ pressed }) => ({
                      borderWidth: isIOS ? 0.33 : 1,
                      borderColor: T.colors.separator,
                      backgroundColor: T.colors.cardBackground,
                      paddingHorizontal: 40,
                      paddingVertical: 32,
                      alignItems: 'center' as const,
                      borderRadius: T.radius.card,
                      ...T.shadow.card,
                      opacity: pressed ? 0.82 : 1,
                      transform: [{ scale: pressed ? 0.96 : 1 }],
                    })}
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
                  </Pressable>
                ))}
              </View>
              <Text style={{ fontSize: 12, color: T.colors.tertiaryLabel, marginTop: 40, letterSpacing: 0.3 }}>
                {isIOS ? 'Select target vehicle' : 'SELECT TARGET VEHICLE'}
              </Text>
            </>
          )}
        </View>
        {renderUiToast()}
      </View>
    );
  }

  // ── Render: MANIFEST VIEW ─────────────────────────────────────────────────
  if (workspaceMode === 'inbound') {
    return (
      <InboundReturnsPanel
        theme={T}
        isIOS={isIOS}
        isOnline={isOnline}
        onBack={() => setWorkspaceMode('outbound')}
        showToast={(title, message, kind) => showToast(title, message, kind === 'success' ? 'success' : kind === 'error' ? 'error' : 'info')}
      />
    );
  }

  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background, flexDirection: 'column' }}>
      {renderClientPolicyBanner()}
      <View style={{ flex: 1, flexDirection: 'row' }}>

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
            <ConnectionStrip isOnline={isOnline} queuedCount={offlineQueue.length} theme={T} />
          </View>
          <Pressable
            onPress={() => setWorkspaceMode('inbound')}
            style={{ padding: 6, marginRight: 4 }}
          >
            <MaterialIcons name="undo" size={20} color={T.colors.sidebarLabel} />
          </Pressable>
          <Pressable
            onPress={() => {
              setShowExceptionsPanel(true);
              void loadManifestExceptions();
            }}
            style={{ padding: 6, marginRight: 4 }}
          >
            <MaterialIcons name="report-problem" size={20} color={T.colors.sidebarLabel} />
            {manifestExceptions.length > 0 ? (
              <View style={{ position: 'absolute', top: 2, right: 2, backgroundColor: '#F59E0B', borderRadius: 8, minWidth: 16, height: 16, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#FFF', fontSize: 9, fontWeight: '700' }}>
                  {manifestExceptions.length > 99 ? '99+' : manifestExceptions.length}
                </Text>
              </View>
            ) : null}
          </Pressable>
          <Pressable onPress={() => setShowNotifPanel(true)} style={{ padding: 6 }}>
            <MaterialIcons name="notifications" size={20} color={T.colors.sidebarLabel} />
            {unreadCount > 0 && (
              <View style={{ position: 'absolute', top: 2, right: 2, backgroundColor: '#EF4444', borderRadius: 8, minWidth: 16, height: 16, alignItems: 'center', justifyContent: 'center' }}>
                <Text style={{ color: '#FFF', fontSize: 9, fontWeight: '700' }}>{unreadCount > 99 ? '99+' : unreadCount}</Text>
              </View>
            )}
          </Pressable>
        </View>

        {/* Truck toggle bar */}
        <View style={{ flexDirection: 'row', borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
          {trucks.map(truck => (
            <Pressable
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
            </Pressable>
          ))}
        </View>

        {batchReadyManifestIds.length > 1 && (
          <View style={{
            paddingHorizontal: 12,
            paddingVertical: 10,
            borderBottomWidth: 0.5,
            borderBottomColor: T.colors.sidebarSeparator,
            backgroundColor: `${T.colors.accent}12`,
          }}>
            <WorkflowSectionHeader
              onDark
              subtitle={`${batchReadyManifestIds.length} trucks ready to finalize`}
              theme={T}
              title="Batch seal"
            />
            <Pressable
              onPress={handleFinalizeBatchSeal}
              disabled={batchSealing}
              style={{
                paddingVertical: 10,
                alignItems: 'center',
                backgroundColor: batchSealing ? T.colors.fillSecondary : T.colors.accent,
                borderRadius: T.radius.button,
              }}
            >
              <Text style={{ fontWeight: '700', fontSize: 11, color: batchSealing ? T.colors.tertiaryLabel : '#FFFFFF' }}>
                {batchSealing ? (isIOS ? 'Finalizing…' : 'FINALIZING…') : (isIOS ? 'Seal all trucks' : 'SEAL ALL TRUCKS')}
              </Text>
            </Pressable>
          </View>
        )}

        {/* LEO: Volume Progress Bar + Manifest State */}
        {manifestId && (
          <View style={{ paddingHorizontal: 16, paddingVertical: 10, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
            <ManifestKpiGrid
              compact
              manifestId={manifestId}
              maxVolumeVu={manifestMaxVolume}
              regionCode={manifestRegionCode}
              state={manifestState}
              stopCount={manifestStopCount}
              theme={T}
              totalVolumeVu={manifestVolume}
            />
            {manifestState === 'DRAFT' && (
              <Pressable
                onPress={handleStartLoading}
                disabled={isStartingLoad}
                style={{
                  marginTop: 10,
                  paddingVertical: 10,
                  alignItems: 'center',
                  backgroundColor: isStartingLoad ? T.colors.fillSecondary : T.colors.accent,
                  borderRadius: T.radius.button,
                }}
              >
                <Text style={{ fontWeight: '600', fontSize: 12, letterSpacing: 0.5, color: isStartingLoad ? T.colors.tertiaryLabel : '#FFFFFF' }}>
                  {isStartingLoad ? (isIOS ? 'Starting...' : 'STARTING...') : (isIOS ? 'Start Loading' : 'START LOADING')}
                </Text>
              </Pressable>
            )}
          </View>
        )}

        {/* Order list */}
        <ScrollView>
          {isLoading ? (
            <View className="p-6 items-center">
              <PayloadStatePanel
                theme={T}
                variant="manifest"
                title={isIOS ? 'Fetching manifest...' : 'FETCHING MANIFEST...'}
                message={isIOS ? 'Loading the active checklist for this truck.' : 'LOADING THE ACTIVE CHECKLIST FOR THIS TRUCK.'}
                compact
              />
            </View>
          ) : orders.length === 0 ? (
            <View className="p-6 items-center">
              <PayloadStatePanel
                theme={T}
                variant="manifest"
                title={isIOS ? 'No pending orders' : 'NO PENDING ORDERS'}
                message={isIOS ? 'This truck has no checklist items waiting to load.' : 'THIS TRUCK HAS NO CHECKLIST ITEMS WAITING TO LOAD.'}
                compact
              />
            </View>
          ) : (
            orders.map(order => {
              const isSealed = sealedOrderIds.has(order.order_id);
              const isActive = order.order_id === selectedOrderId;
              return (
                <Pressable
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
                    <StatusBadge compact label={isIOS ? 'Cleared' : 'CLEARED'} theme={T} tone="success" />
                  )}
                </Pressable>
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
              <Pressable
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
              </Pressable>

              {/* LEO: Exception buttons — remove order from manifest */}
              {manifestState === 'LOADING' && selectedOrderId && (
                <View style={{ flexDirection: 'row', marginLeft: 8, gap: 4 }}>
                  {(['OVERFLOW', 'DAMAGED', 'MANUAL'] as const).map(reason => (
                    <Pressable
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
                      style={({ pressed }) => ({
                        paddingHorizontal: 8,
                        paddingVertical: 6,
                        borderRadius: T.radius.checkbox,
                        borderWidth: isIOS ? 0.33 : 1,
                        borderColor: reason === 'DAMAGED' ? '#EF4444' : reason === 'OVERFLOW' ? '#F59E0B' : T.colors.separator,
                        backgroundColor: T.colors.fillTertiary,
                        opacity: pressed ? 0.75 : 1,
                        transform: [{ scale: pressed ? 0.95 : 1 }],
                      })}
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
                    </Pressable>
                  ))}
                </View>
              )}
            </View>

            {/* Manifest checklist */}
            <View style={{ paddingHorizontal: 32, paddingTop: 16, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' }}>
              <WorkflowSectionHeader
                subtitle={isIOS ? 'Tap each line to confirm load verification.' : 'TAP EACH LINE TO CONFIRM LOAD VERIFICATION.'}
                theme={T}
                title="Load checklist"
              />
              <Pressable
                onPress={() => setShowProductScanner(true)}
                style={{
                  paddingHorizontal: 12,
                  paddingVertical: 8,
                  borderRadius: T.radius.button,
                  borderWidth: 1,
                  borderColor: T.colors.accent,
                }}
              >
                <Text style={{ fontSize: 11, fontWeight: '700', color: T.colors.accent }}>
                  {isIOS ? 'Scan product' : 'SCAN PRODUCT'}
                </Text>
              </Pressable>
            </View>
            <ScrollView className="flex-1 px-8 py-2">
              {selectedManifest.map(item => (
                <Pressable
                  key={item.id}
                  onPress={() => toggleCheck(item.id)}
                  style={({ pressed }) => ({
                    flexDirection: 'row' as const,
                    alignItems: 'center' as const,
                    paddingVertical: 16,
                    borderBottomWidth: isIOS ? 0.33 : 1,
                    borderBottomColor: T.colors.separator,
                    opacity: item.scanned ? 0.4 : pressed ? 0.75 : 1,
                    transform: [{ scale: pressed ? 0.99 : 1 }],
                  })}
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
                </Pressable>
              ))}
            </ScrollView>

            {/* Seal button — per-order (legacy) + manifest-level (LEO) */}
            <View style={{ paddingHorizontal: 32, paddingVertical: 20, borderTopWidth: isIOS ? 0.33 : 1, borderTopColor: T.colors.separator }}>
              <WorkflowSectionHeader
                subtitle={manifestId ? (isIOS ? 'Inject orders or seal the manifest when loading is complete.' : 'INJECT ORDERS OR SEAL THE MANIFEST WHEN LOADING IS COMPLETE.') : (isIOS ? 'Seal each order after checklist verification.' : 'SEAL EACH ORDER AFTER CHECKLIST VERIFICATION.')}
                theme={T}
                title={manifestId ? 'Manifest workflow' : 'Order seal'}
              />
              <View style={{ marginTop: 12 }}>
              {/* Per-order seal (legacy — when no manifest entity exists) */}
              {!manifestId && (
                <Pressable
                  onPress={handleSeal}
                  disabled={!allChecked || isSealing}
                  style={({ pressed }) => ({
                    paddingVertical: 16,
                    alignItems: 'center' as const,
                    backgroundColor: allChecked && !isSealing ? T.colors.accent : T.colors.fillSecondary,
                    borderRadius: T.radius.button,
                    opacity: pressed ? 0.82 : 1,
                    transform: [{ scale: pressed ? 0.97 : 1 }],
                  })}
                >
                  <Text style={{
                    fontWeight: '600',
                    fontSize: 14,
                    letterSpacing: isIOS ? 0.3 : 1,
                    color: allChecked && !isSealing ? '#FFFFFF' : T.colors.tertiaryLabel,
                  }}>
                    {isSealing ? (isIOS ? 'Sealing...' : 'SEALING...') : (isIOS ? 'Mark as Loaded' : 'MARK AS LOADED')}
                  </Text>
                </Pressable>
              )}
              {/* Manifest-level seal (LEO — slide to seal entire manifest) */}
              {manifestId && manifestState === 'LOADING' && (
                <View style={{ gap: 10 }}>
                  {/* Inject order button */}
                  <Pressable
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
                  </Pressable>
                  {/* Seal manifest button */}
                  <Pressable
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
                  </Pressable>
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
            </View>
          </>
        ) : (
          <View className="flex-1 items-center justify-center p-8">
            <PayloadStatePanel
              compact
              message={isIOS ? 'Choose an order from the sidebar to review its checklist.' : 'CHOOSE AN ORDER FROM THE SIDEBAR TO REVIEW ITS CHECKLIST.'}
              theme={T}
              title={isLoading ? (isIOS ? 'Fetching manifest...' : 'FETCHING MANIFEST...') : (isIOS ? 'Select order from manifest' : 'SELECT ORDER FROM MANIFEST')}
              variant="manifest"
            />
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
                Scan an order label or enter the order ID to inject into the active loading session.
              </Text>
            </View>
            <View style={{ padding: 24, gap: 16 }}>
              <Pressable
                onPress={() => setShowInjectScanner((v) => !v)}
                style={{
                  paddingVertical: 10,
                  alignItems: 'center',
                  borderRadius: T.radius.button,
                  borderWidth: 1,
                  borderColor: T.colors.accent,
                }}
              >
                <Text style={{ fontWeight: '600', fontSize: 13, color: T.colors.accent }}>
                  {showInjectScanner ? (isIOS ? 'Hide scanner' : 'HIDE SCANNER') : (isIOS ? 'Scan order label' : 'SCAN ORDER LABEL')}
                </Text>
              </Pressable>
              {showInjectScanner ? (
                <CameraView
                  style={{ height: 160, borderRadius: 12, overflow: 'hidden' }}
                  barcodeScannerSettings={{ barcodeTypes: ['ean13', 'ean8', 'code128', 'qr'] }}
                  onBarcodeScanned={({ data }) => {
                    setInjectOrderId(data.trim());
                    setShowInjectScanner(false);
                  }}
                />
              ) : null}
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
                <Pressable
                  onPress={() => { setShowInjectOrder(false); setInjectOrderId(''); }}
                  style={{ flex: 1, paddingVertical: 14, alignItems: 'center', backgroundColor: T.colors.fillSecondary, borderRadius: T.radius.button }}
                >
                  <Text style={{ fontWeight: '600', fontSize: 14, color: T.colors.secondaryLabel }}>Cancel</Text>
                </Pressable>
                <Pressable
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
                </Pressable>
              </View>
            </View>
          </View>
        </View>
      </Modal>

      <Modal visible={showProductScanner} transparent animationType="fade" onRequestClose={() => setShowProductScanner(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.5)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 420, backgroundColor: T.colors.background, borderRadius: isIOS ? 14 : 16, overflow: 'hidden', padding: 16, gap: 12 }}>
            <Text style={{ fontWeight: '700', fontSize: 16, color: T.colors.label }}>
              {isIOS ? 'Scan product EAN' : 'SCAN PRODUCT EAN'}
            </Text>
            <CameraView
              style={{ height: 200, borderRadius: 12, overflow: 'hidden' }}
              barcodeScannerSettings={{ barcodeTypes: ['ean13', 'ean8'] }}
              onBarcodeScanned={({ data }) => { void handleProductBarcodeScan(data); }}
            />
            <Pressable onPress={() => setShowProductScanner(false)} style={{ paddingVertical: 12, alignItems: 'center' }}>
              <Text style={{ color: T.colors.secondaryLabel }}>{isIOS ? 'Close' : 'CLOSE'}</Text>
            </Pressable>
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
              <Pressable onPress={() => { setShowReDispatch(false); setReDispatchOrderId(null); }} style={{ padding: 8 }}>
                <MaterialIcons name="close" size={22} color={T.colors.tertiaryLabel} />
              </Pressable>
            </View>

            {/* Recommendation list */}
            {isLoadingRecs ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <PayloadStatePanel
                  theme={T}
                  variant="dispatch"
                  title={isIOS ? 'Analyzing fleet positions...' : 'ANALYZING FLEET POSITIONS...'}
                  message={isIOS ? 'Scoring nearby trucks for the best reassignment path.' : 'SCORING NEARBY TRUCKS FOR THE BEST REASSIGNMENT PATH.'}
                  compact
                />
              </View>
            ) : recommendations.length === 0 ? (
              <View style={{ padding: 48, alignItems: 'center' }}>
                <PayloadStatePanel
                  theme={T}
                  variant="dispatch"
                  title={isIOS ? 'No available trucks found' : 'NO AVAILABLE TRUCKS FOUND'}
                  message={isIOS ? 'No nearby fleet target can accept this order right now.' : 'NO NEARBY FLEET TARGET CAN ACCEPT THIS ORDER RIGHT NOW.'}
                  compact
                  tone="warning"
                />
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
                    <Pressable
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
                            <StatusBadge compact label={isIOS ? 'Best' : 'BEST'} theme={T} tone="accent" />
                          )}
                          {isMaintenance && (
                            <StatusBadge compact label={isIOS ? 'Maintenance' : 'MAINTENANCE'} theme={T} tone="danger" />
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
                    </Pressable>
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
                <Pressable onPress={markAllNotifsRead} style={{ marginRight: 12 }}>
                  <Text style={{ fontSize: 12, color: T.colors.accent, fontWeight: '600' }}>Mark all read</Text>
                </Pressable>
              )}
              <Pressable onPress={() => setShowNotifPanel(false)}>
                <MaterialIcons name="close" size={20} color={T.colors.sidebarSecondary} />
              </Pressable>
            </View>
            {/* Notification list */}
            <FlatList
              data={notifications}
              keyExtractor={item => item.id}
              ListEmptyComponent={
                <View style={{ padding: 40, alignItems: 'center' }}>
                  <PayloadStatePanel
                    theme={T}
                    variant="notifications"
                    title={isIOS ? 'No notifications' : 'NO NOTIFICATIONS'}
                    message={isIOS ? 'Payload alerts and sync events will appear here.' : 'PAYLOAD ALERTS AND SYNC EVENTS WILL APPEAR HERE.'}
                    compact
                  />
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
                  <Pressable
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
                  </Pressable>
                );
              }}
            />
          </View>
        </View>
      </Modal>

      {/* ── Manifest Exceptions Panel Modal ──────────────────────────────── */}
      <Modal visible={showExceptionsPanel} transparent animationType="fade" onRequestClose={() => setShowExceptionsPanel(false)}>
        <View style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.4)', justifyContent: 'center', alignItems: 'center' }}>
          <View style={{ width: 420, maxHeight: '80%', backgroundColor: T.colors.sidebarBackground, borderRadius: 12, overflow: 'hidden' }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 14, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
              <Text style={{ flex: 1, fontWeight: '700', fontSize: 15, color: T.colors.sidebarLabel, letterSpacing: 0.3 }}>
                {isIOS ? 'Manifest exceptions' : 'MANIFEST EXCEPTIONS'}
              </Text>
              <Pressable onPress={() => void loadManifestExceptions()} style={{ marginRight: 12 }}>
                <MaterialIcons name="refresh" size={20} color={T.colors.accent} />
              </Pressable>
              <Pressable onPress={() => setShowExceptionsPanel(false)}>
                <MaterialIcons name="close" size={20} color={T.colors.sidebarSecondary} />
              </Pressable>
            </View>
            {loadingExceptions && manifestExceptions.length === 0 ? (
              <SkeletonList count={4} theme={T} />
            ) : (
              <FlatList
                data={manifestExceptions}
                keyExtractor={item => item.exception_id}
                ListEmptyComponent={
                  <View style={{ padding: 40, alignItems: 'center' }}>
                    <PayloadStatePanel
                      theme={T}
                      variant="manifest"
                      title={isIOS ? 'No exceptions' : 'NO EXCEPTIONS'}
                      message={isIOS ? 'Overflow, damaged, and manual removals appear here.' : 'OVERFLOW, DAMAGED, AND MANUAL REMOVALS APPEAR HERE.'}
                      compact
                    />
                  </View>
                }
                renderItem={({ item }) => (
                  <View style={{ paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 0.5, borderBottomColor: T.colors.sidebarSeparator }}>
                    <View style={{ flexDirection: 'row', alignItems: 'center', marginBottom: 4, gap: 8 }}>
                      <StatusBadge compact label={item.reason} theme={T} tone={exceptionReasonTone(item.reason)} />
                      {item.escalated ? (
                        <StatusBadge compact label={isIOS ? 'Escalated' : 'ESCALATED'} theme={T} tone="danger" />
                      ) : null}
                    </View>
                    <Text style={{ fontSize: 12, color: T.colors.sidebarSecondary }}>
                      {isIOS ? 'Order' : 'ORDER'} {item.order_id.slice(0, 8)} · {isIOS ? 'Manifest' : 'MANIFEST'} {item.manifest_id.slice(0, 8)}
                    </Text>
                    <Text style={{ fontSize: 10, color: T.colors.tertiaryLabel, marginTop: 4 }}>
                      {isIOS ? 'Attempts' : 'ATTEMPTS'} {item.attempt_count}
                    </Text>
                  </View>
                )}
              />
            )}
          </View>
        </View>
      </Modal>

      {renderUiToast()}
      </View>
    </View>
  );
}
