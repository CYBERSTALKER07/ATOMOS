'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { getAdminToken } from '@/lib/auth';
import {
  RotateCcw,
  Zap,
  RefreshCw,
  ArrowRightLeft,
} from 'lucide-react';

// ── Quick Actions Cell — The Control (2×1) ──────────────────────────────────
// High-consequence operational buttons. Each triggers a backend action with
// a confirmation step to prevent misfire.

type ActionConfig = {
  id: string;
  label: string;
  sub: string;
  icon: React.ReactNode;
  endpoint: string;
  method: string;
  confirm: string;
  dangerLevel: 'normal' | 'caution' | 'danger';
  navigateTo?: string;
};

const ACTIONS: ActionConfig[] = [
  {
    id: 'emergency-reroute',
    label: 'Emergency Reroute',
    sub: 'Re-dispatch all active routes',
    icon: <Zap size={16} strokeWidth={2} />,
    endpoint: '/v1/fleet/dispatch',
    method: 'POST',
    confirm: 'This will re-dispatch ALL active routes. Continue?',
    dangerLevel: 'danger',
  },
  {
    id: 'inventory-reset',
    label: 'Reset Loads',
    sub: 'Clear stuck truck loads',
    icon: <RotateCcw size={16} strokeWidth={2} />,
    endpoint: '/v1/fleet/capacity',
    method: 'POST',
    confirm: 'Reset all stuck truck load states?',
    dangerLevel: 'caution',
  },
  {
    id: 'sync-fleet',
    label: 'Sync Fleet',
    sub: 'Force telemetry refresh',
    icon: <RefreshCw size={16} strokeWidth={2} />,
    endpoint: '/v1/fleet/active',
    method: 'GET',
    confirm: '',
    dangerLevel: 'normal',
  },
  {
    id: 'reassign',
    label: 'Bulk Reassign',
    sub: 'Move orders between trucks',
    icon: <ArrowRightLeft size={16} strokeWidth={2} />,
    // /v1/fleet/reassign requires {order_ids, new_route_id} — can't be triggered
    // from a blank-body tile. Navigate to the orders page which has the proper
    // multi-select + target-truck dialog.
    endpoint: '/supplier/orders',
    method: 'NAVIGATE',
    confirm: '',
    dangerLevel: 'caution',
    navigateTo: '/supplier/orders',
  },
];

export default function QuickActionsCell() {
  const router = useRouter();
  const [executing, setExecuting] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<{ id: string; ok: boolean; msg: string } | null>(null);

  const execute = async (action: ActionConfig) => {
    if (action.navigateTo) {
      router.push(action.navigateTo);
      return;
    }
    if (action.confirm && !window.confirm(action.confirm)) return;

    setExecuting(action.id);
    setFeedback(null);

    try {
      const token = await getAdminToken();
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${action.endpoint}`, {
        method: action.method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: action.method !== 'GET' ? JSON.stringify({}) : undefined,
      });

      setFeedback({
        id: action.id,
        ok: res.ok,
        msg: res.ok ? 'Done' : `Error ${res.status}`,
      });
    } catch {
      setFeedback({ id: action.id, ok: false, msg: 'Network error' });
    } finally {
      setExecuting(null);
      // Clear feedback after 3s
      setTimeout(() => setFeedback(null), 3000);
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="bento-card-header">
        <span className="bento-card-title">Quick Actions</span>
      </div>

      {/* Action Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 flex-1 content-start">
        {ACTIONS.map((action) => {
          const isExecuting = executing === action.id;
          const fb = feedback?.id === action.id ? feedback : null;

          return (
            <button
              key={action.id}
              onClick={() => execute(action)}
              disabled={isExecuting}
              className="flex flex-col items-center justify-center gap-1.5 p-3 rounded transition-all cursor-pointer disabled:opacity-50"
              style={{
                border: '1px solid var(--border)',
                background: fb
                  ? fb.ok
                    ? 'var(--success)'
                    : 'var(--danger)'
                  : 'transparent',
                color: fb ? 'var(--background)' : 'var(--foreground)',
              }}
            >
              <span style={{ opacity: isExecuting ? 0.4 : 1 }}>
                {action.icon}
              </span>
              <span className="md-typescale-label-small font-semibold text-center leading-tight">
                {fb ? fb.msg : action.label}
              </span>
              <span
                className="md-typescale-label-small text-center leading-tight"
                style={{ color: fb ? 'inherit' : 'var(--muted)', fontSize: '10px' }}
              >
                {action.sub}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
