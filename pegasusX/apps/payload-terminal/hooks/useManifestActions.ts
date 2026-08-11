import { useState, type Dispatch, type SetStateAction } from 'react';
import * as Haptics from 'expo-haptics';
import * as SecureStore from 'expo-secure-store';

import { PayloadTerminalApi } from '../api';
import { authFetch } from '../authSession';
import { ApiExplainError, type StatusExplain } from '../explainBanner';
import { extractProblemMessage } from '../localization';
import { isIOS } from '../theme';
import {
  payloadManifestExceptionKey,
  payloadOrderSealKey,
  payloadSealCompletedKey,
  payloadSupplierInjectKey,
  payloadSupplierStartLoadingKey,
} from '../utils/idempotency';
import type { Locale } from '../../../packages/i18n/locales';
import type { useManifestData } from './useManifestData';
import type { QueuedAction } from './useOfflineQueue';
import type { ShowToast } from './useToast';

// ─── Manifest actions ─────────────────────────────────────────────────────────
// Seal / exception / inject / checklist mutations against the manifest domain.

export function useManifestActions({
  token,
  locale,
  tx,
  showToast,
  data,
  queue,
}: {
  token: string | null;
  locale: Locale;
  tx: (key: string) => string;
  showToast: ShowToast;
  data: ReturnType<typeof useManifestData>;
  queue: {
    offlineQueue: QueuedAction[];
    setOfflineQueue: Dispatch<SetStateAction<QueuedAction[]>>;
  };
}) {
  const {
    activeTruck,
    allChecked,
    batchReadyManifestIds,
    countdownRef,
    fetchManifest,
    fetchTruckManifest,
    isInjecting,
    setIsInjecting,
    isOnline,
    manifestId,
    orders,
    sealedOrderIds,
    selectedManifest,
    selectedOrderId,
    setAllSealed,
    setBatchReadyManifestIds,
    setDispatchCodes,
    setIsSealingManifest,
    setIsStartingLoad,
    setManifest,
    setManifestState,
    setOrders,
    setPostSealCountdown,
    setPostSealOrderId,
    setSealedOrderIds,
    setSealedOrdersByTruck,
    setSelectedOrderId,
  } = data;
  const { offlineQueue, setOfflineQueue } = queue;

  const [isSealing, setIsSealing] = useState(false);
  const [sealExplain, setSealExplain] = useState<StatusExplain | null>(null);
  const [exceptionLoading, setExceptionLoading] = useState<string | null>(null); // orderId being excepted
  const [showInjectOrder, setShowInjectOrder] = useState(false);
  const [showProductScanner, setShowProductScanner] = useState(false);
  const [showInjectScanner, setShowInjectScanner] = useState(false);
  const [injectOrderId, setInjectOrderId] = useState('');
  const [batchSealing, setBatchSealing] = useState(false);
  const [batchSealFailures, setBatchSealFailures] = useState<Array<{
    manifest_id?: string;
    status?: string;
    explain?: StatusExplain;
  }>>([]);

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
    setSealExplain(null);
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
      if (e instanceof ApiExplainError) {
        setSealExplain(e.explain);
        showToast(tx('payload.alert.seal_failed'), e.message, 'error');
      } else {
        showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
      }
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

  // ── Checkbox toggle ───────────────────────────────────────────────────────
  const toggleCheck = (itemId: string) => {
    setManifest(prev =>
      prev.map(item => {
        if (item.id === itemId) {
          const isComplete = !item.scanned;
          return {
            ...item,
            scanned: isComplete,
            verifiedQuantity: isComplete ? item.quantity : 0
          };
        }
        return item;
      })
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

      const unverifiedMatch = selectedManifest.find((item) => item.brand === sku && item.verifiedQuantity < item.quantity);
      if (!unverifiedMatch) {
        const fullMatch = selectedManifest.find((item) => item.brand === sku);
        if (fullMatch) {
            showToast(isIOS ? 'Item fully scanned' : 'ITEM FULLY SCANNED', '', 'info', 2400);
            return;
        }
        showToast(isIOS ? 'SKU not on this order' : 'SKU NOT ON THIS ORDER', '', 'warning', 2400);
        return;
      }

      setManifest(prev =>
        prev.map(item => {
          if (item.id === unverifiedMatch.id) {
            const newQty = item.verifiedQuantity + 1;
            return {
              ...item,
              verifiedQuantity: newQty,
              scanned: newQty >= item.quantity
            };
          }
          return item;
        })
      );

      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      showToast(isIOS ? 'Checked item' : 'CHECKED ITEM', product.name ?? sku, 'success', 2200);

      // Let the user scan more without closing the modal
      // setShowProductScanner(false);
    } catch (e: unknown) {
      showToast(
        isIOS ? 'Barcode lookup failed' : 'BARCODE LOOKUP FAILED',
        e instanceof Error ? e.message : tx('common.error.unknown'),
        'error',
      );
    }
  };

  // ── Seal & dispatch ───────────────────────────────────────────────────────
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
      if (e instanceof ApiExplainError) {
        setSealExplain(e.explain);
        showToast(tx('payload.alert.seal_failed'), e.message, 'error');
      } else {
        showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
      }
    } finally {
      setIsSealing(false);
    }
  };

  const handleFinalizeBatchSeal = async () => {
    if (batchReadyManifestIds.length < 2 || batchSealing || !token) return;
    setBatchSealing(true);
    setSealExplain(null);
    setBatchSealFailures([]);
    const manifestCount = batchReadyManifestIds.length;
    try {
      const data = await PayloadTerminalApi.sealCompletedManifests(
        token,
        batchReadyManifestIds,
        payloadSealCompletedKey(batchReadyManifestIds),
      );
      const failures = Array.isArray(data.results)
        ? data.results.filter((row: { status?: string }) => row.status && row.status !== 'sealed')
        : [];
      if (failures.length > 0) {
        setBatchSealFailures(failures);
        const firstExplain = failures.find((row: { explain?: StatusExplain }) => row.explain)?.explain ?? null;
        setSealExplain(firstExplain);
        showToast(
          tx('payload.alert.seal_failed'),
          failures.map((row: { manifest_id?: string; status?: string; explain?: StatusExplain }) =>
            row.explain?.title ?? `${row.manifest_id ?? 'manifest'}: ${row.status ?? 'failed'}`
          ).join(' · '),
          'error',
        );
        return;
      }
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
      if (e instanceof ApiExplainError) {
        setSealExplain(e.explain);
        showToast(tx('payload.alert.seal_failed'), e.message, 'error');
      } else {
        showToast(tx('payload.alert.seal_failed'), e instanceof Error ? e.message : tx('common.error.unknown'), 'error');
      }
    } finally {
      setBatchSealing(false);
    }
  };

  return {
    isSealing,
    sealExplain,
    exceptionLoading,
    showInjectOrder,
    setShowInjectOrder,
    showProductScanner,
    setShowProductScanner,
    showInjectScanner,
    setShowInjectScanner,
    injectOrderId,
    setInjectOrderId,
    batchSealing,
    batchSealFailures,
    handleStartLoading,
    handleException,
    handleManifestSeal,
    handleInjectOrder,
    toggleCheck,
    handleProductBarcodeScan,
    handleSeal,
    handleFinalizeBatchSeal,
  };
}
