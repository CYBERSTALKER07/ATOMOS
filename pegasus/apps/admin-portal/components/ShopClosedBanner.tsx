'use client';

import { useState, useEffect, useCallback } from 'react';
import { useToken, readTokenFromCookie } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import StatusBadge from './StatusBadge';
import Icon from './Icon';
import { Button } from '@heroui/react';
import { useToast } from './Toast';

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

  // WS listener for SHOP_CLOSED_ESCALATED events
  useEffect(() => {
    if (isTauri() || !token) return;

    let disposed = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let backoff = 1000;

    const connect = () => {
      if (disposed) return;
      const wsBase = API.replace(/^http/, 'ws');
      const url = new URL('/ws/telemetry', wsBase);
      const wsToken = readTokenFromCookie() || token;
      if (wsToken) url.searchParams.set('token', wsToken);

      const ws = new WebSocket(url.toString());
      ws.onopen = () => { backoff = 1000; };
      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'SHOP_CLOSED_ESCALATED' && data.order_id) {
            setEscalations(prev => {
              if (prev.some(e => e.order_id === data.order_id)) return prev;
              return [{
                order_id: data.order_id,
                driver_id: data.driver_id || '',
                retailer_id: data.retailer_id || '',
                retailer_name: data.retailer_name || '',
                attempt_id: data.attempt_id || '',
                escalated_at: new Date().toISOString(),
              }, ...prev];
            });
          }
        } catch { /* ignore */ }
      };
      ws.onclose = () => {
        if (disposed) return;
        reconnectTimer = setTimeout(() => connect(), backoff);
        backoff = Math.min(backoff * 2, 30_000);
      };
      ws.onerror = () => {};
      return ws;
    };

    const ws = connect();
    return () => {
      disposed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, [token]);

  const resolve = useCallback(async (orderId: string, action: 'INSTRUCT_WAIT' | 'ISSUE_BYPASS' | 'RETURN_TO_DEPOT') => {
    if (!token) return;
    setResolving(orderId);
    try {
      const res = await fetch(`${API}/v1/admin/shop-closed/resolve`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
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
              onPress={() => resolve(esc.order_id, 'INSTRUCT_WAIT')}
            >
              Wait
            </Button>
            <Button
              size="sm"
              variant="primary"
              isDisabled={resolving === esc.order_id}
              onPress={() => resolve(esc.order_id, 'ISSUE_BYPASS')}
            >
              Bypass
            </Button>
            <Button
              size="sm"
              variant="danger"
              isDisabled={resolving === esc.order_id}
              onPress={() => resolve(esc.order_id, 'RETURN_TO_DEPOT')}
            >
              Return
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
