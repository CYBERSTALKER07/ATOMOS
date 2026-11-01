'use client';

import React, { useState } from 'react';
import { TIYIN_PER_UZS, STATUTORY_AML_CASH_CEILING_MINOR, type AMLCheckResponse, type CashierSessionView } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';

export default function CashierVaultMonitor({ token }: { token: string }) {
  // AML Exposure Check State
  const [tin, setTin] = useState('301234567');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [calendarDate, setCalendarDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [cashAmountUZS, setCashAmountUZS] = useState(5000000);
  const [amlChecking, setAmlChecking] = useState(false);
  const [amlResult, setAmlResult] = useState<AMLCheckResponse | null>(null);

  // Cashier Session Test State
  const [bagSeal, setBagSeal] = useState('SEAL-ADMIN-9901');
  const [driverId, setDriverId] = useState('drv_anvar_01');
  const [expectedUZS, setExpectedUZS] = useState(2000000);
  const [startingSession, setStartingSession] = useState(false);
  const [session, setSession] = useState<CashierSessionView | null>(null);

  const [countUZS, setCountUZS] = useState(1900000); // 100,000 UZS variance
  const [submittingCount, setSubmittingCount] = useState(false);

  const [error, setError] = useState<string | null>(null);

  // AML check
  const handleCheckAML = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setAmlChecking(true);
    try {
      const res = await foundryApi.amlCheck(token, {
        legal_entity_tin: tin.trim(),
        supplier_id: supplierId.trim(),
        calendar_date: calendarDate,
        cash_minor: cashAmountUZS * TIYIN_PER_UZS,
      });
      setAmlResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AML check failed');
    } finally {
      setAmlChecking(false);
    }
  };

  // Start Session
  const handleStartSession = async () => {
    setError(null);
    setStartingSession(true);
    try {
      const s = await foundryApi.startCashierSession(token, {
        cell_id: 'cell-uz',
        supplier_id: supplierId,
        depot_id: 'depot_central',
        driver_id: driverId,
        route_id: 'route_101',
        bag_seal_barcode: bagSeal,
        expected_cash_minor: expectedUZS * TIYIN_PER_UZS,
      });
      setSession(s);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start session');
    } finally {
      setStartingSession(false);
    }
  };

  // Submit Count
  const handleSubmitCount = async () => {
    if (!session) return;
    setError(null);
    setSubmittingCount(true);
    try {
      const noteCount = Math.floor(countUZS / 100000);
      const updated = await foundryApi.submitBlindCount(token, session.session_id, {
        denominations_uzs: { 100000: noteCount },
        tolerance_threshold_minor: 0,
      });
      setSession(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Count submission failed');
    } finally {
      setSubmittingCount(false);
    }
  };

  const ceilingUZS = Math.floor(STATUTORY_AML_CASH_CEILING_MINOR / TIYIN_PER_UZS);

  return (
    <div className="space-y-6">
      {/* AML Exposure Checker Section */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Multi-Tender Settlement Exposure Monitor (Uncapped Policy)
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Audit cumulative daily B2B cash settlement across legal entity storefronts.
          </p>
        </div>

        <form onSubmit={handleCheckAML} className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-700">Legal Entity TIN</label>
            <input
              type="text"
              required
              value={tin}
              onChange={(e) => setTin(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Calendar Date</label>
            <input
              type="date"
              required
              value={calendarDate}
              onChange={(e) => setCalendarDate(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">New Cash Amount (UZS)</label>
            <input
              type="number"
              min="0"
              value={cashAmountUZS}
              onChange={(e) => setCashAmountUZS(parseInt(e.target.value || '0', 10))}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div className="flex items-end">
            <button
              type="submit"
              disabled={amlChecking}
              className="w-full rounded bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {amlChecking ? 'Auditing...' : 'Audit AML Ceiling'}
            </button>
          </div>
        </form>

        {amlResult && (
          <div className="rounded-lg border bg-gray-50 p-4 space-y-2 text-xs">
            <div className="flex justify-between font-semibold">
              <span>TIN: {amlResult.legal_entity_tin}</span>
              <span className="font-mono">
                Accumulated: {Math.floor(amlResult.accumulated_cash_minor / TIYIN_PER_UZS).toLocaleString()} UZS (Uncapped)
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2 overflow-hidden">
              <div
                className="h-full bg-emerald-500"
                style={{
                  width: '100%',
                }}
              />
            </div>
            <div className="text-emerald-700 font-semibold mt-1">
              Enterprise Settlement Active: Uncapped multi-tender active. Daily cash volume tracked for fiscal reporting.
            </div>
          </div>
        )}
      </div>

      {/* Depot Cashier Session & Ratchet Oversight */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Depot Cashier Turn-In & Dual-Blind Recount Ratchet Monitor
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Monitor blind count submissions, expected cash masking, and recount ratchet state transitions.
          </p>
        </div>

        {!session ? (
          <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-700">Bag Seal Barcode</label>
              <input
                type="text"
                value={bagSeal}
                onChange={(e) => setBagSeal(e.target.value)}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Driver ID</label>
              <input
                type="text"
                value={driverId}
                onChange={(e) => setDriverId(e.target.value)}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Expected Cash (UZS)</label>
              <input
                type="number"
                value={expectedUZS}
                onChange={(e) => setExpectedUZS(parseInt(e.target.value || '0', 10))}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div className="flex items-end">
              <button
                type="button"
                onClick={handleStartSession}
                disabled={startingSession}
                className="w-full rounded bg-emerald-600 px-4 py-2 text-xs font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
              >
                {startingSession ? 'Initializing...' : 'Initialize Test Session'}
              </button>
            </div>
          </div>
        ) : (
          <div className="rounded-lg border bg-gray-50 p-4 space-y-3 text-xs">
            <div className="flex justify-between items-center border-b pb-2">
              <span className="font-bold">Session: {session.session_id}</span>
              <span className="font-mono bg-indigo-100 text-indigo-800 px-2 py-0.5 rounded font-bold">
                {session.status} (Attempt #{session.attempt_number})
              </span>
            </div>

            <div className="grid grid-cols-3 gap-3 font-mono">
              <div>
                <span className="text-gray-500 block">Counted:</span>
                <span className="font-bold">{Math.floor((session.counted_cash_minor || 0) / TIYIN_PER_UZS).toLocaleString()} UZS</span>
              </div>
              <div>
                <span className="text-gray-500 block">Expected:</span>
                <span>
                  {session.expected_cash_minor !== undefined && session.expected_cash_minor !== null
                    ? `${Math.floor(session.expected_cash_minor / TIYIN_PER_UZS).toLocaleString()} UZS`
                    : '🔒 MASKED (BLIND)'}
                </span>
              </div>
              <div>
                <span className="text-gray-500 block">Variance:</span>
                <span>
                  {session.variance_minor !== undefined && session.variance_minor !== null
                    ? `${Math.floor(session.variance_minor / TIYIN_PER_UZS).toLocaleString()} UZS`
                    : '🔒 MASKED'}
                </span>
              </div>
            </div>

            {(session.status === 'AWAITING_BLIND_COUNT_1' || session.status === 'BLIND_RECOUNT_REQUIRED') && (
              <div className="pt-2 flex items-center gap-3">
                <input
                  type="number"
                  value={countUZS}
                  onChange={(e) => setCountUZS(parseInt(e.target.value || '0', 10))}
                  className="rounded border px-3 py-1 text-xs font-mono w-44"
                />
                <button
                  type="button"
                  onClick={handleSubmitCount}
                  disabled={submittingCount}
                  className="rounded bg-indigo-600 px-3 py-1 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
                >
                  {submittingCount ? 'Submitting...' : `Submit Count Attempt #${session.attempt_number}`}
                </button>
              </div>
            )}
          </div>
        )}

        {error && (
          <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </div>
        )}
      </div>
    </div>
  );
}
