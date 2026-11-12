'use client';

import React, { useState, useMemo } from 'react';
import { type CashierSessionView } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { formatGroupedInteger, majorIntToMinor, majorIntToMinorBigInt } from '@/lib/money';
import { X, ShieldAlert, CheckCircle2, Clock, CreditCard, AlertCircle } from 'lucide-react';

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
  const [gatewayPreset, setGatewayPreset] = useState<'HUMO' | 'UZCARD' | 'VISA' | 'MASTERCARD'>('HUMO');
  const [actorId, setActorId] = useState('cashier_admin');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Real-time calculation of timestamp drift against the 15-minute statutory window
  const driftAnalysis = useMemo(() => {
    const epodMs = Date.parse(epodCapturedAt);
    const rrnMs = Date.parse(rrnCapturedAt);
    if (isNaN(epodMs) || isNaN(rrnMs)) {
      return { valid: false, driftMinutes: 0, exceedsLimit: false };
    }
    const diffMs = Math.abs(rrnMs - epodMs);
    const driftMinutes = Math.round(diffMs / 60000);
    const exceedsLimit = diffMs > 15 * 60 * 1000;
    return { valid: true, driftMinutes, exceedsLimit };
  }, [epodCapturedAt, rrnCapturedAt]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    const disputedMinor = majorIntToMinor(disputedUZS);
    if (!orderId.trim()) {
      setError('Order ID is required');
      return;
    }
    if (!gatewayRRN.trim()) {
      setError('Gateway RRN is required');
      return;
    }
    if (disputedMinor <= 0) {
      setError('Disputed cash amount must be > 0 tiyin');
      return;
    }

    if (!driftAnalysis.valid) {
      setError('Invalid date-time format for ePOD or RRN timestamps');
      return;
    }

    if (driftAnalysis.exceedsLimit) {
      setError(
        `RRN proof timestamp drift (${driftAnalysis.driftMinutes}m) exceeds maximum allowable 15-minute statutory window!`,
      );
      return;
    }

    setSubmitting(true);
    try {
      const updated = await foundryWarehouseApi.reclassifyTender({
        session_id: sessionId,
        order_id: orderId.trim(),
        disputed_cash_minor: disputedMinor,
        gateway_rrn: `${gatewayPreset}:${gatewayRRN.trim()}`,
        rrn_amount_minor: disputedMinor, // Strict invariant: rrn_amount_minor == disputed_cash_minor
        epod_captured_at: new Date(Date.parse(epodCapturedAt)).toISOString(),
        rrn_captured_at: new Date(Date.parse(rrnCapturedAt)).toISOString(),
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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm select-none">
      <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-lg shadow-2xl p-6 text-neutral-100">
        <div className="flex items-center justify-between pb-3.5 border-b border-neutral-800">
          <div>
            <div className="flex items-center gap-2">
              <CreditCard size={18} className="text-emerald-400" />
              <h3 className="text-base font-bold text-neutral-100 uppercase tracking-wide">
                RRN Tender Reclassification
              </h3>
            </div>
            <p className="text-xs text-neutral-400 mt-0.5">
              Zero-destructive cash-to-card terminal gateway correction (Humo / Uzcard).
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg border border-neutral-800 text-neutral-400 hover:border-neutral-700 hover:bg-neutral-800 hover:text-neutral-100 transition-colors"
            aria-label="Close modal"
          >
            <X size={16} />
          </button>
        </div>

        {/* Informative banner on zero-destructive ledger append */}
        <div className="mt-4 p-3 rounded-xl bg-neutral-950/80 border border-neutral-800 text-[11px] text-neutral-400 leading-relaxed">
          <span className="font-semibold text-neutral-200">Zero-Destructive Ledger Invariant:</span> Historical transactions are never mutated. This action appends a <code className="font-mono text-amber-300">CASH_REVERSAL</code> and an offsetting <code className="font-mono text-emerald-300">CARD_VERIFIED_RRN</code> ledger entry tied to this session.
        </div>

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                Order ID
              </label>
              <input
                type="text"
                required
                value={orderId}
                onChange={(e) => setOrderId(e.target.value)}
                placeholder="e.g. ord_2026_1001"
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                Payment Terminal Gateway
              </label>
              <select
                value={gatewayPreset}
                onChange={(e) => setGatewayPreset(e.target.value as 'HUMO' | 'UZCARD' | 'VISA' | 'MASTERCARD')}
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
              >
                <option value="HUMO">HUMO (Direct POS)</option>
                <option value="UZCARD">UZCARD (EMV Terminal)</option>
                <option value="VISA">VISA (International Acquirer)</option>
                <option value="MASTERCARD">MASTERCARD (Acquirer)</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Gateway RRN (Retrieval Reference Number)
            </label>
            <input
              type="text"
              required
              value={gatewayRRN}
              onChange={(e) => setGatewayRRN(e.target.value)}
              placeholder="e.g. 987654321012 (12-digit RRN)"
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Disputed Cash Amount (UZS)
            </label>
            <div className="relative">
              <input
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                required
                value={disputedUZS === 0 ? '' : disputedUZS}
                onChange={(e) => {
                  const s = e.target.value.replace(/\D/g, '');
                  setDisputedUZS(Math.max(0, parseInt(s || '0', 10) || 0));
                }}
                placeholder="e.g. 500000"
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
              />
              <span className="absolute right-3 top-2 text-[11px] text-neutral-500 font-mono">
                = {formatGroupedInteger(majorIntToMinorBigInt(disputedUZS))} Tiyin
              </span>
            </div>
          </div>

          {/* Timestamps & 15-Minute Statutory Gate */}
          <div className="p-3.5 rounded-xl border border-neutral-800 bg-neutral-950/60 space-y-3">
            <div className="flex items-center justify-between text-xs">
              <span className="font-semibold text-neutral-300 flex items-center gap-1.5">
                <Clock size={13} className="text-neutral-400" />
                <span>Statutory 15-Minute Window Audit</span>
              </span>
              <span
                className={`px-2 py-0.5 rounded font-mono text-[10px] font-bold border ${
                  driftAnalysis.exceedsLimit
                    ? 'bg-rose-500/10 text-rose-300 border-rose-500/30'
                    : 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30'
                }`}
              >
                Drift: {driftAnalysis.driftMinutes}m / 15m limit
              </span>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-[10px] font-semibold text-neutral-500 uppercase tracking-wider mb-1">
                  ePOD Signature Captured At
                </label>
                <input
                  type="datetime-local"
                  required
                  value={epodCapturedAt}
                  onChange={(e) => setEpodCapturedAt(e.target.value)}
                  className="w-full bg-neutral-900 border border-neutral-750 rounded-lg px-2.5 py-1.5 text-[11px] font-mono text-neutral-200 focus:outline-none focus:border-emerald-500"
                />
              </div>
              <div>
                <label className="block text-[10px] font-semibold text-neutral-500 uppercase tracking-wider mb-1">
                  POS Gateway Slip Captured At
                </label>
                <input
                  type="datetime-local"
                  required
                  value={rrnCapturedAt}
                  onChange={(e) => setRrnCapturedAt(e.target.value)}
                  className="w-full bg-neutral-900 border border-neutral-750 rounded-lg px-2.5 py-1.5 text-[11px] font-mono text-neutral-200 focus:outline-none focus:border-emerald-500"
                />
              </div>
            </div>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Authorizing Cashier / Supervisor ID
            </label>
            <input
              type="text"
              required
              value={actorId}
              onChange={(e) => setActorId(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          {error && (
            <div className="p-3 bg-rose-950/60 border border-rose-800 text-xs text-rose-200 rounded-xl flex items-center gap-2">
              <AlertCircle size={15} className="shrink-0 text-rose-400" />
              <span>{error}</span>
            </div>
          )}

          <div className="flex items-center justify-end gap-3 pt-3 border-t border-neutral-800">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting}
              className="px-4 py-2 rounded-xl text-xs font-semibold text-neutral-400 hover:text-neutral-200"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting || driftAnalysis.exceedsLimit}
              className="px-5 py-2 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-40 transition-all shadow flex items-center gap-1.5"
            >
              {submitting ? 'Reclassifying...' : 'Submit Reclassification'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
