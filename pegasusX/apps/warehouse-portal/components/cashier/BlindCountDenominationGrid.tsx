'use client';

import React, { useState, useMemo, useEffect } from 'react';
import { UZS_BANKNOTE_DENOMINATIONS, type UZSBanknoteDenomination } from '@pegasusx/types';
import { formatGroupedInteger, majorIntToMinor, majorIntToMinorBigInt } from '@/lib/money';
import { Lock, RotateCcw, CheckCircle2, AlertCircle, ShieldAlert } from 'lucide-react';

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
    let totalNotes = 0;
    for (const [denomStr, count] of Object.entries(counts)) {
      const denom = Number(denomStr);
      const safeCount = Math.max(0, count || 0);
      totalUZS += denom * safeCount;
      totalNotes += safeCount;
    }
    const totalMinor = majorIntToMinor(totalUZS);
    return { totalUZS, totalMinor, totalNotes };
  }, [counts]);

  const handleCountChange = (denom: UZSBanknoteDenomination, valStr: string) => {
    const sanitized = valStr.replace(/\D/g, '');
    const val = Math.max(0, parseInt(sanitized || '0', 10) || 0);
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

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setError(null);
    if (totalCalculated.totalMinor <= 0) {
      setError('Total counted cash must be greater than 0 tiyin');
      return;
    }
    setSubmitting(true);
    try {
      const toleranceMinor = majorIntToMinor(Math.max(0, Math.floor(toleranceUZS)));
      await onSubmitCount(counts, totalCalculated.totalMinor, toleranceMinor);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Blind count submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  // Keyboard shortcut ⌘↵ to submit
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        if (!disabled && !submitting && totalCalculated.totalMinor > 0) {
          e.preventDefault();
          handleSubmit();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [disabled, submitting, totalCalculated.totalMinor]);

  const handleReset = () => {
    const resetMap: Record<number, number> = {};
    for (const note of UZS_BANKNOTE_DENOMINATIONS) {
      resetMap[note] = 0;
    }
    setCounts(resetMap);
    setToleranceUZS(0);
    setError(null);
  };

  return (
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-5 shadow-xl backdrop-blur-sm">
      {/* Header section with blind count badge */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-neutral-800">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base font-bold text-neutral-100 tracking-wide">
              Physical Banknote Denomination Count
            </h3>
            <span
              className={`px-2.5 py-0.5 rounded-full text-xs font-mono font-bold border ${
                attemptNumber > 1
                  ? 'bg-amber-500/10 text-amber-300 border-amber-500/30 animate-pulse'
                  : 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30'
              }`}
            >
              Attempt #{attemptNumber} {attemptNumber > 1 ? '(Recount Ratchet)' : ''}
            </span>
          </div>
          <div className="flex items-center gap-1.5 mt-1 text-xs text-neutral-400">
            <Lock size={13} className="text-amber-400 shrink-0" />
            <span>Strict blind count ritual (Guard #11): Expected amount masked on server until verification.</span>
          </div>
        </div>

        <button
          type="button"
          onClick={handleReset}
          disabled={disabled || submitting}
          className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-neutral-400 hover:text-neutral-200 border border-neutral-800 hover:border-neutral-700 rounded-lg bg-neutral-950/60 transition-colors disabled:opacity-40"
        >
          <RotateCcw size={12} />
          <span>Reset Grid</span>
        </button>
      </div>

      <form onSubmit={handleSubmit} className="mt-4">
        {/* Denomination Cards Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {UZS_BANKNOTE_DENOMINATIONS.map((denom) => {
            const count = counts[denom] || 0;
            const subtotalUZS = denom * count;

            return (
              <div
                key={denom}
                className="rounded-xl border border-neutral-800 bg-neutral-950/80 p-3.5 transition-colors hover:border-neutral-700 focus-within:border-emerald-500/60"
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="font-mono text-sm font-bold text-emerald-400">
                    {denom.toLocaleString()} UZS
                  </span>
                  <span className="text-[11px] font-mono text-neutral-500">
                    {formatGroupedInteger(majorIntToMinorBigInt(denom))} Tiyin
                  </span>
                </div>

                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, -1)}
                    disabled={disabled || submitting || count <= 0}
                    className="w-8 h-8 rounded-lg bg-neutral-900 hover:bg-neutral-800 text-neutral-200 disabled:opacity-30 font-bold text-sm flex items-center justify-center transition-colors border border-neutral-800"
                    title="Minus 1 note"
                    aria-label={`Decrease ${denom} UZS count by 1`}
                  >
                    -
                  </button>
                  <input
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={count === 0 ? '' : count}
                    placeholder="0"
                    disabled={disabled || submitting}
                    onChange={(e) => handleCountChange(denom, e.target.value)}
                    aria-label={`${denom} UZS banknote count`}
                    className="w-full text-center font-mono text-base font-bold bg-neutral-900 border border-neutral-750 rounded-lg py-1 px-2 text-neutral-100 focus:outline-none focus:border-emerald-500 transition-colors"
                  />
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, 1)}
                    disabled={disabled || submitting}
                    className="w-8 h-8 rounded-lg bg-neutral-900 hover:bg-neutral-800 text-neutral-200 disabled:opacity-30 font-bold text-sm flex items-center justify-center transition-colors border border-neutral-800"
                    title="Plus 1 note"
                    aria-label={`Increase ${denom} UZS count by 1`}
                  >
                    +
                  </button>
                </div>

                {/* Cashier strap & bundle quick increments */}
                <div className="flex items-center justify-center gap-1 mt-2">
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, 10)}
                    disabled={disabled || submitting}
                    className="px-2 py-0.5 text-[10px] font-mono font-medium rounded bg-neutral-900 hover:bg-neutral-800 text-neutral-400 hover:text-neutral-200 border border-neutral-800 transition-colors"
                    title="Add 1 strap (10 notes)"
                    aria-label={`Add 10 notes of ${denom} UZS`}
                  >
                    +10
                  </button>
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, 50)}
                    disabled={disabled || submitting}
                    className="px-2 py-0.5 text-[10px] font-mono font-medium rounded bg-neutral-900 hover:bg-neutral-800 text-neutral-400 hover:text-neutral-200 border border-neutral-800 transition-colors"
                    title="Add half-bundle (50 notes)"
                    aria-label={`Add 50 notes of ${denom} UZS`}
                  >
                    +50
                  </button>
                  <button
                    type="button"
                    onClick={() => handleIncrement(denom, 100)}
                    disabled={disabled || submitting}
                    className="px-2 py-0.5 text-[10px] font-mono font-medium rounded bg-neutral-900 hover:bg-neutral-800 text-neutral-400 hover:text-neutral-200 border border-neutral-800 transition-colors"
                    title="Add full strap bundle (100 notes)"
                    aria-label={`Add 100 notes of ${denom} UZS`}
                  >
                    +100
                  </button>
                </div>

                <div className="flex items-center justify-between text-[11px] font-mono text-neutral-400 mt-2 pt-2 border-t border-neutral-800/80">
                  <span>{count.toLocaleString()} notes</span>
                  <span className="text-neutral-300 font-medium">
                    = {subtotalUZS.toLocaleString()} UZS
                  </span>
                </div>
              </div>
            );
          })}
        </div>

        {/* Footer controls: Tolerance threshold + Total cash tally + Submit action */}
        <div className="mt-5 pt-4 border-t border-neutral-800 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex items-center gap-3 w-full md:w-auto">
            <label className="text-xs font-medium text-neutral-400 whitespace-nowrap">
              Acceptable Tolerance (UZS):
            </label>
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              value={toleranceUZS === 0 ? '' : toleranceUZS}
              placeholder="0"
              disabled={disabled || submitting}
              onChange={(e) => {
                const s = e.target.value.replace(/\D/g, '');
                setToleranceUZS(Math.max(0, parseInt(s || '0', 10) || 0));
              }}
              className="w-32 text-right font-mono text-xs bg-neutral-950 border border-neutral-750 rounded-lg py-1.5 px-2.5 text-neutral-200 focus:outline-none focus:border-emerald-500"
            />
            <span className="text-[11px] font-mono text-neutral-500">
              = {formatGroupedInteger(majorIntToMinorBigInt(Math.max(0, toleranceUZS)))} Tiyin
            </span>
          </div>

          <div className="flex items-center justify-between md:justify-end gap-5 w-full md:w-auto">
            <div className="text-right">
              <div className="text-[10px] uppercase font-mono tracking-wider text-neutral-500 font-semibold">
                Total Counted ({totalCalculated.totalNotes.toLocaleString()} notes)
              </div>
              <div className="text-xl font-bold font-mono text-emerald-400 leading-tight">
                {totalCalculated.totalUZS.toLocaleString()} UZS
              </div>
              <div className="text-xs font-mono text-neutral-400">
                {totalCalculated.totalMinor.toLocaleString()} Tiyin
              </div>
            </div>

            <button
              type="submit"
              disabled={disabled || submitting || totalCalculated.totalMinor <= 0}
              className="px-5 py-2.5 rounded-xl font-bold text-xs bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-40 disabled:cursor-not-allowed transition-all shadow-md flex items-center gap-2 whitespace-nowrap"
            >
              {submitting ? (
                <>
                  <span className="w-3.5 h-3.5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  <span>Submitting Count...</span>
                </>
              ) : (
                <>
                  <CheckCircle2 size={15} />
                  <span>Confirm Attempt #{attemptNumber}</span>
                  <kbd className="hidden sm:inline-flex text-[10px] bg-emerald-700/60 px-1 py-0.5 rounded text-emerald-100">
                    ⌘↵
                  </kbd>
                </>
              )}
            </button>
          </div>
        </div>

        {error && (
          <div className="mt-4 p-3 rounded-xl bg-rose-950/60 border border-rose-800/80 text-rose-200 text-xs flex items-center gap-2.5">
            <AlertCircle size={16} className="text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}
      </form>
    </div>
  );
}
