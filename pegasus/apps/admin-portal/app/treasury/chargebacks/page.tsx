'use client';

import { useCallback, useEffect, useState } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ReconciliationRecord {
  order_id: string;
  retailer_id: string;
  spanner_amount: number;
  gateway_amount: number;
  gateway_provider: string;
  status: string;
  timestamp: string;
}

interface ReconciliationResponse {
  status: string;
  data: ReconciliationRecord[];
}

export default function ChargebacksPage() {
  const token = useToken();
  const { toast } = useToast();

  const [rows, setRows] = useState<ReconciliationRecord[]>([]);
  const [loading, setLoading] = useState(true);

  const [orderId, setOrderId] = useState('');
  const [retailerId, setRetailerId] = useState('');
  const [gateway, setGateway] = useState('CASH');
  const [amount, setAmount] = useState('0');
  const [sessionId, setSessionId] = useState('');

  const load = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch(`${API}/v1/admin/reconciliation`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error('Failed to load reconciliation queue');
      const payload = (await res.json()) as ReconciliationResponse;
      setRows((payload.data || []).filter((r) => r.status === 'CHARGEBACK' || r.status === 'REVERSAL'));
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Failed to load chargebacks', 'error');
    } finally {
      setLoading(false);
    }
  }, [token, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const createChargeback = useCallback(async () => {
    if (!token) return;
    if (!orderId || !retailerId || Number(amount) <= 0) {
      toast('order_id, retailer_id and amount are required', 'error');
      return;
    }
    try {
      const res = await fetch(`${API}/v1/global_paynt/chargeback`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          order_id: orderId,
          retailer_id: retailerId,
          gateway,
          amount_uzs: Number(amount),
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast('Chargeback recorded', 'success');
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Failed to record chargeback', 'error');
    }
  }, [amount, gateway, load, orderId, retailerId, token, toast]);

  const createReversal = useCallback(async () => {
    if (!token || !sessionId) {
      toast('session_id is required', 'error');
      return;
    }
    try {
      const res = await fetch(`${API}/v1/global_paynt/chargeback/reversal`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast('Reversal recorded', 'success');
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Failed to record reversal', 'error');
    }
  }, [load, sessionId, token, toast]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      <div>
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>Chargebacks</h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          Manage provider-initiated chargebacks and settlement reversals.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="md-card md-elevation-1 md-shape-md p-4" style={{ background: 'var(--color-md-surface)' }}>
          <h2 className="md-typescale-title-small mb-3">Record Chargeback</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <input className="md-input-outlined px-3 py-2" placeholder="Order ID" value={orderId} onChange={(e) => setOrderId(e.target.value)} />
            <input className="md-input-outlined px-3 py-2" placeholder="Retailer ID" value={retailerId} onChange={(e) => setRetailerId(e.target.value)} />
            <select className="md-input-outlined px-3 py-2" value={gateway} onChange={(e) => setGateway(e.target.value)}>
              <option value="CASH">CASH</option>
              <option value="GLOBAL_PAY">GLOBAL_PAY</option>
              <option value="GLOBAL_PAY">GLOBAL_PAY</option>
            </select>
            <input className="md-input-outlined px-3 py-2" placeholder="Amount UZS" type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} />
          </div>
          <Button variant="primary" className="mt-3" onPress={createChargeback}>Record</Button>
        </div>

        <div className="md-card md-elevation-1 md-shape-md p-4" style={{ background: 'var(--color-md-surface)' }}>
          <h2 className="md-typescale-title-small mb-3">Record Reversal</h2>
          <input className="md-input-outlined px-3 py-2 w-full" placeholder="GlobalPaynt Session ID" value={sessionId} onChange={(e) => setSessionId(e.target.value)} />
          <Button variant="outline" className="mt-3" onPress={createReversal}>Record Reversal</Button>
        </div>
      </div>

      {loading ? (
        <div className="md-card md-elevation-1 md-shape-md p-6" style={{ background: 'var(--color-md-surface)' }}>Loading…</div>
      ) : rows.length === 0 ? (
        <EmptyState icon="reconcile" headline="No chargeback anomalies" body="No CHARGEBACK or REVERSAL anomalies in the reconciliation queue." />
      ) : (
        <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <th className="text-left px-4 py-3">Order</th>
                <th className="text-left px-4 py-3">Retailer</th>
                <th className="text-right px-4 py-3">Spanner</th>
                <th className="text-right px-4 py-3">Gateway</th>
                <th className="text-left px-4 py-3">Provider</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-left px-4 py-3">Detected</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={`${r.order_id}-${r.timestamp}`} className="border-b last:border-b-0" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <td className="px-4 py-3 font-mono text-xs">{r.order_id}</td>
                  <td className="px-4 py-3 font-mono text-xs">{r.retailer_id}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{r.spanner_amount}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{r.gateway_amount}</td>
                  <td className="px-4 py-3">{r.gateway_provider}</td>
                  <td className="px-4 py-3">{r.status}</td>
                  <td className="px-4 py-3 text-xs">{new Date(r.timestamp).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
