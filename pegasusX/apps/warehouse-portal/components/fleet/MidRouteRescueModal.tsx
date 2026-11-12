'use client';

import React, { useState } from 'react';
import { type RescueToteItem, type MidRouteRescueResult } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { minorToWholeTrunc } from '@/lib/money';
import { X, ShieldAlert, AlertTriangle, Snowflake, CheckCircle2, ArrowRightLeft, AlertCircle } from 'lucide-react';

interface MidRouteRescueModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (result: MidRouteRescueResult) => void;
}

export function MidRouteRescueModal({
  isOpen,
  onClose,
  onSuccess,
}: MidRouteRescueModalProps) {
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [originalRouteId, setOriginalRouteId] = useState('route_yashnobod_101');
  const [brokenVehicleId, setBrokenVehicleId] = useState('trk_isuzu_04');
  const [originalDriverId, setOriginalDriverId] = useState('drv_rustam_09');
  const [rescueVehicleId, setRescueVehicleId] = useState('trk_hyundai_12');
  const [rescueDriverId, setRescueDriverId] = useState('drv_anvar_01');
  const [lastCompletedStopSeq, setLastCompletedStopSeq] = useState<number>(3);
  const [newRescueBoltSeal, setNewRescueBoltSeal] = useState('RESCUE-SEAL-99881');

  // Pre-populated default totes demonstrating cold-chain excursion check
  const [totes, setTotes] = useState<RescueToteItem[]>([
    {
      tote_id: 'tote_dry_01',
      stop_sequence: 1,
      is_cold_chain: false,
      expected_cash_minor: 40000000, // 400,000 UZS
      thermal_excursion_minutes: 0,
      max_allowed_excursion_mins: 120,
    },
    {
      tote_id: 'tote_dry_02',
      stop_sequence: 2,
      is_cold_chain: false,
      expected_cash_minor: 60000000, // 600,000 UZS
      thermal_excursion_minutes: 0,
      max_allowed_excursion_mins: 120,
    },
    {
      tote_id: 'tote_dairy_spoiled',
      stop_sequence: 4,
      is_cold_chain: true,
      expected_cash_minor: 85000000, // 850,000 UZS
      thermal_excursion_minutes: 55, // EXCEEDS 45 mins!
      max_allowed_excursion_mins: 45,
    },
    {
      tote_id: 'tote_dairy_safe',
      stop_sequence: 5,
      is_cold_chain: true,
      expected_cash_minor: 95000000, // 950,000 UZS
      thermal_excursion_minutes: 20, // SAFE (< 45 mins)
      max_allowed_excursion_mins: 45,
    },
  ]);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<MidRouteRescueResult | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setResult(null);

    if (!rescueVehicleId.trim() || !rescueDriverId.trim()) {
      setError('Rescue vehicle and driver are required');
      return;
    }
    if (!newRescueBoltSeal.trim()) {
      setError('New cryptographic rescue bolt seal is mandatory');
      return;
    }

    setSubmitting(true);
    try {
      const res = await foundryWarehouseApi.midRouteRescue({
        cell_id: cellId,
        supplier_id: supplierId,
        original_route_id: originalRouteId,
        broken_vehicle_id: brokenVehicleId,
        original_driver_id: originalDriverId,
        rescue_vehicle_id: rescueVehicleId.trim(),
        rescue_driver_id: rescueDriverId.trim(),
        last_completed_stop_seq: lastCompletedStopSeq,
        new_rescue_bolt_seal: newRescueBoltSeal.trim(),
        totes,
      });
      setResult(res);
      onSuccess?.(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Mid-route rescue execution failed');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm overflow-y-auto select-none">
      <div className="bg-neutral-900 border border-neutral-800 rounded-2xl w-full max-w-3xl shadow-2xl p-6 text-neutral-100 my-8">
        <div className="flex items-center justify-between pb-3.5 border-b border-neutral-800">
          <div>
            <div className="flex items-center gap-2">
              <ArrowRightLeft size={18} className="text-amber-400" />
              <h3 className="text-base font-bold text-neutral-100 uppercase tracking-wide">
                Mid-Route Breakdown Rescue & Thermal Quarantine Console
              </h3>
            </div>
            <p className="text-xs text-neutral-400 mt-0.5">
              Custody transfer, Driver 1 completed stop liability freeze, and cold-chain excursion evaluation.
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

        <form onSubmit={handleSubmit} className="mt-4 space-y-4">
          {/* Breakdown context */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 p-3.5 bg-neutral-950/80 rounded-xl border border-neutral-800 text-xs">
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase font-semibold">Disabled Vehicle</span>
              <span className="font-mono font-bold text-rose-400">{brokenVehicleId}</span>
            </div>
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase font-semibold">Original Driver</span>
              <span className="font-mono font-bold text-neutral-200">{originalDriverId}</span>
            </div>
            <div>
              <span className="text-neutral-500 block font-mono text-[10px] uppercase font-semibold">Last Completed Stop</span>
              <span className="font-mono font-bold text-amber-300">Stop #{lastCompletedStopSeq}</span>
            </div>
          </div>

          {/* Rescue assignment */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                Rescue Vehicle ID (DVIR Cleared)
              </label>
              <input
                type="text"
                required
                value={rescueVehicleId}
                onChange={(e) => setRescueVehicleId(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                Rescue Driver ID
              </label>
              <input
                type="text"
                required
                value={rescueDriverId}
                onChange={(e) => setRescueDriverId(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                New Rescue Bolt Seal Barcode
              </label>
              <input
                type="text"
                required
                value={newRescueBoltSeal}
                onChange={(e) => setNewRescueBoltSeal(e.target.value)}
                className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-amber-500"
              />
            </div>
          </div>

          {/* Totes inventory table */}
          <div>
            <span className="block text-[11px] font-semibold uppercase tracking-wider text-neutral-400 mb-2">
              Route Totes & Cold-Chain Excursion Assessment
            </span>
            <div className="rounded-xl border border-neutral-800 overflow-hidden divide-y divide-neutral-800 text-xs">
              <div className="bg-neutral-950 p-2.5 font-semibold text-neutral-400 grid grid-cols-5 gap-2 text-[11px] font-mono">
                <span>Tote ID</span>
                <span>Stop #</span>
                <span>Category</span>
                <span>Thermal Excursion</span>
                <span>Expected Cash</span>
              </div>
              {totes.map((t) => {
                const isOverLimit = t.is_cold_chain && t.thermal_excursion_minutes > t.max_allowed_excursion_mins;
                const isCompleted = t.stop_sequence <= lastCompletedStopSeq;
                return (
                  <div
                    key={t.tote_id}
                    className={`p-2.5 grid grid-cols-5 gap-2 items-center ${
                      isOverLimit
                        ? 'bg-rose-950/40 text-rose-200'
                        : isCompleted
                          ? 'bg-neutral-950/40 text-neutral-500'
                          : 'bg-neutral-900/60 text-neutral-200'
                    }`}
                  >
                    <span className="font-mono font-semibold">{t.tote_id}</span>
                    <span className="font-mono">
                      Stop {t.stop_sequence} {isCompleted && '(Delivered)'}
                    </span>
                    <span className="flex items-center gap-1">
                      {t.is_cold_chain ? (
                        <>
                          <Snowflake size={12} className="text-cyan-400" />
                          <span>Cold Chain</span>
                        </>
                      ) : (
                        'Dry Goods'
                      )}
                    </span>
                    <span className="font-mono text-[11px]">
                      {t.is_cold_chain ? (
                        <span className={isOverLimit ? 'font-bold text-rose-400' : 'text-emerald-400'}>
                          {t.thermal_excursion_minutes}m / {t.max_allowed_excursion_mins}m limit
                        </span>
                      ) : (
                        'N/A'
                      )}
                    </span>
                    <span className="font-mono font-semibold">
                      {minorToWholeTrunc(t.expected_cash_minor).toLocaleString()} UZS
                    </span>
                  </div>
                );
              })}
            </div>
          </div>

          {error && (
            <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200 flex items-center gap-2">
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
              disabled={submitting}
              className="px-5 py-2.5 rounded-xl text-xs font-bold bg-amber-600 hover:bg-amber-500 text-white disabled:opacity-40 transition-all shadow"
            >
              {submitting ? 'Executing Rescue Protocol...' : 'Authorize Rescue Transfer & Split Liability'}
            </button>
          </div>
        </form>

        {/* Results Receipt */}
        {result && (
          <div className="mt-4 p-4 rounded-xl bg-neutral-950 border border-emerald-500/50 space-y-3 text-xs">
            <div className="flex items-center justify-between pb-2 border-b border-neutral-800">
              <span className="font-bold text-emerald-400 flex items-center gap-1.5">
                <CheckCircle2 size={16} />
                <span>Rescue Protocol Complete: Route {result.rescue_route_id}</span>
              </span>
              <span className="font-mono text-neutral-400 text-[11px]">Bolt Seal: {result.new_rescue_bolt_seal}</span>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="p-3 rounded-xl bg-neutral-900 border border-neutral-800">
                <span className="text-neutral-500 block font-mono text-[10px] uppercase font-semibold mb-1">
                  Driver 1 (Broken) Locked Cash Liability:
                </span>
                <div className="font-mono text-base font-bold text-amber-300">
                  {minorToWholeTrunc(result.original_driver_liability_minor).toLocaleString()} UZS
                </div>
                <span className="text-[10px] text-neutral-500 font-mono">
                  Strictly bounded to completed stops (≤ #{lastCompletedStopSeq})
                </span>
              </div>

              <div className="p-3 rounded-xl bg-neutral-900 border border-neutral-800">
                <span className="text-neutral-500 block font-mono text-[10px] uppercase font-semibold mb-1">
                  Rescue Driver 2 Transferred Cash Liability:
                </span>
                <div className="font-mono text-base font-bold text-emerald-400">
                  {minorToWholeTrunc(result.rescue_driver_liability_minor).toLocaleString()} UZS
                </div>
                <span className="text-[10px] text-neutral-500 font-mono">
                  For remaining un-quarantined delivery stops
                </span>
              </div>
            </div>

            {/* Quarantined Spoiled Totes Banner */}
            {result.quarantined_spoiled_tote_ids.length > 0 && (
              <div className="p-3.5 rounded-xl bg-rose-950/60 border border-rose-600 text-rose-200 space-y-1">
                <div className="font-bold flex items-center gap-1.5 text-xs">
                  <ShieldAlert size={16} className="text-rose-400 shrink-0" />
                  <span>Spoiled Totes Blocked from Custody Transfer (Thermal Excursion)</span>
                </div>
                <div className="font-mono text-xs">
                  Quarantined IDs: {result.quarantined_spoiled_tote_ids.join(', ')}
                </div>
                <p className="text-[11px] text-rose-300/80">
                  Stamped with <code className="bg-rose-900/60 px-1 py-0.5 rounded font-mono text-rose-100">MARKING:THERMAL_QUARANTINE_SPOILED</code>. Route return to warehouse for claims disposal.
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
