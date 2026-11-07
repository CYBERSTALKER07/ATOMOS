'use client';

import React, { useState } from 'react';
import { TIYIN_PER_UZS, STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface AMLExposureCheckProps {
  initialTin?: string;
  supplierId?: string;
}

export function AMLExposureCheck({
  initialTin = '301234567',
  supplierId = 'sup_samarkand_dairy',
}: AMLExposureCheckProps) {
  const [tin, setTin] = useState(initialTin);
  const [calendarDate, setCalendarDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [cashUZS, setCashUZS] = useState<number>(0);
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState<AMLCheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const ceilingUZS = Math.floor(STATUTORY_AML_CASH_CEILING_MINOR / TIYIN_PER_UZS);

  const handleCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!tin.trim()) {
      setError('Legal Entity TIN is required');
      return;
    }
    setChecking(true);
    try {
      const cashMinor = Math.floor(cashUZS * TIYIN_PER_UZS);
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

  const accumulatedUZS = result ? Math.floor(result.accumulated_cash_minor / TIYIN_PER_UZS) : 0;
  const percentage = Math.min(100, Math.round((accumulatedUZS / ceilingUZS) * 100));
  const isCeilingBreached = result ? result.accumulated_cash_minor >= STATUTORY_AML_CASH_CEILING_MINOR : false;
  const isAmberWarning = percentage >= 80 && !isCeilingBreached;

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-5">
      <div>
        <h3 className="text-lg font-bold text-slate-100 flex items-center gap-2">
          <span>🏛️</span>
          <span>Multi-Tender Settlement Exposure Check (Uncapped Policy)</span>
        </h3>
        <p className="text-xs text-slate-400 mt-1">
          Audit daily cash settlement exposure across all storefronts sharing a parent Legal Entity TIN under uncapped multi-tender rules.
        </p>
      </div>

      <form onSubmit={handleCheck} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Legal Entity TIN (INN)
            </label>
            <input
              type="text"
              required
              value={tin}
              onChange={(e) => setTin(e.target.value)}
              placeholder="e.g. 301234567"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Calendar Date
            </label>
            <input
              type="date"
              required
              value={calendarDate}
              onChange={(e) => setCalendarDate(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              New Transaction Cash (UZS)
            </label>
            <input
              type="number"
              min="0"
              value={cashUZS === 0 ? '' : cashUZS}
              onChange={(e) => setCashUZS(Math.max(0, parseInt(e.target.value || '0', 10) || 0))}
              placeholder="0 (check balance only)"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>
        </div>

        {error && (
          <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={checking}
            className="px-5 py-2 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-all shadow"
          >
            {checking ? 'Evaluating Exposure...' : 'Check AML Limit'}
          </button>
        </div>
      </form>

      {/* Result Display */}
      {result && (
        <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 space-y-3">
          <div className="flex items-center justify-between text-xs">
            <span className="font-semibold text-slate-300">
              TIN: <span className="font-mono text-slate-100">{result.legal_entity_tin}</span>
            </span>
            <span className="font-mono text-slate-400">
              Limit: {ceilingUZS.toLocaleString()} UZS (2,500,000,000 Tiyin)
            </span>
          </div>

          {/* Progress Bar */}
          <div className="w-full bg-slate-950 rounded-full h-3 border border-slate-800 overflow-hidden">
            <div
              className="h-full transition-all duration-500 rounded-full bg-emerald-500"
              style={{ width: '100%' }}
            />
          </div>

          <div className="flex items-center justify-between text-xs font-mono">
            <span className="text-slate-300 font-bold">
              Accumulated Today: {accumulatedUZS.toLocaleString()} UZS
            </span>
            <span className="text-emerald-400 font-bold">Uncapped Multi-Tender Active</span>
          </div>

          <div className="p-3 rounded-lg bg-emerald-950/60 border border-emerald-600 text-emerald-200 text-xs font-semibold flex items-center gap-2">
            <span>✅</span>
            <span>
              Settlement Notice: Uncapped multi-tender active. Cash exposure recorded for depot vault audit without restriction.
            </span>
          </div>
        </div>
      )}
    </div>
  );
}
