import { useCallback, useState } from 'react';
import * as Haptics from 'expo-haptics';

import { authFetch } from '../authSession';
import type { TruckRecommendation } from '../components/RecommendationCard';
import { extractProblemMessage } from '../localization';
import {
  payloadApplyReassignKey,
  payloadRecommendReassignKey,
} from '../utils/idempotency';
import type { Locale } from '../../../packages/i18n/locales';
import type { useManifestData } from './useManifestData';
import type { ShowToast } from './useToast';

// ─── Re-dispatch (reassign order to another truck) ────────────────────────────

export function useReDispatch({
  token,
  locale,
  tx,
  showToast,
  authHeaders,
  data,
}: {
  token: string | null;
  locale: Locale;
  tx: (key: string) => string;
  showToast: ShowToast;
  authHeaders: Record<string, string | undefined>;
  data: ReturnType<typeof useManifestData>;
}) {
  const {
    orders,
    sealedOrderIds,
    selectedOrderId,
    setManifest,
    setOrders,
    setSelectedOrderId,
  } = data;

  // Re-dispatch state
  const [showReDispatch, setShowReDispatch] = useState(false);
  const [reDispatchOrderId, setReDispatchOrderId] = useState<string | null>(null);
  const [reDispatchRetailer, setReDispatchRetailer] = useState('');
  const [reDispatchVolume, setReDispatchVolume] = useState(0);
  const [recommendations, setRecommendations] = useState<TruckRecommendation[]>([]);
  const [isLoadingRecs, setIsLoadingRecs] = useState(false);
  const [isReassigning, setIsReassigning] = useState(false);

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

  return {
    showReDispatch,
    setShowReDispatch,
    reDispatchOrderId,
    setReDispatchOrderId,
    reDispatchRetailer,
    reDispatchVolume,
    recommendations,
    isLoadingRecs,
    isReassigning,
    openReDispatch,
    handleReassign,
  };
}
