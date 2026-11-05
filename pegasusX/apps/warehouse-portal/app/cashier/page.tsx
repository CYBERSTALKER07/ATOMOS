'use client';

import React, { useState } from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { BlindCountDenominationGrid } from '@/components/cashier/BlindCountDenominationGrid';
import { CashierSessionView } from '@/components/cashier/CashierSessionView';
import { RRNReclassificationModal } from '@/components/cashier/RRNReclassificationModal';
import { CITHandoverManifest } from '@/components/cashier/CITHandoverManifest';
import { AMLExposureCheck } from '@/components/cashier/AMLExposureCheck';

export default function CashierPage() {
  const [activeTab, setActiveTab] = useState<'count' | 'cit' | 'aml'>('count');

  // Bag Scan / Session Start State
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [depotId, setDepotId] = useState('depot_tashkent_central');
  const [driverId, setDriverId] = useState('drv_anvar_01');
  const [routeId, setRouteId] = useState('route_yashnobod_101');
  const [bagSealBarcode, setBagSealBarcode] = useState('');

  const [session, setSession] = useState<CashierSessionViewType | null>(null);
  const [startingSession, setStartingSession] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);

  // RRN Modal State
  const [isRRNModalOpen, setIsRRNModalOpen] = useState(false);

  // Start new bag blind count session
  const handleStartSession = async (e: React.FormEvent) => {
    e.preventDefault();
    setStartError(null);
    if (!bagSealBarcode.trim()) {
      setStartError('Bag seal barcode is required');
      return;
    }

    setStartingSession(true);
    try {
      // Expected cash is resolved server-side from Spanner route orders and masked during blind count.
      const newSession = await foundryWarehouseApi.startCashierSession({
        cell_id: cellId,
        supplier_id: supplierId,
        depot_id: depotId,
        driver_id: driverId.trim(),
        route_id: routeId.trim(),
        bag_seal_barcode: bagSealBarcode.trim(),
        expected_cash_minor: 0,
      });
      setSession(newSession);
    } catch (err) {
      setStartError(err instanceof Error ? err.message : 'Failed to start cashier session');
    } finally {
      setStartingSession(false);
    }
  };

  // Submit Blind Count
  const handleSubmitBlindCount = async (
    denominations: Record<number, number>,
    _totalMinor: number,
    toleranceMinor: number,
  ) => {
    if (!session) return;
    const updated = await foundryWarehouseApi.submitBlindCount(session.session_id, {
      denominations_uzs: denominations,
      tolerance_threshold_minor: toleranceMinor,
    });
    setSession(updated);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <span>💰</span>
            <span>Role 7: Depot Cashier & Vault Operations</span>
          </h1>
          <p className="text-sm text-slate-400 mt-1">
            Physical-to-fiscal reconciliation, dual-blind recount ratchet, RRN tender corrections, CIT armored handovers, and Central Bank AML limits.
          </p>
        </div>

        {/* Tab navigation buttons */}
        <div className="flex items-center gap-1.5 p-1 bg-slate-900 rounded-xl border border-slate-800">
          <button
            type="button"
            onClick={() => setActiveTab('count')}
            className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
              activeTab === 'count'
                ? 'bg-emerald-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Driver Bag Turn-In
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('cit')}
            className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
              activeTab === 'cit'
                ? 'bg-amber-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            CIT Armored Handover
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('aml')}
            className={`px-4 py-2 rounded-lg text-xs font-semibold transition-all ${
              activeTab === 'aml'
                ? 'bg-blue-600 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            AML Exposure Check
          </button>
        </div>
      </div>

      {/* Main Tab Content */}
      {activeTab === 'count' && (
        <div className="space-y-6">
          {!session ? (
            /* Bag Seal Scanning & Session Start Form */
            <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-4">
              <div>
                <h3 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                  <span>🏷️</span>
                  <span>Step 1: Scan Driver Deposit Bag Seal</span>
                </h3>
                <p className="text-xs text-slate-400 mt-0.5">
                  Scan the serialized tamper-evident barcode from the driver deposit pouch. Duplicate barcodes are rejected.
                </p>
              </div>

              <form onSubmit={handleStartSession} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      Bag Seal Barcode
                    </label>
                    <input
                      type="text"
                      required
                      value={bagSealBarcode}
                      onChange={(e) => setBagSealBarcode(e.target.value)}
                      placeholder="e.g. SEAL-UZ-99214"
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      Driver ID
                    </label>
                    <input
                      type="text"
                      required
                      value={driverId}
                      onChange={(e) => setDriverId(e.target.value)}
                      placeholder="e.g. drv_anvar_01"
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      Route ID
                    </label>
                    <input
                      type="text"
                      required
                      value={routeId}
                      onChange={(e) => setRouteId(e.target.value)}
                      placeholder="e.g. route_101"
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      System Expected Cash (UZS)
                    </label>
                    <div className="w-full bg-slate-950/70 border border-slate-800 rounded-xl px-3 py-2 text-xs font-mono text-emerald-400">
                      🔒 Server-Derived (Masked for Blind Count)
                    </div>
                    <span className="text-[11px] text-slate-400 mt-0.5 block">
                      Computed authoritatively from Spanner route orders; never exposed before count completion.
                    </span>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      Depot
                    </label>
                    <input
                      type="text"
                      required
                      value={depotId}
                      onChange={(e) => setDepotId(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
                      Supplier ID
                    </label>
                    <input
                      type="text"
                      required
                      value={supplierId}
                      onChange={(e) => setSupplierId(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>
                </div>

                {startError && (
                  <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
                    {startError}
                  </div>
                )}

                <div className="flex justify-end pt-2">
                  <button
                    type="submit"
                    disabled={startingSession}
                    className="px-6 py-2.5 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-50 transition-all shadow-md"
                  >
                    {startingSession ? 'Initializing Blind Session...' : 'Bind Seal & Begin Blind Count'}
                  </button>
                </div>
              </form>
            </div>
          ) : (
            /* Active Blind Session Display */
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <span className="text-xs text-slate-400">Active Bag Turn-In Session</span>
                <button
                  type="button"
                  onClick={() => setSession(null)}
                  className="text-xs text-slate-400 hover:text-slate-200 underline"
                >
                  Start another driver bag
                </button>
              </div>

              <CashierSessionView
                session={session}
                onOpenReclassifyModal={() => setIsRRNModalOpen(true)}
                onRequestSupervisorOverride={() => {
                  alert(
                    `Supervisor override requested for Session ${session.session_id}. Driver ${session.driver_id} remains locked under MARKING:DISCREPANT_QUARANTINE.`,
                  );
                }}
              />

              {/* Only show Denomination Grid if session is pending count */}
              {(session.status === 'AWAITING_BLIND_COUNT_1' || session.status === 'BLIND_RECOUNT_REQUIRED') && (
                <BlindCountDenominationGrid
                  attemptNumber={session.attempt_number}
                  onSubmitCount={handleSubmitBlindCount}
                />
              )}
            </div>
          )}

          {/* RRN Tender Reclassification Modal */}
          {session && (
            <RRNReclassificationModal
              sessionId={session.session_id}
              isOpen={isRRNModalOpen}
              onClose={() => setIsRRNModalOpen(false)}
              onSuccess={(updated) => setSession(updated)}
            />
          )}
        </div>
      )}

      {activeTab === 'cit' && (
        <CITHandoverManifest depotId={depotId} supplierId={supplierId} />
      )}

      {activeTab === 'aml' && (
        <AMLExposureCheck supplierId={supplierId} />
      )}
    </div>
  );
}
