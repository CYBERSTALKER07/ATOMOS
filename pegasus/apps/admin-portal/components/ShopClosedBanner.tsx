'use client';

import { useState, useCallback } from 'react';
import { useToken } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import { useTelemetry } from '@/hooks/useTelemetry';
import type { TelemetryMessage } from '@/hooks/useTelemetry';
import StatusBadge from './StatusBadge';
import Icon from './Icon';
import { Button } from '@heroui/react';
import { useToast } from './Toast';
import { buildSupplierShopClosedResolveIdempotencyKey } from '../app/supplier/_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ShopClosedEscalation {
  order_id: string;
  driver_id: string;
  retailer_id: string;
  retailer_name?: string;
  attempt_id: string;
  escalated_at: string;
}

export default function ShopClosedBanner() {
  const token = useToken();
  const { toast } = useToast();
  const [escalations, setEscalations] = useState<ShopClosedEscalation[]>([]);
  const [resolving, setResolving] = useState<string | null>(null);

  useTelemetry(
    useCallback((data: TelemetryMessage) => {
      const orderId = typeof data.order_id === 'string' ? data.order_id : '';
      if (data.type !== 'SHOP_CLOSED_ESCALATED' || !orderId) {
        return;
      }
      setEscalations((prev) => {
        if (prev.some((escalation) => escalation.order_id === orderId)) {
          return prev;
        }
        return [{
          order_id: orderId,
          driver_id: typeof data.driver_id === 'string' ? data.driver_id : '',
          retailer_id: typeof data.retailer_id === 'string' ? data.retailer_id : '',
          retailer_name: typeof data.retailer_name === 'string' ? data.retailer_name : '',
          attempt_id: typeof data.attempt_id === 'string' ? data.attempt_id : '',
          escalated_at: new Date().toISOString(),
        }, ...prev];
      });
    }, []),
    { enabled: !isTauri() && Boolean(token) },
  );

  const resolve = useCallback(async (
    attemptId: string,
    orderId: string,
    action: 'INSTRUCT_WAIT' | 'ISSUE_BYPASS' | 'RETURN_TO_DEPOT',
  ) => {
    if (!token) return;
    setResolving(orderId);
    try {
      const res = await fetch(`${API}/v1/admin/shop-closed/resolve`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierShopClosedResolveIdempotencyKey(attemptId || orderId, action),
        },
        body: JSON.stringify({ order_id: orderId, action }),
      });
      if (!res.ok) throw new Error('Failed to resolve');
      setEscalations(prev => prev.filter(e => e.order_id !== orderId));
      toast(`Order ${orderId.slice(0, 12)}… — ${action.replace(/_/g, ' ').toLowerCase()}`, 'success');
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setResolving(null);
    }
  }, [token, toast]);

  if (escalations.length === 0) return null;

  return (
    <div className="space-y-2 mb-4">
      {escalations.map(esc => (
        <div
          key={esc.order_id}
          className="flex items-center gap-4 px-4 py-3 md-shape-md"
          style={{
            background: 'var(--color-md-warning-container, rgba(234,179,8,0.12))',
            border: '1px solid var(--color-md-warning, #eab308)',
          }}
        >
          <Icon name="warning" className="w-5 h-5 shrink-0 text-warning" />

          <div className="flex-1 min-w-0">
            <p className="md-typescale-label-medium font-semibold" style={{ color: 'var(--color-md-on-surface)' }}>
              Shop Closed Escalation
            </p>
            <p className="md-typescale-body-small text-muted truncate">
              Order {esc.order_id.slice(0, 16)}… • {esc.retailer_name || esc.retailer_id}
            </p>
          </div>

          <StatusBadge state="ARRIVED_SHOP_CLOSED" />

          <div className="flex items-center gap-2 shrink-0">
            <Button
              size="sm"
              variant="secondary"
              isDisabled={resolving === esc.order_id}
              onPress={() => resolve(esc.attempt_id, esc.order_id, 'INSTRUCT_WAIT')}
            >
              Wait
            </Button>
            <Button
              size="sm"
              variant="primary"
              isDisabled={resolving === esc.order_id}
              onPress={() => resolve(esc.attempt_id, esc.order_id, 'ISSUE_BYPASS')}
            >
              Bypass
            </Button>
            <Button
              size="sm"
              variant="danger"
              isDisabled={resolving === esc.order_id}
              onPress={() => resolve(esc.attempt_id, esc.order_id, 'RETURN_TO_DEPOT')}
            >
              Return
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
