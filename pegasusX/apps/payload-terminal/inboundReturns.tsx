import React, { useCallback, useEffect, useRef, useState } from 'react';
import { View, Text, TextInput, Pressable, ScrollView, ActivityIndicator } from 'react-native';
import { MaterialIcons } from '@expo/vector-icons';
import { CameraView, useCameraPermissions } from 'expo-camera';
import * as Haptics from 'expo-haptics';
import * as SecureStore from 'expo-secure-store';
import { authFetch } from './authSession';
import { payloadInboundConfirmKey, payloadInboundScanKey } from './utils/idempotency';
import { InboundReturnsList } from './components/InboundReturnsList';

export type InboundRow = {
  return_id: string;
  order_id: string;
  sku_id: string;
  product_name: string;
  image_url?: string;
  barcode?: string;
  expected_qty: number;
  received_qty: number;
  reason: string;
  physical_status: string;
  manifest_id?: string;
  driver_name?: string;
  suggested_disposition: string;
};

type ScanMode = 'manual' | 'camera';
type Tab = 'queue' | 'history';

type QueuedScan = {
  id: string;
  endpoint: string;
  method: string;
  body: string;
  createdAt: number;
};

type Props = {
  theme: any;
  isIOS: boolean;
  isOnline?: boolean;
  onBack: () => void;
  showToast: (title: string, message: string, kind: 'success' | 'error' | 'info') => void;
};

const OFFLINE_QUEUE_KEY = 'offline_queue';

async function loadInboundQueue(): Promise<QueuedScan[]> {
  try {
    const raw = await SecureStore.getItemAsync(OFFLINE_QUEUE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as QueuedScan[];
    return parsed.filter(item => item.endpoint.includes('returns/inbound/scan'));
  } catch {
    return [];
  }
}

async function enqueueInboundScan(scan: QueuedScan): Promise<number> {
  const raw = await SecureStore.getItemAsync(OFFLINE_QUEUE_KEY);
  const existing = raw ? (JSON.parse(raw) as QueuedScan[]) : [];
  const updated = [...existing, scan];
  await SecureStore.setItemAsync(OFFLINE_QUEUE_KEY, JSON.stringify(updated));
  return updated.filter(item => item.endpoint.includes('returns/inbound/scan')).length;
}

export function InboundReturnsPanel({ theme: T, isIOS, isOnline = true, onBack, showToast }: Props) {
  const [tab, setTab] = useState<Tab>('queue');
  const [rows, setRows] = useState<InboundRow[]>([]);
  const [history, setHistory] = useState<InboundRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [barcode, setBarcode] = useState('');
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [scanMode, setScanMode] = useState<ScanMode>('manual');
  const [cameraEnabled, setCameraEnabled] = useState(true);
  const [queuedScans, setQueuedScans] = useState(0);
  const [permission, requestPermission] = useCameraPermissions();
  const lastScanRef = useRef<{ code: string; at: number }>({ code: '', at: 0 });

  const refreshQueueCount = useCallback(async () => {
    setQueuedScans((await loadInboundQueue()).length);
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [inboundRes, historyRes] = await Promise.all([
        authFetch('/v1/returns/inbound?physical_status=ARRIVED&limit=100'),
        authFetch('/v1/returns/history?limit=50'),
      ]);
      if (!inboundRes.ok) throw new Error('load_failed');
      const inboundData = await inboundRes.json();
      setRows(inboundData.data ?? []);
      if (historyRes.ok) {
        const historyData = await historyRes.json();
        setHistory(historyData.data ?? []);
      }
      await refreshQueueCount();
    } catch (e: unknown) {
      showToast('Inbound returns', e instanceof Error ? e.message : 'load_failed', 'error');
    } finally {
      setLoading(false);
    }
  }, [refreshQueueCount, showToast]);

  useEffect(() => { void load(); }, [load]);

  const ensureSession = async () => {
    if (sessionId) return sessionId;
    const res = await authFetch('/v1/returns/inbound/sessions', { method: 'POST', body: '{}' });
    if (!res.ok) throw new Error('session_failed');
    const data = await res.json();
    setSessionId(data.session_id);
    return data.session_id as string;
  };

  const handleScan = async (value?: string) => {
    const trimmed = (value ?? barcode).trim();
    if (!trimmed) return;
    setScanning(true);
    try {
      const sid = sessionId ?? (await ensureSession());
      const idempotencyKey = payloadInboundScanKey(trimmed, sid);

      if (!isOnline) {
        const body = JSON.stringify({ barcode: trimmed, qty: 1, session_id: sid });
        const count = await enqueueInboundScan({
          id: idempotencyKey,
          endpoint: '/v1/returns/inbound/scan',
          method: 'POST',
          body,
          createdAt: Date.now(),
        });
        setQueuedScans(count);
        showToast('Queued offline', 'Scan will sync when back online.', 'info');
        setBarcode('');
        return;
      }

      const res = await authFetch('/v1/returns/inbound/scan', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify({ barcode: trimmed, qty: 1, session_id: sid }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || data.message || 'scan_failed');
      showToast('Scanned', data.message || data.return_id, data.variance ? 'error' : 'success');
      setBarcode('');
      setCameraEnabled(false);
      await load();
      setTimeout(() => setCameraEnabled(true), 1500);
    } catch (e: unknown) {
      showToast('Scan failed', e instanceof Error ? e.message : 'scan_failed', 'error');
    } finally {
      setScanning(false);
    }
  };

  const onBarcodeScanned = (result: { data: string }) => {
    if (!cameraEnabled || scanning) return;
    const now = Date.now();
    const code = result.data.trim();
    if (!code) return;
    if (code === lastScanRef.current.code && now - lastScanRef.current.at < 1500) return;
    lastScanRef.current = { code, at: now };
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    setBarcode(code);
    void handleScan(code);
  };

  const handleConfirm = async (disposition: 'RESTOCK' | 'WRITE_OFF') => {
    const targets = rows.filter(r => selected.has(r.return_id));
    if (targets.length === 0) {
      showToast('Select lines', 'Tap rows to select, or scan first.', 'info');
      return;
    }
    try {
      const sid = sessionId ?? await ensureSession();
      const returnIds = targets.map(r => r.return_id);
      const res = await authFetch('/v1/returns/inbound/confirm', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': payloadInboundConfirmKey(returnIds, disposition),
        },
        body: JSON.stringify({
          session_id: sid,
          lines: targets.map(r => ({
            return_id: r.return_id,
            disposition,
            qty: r.received_qty > 0 ? r.received_qty : r.expected_qty,
          })),
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'confirm_failed');
      showToast('Confirmed', `${disposition} — ${(data.return_ids ?? []).length} line(s)`, 'success');
      setSelected(new Set());
      await load();
    } catch (e: unknown) {
      showToast('Confirm failed', e instanceof Error ? e.message : 'confirm_failed', 'error');
    }
  };

  const toggle = (id: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const list = tab === 'queue' ? rows : history;

  return (
    <View style={{ flex: 1, backgroundColor: T.colors.background }}>
      <View style={{ flexDirection: 'row', alignItems: 'center', padding: 16, borderBottomWidth: 0.5, borderBottomColor: T.colors.separator }}>
        <Pressable onPress={onBack} style={{ padding: 8, marginRight: 8 }}>
          <MaterialIcons name="arrow-back" size={22} color={T.colors.label} />
        </Pressable>
        <Text style={{ fontSize: 16, fontWeight: '700', color: T.colors.label, flex: 1 }}>
          {isIOS ? 'Inbound Returns' : 'INBOUND RETURNS'}
        </Text>
        <Pressable onPress={() => void load()} style={{ padding: 8 }}>
          <MaterialIcons name="refresh" size={20} color={T.colors.secondaryLabel} />
        </Pressable>
      </View>

      <View style={{ padding: 16, gap: 8, borderBottomWidth: 0.5, borderBottomColor: T.colors.separator }}>
        <View style={{ flexDirection: 'row', gap: 8 }}>
          <Pressable
            onPress={() => setTab('queue')}
            style={{ flex: 1, padding: 8, borderRadius: 8, backgroundColor: tab === 'queue' ? T.colors.tint : T.colors.secondaryBackground }}
          >
            <Text style={{ textAlign: 'center', color: tab === 'queue' ? '#fff' : T.colors.label, fontWeight: '600' }}>Gate queue</Text>
          </Pressable>
          <Pressable
            onPress={() => setTab('history')}
            style={{ flex: 1, padding: 8, borderRadius: 8, backgroundColor: tab === 'history' ? T.colors.tint : T.colors.secondaryBackground }}
          >
            <Text style={{ textAlign: 'center', color: tab === 'history' ? '#fff' : T.colors.label, fontWeight: '600' }}>History</Text>
          </Pressable>
        </View>
        {tab === 'queue' && (
          <>
            <View style={{ flexDirection: 'row', gap: 8 }}>
              <Pressable
                onPress={() => setScanMode('manual')}
                style={{ flex: 1, padding: 8, borderRadius: 8, backgroundColor: scanMode === 'manual' ? T.colors.tint : T.colors.secondaryBackground }}
              >
                <Text style={{ textAlign: 'center', color: scanMode === 'manual' ? '#fff' : T.colors.label, fontWeight: '600' }}>Manual</Text>
              </Pressable>
              <Pressable
                onPress={() => {
                  setScanMode('camera');
                  if (!permission?.granted) void requestPermission();
                }}
                style={{ flex: 1, padding: 8, borderRadius: 8, backgroundColor: scanMode === 'camera' ? T.colors.tint : T.colors.secondaryBackground }}
              >
                <Text style={{ textAlign: 'center', color: scanMode === 'camera' ? '#fff' : T.colors.label, fontWeight: '600' }}>Camera</Text>
              </Pressable>
            </View>
            {scanMode === 'camera' && permission?.granted && cameraEnabled && (
              <CameraView
                style={{ height: 160, borderRadius: 8, overflow: 'hidden' }}
                facing="back"
                barcodeScannerSettings={{ barcodeTypes: ['ean13', 'ean8'] }}
                onBarcodeScanned={onBarcodeScanned}
              />
            )}
            {scanMode === 'camera' && !permission?.granted && (
              <Pressable onPress={() => void requestPermission()} style={{ padding: 12, backgroundColor: T.colors.secondaryBackground, borderRadius: 8 }}>
                <Text style={{ textAlign: 'center', color: T.colors.label }}>Grant camera permission to scan EAN</Text>
              </Pressable>
            )}
            <Text style={{ fontSize: 12, color: T.colors.secondaryLabel }}>EAN / barcode</Text>
            <View style={{ flexDirection: 'row', gap: 8 }}>
              <TextInput
                value={barcode}
                onChangeText={setBarcode}
                placeholder="Scan or type EAN"
                style={{ flex: 1, borderWidth: 1, borderColor: T.colors.separator, borderRadius: 8, padding: 12, color: T.colors.label }}
                onSubmitEditing={() => void handleScan()}
              />
              <Pressable
                onPress={() => void handleScan()}
                disabled={scanning}
                style={{ backgroundColor: T.colors.tint, paddingHorizontal: 16, justifyContent: 'center', borderRadius: 8, opacity: scanning ? 0.6 : 1 }}
              >
                {scanning ? <ActivityIndicator color="#fff" /> : <Text style={{ color: '#fff', fontWeight: '700' }}>Scan</Text>}
              </Pressable>
            </View>
            <View style={{ flexDirection: 'row', gap: 8, marginTop: 8 }}>
              <Pressable onPress={() => void handleConfirm('RESTOCK')} style={{ flex: 1, backgroundColor: '#16a34a', padding: 12, borderRadius: 8 }}>
                <Text style={{ color: '#fff', textAlign: 'center', fontWeight: '700' }}>Restock</Text>
              </Pressable>
              <Pressable onPress={() => void handleConfirm('WRITE_OFF')} style={{ flex: 1, backgroundColor: '#dc2626', padding: 12, borderRadius: 8 }}>
                <Text style={{ color: '#fff', textAlign: 'center', fontWeight: '700' }}>Write off</Text>
              </Pressable>
            </View>
            {queuedScans > 0 && (
              <Text style={{ fontSize: 12, color: '#ea580c' }}>
                {queuedScans} scan(s) queued offline
              </Text>
            )}
          </>
        )}
      </View>

      {loading ? (
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
          <ActivityIndicator size="large" color={T.colors.tint} />
        </View>
      ) : (
        <InboundReturnsList 
          theme={T} 
          list={list} 
          selectable={tab === 'queue'} 
          selected={selected} 
          onToggle={toggle} 
          emptyText={tab === 'queue' ? 'No trucks awaiting gate receive.' : 'No completed receives yet.'} 
        />
      )}
    </View>
  );
}
