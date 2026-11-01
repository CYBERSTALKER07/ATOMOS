'use client';

import React, { useState } from 'react';
import type { PredictiveReeferResponse, DispatchCheckResponse } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface PredictiveReeferCardProps {
  defaultVehicleId?: string;
}

export function PredictiveReeferCard({
  defaultVehicleId = 'trk_isuzu_04',
}: PredictiveReeferCardProps) {
  const [vehicleId, setVehicleId] = useState(defaultVehicleId);
  const [coolingRate, setCoolingRate] = useState<number>(0.22);
  const [minRequiredRate, setMinRequiredRate] = useState<number>(0.30);
  const [evaluating, setEvaluating] = useState(false);
  const [reeferResult, setReeferResult] = useState<PredictiveReeferResponse | null>(null);
  const [dispatchResult, setDispatchResult] = useState<DispatchCheckResponse | null>(null);
  const [checkingDispatch, setCheckingDispatch] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleEvaluate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setReeferResult(null);
    setEvaluating(true);
    try {
      const res = await foundryWarehouseApi.predictiveReefer({
        vehicle_id: vehicleId.trim(),
        cooling_rate_deg_per_min: coolingRate,
        min_required_rate_deg_per_min: minRequiredRate,
      });
      setReeferResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Predictive reefer evaluation failed');
    } finally {
      setEvaluating(false);
    }
  };

  const handleDispatchCheck = async () => {
    setError(null);
    setDispatchResult(null);
    setCheckingDispatch(true);
    try {
      const res = await foundryWarehouseApi.dispatchCheck(vehicleId.trim());
      setDispatchResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Dispatch check failed');
    } finally {
      setCheckingDispatch(false);
    }
  };

  const rateDelta = coolingRate - minRequiredRate;
  const isDegraded = rateDelta < 0;

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-slate-700/60">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
            <span>❄️</span>
            <span>Predictive Reefer Compressor Telemetry & Dispatch Gate</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Detects refrigeration compressor decay curves before cold-chain cargo is loaded. Enforces pre-failure maintenance holds.
          </p>
        </div>

        <button
          type="button"
          onClick={handleDispatchCheck}
          disabled={checkingDispatch || !vehicleId.trim()}
          className="px-4 py-2 rounded-xl text-xs font-bold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-colors shadow flex items-center gap-2 disabled:opacity-50"
        >
          {checkingDispatch ? 'Checking Authorization...' : 'Verify Vehicle Dispatch Auth'}
        </button>
      </div>

      {dispatchResult && (
        <div
          className={`p-4 rounded-xl border text-xs flex items-center justify-between ${
            dispatchResult.dispatch_authorized
              ? 'bg-emerald-950/40 border-emerald-500/50 text-emerald-200'
              : 'bg-rose-950/40 border-rose-500/50 text-rose-200'
          }`}
        >
          <div className="flex items-center gap-2">
            <span className="text-base">{dispatchResult.dispatch_authorized ? '✅' : '🛑'}</span>
            <span className="font-semibold">
              Vehicle {dispatchResult.vehicle_id}:{' '}
              {dispatchResult.dispatch_authorized ? 'DISPATCH AUTHORIZED' : 'DISPATCH BLOCKED'}
            </span>
          </div>
          <span className="font-mono text-[11px]">
            {dispatchResult.dispatch_authorized
              ? 'Pre-Trip DVIR Clear & No Maintenance Holds'
              : 'Red-Tagged or Reefer Hold Active'}
          </span>
        </div>
      )}

      <form onSubmit={handleEvaluate} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Vehicle ID
            </label>
            <input
              type="text"
              required
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Observed Cooling Rate (°C/min)
            </label>
            <input
              type="number"
              step="0.01"
              required
              value={coolingRate}
              onChange={(e) => setCoolingRate(parseFloat(e.target.value || '0'))}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Minimum Required Rate (°C/min)
            </label>
            <input
              type="number"
              step="0.01"
              required
              value={minRequiredRate}
              onChange={(e) => setMinRequiredRate(parseFloat(e.target.value || '0'))}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-cyan-500"
            />
          </div>
        </div>

        {error && (
          <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={evaluating}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-cyan-600 hover:bg-cyan-500 text-white disabled:opacity-50 transition-all shadow-md flex items-center gap-2"
          >
            {evaluating ? 'Analyzing Curve...' : 'Evaluate Telemetry & Enforce Reefer Hold'}
          </button>
        </div>
      </form>

      {/* Evaluation Results */}
      {reeferResult && (
        <div className="space-y-3 pt-3 border-t border-slate-800">
          {reeferResult.pre_failure_maintenance_hold ? (
            /* PRE-FAILURE MAINTENANCE HOLD ALERT */
            <div className="p-5 rounded-2xl bg-amber-950/60 border-2 border-amber-500 space-y-3 shadow-xl">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 font-bold text-amber-200 text-base">
                  <span className="text-2xl">⚠️</span>
                  <span>PRE_FAILURE_MAINTENANCE_HOLD APPLIED</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-amber-900 text-amber-100">
                  DEGRADED COOLING CURVE
                </span>
              </div>

              <p className="text-xs text-amber-300 leading-relaxed">
                Reefer compressor on vehicle <strong>{reeferResult.vehicle_id}</strong> cools at only{' '}
                <span className="font-mono font-bold underline">{coolingRate} °C/min</span>, failing the minimum threshold of{' '}
                <span className="font-mono font-bold">{minRequiredRate} °C/min</span>. Vehicle is locked on pre-failure maintenance hold to protect perishable cargo.
              </p>

              <div className="p-3 bg-slate-950/80 rounded-xl border border-amber-800 text-xs text-slate-300 space-y-1">
                <div>• Perishable freight staging blocked for this vehicle.</div>
                <div>• Driver uncoupled until refrigeration technician calibration pass.</div>
              </div>
            </div>
          ) : (
            /* HEALTHY REEFER STATUS */
            <div className="p-5 rounded-2xl bg-emerald-950/50 border border-emerald-500/60 space-y-2 shadow-lg">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 font-bold text-emerald-200 text-base">
                  <span className="text-2xl">❄️</span>
                  <span>Reefer Compressor Operational</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-emerald-900 text-emerald-100">
                  RATE OK: +{rateDelta.toFixed(2)} °C/min margin
                </span>
              </div>
              <p className="text-xs text-emerald-300/80">
                Vehicle {reeferResult.vehicle_id} meets refrigeration SLA standards and is approved for cold chain loading.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
