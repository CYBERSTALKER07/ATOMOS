'use client';

import React, { useState } from 'react';
import { TIYIN_PER_UZS, type DepotVaultLedgerEntry } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface CITHandoverManifestProps {
  depotId?: string;
  supplierId?: string;
  onHandoverComplete?: (entry: DepotVaultLedgerEntry) => void;
}

export function CITHandoverManifest({
  depotId = 'depot_tashkent_central',
  supplierId = 'sup_samarkand_dairy',
  onHandoverComplete,
}: CITHandoverManifestProps) {
  const [courierBadgeId, setCourierBadgeId] = useState('');
  const [citSealBarcode, setCitSealBarcode] = useState('');
  const [handoverUZS, setHandoverUZS] = useState<number>(0);
  const [sessionIdsInput, setSessionIdsInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [completedEntry, setCompletedEntry] = useState<DepotVaultLedgerEntry | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCompletedEntry(null);

    const amountMinor = Math.floor(handoverUZS * TIYIN_PER_UZS);
    if (amountMinor <= 0) {
      setError('Handover cash amount must be greater than 0');
      return;
    }
    if (!courierBadgeId.trim()) {
      setError('Armored courier badge ID is required');
      return;
    }
    if (!citSealBarcode.trim()) {
      setError('Armored deposit pouch seal barcode is required');
      return;
    }

    const sessionIds = sessionIdsInput
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);

    setSubmitting(true);
    try {
      const entry = await foundryWarehouseApi.citVaultHandover({
        depot_id: depotId,
        supplier_id: supplierId,
        courier_badge_id: courierBadgeId.trim(),
        cit_seal_barcode: citSealBarcode.trim(),
        amount_minor: amountMinor,
        session_ids: sessionIds.length > 0 ? sessionIds : ['sess_auto_reconcile'],
      });
      setCompletedEntry(entry);
      onHandoverComplete?.(entry);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'CIT Armored Handover failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-5">
      <div>
        <h3 className="text-lg font-bold text-slate-100 flex items-center gap-2">
          <span>🛡️</span>
          <span>CIT Armored Courier Handover Manifest (Inkassatsiya)</span>
        </h3>
        <p className="text-xs text-slate-400 mt-1">
          Consolidate balanced depot cashier safe balances for physical armored pickup.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Armored Courier Badge ID
            </label>
            <input
              type="text"
              required
              value={courierBadgeId}
              onChange={(e) => setCourierBadgeId(e.target.value)}
              placeholder="e.g. INK-77821"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              CIT Armored Bag Seal Barcode
            </label>
            <input
              type="text"
              required
              value={citSealBarcode}
              onChange={(e) => setCitSealBarcode(e.target.value)}
              placeholder="e.g. CIT-SEAL-889900"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
            Handover Amount (UZS)
          </label>
          <div className="relative">
            <input
              type="number"
              min="1"
              required
              value={handoverUZS === 0 ? '' : handoverUZS}
              onChange={(e) => setHandoverUZS(Math.max(0, parseInt(e.target.value || '0', 10) || 0))}
              placeholder="e.g. 15000000"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
            <span className="absolute right-3 top-2.5 text-xs text-slate-400 font-mono">
              = {(handoverUZS * TIYIN_PER_UZS).toLocaleString()} Tiyin
            </span>
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
            Associated Session IDs (comma-separated, optional)
          </label>
          <input
            type="text"
            value={sessionIdsInput}
            onChange={(e) => setSessionIdsInput(e.target.value)}
            placeholder="sess_101, sess_102"
            className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
          />
        </div>

        {error && (
          <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-amber-600 hover:bg-amber-500 text-white disabled:opacity-50 transition-all shadow-md flex items-center gap-2"
          >
            {submitting ? 'Executing Armored Handover...' : 'Generate CIT Handover & Deduct Vault'}
          </button>
        </div>
      </form>

      {/* Completed Handover Receipt */}
      {completedEntry && (
        <div className="mt-4 p-4 rounded-xl bg-slate-900 border border-emerald-500/50 space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-slate-800">
            <span className="text-xs font-bold text-emerald-400 flex items-center gap-1.5">
              <span>✅</span>
              <span>CIT Armored Outflow Ledger Entry Created</span>
            </span>
            <span className="text-[11px] font-mono text-slate-400">{completedEntry.vault_entry_id}</span>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
            <div>
              <span className="text-slate-400 block">Outflow Amount:</span>
              <span className="font-mono font-bold text-slate-100">
                {Math.abs(Math.floor(completedEntry.amount_minor / TIYIN_PER_UZS)).toLocaleString()} UZS
              </span>
            </div>
            <div>
              <span className="text-slate-400 block">Remaining Safe Balance:</span>
              <span className="font-mono font-bold text-emerald-400">
                {Math.floor(completedEntry.running_balance_minor / TIYIN_PER_UZS).toLocaleString()} UZS
              </span>
            </div>
            <div>
              <span className="text-slate-400 block">Courier Badge:</span>
              <span className="font-mono text-slate-200">{completedEntry.courier_badge_id}</span>
            </div>
            <div>
              <span className="text-slate-400 block">Bag Seal:</span>
              <span className="font-mono text-slate-200">{completedEntry.seal_barcode}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
