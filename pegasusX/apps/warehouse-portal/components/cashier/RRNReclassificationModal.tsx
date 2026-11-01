'use client';

import React, { useState } from 'react';
import { TIYIN_PER_UZS, type CashierSessionView } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface RRNReclassificationModalProps {
  sessionId: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (updatedSession: CashierSessionView) => void;
}

export function RRNReclassificationModal({
  sessionId,
  isOpen,
  onClose,
  onSuccess,
}: RRNReclassificationModalProps) {
  const [orderId, setOrderId] = useState('');
  const [gatewayRRN, setGatewayRRN] = useState('');
  const [disputedUZS, setDisputedUZS] = useState<number>(0);
  const [epodCapturedAt, setEpodCapturedAt] = useState(() => new Date(Date.now() - 5 * 60000).toISOString().slice(0, 16));
  const [rrnCapturedAt, setRrnCapturedAt] = useState(() => new Date().toISOString().slice(0, 16));
  const [actorId, setActorId] = useState('cashier_admin');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const disputedMinor = Math.floor(disputedUZS * TIYIN_PER_UZS);
    if (!orderId.trim()) {
      setError('Order ID is required');
      return;
    }
    if (!gatewayRRN.trim()) {
      setError('Gateway RRN is required');
      return;
    }
    if (disputedMinor <= 0) {
      setError('Disputed cash amount must be > 0');
      return;
    }

    const epodMs = Date.parse(epodCapturedAt);
    const rrnMs = Date.parse(rrnCapturedAt);
    if (isNaN(epodMs) || isNaN(rrnMs)) {
      setError('Invalid date-time format for ePOD or RRN timestamps');
      return;
    }

    const driftMs = Math.abs(rrnMs - epodMs);
    const maxDriftMs = 15 * 60 * 1000; // 15 minutes statutory rule
    if (driftMs > maxDriftMs) {
      setError(
        `RRN proof timestamp drift (${Math.round(driftMs / 60000)}m) exceeds maximum allowable 15-minute window!`,
      );
      return;
    }

    setSubmitting(true);
    try {
      const updated = await foundryWarehouseApi.reclassifyTender({
        session_id: sessionId,
        order_id: orderId.trim(),
        disputed_cash_minor: disputedMinor,
        gateway_rrn: gatewayRRN.trim(),
        rrn_amount_minor: disputedMinor, // Strict invariant: rrn_amount_minor == disputed_cash_minor
        epod_captured_at: new Date(epodMs).toISOString(),
        rrn_captured_at: new Date(rrnMs).toISOString(),
        actor_id: actorId.trim() || 'cashier_desk',
      });
      onSuccess(updated);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'RRN tender reclassification failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
      <div className="bg-slate-900 border border-slate-700 rounded-2xl w-full max-w-lg shadow-2xl p-6 text-slate-100">
        <div className="flex items-center justify-between pb-3 border-b border-slate-800">
          <div>
            <h3 className="text-lg font-bold text-slate-100">RRN Tender Reclassification</h3>
            <p className="text-xs text-slate-400 mt-0.5">
              Zero-destructive cash-to-card Humo/Uzcard gateway correction.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200 text-lg p-1"
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Order ID
            </label>
            <input
              type="text"
              required
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
              placeholder="e.g. ord_2026_1001"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Gateway RRN (Retrieval Reference Number)
            </label>
            <input
              type="text"
              required
              value={gatewayRRN}
              onChange={(e) => setGatewayRRN(e.target.value)}
              placeholder="e.g. 987654321012"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Disputed Amount (UZS)
            </label>
            <div className="relative">
              <input
                type="number"
                min="1"
                required
                value={disputedUZS === 0 ? '' : disputedUZS}
                onChange={(e) => setDisputedUZS(Math.max(0, parseInt(e.target.value || '0', 10) || 0))}
                placeholder="e.g. 500000"
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
              />
              <span className="absolute right-3 top-2.5 text-xs text-slate-400 font-mono">
                = {(disputedUZS * TIYIN_PER_UZS).toLocaleString()} Tiyin
              </span>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                ePOD Captured At
              </label>
              <input
                type="datetime-local"
                required
                value={epodCapturedAt}
                onChange={(e) => setEpodCapturedAt(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                RRN Terminal Timestamp
              </label>
              <input
                type="datetime-local"
                required
                value={rrnCapturedAt}
                onChange={(e) => setRrnCapturedAt(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Cashier / Actor ID
            </label>
            <input
              type="text"
              required
              value={actorId}
              onChange={(e) => setActorId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          {error && (
            <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
              {error}
            </div>
          )}

          <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-800">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting}
              className="px-4 py-2 rounded-xl text-xs font-semibold text-slate-400 hover:text-slate-200"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="px-5 py-2 rounded-xl text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-50"
            >
              {submitting ? 'Verifying & Reclassifying...' : 'Submit Reclassification'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
