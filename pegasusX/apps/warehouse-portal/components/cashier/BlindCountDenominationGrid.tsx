'use client';

import React, { useState, useMemo } from 'react';
import { UZS_BANKNOTE_DENOMINATIONS, TIYIN_PER_UZS, type UZSBanknoteDenomination } from '@pegasusx/types';

interface BlindCountDenominationGridProps {
  disabled?: boolean;
  attemptNumber: number;
  onSubmitCount: (denominations: Record<number, number>, totalMinor: number, toleranceMinor: number) => Promise<void>;
}

export function BlindCountDenominationGrid({
  disabled = false,
  attemptNumber,
  onSubmitCount,
}: BlindCountDenominationGridProps) {
  const [counts, setCounts] = useState<Record<number, number>>(() => {
    const initial: Record<number, number> = {};
    for (const note of UZS_BANKNOTE_DENOMINATIONS) {
      initial[note] = 0;
    }
    return initial;
  });
  const [toleranceUZS, setToleranceUZS] = useState<number>(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const totalCalculated = useMemo(() => {
    let totalUZS = 0;
    for (const [denomStr, count] of Object.entries(counts)) {
      const denom = Number(denomStr);
      totalUZS += denom * (count || 0);
    }
    const totalMinor = totalUZS * TIYIN_PER_UZS;
    return { totalUZS, totalMinor };
  }, [counts]);

  const handleCountChange = (denom: UZSBanknoteDenomination, valStr: string) => {
    const val = Math.max(0, parseInt(valStr || '0', 10) || 0);
    setCounts((prev) => ({
      ...prev,
      [denom]: val,
    }));
  };

  const handleIncrement = (denom: UZSBanknoteDenomination, delta: number) => {
    setCounts((prev) => ({
      ...prev,
      [denom]: Math.max(0, (prev[denom] || 0) + delta),
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (totalCalculated.totalMinor <= 0) {
      setError('Total counted cash must be greater than 0');
      return;
    }
    setSubmitting(true);
    try {
      const toleranceMinor = Math.max(0, toleranceUZS) * TIYIN_PER_UZS;
      await onSubmitCount(counts, totalCalculated.totalMinor, toleranceMinor);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Blind count submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleReset = () => {
    const resetMap: Record<number, number> = {};
    for (const note of UZS_BANKNOTE_DENOMINATIONS) {
      resetMap[note] = 0;
    }
    setCounts(resetMap);
    setError(null);
  };

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-5 rounded-2xl border border-slate-700/60 shadow-lg">
      <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-700/60">
        <div>
          <h3 className="text-lg font-semibold text-slate-100 flex items-center gap-2">
            <span>Physical Banknote Denomination Count</span>
            <span className="text-xs px-2 py-0.5 rounded-full font-medium bg-amber-500/20 text-amber-300 border border-amber-500/30">
              Attempt #{attemptNumber}
            </span>
          </h3>
          <p className="text-xs text-slate-400 mt-0.5">
            Strict blind count ritual: Expected amount is hidden. Count physical bills directly.
          </p>
        </div>
        <button
          type="button"
          onClick={handleReset}
          disabled={disabled || submitting}
          className="text-xs text-slate-400 hover:text-slate-200 transition-colors"
        >
          Reset grid
        </button>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {UZS_BANKNOTE_DENOMINATIONS.map((denom) => {
            const count = counts[denom] || 0;
            const subtotalUZS = denom * count;
            return (
              <div
                key={denom}
                className="p-3 bg-slate-900/60 rounded-xl border border-slate-800 hover:border-slate-700 transition-colors"
              >
                <div className="flex items-center justify-between mb-1.5">
                  <span className="font-mono text-sm font-bold text-emerald-400">
                    {denom.toLocaleString()} UZS
                  </span>
                  <span className="text-xs font-mono text-slate-400">
                    {(denom * TIYIN_PER_UZS).toLocaleString()} Tiyin
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, -1)}
                    disabled={disabled || submitting || count <= 0}
                    className="w-7 h-7 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 disabled:opacity-40 font-bold flex items-center justify-center transition-colors"
                  >
                    -
                  </button>
                  <input
                    type="number"
                    min="0"
                    value={count === 0 ? '' : count}
                    placeholder="0"
                    disabled={disabled || submitting}
                    onChange={(e) => handleCountChange(denom, e.target.value)}
                    className="w-full text-center font-mono text-base font-semibold bg-slate-950 border border-slate-700 rounded-lg py-1 px-2 text-slate-100 focus:outline-none focus:border-emerald-500"
                  />
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, 1)}
                    disabled={disabled || submitting}
                    className="w-7 h-7 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 disabled:opacity-40 font-bold flex items-center justify-center transition-colors"
                  >
                    +
                  </button>
                </div>
                <div className="text-right text-[11px] font-mono text-slate-400 mt-1.5">
                  = {subtotalUZS.toLocaleString()} UZS
                </div>
              </div>
            );
          })}
        </div>

        <div className="mt-4 pt-4 border-t border-slate-800 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <label className="text-xs text-slate-400">
              Tolerance threshold (UZS):
            </label>
            <input
              type="number"
              min="0"
              value={toleranceUZS}
              onChange={(e) => setToleranceUZS(Math.max(0, parseInt(e.target.value || '0', 10) || 0))}
              className="w-28 text-right font-mono text-xs bg-slate-950 border border-slate-700 rounded-md py-1 px-2 text-slate-200"
            />
          </div>

          <div className="flex items-center gap-4">
            <div className="text-right">
              <div className="text-xs text-slate-400 uppercase tracking-wider font-semibold">Total Counted</div>
              <div className="text-lg font-bold font-mono text-emerald-400">
                {totalCalculated.totalUZS.toLocaleString()} UZS
              </div>
              <div className="text-[11px] font-mono text-slate-400">
                ({totalCalculated.totalMinor.toLocaleString()} Tiyin)
              </div>
            </div>

            <button
              type="submit"
              disabled={disabled || submitting || totalCalculated.totalMinor <= 0}
              className="px-5 py-2.5 rounded-xl font-semibold text-sm bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-md flex items-center gap-2"
            >
              {submitting ? 'Submitting Count...' : `Confirm Attempt #${attemptNumber}`}
            </button>
          </div>
        </div>

        {error && (
          <div className="mt-3 p-3 rounded-lg bg-red-900/40 border border-red-700/60 text-red-200 text-xs">
            {error}
          </div>
        )}
      </form>
    </div>
  );
}
