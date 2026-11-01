'use client';

import React, { useState } from 'react';
import type { FleetDVIRInspection, DVIRChecklist8Point } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface FleetDVIRScreenProps {
  onInspectionComplete?: (inspection: FleetDVIRInspection) => void;
}

export function FleetDVIRScreen({ onInspectionComplete }: FleetDVIRScreenProps) {
  const [vehicleId, setVehicleId] = useState('trk_isuzu_04');
  const [driverId, setDriverId] = useState('drv_rustam_09');
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [inspectionType, setInspectionType] = useState<'PRE_TRIP' | 'POST_TRIP'>('PRE_TRIP');
  const [requiresColdChain, setRequiresColdChain] = useState(true);
  const [odometerKm, setOdometerKm] = useState<number>(45200);
  const [reeferTempCelsius, setReeferTempCelsius] = useState<number>(3.5);

  const [checklist, setChecklist] = useState<DVIRChecklist8Point>({
    steering_and_horn_ok: true,
    service_and_parking_brakes_ok: true,
    tires_tread_and_pressure_ok: true,
    lights_and_hazards_ok: true,
    wipers_and_fluids_ok: true,
    emergency_kit_and_extinguisher_ok: true,
    cargo_box_sanitation_ok: true,
    reefer_temp_calibrated_ok: true,
  });

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<FleetDVIRInspection | null>(null);

  const toggleCheck = (key: keyof DVIRChecklist8Point) => {
    setChecklist((prev) => ({
      ...prev,
      [key]: !prev[key],
    }));
  };

  const setAllPass = () => {
    setChecklist({
      steering_and_horn_ok: true,
      service_and_parking_brakes_ok: true,
      tires_tread_and_pressure_ok: true,
      lights_and_hazards_ok: true,
      wipers_and_fluids_ok: true,
      emergency_kit_and_extinguisher_ok: true,
      cargo_box_sanitation_ok: true,
      reefer_temp_calibrated_ok: true,
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setResult(null);

    setSubmitting(true);
    try {
      const inspection = await foundryWarehouseApi.submitDVIR({
        cell_id: cellId,
        supplier_id: supplierId,
        vehicle_id: vehicleId.trim(),
        driver_id: driverId.trim(),
        inspection_type: inspectionType,
        requires_cold_chain: requiresColdChain,
        checklist,
        odometer_km: odometerKm,
        reefer_temp_celsius: reeferTempCelsius,
      });
      setResult(inspection);
      onInspectionComplete?.(inspection);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'DVIR submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  const checkItems: { key: keyof DVIRChecklist8Point; label: string; desc: string; coldChainOnly?: boolean }[] = [
    { key: 'steering_and_horn_ok', label: '1. Steering & Horn', desc: 'Steering rack freeplay and horn audibility (90+ dB)' },
    { key: 'service_and_parking_brakes_ok', label: '2. Service & Parking Brakes', desc: 'Firm pedal hydraulic resistance & parking lock hold' },
    { key: 'tires_tread_and_pressure_ok', label: '3. Tires & Pressure', desc: 'Tread depth ≥ 3.5mm, cold inflation pressure verified' },
    { key: 'lights_and_hazards_ok', label: '4. Lights & Signals', desc: 'High/low beams, brake lights, indicators, hazards functional' },
    { key: 'wipers_and_fluids_ok', label: '5. Wipers & Fluids', desc: 'Windshield washer, engine oil, coolant, brake fluid levels OK' },
    { key: 'emergency_kit_and_extinguisher_ok', label: '6. Emergency Equipment', desc: 'First-aid kit, certified ABC fire extinguisher, warning triangle' },
    { key: 'cargo_box_sanitation_ok', label: '7. Cargo Box Sanitation', desc: 'Clean food-grade interior, dry flooring, free from chemical odors' },
    {
      key: 'reefer_temp_calibrated_ok',
      label: '8. Reefer Calibration',
      desc: 'Thermoking / Carrier micro-controller calibrated and probe active',
      coldChainOnly: true,
    },
  ];

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-slate-700/60">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
            <span>📋</span>
            <span>Mandatory 8-Point Pre-Trip DVIR Walkaround</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Enforces the strict Red-Tag Grounding Gate (Guard #12). Any defect applies RED_TAG_QUARANTINE.
          </p>
        </div>

        <button
          type="button"
          onClick={setAllPass}
          className="text-xs px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium transition-colors"
        >
          Check All Pass
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Vehicle & Trip Details */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-4 bg-slate-900/50 rounded-xl border border-slate-800">
          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Vehicle ID
            </label>
            <input
              type="text"
              required
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Driver ID
            </label>
            <input
              type="text"
              required
              value={driverId}
              onChange={(e) => setDriverId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Odometer (KM)
            </label>
            <input
              type="number"
              required
              value={odometerKm}
              onChange={(e) => setOdometerKm(Math.max(0, parseInt(e.target.value || '0', 10)))}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Inspection Type
            </label>
            <select
              value={inspectionType}
              onChange={(e) => setInspectionType(e.target.value as 'PRE_TRIP' | 'POST_TRIP')}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            >
              <option value="PRE_TRIP">PRE_TRIP (Pre-Dispatch)</option>
              <option value="POST_TRIP">POST_TRIP (Yard Inbound)</option>
            </select>
          </div>
        </div>

        {/* Cold chain configuration */}
        <div className="flex items-center justify-between p-4 bg-slate-900/40 rounded-xl border border-slate-800">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="cold_chain_toggle"
              checked={requiresColdChain}
              onChange={(e) => setRequiresColdChain(e.target.checked)}
              className="w-4 h-4 rounded text-emerald-500 focus:ring-emerald-400"
            />
            <label htmlFor="cold_chain_toggle" className="text-sm font-medium text-slate-200 cursor-pointer">
              Perishable Cold Chain Freight (Mandatory Reefer Probe Calibration)
            </label>
          </div>

          {requiresColdChain && (
            <div className="flex items-center gap-2">
              <label className="text-xs text-slate-400">Box Temp (°C):</label>
              <input
                type="number"
                step="0.1"
                value={reeferTempCelsius}
                onChange={(e) => setReeferTempCelsius(parseFloat(e.target.value || '0'))}
                className="w-20 bg-slate-950 border border-slate-700 rounded-lg px-2 py-1 text-xs font-mono text-slate-100 text-right"
              />
            </div>
          )}
        </div>

        {/* 8-Point Checklist Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {checkItems.map(({ key, label, desc, coldChainOnly }) => {
            const isApplicable = !coldChainOnly || requiresColdChain;
            const isPass = checklist[key];

            return (
              <div
                key={key}
                onClick={() => isApplicable && toggleCheck(key)}
                className={`p-4 rounded-xl border transition-all cursor-pointer flex items-start justify-between gap-3 ${
                  !isApplicable
                    ? 'opacity-40 bg-slate-900/20 border-slate-800 cursor-not-allowed'
                    : isPass
                      ? 'bg-slate-900/60 border-slate-800 hover:border-emerald-500/50'
                      : 'bg-rose-950/30 border-rose-600/70 hover:border-rose-500'
                }`}
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-slate-100">{label}</span>
                    {coldChainOnly && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-cyan-950 text-cyan-300 border border-cyan-800">
                        Cold Chain
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-slate-400 mt-1 leading-snug">{desc}</p>
                </div>

                <div className="shrink-0 mt-0.5">
                  {isPass ? (
                    <span className="px-2.5 py-1 rounded-lg text-xs font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40">
                      PASS
                    </span>
                  ) : (
                    <span className="px-2.5 py-1 rounded-lg text-xs font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40">
                      FAIL (RED-TAG)
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        {error && (
          <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-50 transition-all shadow-md flex items-center gap-2"
          >
            {submitting ? 'Submitting & Evaluating Gates...' : 'Submit 8-Point DVIR Inspection'}
          </button>
        </div>
      </form>

      {/* Result Status Display */}
      {result && (
        <div className="space-y-4 pt-4 border-t border-slate-800">
          {result.red_tag_applied ? (
            /* RED-TAG GROUNDING GATE ALERT */
            <div className="p-5 rounded-2xl bg-rose-950/60 border-2 border-rose-600 space-y-3 shadow-xl">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 font-bold text-rose-200 text-base">
                  <span className="text-2xl">🛑</span>
                  <span>VEHICLE GROUNDED: RED_TAG_QUARANTINE (OUT_OF_SERVICE)</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-rose-900 text-rose-100">
                  CRITICAL DEFECT DETECTED
                </span>
              </div>

              <p className="text-xs text-rose-300 leading-relaxed">
                Vehicle <strong>{result.vehicle_id}</strong> failed mandatory safety points:{' '}
                <span className="font-mono text-white underline">
                  {result.failed_points?.join(', ') || 'safety defect'}
                </span>
                . The Red-Tag Grounding Gate has severed driver pairing, generated a workshop work order,
                and locked dispatch authorization.
              </p>

              <div className="grid grid-cols-2 md:grid-cols-3 gap-3 p-3 bg-slate-950/80 rounded-xl border border-rose-800/60 text-xs">
                <div>
                  <span className="text-slate-400 block">Maintenance Work Order:</span>
                  <span className="font-mono font-bold text-amber-300">{result.maintenance_work_order_id}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Driver Pairing:</span>
                  <span className="font-mono text-rose-300 font-bold">SEVERED & UNCOUPLED</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Ontology Marking:</span>
                  <span className="font-mono text-purple-300">MARKING:DISCREPANT_QUARANTINE</span>
                </div>
              </div>
            </div>
          ) : (
            /* CLEAR DISPATCH PASS */
            <div className="p-5 rounded-2xl bg-emerald-950/50 border border-emerald-500/60 space-y-2 shadow-lg">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 font-bold text-emerald-200 text-base">
                  <span className="text-2xl">✅</span>
                  <span>DISPATCH_AUTHORIZED: 8-Point Safety Verification Passed</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-emerald-900 text-emerald-100">
                  ALL 8 CHECKS OK
                </span>
              </div>
              <p className="text-xs text-emerald-300/80">
                Vehicle {result.vehicle_id} is cleared for manifest loading and driver route departure.
              </p>
              {result.preventative_service_due && (
                <div className="p-2.5 rounded-lg bg-amber-950/40 border border-amber-600/60 text-amber-300 text-xs flex items-center gap-2">
                  <span>ℹ️</span>
                  <span>Preventative service milestone reached ({result.odometer_km.toLocaleString()} km). Schedule workshop inspection post-shift.</span>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
