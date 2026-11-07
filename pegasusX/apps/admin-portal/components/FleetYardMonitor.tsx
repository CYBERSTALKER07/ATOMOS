'use client';

import React, { useState } from 'react';
import type { FleetDVIRInspection, YardGateCheckEvent, PredictiveReeferResponse, DispatchCheckResponse } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';

export default function FleetYardMonitor({ token }: { token: string }) {
  const [vehicleId, setVehicleId] = useState('trk_isuzu_04');
  const [driverId, setDriverId] = useState('drv_rustam_09');

  // DVIR state
  const [dvirSubmitting, setDvirSubmitting] = useState(false);
  const [dvirResult, setDvirResult] = useState<FleetDVIRInspection | null>(null);

  // Gate state
  const [direction, setDirection] = useState<'GATE_IN' | 'GATE_OUT'>('GATE_OUT');
  const [licensePlate, setLicensePlate] = useState('01 A 777 AA');
  const [boltSeal, setBoltSeal] = useState('BOLT-998811');
  const [gateSubmitting, setGateSubmitting] = useState(false);
  const [gateEvent, setGateEvent] = useState<YardGateCheckEvent | null>(null);

  // Reefer & Dispatch Check
  const [coolingRate, setCoolingRate] = useState(0.24);
  const [reeferEvaluating, setReeferEvaluating] = useState(false);
  const [reeferResult, setReeferResult] = useState<PredictiveReeferResponse | null>(null);

  const [dispatchChecking, setDispatchChecking] = useState(false);
  const [dispatchResult, setDispatchResult] = useState<DispatchCheckResponse | null>(null);

  const [error, setError] = useState<string | null>(null);

  // Submit test DVIR with defect to demonstrate Red-Tag Grounding Gate
  const handleTestDVIR = async (failCheck: boolean) => {
    setError(null);
    setDvirResult(null);
    setDvirSubmitting(true);
    try {
      const res = await foundryApi.submitDVIR(token, {
        cell_id: 'cell-uz',
        supplier_id: 'sup_samarkand_dairy',
        vehicle_id: vehicleId.trim(),
        driver_id: driverId.trim(),
        inspection_type: 'PRE_TRIP',
        requires_cold_chain: true,
        checklist: {
          steering_and_horn_ok: !failCheck, // If failCheck, steering fails!
          service_and_parking_brakes_ok: true,
          tires_tread_and_pressure_ok: true,
          lights_and_hazards_ok: true,
          wipers_and_fluids_ok: true,
          emergency_kit_and_extinguisher_ok: true,
          cargo_box_sanitation_ok: true,
          reefer_temp_calibrated_ok: true,
        },
        odometer_km: 52100,
        reefer_temp_celsius: 4.0,
      });
      setDvirResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'DVIR submission failed');
    } finally {
      setDvirSubmitting(false);
    }
  };

  // Record Gate Event
  const handleGateEvent = async () => {
    setError(null);
    setGateSubmitting(true);
    try {
      const ev = await foundryApi.recordGateEvent(token, {
        depot_id: 'depot_central',
        supplier_id: 'sup_samarkand_dairy',
        vehicle_id: vehicleId.trim(),
        license_plate: licensePlate.trim(),
        direction,
        vehicle_purpose: 'OUTBOUND_ROUTE',
        bolt_seal_barcode: boltSeal.trim(),
        seal_intact: true,
      });
      setGateEvent(ev);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gate event failed');
    } finally {
      setGateSubmitting(false);
    }
  };

  // Evaluate Reefer
  const handleReeferCheck = async () => {
    setError(null);
    setReeferEvaluating(true);
    try {
      const res = await foundryApi.predictiveReefer(token, {
        vehicle_id: vehicleId.trim(),
        cooling_rate_deg_per_min: coolingRate,
        min_required_rate_deg_per_min: 0.30,
      });
      setReeferResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Predictive reefer evaluation failed');
    } finally {
      setReeferEvaluating(false);
    }
  };

  // Dispatch Check
  const handleDispatchCheck = async () => {
    setError(null);
    setDispatchChecking(true);
    try {
      const res = await foundryApi.dispatchCheck(token, vehicleId.trim());
      setDispatchResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Dispatch check failed');
    } finally {
      setDispatchChecking(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* 8-Point DVIR Red-Tag Grounding Gate Monitor */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Pre-Trip 8-Point DVIR & Red-Tag Grounding Gate Auditor
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Audit walkaround safety inspections. Any failed checkpoint triggers RED_TAG_QUARANTINE.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <input
            type="text"
            value={vehicleId}
            onChange={(e) => setVehicleId(e.target.value)}
            className="rounded border px-3 py-1.5 text-xs font-mono"
            placeholder="Vehicle ID"
          />
          <input
            type="text"
            value={driverId}
            onChange={(e) => setDriverId(e.target.value)}
            className="rounded border px-3 py-1.5 text-xs font-mono"
            placeholder="Driver ID"
          />

          <button
            type="button"
            onClick={() => handleTestDVIR(false)}
            disabled={dvirSubmitting}
            className="rounded bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
          >
            {dvirSubmitting ? 'Evaluating...' : 'Simulate Clean DVIR (PASS)'}
          </button>

          <button
            type="button"
            onClick={() => handleTestDVIR(true)}
            disabled={dvirSubmitting}
            className="rounded bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
          >
            {dvirSubmitting ? 'Evaluating...' : 'Simulate Defect (Trigger RED-TAG)'}
          </button>

          <button
            type="button"
            onClick={handleDispatchCheck}
            disabled={dispatchChecking}
            className="rounded border px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            {dispatchChecking ? 'Checking...' : 'Check Dispatch Clearance'}
          </button>
        </div>

        {dispatchResult && (
          <div className="p-3 rounded border bg-gray-50 text-xs flex justify-between items-center font-mono">
            <span>Vehicle: {dispatchResult.vehicle_id}</span>
            <span className={dispatchResult.dispatch_authorized ? 'text-emerald-700 font-bold' : 'text-red-700 font-bold'}>
              {dispatchResult.dispatch_authorized ? '✅ DISPATCH AUTHORIZED' : '🛑 DISPATCH BLOCKED'}
            </span>
          </div>
        )}

        {dvirResult && (
          <div
            className={`p-4 rounded-lg border text-xs space-y-2 ${
              dvirResult.red_tag_applied
                ? 'bg-red-50 border-red-300 text-red-900'
                : 'bg-emerald-50 border-emerald-300 text-emerald-900'
            }`}
          >
            <div className="flex justify-between items-center font-bold">
              <span>Status: {dvirResult.status}</span>
              {dvirResult.red_tag_applied && (
                <span className="bg-red-200 text-red-800 px-2 py-0.5 rounded font-mono">
                  RED_TAG_QUARANTINE
                </span>
              )}
            </div>

            {dvirResult.red_tag_applied ? (
              <div className="space-y-1">
                <div>Failed safety points: <span className="font-mono font-bold">{dvirResult.failed_points?.join(', ')}</span></div>
                <div>Maintenance Work Order: <span className="font-mono font-bold">{dvirResult.maintenance_work_order_id}</span></div>
                <div className="text-[11px] text-red-700 font-semibold">• Driver pairing severed • Outbound gate departure locked</div>
              </div>
            ) : (
              <div>Vehicle passed all 8 inspection points. Cleared for manifest loading.</div>
            )}
          </div>
        )}

        {error && (
          <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </div>
        )}
      </div>

      {/* Yard Gate ALPR & Outbound Seal Auditor */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            ALPR Yard Security Gate & Outbound Bolt Seal Auditor
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Logs automated gate crossings and verifies cryptographic bolt seal integrity.
          </p>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-700">License Plate</label>
            <input
              type="text"
              value={licensePlate}
              onChange={(e) => setLicensePlate(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono uppercase"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Direction</label>
            <select
              value={direction}
              onChange={(e) => setDirection(e.target.value as 'GATE_IN' | 'GATE_OUT')}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            >
              <option value="GATE_IN">GATE_IN</option>
              <option value="GATE_OUT">GATE_OUT</option>
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Bolt Seal Barcode</label>
            <input
              type="text"
              value={boltSeal}
              onChange={(e) => setBoltSeal(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div className="flex items-end">
            <button
              type="button"
              onClick={handleGateEvent}
              disabled={gateSubmitting}
              className="w-full rounded bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {gateSubmitting ? 'Recording...' : 'Record Gate Event'}
            </button>
          </div>
        </div>

        {gateEvent && (
          <div className="p-3 rounded border bg-gray-50 text-xs flex justify-between items-center font-mono">
            <span>
              {gateEvent.direction} {gateEvent.license_plate}
            </span>
            <span className="font-bold text-indigo-700">Bay: {gateEvent.assigned_bay_slot}</span>
            <span>Seal: {gateEvent.bolt_seal_barcode || 'None'}</span>
          </div>
        )}
      </div>

      {/* Predictive Reefer Telemetry */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            Predictive Reefer Maintenance Hold Monitor
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Evaluates cooling rate degradation before spoiled freight can be staged.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <input
            type="number"
            step="0.01"
            value={coolingRate}
            onChange={(e) => setCoolingRate(parseFloat(e.target.value || '0'))}
            className="rounded border px-3 py-1.5 text-xs font-mono w-32"
          />
          <span className="text-xs text-gray-500">Observed cooling rate (°C/min)</span>

          <button
            type="button"
            onClick={handleReeferCheck}
            disabled={reeferEvaluating}
            className="rounded bg-cyan-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-cyan-700 disabled:opacity-50"
          >
            {reeferEvaluating ? 'Evaluating...' : 'Evaluate Cooling Health'}
          </button>
        </div>

        {reeferResult && (
          <div
            className={`p-3 rounded border text-xs flex justify-between items-center font-mono ${
              reeferResult.pre_failure_maintenance_hold
                ? 'bg-amber-50 border-amber-300 text-amber-900 font-bold'
                : 'bg-emerald-50 border-emerald-300 text-emerald-900'
            }`}
          >
            <span>Vehicle: {reeferResult.vehicle_id}</span>
            <span>
              {reeferResult.pre_failure_maintenance_hold
                ? '⚠️ PRE_FAILURE_MAINTENANCE_HOLD APPLIED (Cooling curve degraded)'
                : '✅ REEFER HEALTHY (Cooling rate satisfies SLA)'}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
