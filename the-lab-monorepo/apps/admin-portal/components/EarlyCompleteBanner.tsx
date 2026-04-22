'use client';

import { useState, useEffect, useCallback } from 'react';
import { useToken, readTokenFromCookie } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import Icon from './Icon';
import { Button } from '@heroui/react';
import { useToast } from './Toast';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface EarlyCompleteRequest {
  driver_id: string;
  driver_name?: string;
  reason: string;
  note?: string;
  remaining_orders: number;
  requested_at: string;
}

export default function EarlyCompleteBanner() {
  const token = useToken();
  const { toast } = useToast();
  const [requests, setRequests] = useState<EarlyCompleteRequest[]>([]);
  const [resolving, setResolving] = useState<string | null>(null);

  // WS listener for EARLY_COMPLETE_REQUESTED events
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
          if (data.type === 'EARLY_COMPLETE_REQUESTED' && data.driver_id) {
            setRequests(prev => {
              if (prev.some(r => r.driver_id === data.driver_id)) return prev;
              return [{
                driver_id: data.driver_id,
                driver_name: data.driver_name || '',
                reason: data.reason || '',
                note: data.note || '',
                remaining_orders: data.remaining_orders || 0,
                requested_at: new Date().toISOString(),
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

  const approve = useCallback(async (driverId: string) => {
    if (!token) return;
    setResolving(driverId);
    try {
      const res = await fetch(`${API}/v1/admin/route/approve-early-complete`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ driver_id: driverId }),
      });
      if (!res.ok) throw new Error('Failed to approve');
      setRequests(prev => prev.filter(r => r.driver_id !== driverId));
      toast(`Early route complete approved for ${driverId.slice(0, 12)}…`, 'success');
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setResolving(null);
    }
  }, [token, toast]);

  const dismiss = useCallback((driverId: string) => {
    setRequests(prev => prev.filter(r => r.driver_id !== driverId));
  }, []);

  if (requests.length === 0) return null;

  return (
    <div className="space-y-2 mb-4">
      {requests.map(req => (
        <div
          key={req.driver_id}
          className="flex items-center gap-4 px-4 py-3 md-shape-md"
          style={{
            background: 'var(--color-md-error-container, rgba(220,38,38,0.08))',
            border: '1px solid var(--color-md-error, #dc2626)',
          }}
        >
          <Icon name="warning" className="w-5 h-5 shrink-0 text-danger" />

          <div className="flex-1 min-w-0">
            <p className="md-typescale-label-medium font-semibold" style={{ color: 'var(--color-md-on-surface)' }}>
              Early Route Complete Request
            </p>
            <p className="md-typescale-body-small text-muted truncate">
              {req.driver_name || req.driver_id.slice(0, 16)} — {req.reason.replace(/_/g, ' ')}
              {req.note ? ` • "${req.note}"` : ''}
              {req.remaining_orders > 0 && ` • ${req.remaining_orders} orders will be quarantined`}
            </p>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <Button
              size="sm"
              variant="danger"
              isDisabled={resolving === req.driver_id}
              onPress={() => approve(req.driver_id)}
            >
              Approve
            </Button>
            <Button
              size="sm"
              variant="secondary"
              isDisabled={resolving === req.driver_id}
              onPress={() => dismiss(req.driver_id)}
            >
              Dismiss
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
