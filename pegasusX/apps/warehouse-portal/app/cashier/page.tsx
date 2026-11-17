'use client';

import React, { useEffect, useState } from 'react';
import type { CashierSessionView as CashierSessionViewType } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import {
  buildCashierSelectorOptions,
  resolveCashierScope,
  type CashierDriverOption,
  type CashierDriverRecord,
  type CashierManifestRecord,
  type CashierRouteOption,
} from '@/lib/cashier-helpers';
import { BlindCountDenominationGrid } from '@/components/cashier/BlindCountDenominationGrid';
import { CashierSessionView } from '@/components/cashier/CashierSessionView';
import { RRNReclassificationModal } from '@/components/cashier/RRNReclassificationModal';
import { CITHandoverManifest } from '@/components/cashier/CITHandoverManifest';
import { AMLExposureCheck } from '@/components/cashier/AMLExposureCheck';
import { AlertTriangle, Coins, Shield, Building2, Barcode, PlusCircle } from 'lucide-react';

interface SupervisorOverrideBanner {
  sessionId: string;
  driverId: string;
  requestedAt: string;
}

export default function CashierPage() {
  const [activeTab, setActiveTab] = useState<'count' | 'cit' | 'aml'>('count');

  // Scope & Session Start State (hydrated from warehouse auth/scope; zero fake presets)
  const [cellId, setCellId] = useState('');
  const [supplierId, setSupplierId] = useState('');
  const [depotId, setDepotId] = useState('');
  const [actorId, setActorId] = useState('');
  const [driverId, setDriverId] = useState('');
  const [routeId, setRouteId] = useState('');
  const [bagSealBarcode, setBagSealBarcode] = useState('');

  const [driverOptions, setDriverOptions] = useState<CashierDriverOption[]>([]);
  const [routeOptions, setRouteOptions] = useState<CashierRouteOption[]>([]);
  const [selectorsLoading, setSelectorsLoading] = useState(true);
  const [selectorsError, setSelectorsError] = useState<string | null>(null);
  const [manualEntryMode, setManualEntryMode] = useState(false);

  const [session, setSession] = useState<CashierSessionViewType | null>(null);
  const [startingSession, setStartingSession] = useState(false);
  const [startError, setStartError] = useState<string | null>(null);
  const [overrideBanner, setOverrideBanner] = useState<SupervisorOverrideBanner | null>(null);

  // RRN Modal State
  const [isRRNModalOpen, setIsRRNModalOpen] = useState(false);

  useEffect(() => {
    const scope = resolveCashierScope();
    setCellId(scope.cellId);
    setDepotId(scope.depotId);
    setSupplierId(scope.supplierId);
    setActorId(scope.actorId);

    let cancelled = false;
    const loadSelectors = async () => {
      setSelectorsLoading(true);
      setSelectorsError(null);
      const today = new Date().toISOString().slice(0, 10);
      const [driversRes, manifestsRes, liveMapRes] = await Promise.allSettled([
        apiFetch('/v1/warehouse/ops/drivers').then(async (res) => {
          if (!res.ok) throw new Error(`drivers_${res.status}`);
          return (await res.json()) as { drivers?: CashierDriverRecord[] };
        }),
        apiFetch(`/v1/warehouse/ops/manifests?date=${today}`).then(async (res) => {
          if (!res.ok) throw new Error(`manifests_${res.status}`);
          return (await res.json()) as { manifests?: CashierManifestRecord[] };
        }),
        warehouseApi.getWarehouseFleetLiveMap(scope.depotId ? { warehouse_id: scope.depotId } : {}),
      ]);

      if (cancelled) return;

      const drivers = driversRes.status === 'fulfilled' ? driversRes.value.drivers ?? [] : [];
      const manifests = manifestsRes.status === 'fulfilled' ? manifestsRes.value.manifests ?? [] : [];
      const fleetRoutes = liveMapRes.status === 'fulfilled' ? liveMapRes.value.routes ?? [] : [];

      const built = buildCashierSelectorOptions({ drivers, manifests, fleetRoutes });
      setDriverOptions(built.driverOptions);
      setRouteOptions(built.routeOptions);
      if (built.driverOptions.length === 0 && built.routeOptions.length === 0) {
        setManualEntryMode(true);
      }
      if (
        driversRes.status === 'rejected' &&
        manifestsRes.status === 'rejected' &&
        liveMapRes.status === 'rejected'
      ) {
        setSelectorsError('Live driver and route lists are unavailable; manual ID entry is enabled.');
      }
      setSelectorsLoading(false);
    };

    void loadSelectors();
    return () => {
      cancelled = true;
    };
  }, []);

  // Start new bag blind count session
  const handleStartSession = async (e: React.FormEvent) => {
    e.preventDefault();
    setStartError(null);
    if (!bagSealBarcode.trim()) {
      setStartError('Bag seal barcode is required');
      return;
    }
    if (!driverId.trim()) {
      setStartError('Driver selection or ID is required');
      return;
    }
    if (!routeId.trim()) {
      setStartError('Route selection or ID is required');
      return;
    }
    if (!depotId.trim() || !supplierId.trim()) {
      setStartError('Depot node and supplier scope are required');
      return;
    }

    setStartingSession(true);
    try {
      // Expected cash is resolved server-side from Spanner route orders and masked during blind count.
      const newSession = await foundryWarehouseApi.startCashierSession({
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        depot_id: depotId.trim(),
        driver_id: driverId.trim(),
        route_id: routeId.trim(),
        bag_seal_barcode: bagSealBarcode.trim(),
        expected_cash_minor: 0,
      });
      setOverrideBanner(null);
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
          <h1 className="text-2xl font-bold text-neutral-100 flex items-center gap-2">
            <Coins className="text-emerald-400" size={24} />
            <span>Role 7: Depot Cashier & Vault Operations</span>
          </h1>
          <p className="text-xs text-neutral-400 mt-1">
            Physical-to-fiscal reconciliation, dual-blind recount ratchet, RRN tender corrections, CIT armored handovers, and Central Bank AML limits.
          </p>
        </div>

        {/* Tab navigation buttons */}
        <div className="flex items-center gap-1.5 p-1 bg-neutral-900/90 rounded-xl border border-neutral-800">
          <button
            type="button"
            onClick={() => setActiveTab('count')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-bold transition-all ${
              activeTab === 'count'
                ? 'bg-emerald-600 text-white shadow'
                : 'text-neutral-400 hover:text-neutral-200'
            }`}
          >
            <Coins size={14} />
            <span>Driver Bag Turn-In</span>
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('cit')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-bold transition-all ${
              activeTab === 'cit'
                ? 'bg-amber-600 text-white shadow'
                : 'text-neutral-400 hover:text-neutral-200'
            }`}
          >
            <Shield size={14} />
            <span>CIT Armored Handover</span>
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('aml')}
            className={`flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-bold transition-all ${
              activeTab === 'aml'
                ? 'bg-cyan-600 text-white shadow'
                : 'text-neutral-400 hover:text-neutral-200'
            }`}
          >
            <Building2 size={14} />
            <span>AML Exposure Check</span>
          </button>
        </div>
      </div>

      {/* Main Tab Content */}
      {activeTab === 'count' && (
        <div className="space-y-6">
          {!session ? (
            /* Bag Seal Scanning & Session Start Form */
            <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-4 backdrop-blur-sm">
              <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-neutral-800">
                <div>
                  <h3 className="text-base font-bold text-neutral-100 flex items-center gap-2">
                    <Barcode size={18} className="text-emerald-400" />
                    <span>Step 1: Scan Driver Deposit Bag Seal</span>
                  </h3>
                  <p className="text-xs text-neutral-400 mt-0.5">
                    Scan the serialized tamper-evident barcode from the driver deposit pouch. Duplicate barcodes are rejected.
                  </p>
                </div>

                {/* Driver / Route Entry Mode Toggle */}
                <div className="flex items-center gap-2">
                  <span className="text-[11px] font-mono text-neutral-500">
                    {selectorsLoading
                      ? 'Loading live fleet...'
                      : `${driverOptions.length} drivers · ${routeOptions.length} routes`}
                  </span>
                  <button
                    type="button"
                    onClick={() => setManualEntryMode((prev) => !prev)}
                    className="px-2.5 py-1 rounded text-[11px] font-mono bg-neutral-800 hover:bg-neutral-700 text-neutral-200 border border-neutral-700"
                  >
                    {manualEntryMode ? 'Use Live Selectors' : 'Manual ID Fallback'}
                  </button>
                </div>
              </div>

              {selectorsError && (
                <div className="p-2.5 bg-amber-950/50 border border-amber-800/70 rounded-xl text-xs text-amber-200">
                  {selectorsError}
                </div>
              )}

              <form onSubmit={handleStartSession} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Bag Seal Barcode
                    </label>
                    <div className="relative">
                      <input
                        type="text"
                        required
                        value={bagSealBarcode}
                        onChange={(e) => setBagSealBarcode(e.target.value)}
                        placeholder="Scan or enter tamper seal barcode"
                        className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500 pr-9"
                      />
                      <Barcode size={16} className="absolute right-3 top-2.5 text-neutral-500" />
                    </div>
                  </div>

                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Driver
                    </label>
                    {!manualEntryMode && driverOptions.length > 0 ? (
                      <select
                        required
                        aria-label="Driver"
                        value={driverId}
                        onChange={(e) => {
                          const nextDriver = e.target.value;
                          setDriverId(nextDriver);
                          const match = driverOptions.find((d) => d.driverId === nextDriver);
                          if (match?.routeId && !routeId) {
                            setRouteId(match.routeId);
                          }
                        }}
                        className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                      >
                        <option value="">Select active driver...</option>
                        {driverOptions.map((d) => (
                          <option key={d.driverId} value={d.driverId}>
                            {d.label}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type="text"
                        required
                        aria-label="Driver ID"
                        value={driverId}
                        onChange={(e) => setDriverId(e.target.value)}
                        placeholder="Enter driver ID"
                        className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                      />
                    )}
                  </div>

                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Route / Manifest
                    </label>
                    {!manualEntryMode && routeOptions.length > 0 ? (
                      <select
                        required
                        aria-label="Route"
                        value={routeId}
                        onChange={(e) => {
                          const nextRoute = e.target.value;
                          setRouteId(nextRoute);
                          const match = routeOptions.find((r) => r.routeId === nextRoute);
                          if (match?.driverId && !driverId) {
                            setDriverId(match.driverId);
                          }
                        }}
                        className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                      >
                        <option value="">Select active route...</option>
                        {routeOptions.map((r) => (
                          <option key={r.routeId} value={r.routeId}>
                            {r.label}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type="text"
                        required
                        aria-label="Route ID"
                        value={routeId}
                        onChange={(e) => setRouteId(e.target.value)}
                        placeholder="Enter route or manifest ID"
                        className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                      />
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      System Expected Cash (so&apos;m)
                    </label>
                    <div className="w-full bg-neutral-950/70 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-emerald-400">
                      🔒 Server-Derived (Masked for Blind Count)
                    </div>
                    <span className="text-[10px] text-neutral-500 mt-0.5 block">
                      Authoritatively derived from Spanner route orders in so&apos;m; masked until count submission.
                    </span>
                  </div>

                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Depot Node
                    </label>
                    <input
                      type="text"
                      required
                      value={depotId}
                      onChange={(e) => setDepotId(e.target.value)}
                      placeholder="Resolved from warehouse scope"
                      className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Supplier ID
                    </label>
                    <input
                      type="text"
                      required
                      value={supplierId}
                      onChange={(e) => setSupplierId(e.target.value)}
                      placeholder="Resolved from auth claims"
                      className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>

                  <div>
                    <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
                      Routing Cell
                    </label>
                    <input
                      type="text"
                      value={cellId}
                      onChange={(e) => setCellId(e.target.value)}
                      placeholder="Resolved from home_cell"
                      className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
                    />
                  </div>
                </div>

                {startError && (
                  <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200">
                    {startError}
                  </div>
                )}

                <div className="flex justify-end pt-2">
                  <button
                    type="submit"
                    disabled={startingSession}
                    className="px-6 py-2.5 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-40 transition-all shadow flex items-center gap-1.5"
                  >
                    <PlusCircle size={15} />
                    <span>{startingSession ? 'Initializing Blind Session...' : 'Bind Seal & Begin Blind Count'}</span>
                  </button>
                </div>
              </form>
            </div>
          ) : (
            /* Active Blind Session Display */
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono text-neutral-400">Active Turn-In Session: {session.session_id}</span>
                <button
                  type="button"
                  onClick={() => {
                    setSession(null);
                    setOverrideBanner(null);
                  }}
                  className="text-xs text-neutral-400 hover:text-neutral-200 underline"
                >
                  Start another driver bag
                </button>
              </div>

              {overrideBanner && (
                <div
                  role="status"
                  className="p-4 rounded-xl bg-amber-950/60 border border-amber-700/60 flex flex-wrap items-start justify-between gap-3 text-xs text-amber-100"
                >
                  <div className="flex items-start gap-2.5">
                    <AlertTriangle size={16} className="text-amber-400 shrink-0 mt-0.5" />
                    <div>
                      <div className="font-bold text-amber-200">
                        Supervisor Override Request Queued
                      </div>
                      <p className="text-amber-100/90 mt-0.5 font-mono">
                        Session {overrideBanner.sessionId} · Driver {overrideBanner.driverId} remains locked under MARKING:DISCREPANT_QUARANTINE pending supervisor review ({overrideBanner.requestedAt}).
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => setOverrideBanner(null)}
                    className="px-2.5 py-1 rounded-lg bg-amber-900/60 hover:bg-amber-800/70 text-[11px] font-mono text-amber-100 border border-amber-700/50"
                  >
                    Dismiss
                  </button>
                </div>
              )}

              <CashierSessionView
                session={session}
                onOpenReclassifyModal={() => setIsRRNModalOpen(true)}
                onRequestSupervisorOverride={() => {
                  setOverrideBanner({
                    sessionId: session.session_id,
                    driverId: session.driver_id,
                    requestedAt: new Date().toISOString(),
                  });
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
              defaultActorId={actorId}
              onClose={() => setIsRRNModalOpen(false)}
              onSuccess={(updated) => setSession(updated)}
            />
          )}
        </div>
      )}

      {activeTab === 'cit' && (
        <CITHandoverManifest
          depotId={depotId}
          supplierId={supplierId}
          defaultSessionId={session?.session_id ?? ''}
        />
      )}

      {activeTab === 'aml' && (
        <AMLExposureCheck supplierId={supplierId} />
      )}
    </div>
  );
}
