'use client';

import React, { useEffect, useState } from 'react';
import { type DepotVaultLedgerEntry } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { majorIntToMinor } from '@/lib/money';
import { formatWholeSom, formatWholeSomInteger } from '@/lib/cashier-helpers';
import { Shield, CheckCircle2, Barcode, Printer, AlertCircle } from 'lucide-react';

interface CITHandoverManifestProps {
  depotId?: string;
  supplierId?: string;
  defaultSessionId?: string;
  onHandoverComplete?: (entry: DepotVaultLedgerEntry) => void;
}

const CARRIER_PRESETS = [
  { name: 'Inkassatsiya Xizmati DUK (Central Bank Armored Division)' },
  { name: 'NBU Security Logistics Fleet' },
  { name: 'Trastbank Armored Carrier Service' },
];

export function CITHandoverManifest({
  depotId = '',
  supplierId = '',
  defaultSessionId = '',
  onHandoverComplete,
}: CITHandoverManifestProps) {
  const [courierBadgeId, setCourierBadgeId] = useState('');
  const [citSealBarcode, setCitSealBarcode] = useState('');
  const [handoverSom, setHandoverSom] = useState<number>(0);
  const [sessionIdsInput, setSessionIdsInput] = useState(defaultSessionId);
  const [selectedCarrier, setSelectedCarrier] = useState(CARRIER_PRESETS[0].name);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [completedEntry, setCompletedEntry] = useState<DepotVaultLedgerEntry | null>(null);

  useEffect(() => {
    if (!sessionIdsInput && defaultSessionId) {
      setSessionIdsInput(defaultSessionId);
    }
  }, [defaultSessionId, sessionIdsInput]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setCompletedEntry(null);

    const amountMinor = majorIntToMinor(handoverSom);
    if (amountMinor <= 0) {
      setError("Handover cash amount must be greater than 0 so'm");
      return;
    }
    if (!depotId.trim() || !supplierId.trim()) {
      setError('Depot and Supplier scope are required for CIT handover');
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
    if (sessionIds.length === 0) {
      setError('At least one reconciled session ID is required');
      return;
    }

    setSubmitting(true);
    try {
      const entry = await foundryWarehouseApi.citVaultHandover({
        depot_id: depotId.trim(),
        supplier_id: supplierId.trim(),
        courier_badge_id: courierBadgeId.trim(),
        cit_seal_barcode: citSealBarcode.trim(),
        amount_minor: amountMinor,
        session_ids: sessionIds,
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
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-5 backdrop-blur-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-neutral-800">
        <div>
          <div className="flex items-center gap-2">
            <Shield size={20} className="text-amber-400" />
            <h3 className="text-base font-bold text-neutral-100 tracking-wide uppercase">
              CIT Armored Courier Handover Manifest (Inkassatsiya)
            </h3>
          </div>
          <p className="text-xs text-neutral-400 mt-0.5">
            Consolidate cleared depot cashier safe balances for physical armored carrier pickup and vault debit.
          </p>
        </div>

        <span className="px-3 py-1 rounded-full text-xs font-mono font-semibold bg-amber-500/10 text-amber-300 border border-amber-500/30">
          Dual-Custody Vault Outflow
        </span>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Carrier preset selection */}
        <div>
          <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
            Armored Courier Service Provider
          </label>
          <select
            value={selectedCarrier}
            onChange={(e) => setSelectedCarrier(e.target.value)}
            className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
          >
            {CARRIER_PRESETS.map((c) => (
              <option key={c.name} value={c.name}>
                {c.name}
              </option>
            ))}
          </select>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Armored Courier Badge ID
            </label>
            <input
              type="text"
              required
              value={courierBadgeId}
              onChange={(e) => setCourierBadgeId(e.target.value)}
              placeholder="Enter armored courier badge ID"
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              CIT Armored Bag Seal Barcode
            </label>
            <div className="relative">
              <input
                type="text"
                required
                value={citSealBarcode}
                onChange={(e) => setCitSealBarcode(e.target.value)}
                placeholder="Scan armored bag seal barcode"
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500 pr-10"
              />
              <Barcode size={18} className="absolute right-3 top-2 text-neutral-500" />
            </div>
          </div>
        </div>

        <div>
          <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
            Handover Cash Amount (so&apos;m)
          </label>
          <div className="relative">
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              required
              value={handoverSom === 0 ? '' : handoverSom}
              onChange={(e) => {
                const s = e.target.value.replace(/\D/g, '');
                setHandoverSom(Math.max(0, parseInt(s || '0', 10) || 0));
              }}
              placeholder="Enter whole so'm amount"
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
            />
            <span className="absolute right-3 top-2 text-[11px] text-neutral-500 font-mono">
              {formatWholeSomInteger(handoverSom)}
            </span>
          </div>
        </div>

        <div>
          <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
            Associated Turn-In Sessions (Comma-separated)
          </label>
          <input
            type="text"
            value={sessionIdsInput}
            onChange={(e) => setSessionIdsInput(e.target.value)}
            placeholder="Enter session IDs (comma-separated)"
            className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
          />
        </div>

        {error && (
          <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200 flex items-center gap-2">
            <AlertCircle size={15} className="shrink-0 text-rose-400" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-amber-600 hover:bg-amber-500 text-white disabled:opacity-40 transition-all shadow flex items-center gap-2"
          >
            {submitting ? 'Executing Armored Handover...' : 'Generate CIT Handover & Deduct Vault Safe'}
          </button>
        </div>
      </form>

      {/* Completed Handover Receipt */}
      {completedEntry && (
        <div className="mt-5 p-4 rounded-xl bg-neutral-950 border border-emerald-500/50 space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-neutral-800">
            <span className="text-xs font-bold text-emerald-400 flex items-center gap-1.5">
              <CheckCircle2 size={16} />
              <span>CIT Armored Outflow Ledger Entry Created</span>
            </span>
            <div className="flex items-center gap-2">
              <span className="text-[11px] font-mono text-neutral-400">{completedEntry.vault_entry_id}</span>
              <button
                type="button"
                onClick={() => window.print()}
                className="flex items-center gap-1 px-2 py-0.5 rounded bg-neutral-900 border border-neutral-800 text-[10px] text-neutral-300 hover:text-white"
              >
                <Printer size={11} />
                <span>Print</span>
              </button>
            </div>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase">Outflow Amount:</span>
              <span className="font-mono font-bold text-neutral-100">
                {formatWholeSom(Math.abs(completedEntry.amount_minor))}
              </span>
            </div>
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase">Remaining Safe Balance:</span>
              <span className="font-mono font-bold text-emerald-400">
                {formatWholeSom(completedEntry.running_balance_minor)}
              </span>
            </div>
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase">Courier Badge:</span>
              <span className="font-mono text-neutral-200">{completedEntry.courier_badge_id}</span>
            </div>
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase">Bag Seal Barcode:</span>
              <span className="font-mono text-neutral-200">{completedEntry.seal_barcode}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
