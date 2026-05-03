'use client';

import { useState, useCallback } from 'react';
import { apiFetchNoQueue, useToken } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import { useTelemetry } from '@/hooks/useTelemetry';
import type { TelemetryMessage } from '@/hooks/useTelemetry';
import Icon from './Icon';
import { Button } from '@heroui/react';
import { useToast } from './Toast';
import { buildSupplierApproveEarlyCompleteIdempotencyKey } from '../app/supplier/_shared/idempotency';

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

  useTelemetry(
    useCallback((data: TelemetryMessage) => {
      const driverId = typeof data.driver_id === 'string' ? data.driver_id : '';
      if (data.type !== 'EARLY_COMPLETE_REQUESTED' || !driverId) {
        return;
      }
      setRequests((prev) => {
        if (prev.some((request) => request.driver_id === driverId)) {
          return prev;
        }
        return [{
          driver_id: driverId,
          driver_name: typeof data.driver_name === 'string' ? data.driver_name : '',
          reason: typeof data.reason === 'string' ? data.reason : '',
          note: typeof data.note === 'string' ? data.note : '',
          remaining_orders: typeof data.remaining_orders === 'number' ? data.remaining_orders : 0,
          requested_at: new Date().toISOString(),
        }, ...prev];
      });
    }, []),
    { enabled: !isTauri() && Boolean(token) },
  );

  const approve = useCallback(async (driverId: string) => {
    if (!token) return;
    setResolving(driverId);
    try {
      const res = await apiFetchNoQueue('/v1/admin/route/approve-early-complete', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierApproveEarlyCompleteIdempotencyKey(driverId),
        },
        body: JSON.stringify({ driver_id: driverId }),
      });
      const body = await res.json().catch(() => ({} as { error?: string; message?: string }));
      if (!res.ok) throw new Error(body.error || body.message || 'Failed to approve');
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
