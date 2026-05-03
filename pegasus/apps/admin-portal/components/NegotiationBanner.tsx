'use client';

import { useState, useCallback } from 'react';
import { useToken } from '@/lib/auth';
import { isTauri } from '@/lib/bridge';
import { useTelemetry } from '@/hooks/useTelemetry';
import type { TelemetryMessage } from '@/hooks/useTelemetry';
import Icon from './Icon';
import { Button } from '@heroui/react';
import { useToast } from './Toast';
import { buildSupplierNegotiationResolveIdempotencyKey } from '../app/supplier/_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface NegotiationProposal {
  proposal_id: string;
  order_id: string;
  driver_id: string;
  driver_name?: string;
  retailer_name?: string;
  items: { sku_id: string; original_qty: number; proposed_qty: number }[];
  proposed_at: string;
}

export default function NegotiationBanner() {
  const token = useToken();
  const { toast } = useToast();
  const [proposals, setProposals] = useState<NegotiationProposal[]>([]);
  const [resolving, setResolving] = useState<string | null>(null);

  useTelemetry(
    useCallback((data: TelemetryMessage) => {
      const proposalId = typeof data.proposal_id === 'string' ? data.proposal_id : '';
      if (data.type !== 'NEGOTIATION_PROPOSED' || !proposalId) {
        return;
      }
      setProposals((prev) => {
        if (prev.some((proposal) => proposal.proposal_id === proposalId)) {
          return prev;
        }
        return [{
          proposal_id: proposalId,
          order_id: typeof data.order_id === 'string' ? data.order_id : '',
          driver_id: typeof data.driver_id === 'string' ? data.driver_id : '',
          driver_name: typeof data.driver_name === 'string' ? data.driver_name : '',
          retailer_name: typeof data.retailer_name === 'string' ? data.retailer_name : '',
          items: Array.isArray(data.items) ? data.items as NegotiationProposal['items'] : [],
          proposed_at: new Date().toISOString(),
        }, ...prev];
      });
    }, []),
    { enabled: !isTauri() && Boolean(token) },
  );

  const resolve = useCallback(async (proposalId: string, decision: 'APPROVE' | 'REJECT') => {
    if (!token) return;
    setResolving(proposalId);
    try {
      const res = await fetch(`${API}/v1/admin/negotiate/resolve`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierNegotiationResolveIdempotencyKey(proposalId, decision),
        },
        body: JSON.stringify({ proposal_id: proposalId, decision }),
      });
      if (!res.ok) throw new Error('Failed to resolve negotiation');
      setProposals(prev => prev.filter(p => p.proposal_id !== proposalId));
      toast(`Negotiation ${decision === 'APPROVE' ? 'approved' : 'rejected'} — ${proposalId.slice(0, 12)}…`, 'success');
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setResolving(null);
    }
  }, [token, toast]);

  if (proposals.length === 0) return null;

  return (
    <div className="space-y-2 mb-4">
      {proposals.map(prop => (
        <div
          key={prop.proposal_id}
          className="flex items-center gap-4 px-4 py-3 md-shape-md"
          style={{
            background: 'var(--color-md-info-container, rgba(59,130,246,0.08))',
            border: '1px solid var(--color-md-info, #3b82f6)',
          }}
        >
          <Icon name="orders" className="w-5 h-5 shrink-0 text-info" />

          <div className="flex-1 min-w-0">
            <p className="md-typescale-label-medium font-semibold" style={{ color: 'var(--color-md-on-surface)' }}>
              Live Negotiation
            </p>
            <p className="md-typescale-body-small text-muted truncate">
              Order {prop.order_id.slice(0, 12)}… • {prop.driver_name || prop.driver_id.slice(0, 12)}
              {prop.retailer_name && ` → ${prop.retailer_name}`}
              {prop.items.length > 0 && ` • ${prop.items.length} item(s) adjusted`}
            </p>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <Button
              size="sm"
              variant="primary"
              isDisabled={resolving === prop.proposal_id}
              onPress={() => resolve(prop.proposal_id, 'APPROVE')}
            >
              Approve
            </Button>
            <Button
              size="sm"
              variant="danger"
              isDisabled={resolving === prop.proposal_id}
              onPress={() => resolve(prop.proposal_id, 'REJECT')}
            >
              Reject
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
