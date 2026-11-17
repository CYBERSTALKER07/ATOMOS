'use client';

import React, { useState } from 'react';
import { STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { majorIntToMinor, minorToWholeTrunc, ratioPercentInt } from '@/lib/money';
import { formatWholeSomInteger } from '@/lib/cashier-helpers';
import { Building2, ShieldCheck, AlertCircle, CheckCircle2 } from 'lucide-react';

interface AMLExposureCheckProps {
  initialTin?: string;
  supplierId?: string;
}

export function AMLExposureCheck({
  initialTin = '',
  supplierId = '',
}: AMLExposureCheckProps) {
  const [tin, setTin] = useState(initialTin);
  const [calendarDate, setCalendarDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [cashSom, setCashSom] = useState<number>(0);
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState<AMLCheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const ceilingSom = minorToWholeTrunc(STATUTORY_AML_CASH_CEILING_MINOR);

  const handleCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!tin.trim()) {
      setError('Legal Entity TIN is required');
      return;
    }
    setChecking(true);
    try {
      const cashMinor = majorIntToMinor(cashSom);
      const res = await foundryWarehouseApi.amlCheck({
        legal_entity_tin: tin.trim(),
        supplier_id: supplierId,
        calendar_date: calendarDate,
        cash_minor: cashMinor,
      });
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AML check failed');
    } finally {
      setChecking(false);
    }
  };

  const accumulatedSom = result ? minorToWholeTrunc(result.accumulated_cash_minor) : 0;
  const percentage = Math.min(100, ratioPercentInt(accumulatedSom, ceilingSom));

  return (
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-5 backdrop-blur-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-neutral-800">
        <div>
          <div className="flex items-center gap-2">
            <Building2 size={20} className="text-cyan-400" />
            <h3 className="text-base font-bold text-neutral-100 uppercase tracking-wide">
              Multi-Tender Settlement Exposure Check (Uncapped Policy)
            </h3>
          </div>
          <p className="text-xs text-neutral-400 mt-0.5">
            Audit daily cash settlement exposure across all storefronts sharing a parent Legal Entity TIN under uncapped multi-tender rules.
          </p>
        </div>

        <span className="px-3 py-1 rounded-full text-xs font-mono font-semibold bg-cyan-500/10 text-cyan-300 border border-cyan-500/30">
          Central Bank Compliance Gate
        </span>
      </div>

      <form onSubmit={handleCheck} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Legal Entity TIN (INN)
            </label>
            <input
              type="text"
              required
              value={tin}
              onChange={(e) => setTin(e.target.value)}
              placeholder="Enter 9-digit TIN"
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Audit Calendar Date
            </label>
            <input
              type="date"
              required
              value={calendarDate}
              onChange={(e) => setCalendarDate(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Incremental Cash Tender (so&apos;m)
            </label>
            <input
              type="text"
              inputMode="numeric"
              pattern="[0-9]*"
              value={cashSom === 0 ? '' : cashSom}
              onChange={(e) => {
                const s = e.target.value.replace(/\D/g, '');
                setCashSom(Math.max(0, parseInt(s || '0', 10) || 0));
              }}
              placeholder="0 (check balance only)"
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-cyan-500"
            />
          </div>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200 flex items-center gap-2">
            <AlertCircle size={15} className="shrink-0 text-rose-400" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={checking}
            className="px-5 py-2 rounded-xl text-xs font-bold bg-cyan-600 hover:bg-cyan-500 text-white disabled:opacity-40 transition-all shadow"
          >
            {checking ? 'Evaluating Exposure...' : 'Check AML Exposure'}
          </button>
        </div>
      </form>

      {/* Result Display */}
      {result && (
        <div className="p-4 rounded-xl bg-neutral-950 border border-neutral-800 space-y-3">
          <div className="flex flex-wrap items-center justify-between text-xs gap-2">
            <span className="font-semibold text-neutral-300">
              Taxpayer INN: <span className="font-mono text-neutral-100 font-bold">{result.legal_entity_tin}</span>
            </span>
            <span className="font-mono text-neutral-400 text-[11px]">
              Statutory Benchmark: {formatWholeSomInteger(ceilingSom)}
            </span>
          </div>

          {/* Progress Bar */}
          <div className="w-full bg-neutral-900 rounded-full h-2.5 border border-neutral-800 overflow-hidden">
            <div
              className="h-full transition-all duration-500 rounded-full bg-emerald-500"
              style={{ width: `${Math.min(100, percentage)}%` }}
            />
          </div>

          <div className="flex items-center justify-between text-xs font-mono">
            <span className="text-neutral-300 font-bold">
              Accumulated Today: {formatWholeSomInteger(accumulatedSom)}
            </span>
            <span className="text-emerald-400 font-bold">Uncapped Multi-Tender Active</span>
          </div>

          <div className="p-3 rounded-lg bg-emerald-950/40 border border-emerald-600/60 text-emerald-200 text-xs font-medium flex items-center gap-2">
            <CheckCircle2 size={16} className="text-emerald-400 shrink-0" />
            <span>
              Settlement Notice: Uncapped multi-tender active. Cash exposure recorded for depot vault audit without restriction.
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
