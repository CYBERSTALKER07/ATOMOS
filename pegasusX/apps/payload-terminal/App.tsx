import { useEffect, useMemo, useState } from 'react';
import { useWindowDimensions } from 'react-native';
import * as ScreenOrientation from 'expo-screen-orientation';
import * as SecureStore from 'expo-secure-store';
import "./global.css";
import { isIOS, useT } from './theme';
import { getPayloadTranslator, resolvePayloadLocale } from './localization';
import { registerPayloadPushTokens } from './pushRegistration';
import { defaultLocale, type Locale } from '../../packages/i18n/locales';
import { InboundReturnsPanel } from './inboundReturns';
import AllSealedScreen from './components/AllSealedScreen';
import AuthLoadingScreen from './components/AuthLoadingScreen';
import LoginScreen from './components/LoginScreen';
import ManifestWorkspaceScreen from './components/ManifestWorkspaceScreen';
import PostSealCountdownScreen from './components/PostSealCountdownScreen';
import TruckSelectionScreen from './components/TruckSelectionScreen';
import { useClientPolicy } from './hooks/useClientPolicy';
import { useManifestActions } from './hooks/useManifestActions';
import { useManifestData } from './hooks/useManifestData';
import { useManifestExceptions } from './hooks/useManifestExceptions';
import { useNotifications } from './hooks/useNotifications';
import { useOfflineQueue } from './hooks/useOfflineQueue';
import { useOtaUpdates } from './hooks/useOtaUpdates';
import { usePayloaderAuth } from './hooks/usePayloaderAuth';
import { useReDispatch } from './hooks/useReDispatch';
import { useToast } from './hooks/useToast';

// ─── Main Component ───────────────────────────────────────────────────────────

export default function App() {
  const T = useT();
  const { width, height } = useWindowDimensions();
  const isTabletLayout = Math.min(width, height) >= 768;

  // ─── OTA Updates ──────────────────────────────────────────────────────────────
  useOtaUpdates();

  const [locale, setLocale] = useState<Locale>(defaultLocale);
  const tx = useMemo(() => getPayloadTranslator(locale), [locale]);

  // Lightweight in-app toast for non-blocking feedback.
  const { showToast, renderUiToast } = useToast({ theme: T, isTabletLayout });

  // ─── Auth ─────────────────────────────────────────────────────────────────────
  const auth = usePayloaderAuth({ locale, tx, showToast });
  const { token, authLoading } = auth;

  const getAuthHeaders = () => {
    const traceId = crypto.randomUUID();
    return token
      ? { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json', 'X-Trace-Id': traceId }
      : { 'Content-Type': 'application/json', 'X-Trace-Id': traceId };
  };
  const authHeaders = getAuthHeaders();

  // ─── Domain hooks ─────────────────────────────────────────────────────────────
  const queue = useOfflineQueue({ token, tx, showToast });
  const notif = useNotifications({ token });
  const { renderClientPolicyBanner } = useClientPolicy({ token });
  const data = useManifestData({
    token,
    locale,
    tx,
    showToast,
    authHeaders,
    fetchNotifications: notif.fetchNotifications,
    flushOfflineQueue: queue.flushOfflineQueue,
    setNotifications: notif.setNotifications,
    setUnreadCount: notif.setUnreadCount,
  });
  const actions = useManifestActions({ token, locale, tx, showToast, data, queue });
  const redispatch = useReDispatch({ token, locale, tx, showToast, authHeaders, data });
  const exceptions = useManifestExceptions({ token, liveSyncRevision: data.liveSyncRevision, showToast });

  const [workspaceMode, setWorkspaceMode] = useState<'outbound' | 'inbound'>('outbound');

  // Lock tablet to landscape + restore session on mount
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
          auth.setToken(saved);
          auth.setWorkerName(name || 'Payloader');
          if (sid) auth.setSupplierId(sid);
          void registerPayloadPushTokens();
        }
        // Restore offline queue
        const queueStr = await SecureStore.getItemAsync('offline_queue');
        if (queueStr) {
          try { queue.setOfflineQueue(JSON.parse(queueStr)); } catch {}
        }
      } catch {} finally {
        auth.setAuthLoading(false);
      }
    })();
  }, []);

  const handleLogout = async () => {
    await auth.handleLogout();
    data.setActiveTruck(null);
    data.setTrucks([]);
    data.setIsLoadingTrucks(false);
  };

  // ── Render: AUTH LOADING ────────────────────────────────────────────────
  if (authLoading) {
    return <AuthLoadingScreen theme={T} tx={tx} toast={renderUiToast()} />;
  }

  // ── Render: POST-SEAL DOUBLE-CHECK COUNTDOWN (Edge 33) ────────────────
  if (data.postSealOrderId && data.postSealCountdown > 0) {
    return (
      <PostSealCountdownScreen
        theme={T}
        postSealOrderId={data.postSealOrderId}
        postSealCountdown={data.postSealCountdown}
        showToast={showToast}
        toast={renderUiToast()}
      />
    );
  }

  // ── Render: ALL SEALED ────────────────────────────────────────────────────
  if (data.allSealed) {
    return (
      <AllSealedScreen
        theme={T}
        activeTruck={data.activeTruck}
        dispatchCodes={data.dispatchCodes}
        onNewManifest={() => { data.setActiveTruck(null); data.setAllSealed(false); data.setDispatchCodes({}); }}
        toast={renderUiToast()}
      />
    );
  }

  // ── Render: PAYLOADER LOGIN ─────────────────────────────────────────────
  if (!token) {
    return (
      <LoginScreen
        theme={T}
        tx={tx}
        loginMode={auth.loginMode}
        phoneInput={auth.phoneInput}
        onPhoneInputChange={auth.setPhoneInput}
        pinInput={auth.pinInput}
        onPinInputChange={auth.setPinInput}
        otpInput={auth.otpInput}
        onOtpInputChange={auth.setOtpInput}
        otpSent={auth.otpSent}
        isLoggingIn={auth.isLoggingIn}
        onSendOtp={auth.handleSendOtp}
        onVerifyOtp={auth.handleVerifyOtp}
        onLoginPin={auth.handleLoginPin}
        onToggleLoginMode={auth.handleToggleLoginMode}
        policyBanner={renderClientPolicyBanner()}
        toast={renderUiToast()}
      />
    );
  }

  // ── Render: AWAITING TRUCK SELECTION ─────────────────────────────────────
  if (!data.activeTruck) {
    if (workspaceMode === 'inbound') {
      return (
        <InboundReturnsPanel
          theme={T}
          isIOS={isIOS}
          isOnline={data.isOnline}
          onBack={() => setWorkspaceMode('outbound')}
          showToast={(title, message, kind) => showToast(title, message, kind === 'success' ? 'success' : kind === 'error' ? 'error' : 'info')}
        />
      );
    }
    return (
      <TruckSelectionScreen
        theme={T}
        tx={tx}
        workerName={auth.workerName}
        trucks={data.trucks}
        activeTruck={data.activeTruck}
        isLoadingTrucks={data.isLoadingTrucks}
        onTruckSelect={data.handleTruckSelect}
        onLogout={handleLogout}
        onShowInbound={() => setWorkspaceMode('inbound')}
        policyBanner={renderClientPolicyBanner()}
        toast={renderUiToast()}
      />
    );
  }

  // ── Render: INBOUND RETURNS ──────────────────────────────────────────────
  if (workspaceMode === 'inbound') {
    return (
      <InboundReturnsPanel
        theme={T}
        isIOS={isIOS}
        isOnline={data.isOnline}
        onBack={() => setWorkspaceMode('outbound')}
        showToast={(title, message, kind) => showToast(title, message, kind === 'success' ? 'success' : kind === 'error' ? 'error' : 'info')}
      />
    );
  }

  // ── Render: MANIFEST VIEW ─────────────────────────────────────────────────
  return (
    <ManifestWorkspaceScreen
      theme={T}
      tx={tx}
      onShowInbound={() => setWorkspaceMode('inbound')}
      policyBanner={renderClientPolicyBanner()}
      toast={renderUiToast()}
      data={data}
      actions={actions}
      redispatch={redispatch}
      notif={notif}
      exceptions={exceptions}
      queue={queue}
    />
  );
}
