'use client';

import React, { useState } from 'react';
import type { FleetDVIRInspection, DVIRChecklist8Point } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { ClipboardCheck, Truck, Wrench, AlertOctagon, CheckCircle2, ShieldAlert, AlertTriangle } from 'lucide-react';

interface FleetDVIRScreenProps {
  onInspectionComplete?: (inspection: FleetDVIRInspection) => void;
}

export function FleetDVIRScreen({ onInspectionComplete }: FleetDVIRScreenProps) {
  const [vehicleId, setVehicleId] = useState('trk_isuzu_04');
  const [driverId, setDriverId] = useState('drv_rustam_09');
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [inspectionType, setInspectionType] = useState<'PRE_TRIP' | 'POST_TRIP'>('PRE_TRIP');
  const [odometerKm, setOdometerKm] = useState<number>(45200);

  const [checklist, setChecklist] = useState<DVIRChecklist8Point>({
    steering_and_horn_ok: true,
    service_and_parking_brakes_ok: true,
    tires_tread_and_pressure_ok: true,
    lights_and_hazards_ok: true,
    wipers_and_fluids_ok: true,
    emergency_kit_and_extinguisher_ok: true,
    cargo_box_sanitation_ok: true,
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
    });
  };

  const setSimulateDefect = () => {
    setChecklist({
      steering_and_horn_ok: true,
      service_and_parking_brakes_ok: false, // Critical defect!
      tires_tread_and_pressure_ok: true,
      lights_and_hazards_ok: true,
      wipers_and_fluids_ok: true,
      emergency_kit_and_extinguisher_ok: true,
      cargo_box_sanitation_ok: true,
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
        checklist,
        odometer_km: odometerKm,
      });
      setResult(inspection);
      onInspectionComplete?.(inspection);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'DVIR submission failed');
    } finally {
      setSubmitting(false);
    }
  };

  const checkItems: { key: keyof DVIRChecklist8Point; label: string; desc: string }[] = [
    { key: 'steering_and_horn_ok', label: '1. Steering & Horn', desc: 'Steering rack freeplay and horn audibility (90+ dB)' },
    { key: 'service_and_parking_brakes_ok', label: '2. Service & Parking Brakes', desc: 'Firm pedal hydraulic resistance & parking lock hold' },
    { key: 'tires_tread_and_pressure_ok', label: '3. Tires & Pressure', desc: 'Tread depth ≥ 3.5mm, cold inflation pressure verified' },
    { key: 'lights_and_hazards_ok', label: '4. Lights & Signals', desc: 'High/low beams, brake lights, indicators, hazards functional' },
    { key: 'wipers_and_fluids_ok', label: '5. Wipers & Fluids', desc: 'Windshield washer, engine oil, coolant, brake fluid levels OK' },
    { key: 'emergency_kit_and_extinguisher_ok', label: '6. Emergency Equipment', desc: 'First-aid kit, certified ABC fire extinguisher, warning triangle' },
    { key: 'cargo_box_sanitation_ok', label: '7. Cargo Box Sanitation', desc: 'Clean food-grade interior, dry flooring, free from chemical odors' },
  ];

  return (
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-6 backdrop-blur-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-neutral-800">
        <div>
          <div className="flex items-center gap-2">
            <ClipboardCheck size={20} className="text-emerald-400" />
            <h2 className="text-base font-bold text-neutral-100 uppercase tracking-wide">
              Mandatory 8-Point Pre-Trip DVIR Walkaround
            </h2>
          </div>
          <p className="text-xs text-neutral-400 mt-0.5">
            Enforces the strict Red-Tag Grounding Gate (Guard #12). Any critical safety defect applies RED_TAG_QUARANTINE.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={setAllPass}
            className="text-xs px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-750 text-neutral-200 font-medium border border-neutral-700 transition-colors"
          >
            Check All Pass
          </button>
          <button
            type="button"
            onClick={setSimulateDefect}
            className="text-xs px-3 py-1.5 rounded-lg bg-rose-950/60 hover:bg-rose-900/60 text-rose-300 font-medium border border-rose-800 transition-colors"
          >
            Simulate Defect
          </button>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Vehicle & Trip Details */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 p-4 bg-neutral-950/70 rounded-xl border border-neutral-800">
          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Vehicle ID
            </label>
            <input
              type="text"
              required
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full bg-neutral-900 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Driver ID
            </label>
            <input
              type="text"
              required
              value={driverId}
              onChange={(e) => setDriverId(e.target.value)}
              className="w-full bg-neutral-900 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Odometer (KM)
            </label>
            <input
              type="number"
              required
              value={odometerKm}
              onChange={(e) => setOdometerKm(Math.max(0, parseInt(e.target.value || '0', 10)))}
              className="w-full bg-neutral-900 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Inspection Type
            </label>
            <select
              value={inspectionType}
              onChange={(e) => setInspectionType(e.target.value as 'PRE_TRIP' | 'POST_TRIP')}
              className="w-full bg-neutral-900 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-emerald-500"
            >
              <option value="PRE_TRIP">PRE_TRIP (Pre-Dispatch)</option>
              <option value="POST_TRIP">POST_TRIP (Yard Inbound)</option>
            </select>
          </div>
        </div>

        {/* Checklist Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {checkItems.map(({ key, label, desc }) => {
            const isPass = checklist[key];

            return (
              <div
                key={key}
                onClick={() => toggleCheck(key)}
                className={`p-4 rounded-xl border transition-all cursor-pointer flex items-start justify-between gap-3 ${
                  isPass
                    ? 'bg-neutral-950/60 border-neutral-800 hover:border-emerald-500/50'
                    : 'bg-rose-950/40 border-rose-600/80 hover:border-rose-500'
                }`}
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-bold text-neutral-100">{label}</span>
                  </div>
                  <p className="text-[11px] text-neutral-400 mt-1 leading-snug">{desc}</p>
                </div>

                <div className="shrink-0 mt-0.5">
                  {isPass ? (
                    <span className="px-2.5 py-1 rounded-lg text-[10px] font-bold bg-emerald-500/10 text-emerald-300 border border-emerald-500/40">
                      PASS
                    </span>
                  ) : (
                    <span className="px-2.5 py-1 rounded-lg text-[10px] font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40">
                      FAIL (RED-TAG)
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        {error && (
          <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200">
            {error}
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white disabled:opacity-40 transition-all shadow flex items-center gap-2"
          >
            {submitting ? 'Submitting & Evaluating Gates...' : 'Submit 8-Point DVIR Inspection'}
          </button>
        </div>
      </form>

      {/* Result Status Display */}
      {result && (
        <div className="space-y-4 pt-4 border-t border-neutral-800">
          {result.red_tag_applied ? (
            /* RED-TAG GROUNDING GATE ALERT */
            <div className="p-5 rounded-2xl bg-rose-950/70 border-2 border-rose-600 space-y-3 shadow-xl">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2 font-bold text-rose-200 text-sm">
                  <AlertOctagon size={22} className="text-rose-400" />
                  <span>VEHICLE GROUNDED: RED_TAG_QUARANTINE (OUT_OF_SERVICE)</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-rose-900/80 text-rose-100 border border-rose-700">
                  CRITICAL DEFECT DETECTED
                </span>
              </div>

              <p className="text-xs text-rose-200 leading-relaxed">
                Vehicle <strong className="font-mono text-white">{result.vehicle_id}</strong> failed mandatory safety points:{' '}
                <span className="font-mono text-rose-100 underline">
                  {result.failed_points?.join(', ') || 'safety defect'}
                </span>
                . The Red-Tag Grounding Gate has severed driver pairing, generated a workshop work order,
                and locked dispatch authorization.
              </p>

              <div className="grid grid-cols-2 md:grid-cols-3 gap-3 p-3 bg-neutral-950/90 rounded-xl border border-rose-800/60 text-xs">
                <div>
                  <span className="text-neutral-400 block font-mono text-[10px] uppercase">Maintenance Work Order:</span>
                  <span className="font-mono font-bold text-amber-300">{result.maintenance_work_order_id}</span>
                </div>
                <div>
                  <span className="text-neutral-400 block font-mono text-[10px] uppercase">Driver Pairing:</span>
                  <span className="font-mono text-rose-300 font-bold">SEVERED & UNCOUPLED</span>
                </div>
                <div>
                  <span className="text-neutral-400 block font-mono text-[10px] uppercase">Ontology Marking:</span>
                  <span className="font-mono text-purple-300">MARKING:DISCREPANT_QUARANTINE</span>
                </div>
              </div>
            </div>
          ) : (
            /* CLEAR DISPATCH PASS */
            <div className="p-5 rounded-2xl bg-emerald-950/40 border border-emerald-500/60 space-y-2 shadow-lg">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2 font-bold text-emerald-200 text-sm">
                  <CheckCircle2 size={20} className="text-emerald-400" />
                  <span>DISPATCH_AUTHORIZED: 8-Point Safety Verification Passed</span>
                </div>
                <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-emerald-900/60 text-emerald-100 border border-emerald-700">
                  ALL 8 CHECKS OK
                </span>
              </div>
              <p className="text-xs text-emerald-300/80">
                Vehicle <span className="font-mono text-white font-bold">{result.vehicle_id}</span> is cleared for manifest loading and driver route departure.
              </p>
              {result.preventative_service_due && (
                <div className="p-2.5 rounded-lg bg-amber-950/40 border border-amber-600/60 text-amber-300 text-xs flex items-center gap-2">
                  <AlertTriangle size={15} className="shrink-0" />
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
