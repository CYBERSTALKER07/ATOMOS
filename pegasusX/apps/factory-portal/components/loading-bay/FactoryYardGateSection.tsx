'use client';

import React, { useState } from 'react';
import { foundryFactoryApi } from '@/lib/foundry-api';
import type { YardGateCheckEvent, YardBaySlot } from '@pegasusx/types';

interface FactoryYardGateSectionProps {
  depotId?: string;
  supplierId?: string;
}

const BAY_SLOTS: YardBaySlot[] = [
  'INBOUND_STAGING',
  'CROSS_DOCK',
  'OUTBOUND_LOADING',
  'MECHANICAL_SERVICE',
];

export default function FactoryYardGateSection({
  depotId = 'factory-tashkent-01',
  supplierId = 'supp-apex-central',
}: FactoryYardGateSectionProps) {
  const [direction, setDirection] = useState<'GATE_IN' | 'GATE_OUT'>('GATE_IN');
  const [vehicleId, setVehicleId] = useState('TRUCK-UZB-042');
  const [licensePlate, setLicensePlate] = useState('01 777 AAA');
  const [purpose, setPurpose] = useState<'INBOUND_FTL' | 'OUTBOUND_ROUTE' | 'CROSS_DOCK' | 'WORKSHOP'>('INBOUND_FTL');
  const [baySlot, setBaySlot] = useState<YardBaySlot>('INBOUND_STAGING');
  const [boltSeal, setBoltSeal] = useState('BOLT-UZB-84920');
  const [sealIntact, setSealIntact] = useState(true);

  const [submitting, setSubmitting] = useState(false);
  const [gateEvents, setGateEvents] = useState<YardGateCheckEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleRecordGateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccessMsg(null);

    // Security Gate Enforcement:
    if (direction === 'GATE_OUT') {
      if (!boltSeal.trim()) {
        setError('HARD GATE REJECTION: Outbound transport requires a verified bolt-seal barcode.');
        return;
      }
      if (!sealIntact) {
        setError('HARD GATE VIOLATION: Broken or tampered bolt-seal detected. Vehicle departure forbidden until physical re-inspection.');
        return;
      }
    }

    setSubmitting(true);
    try {
      const payload: YardGateCheckEvent = {
        depot_id: depotId,
        supplier_id: supplierId,
        vehicle_id: vehicleId.trim(),
        license_plate: licensePlate.trim(),
        direction,
        vehicle_purpose: purpose,
        bolt_seal_barcode: boltSeal.trim() || undefined,
        seal_intact: sealIntact,
        assigned_bay_slot: baySlot,
        created_at: new Date().toISOString(),
      };

      const result = await foundryFactoryApi.recordGateEvent(payload);
      setGateEvents((prev) => [result, ...prev]);
      setSuccessMsg(`Gate Event recorded successfully: ${result.license_plate} (${result.direction}) assigned to ${result.assigned_bay_slot}`);
      
      // Auto-toggle sample direction for operator convenience
      if (direction === 'GATE_IN') {
        setDirection('GATE_OUT');
        setBaySlot('OUTBOUND_LOADING');
      } else {
        setDirection('GATE_IN');
        setBaySlot('INBOUND_STAGING');
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to record yard gate check event';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="rounded-xl border border-[var(--color-md-outline-variant,rgba(255,255,255,0.1))] bg-[var(--color-md-surface-container,rgba(30,30,30,0.8))] p-6 text-[var(--color-md-on-surface,#f1f5f9)] shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-inherit pb-4 mb-5">
        <div>
          <h3 className="text-base font-bold flex items-center gap-2">
            <span>🚛</span> Factory Loading Bay Yard Gate & Bolt-Seal Verification
          </h3>
          <p className="text-xs text-[var(--color-md-on-surface-variant,#94a3b8)] mt-0.5">
            Role 9 Fleet / Factory gate-events tracking with ALPR verification and tamper-evident bolt-seal gate enforcement.
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs">
          <span className="rounded bg-blue-950/80 px-2.5 py-1 text-blue-300 border border-blue-800 font-mono">
            Depot: {depotId}
          </span>
          <span className="rounded bg-purple-950/80 px-2.5 py-1 text-purple-300 border border-purple-800 font-mono">
            {supplierId}
          </span>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg bg-red-950/60 border border-red-800 p-3 text-xs text-red-200">
          <strong>Gate Error:</strong> {error}
        </div>
      )}

      {successMsg && (
        <div className="mb-4 rounded-lg bg-emerald-950/60 border border-emerald-800 p-3 text-xs text-emerald-200">
          ✓ {successMsg}
        </div>
      )}

      <form onSubmit={handleRecordGateEvent} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-xs">
        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            Gate Direction
          </label>
          <div className="flex rounded-lg border border-neutral-700 bg-neutral-900 p-1">
            <button
              type="button"
              onClick={() => {
                setDirection('GATE_IN');
                setBaySlot('INBOUND_STAGING');
              }}
              className={`flex-1 rounded py-1 font-semibold transition ${
                direction === 'GATE_IN' ? 'bg-emerald-600 text-white shadow' : 'text-neutral-400 hover:text-white'
              }`}
            >
              GATE IN (Ingress)
            </button>
            <button
              type="button"
              onClick={() => {
                setDirection('GATE_OUT');
                setBaySlot('OUTBOUND_LOADING');
              }}
              className={`flex-1 rounded py-1 font-semibold transition ${
                direction === 'GATE_OUT' ? 'bg-blue-600 text-white shadow' : 'text-neutral-400 hover:text-white'
              }`}
            >
              GATE OUT (Departure)
            </button>
          </div>
        </div>

        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            ALPR License Plate
          </label>
          <input
            type="text"
            required
            value={licensePlate}
            onChange={(e) => setLicensePlate(e.target.value)}
            className="w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-white font-mono focus:border-blue-500 focus:outline-none"
            placeholder="01 777 AAA"
          />
        </div>

        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            Vehicle Identifier
          </label>
          <input
            type="text"
            required
            value={vehicleId}
            onChange={(e) => setVehicleId(e.target.value)}
            className="w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-white font-mono focus:border-blue-500 focus:outline-none"
            placeholder="TRUCK-UZB-042"
          />
        </div>

        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            Transport Purpose
          </label>
          <select
            value={purpose}
            onChange={(e) => setPurpose(e.target.value as 'INBOUND_FTL' | 'OUTBOUND_ROUTE' | 'CROSS_DOCK' | 'WORKSHOP')}
            className="w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-white focus:border-blue-500 focus:outline-none"
          >
            <option value="INBOUND_FTL">INBOUND FTL (Raw Material)</option>
            <option value="OUTBOUND_ROUTE">OUTBOUND ROUTE (Finished Goods)</option>
            <option value="CROSS_DOCK">CROSS DOCK (Direct Transfer)</option>
            <option value="WORKSHOP">WORKSHOP (Maintenance Yard)</option>
          </select>
        </div>

        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            Assigned Bay Slot
          </label>
          <select
            value={baySlot}
            onChange={(e) => setBaySlot(e.target.value as YardBaySlot)}
            className="w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-white font-mono focus:border-blue-500 focus:outline-none"
          >
            {BAY_SLOTS.map((bay) => (
              <option key={bay} value={bay}>
                {bay}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block font-medium text-[var(--color-md-on-surface-variant,#94a3b8)] mb-1">
            Bolt-Seal Serial Barcode
          </label>
          <input
            type="text"
            value={boltSeal}
            onChange={(e) => setBoltSeal(e.target.value)}
            className="w-full rounded-lg border border-neutral-700 bg-neutral-900 px-3 py-2 text-white font-mono focus:border-blue-500 focus:outline-none"
            placeholder="BOLT-UZB-84920"
          />
        </div>

        <div className="flex flex-col justify-end">
          <label className="flex items-center gap-2 cursor-pointer py-2">
            <input
              type="checkbox"
              checked={sealIntact}
              onChange={(e) => setSealIntact(e.target.checked)}
              className="h-4 w-4 rounded border-neutral-700 bg-neutral-900 text-blue-600 focus:ring-blue-500"
            />
            <span className="font-semibold text-white">Bolt Seal Verified Intact</span>
          </label>
        </div>

        <div className="flex items-end">
          <button
            type="submit"
            disabled={submitting}
            className={`w-full rounded-lg py-2 text-xs font-bold text-white shadow transition disabled:opacity-50 ${
              direction === 'GATE_IN' ? 'bg-emerald-600 hover:bg-emerald-500' : 'bg-blue-600 hover:bg-blue-500'
            }`}
          >
            {submitting ? 'Verifying Gate...' : `Submit ${direction === 'GATE_IN' ? 'Check-In' : 'Check-Out'}`}
          </button>
        </div>
      </form>

      {/* Log of recorded Gate Events */}
      {gateEvents.length > 0 && (
        <div className="mt-6 border-t border-inherit pt-4">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[var(--color-md-on-surface-variant,#94a3b8)] mb-3">
            Recent Loading Bay Gate Logs ({gateEvents.length})
          </h4>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-neutral-900/60 text-neutral-400">
                <tr>
                  <th className="py-2 px-3">Timestamp</th>
                  <th className="py-2 px-3">Direction</th>
                  <th className="py-2 px-3">Plate / Vehicle</th>
                  <th className="py-2 px-3">Purpose</th>
                  <th className="py-2 px-3">Slot</th>
                  <th className="py-2 px-3">Bolt Seal</th>
                  <th className="py-2 px-3">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-800">
                {gateEvents.map((evt, idx) => (
                  <tr key={evt.gate_event_id || idx} className="hover:bg-neutral-900/30">
                    <td className="py-2 px-3 font-mono text-neutral-400">
                      {evt.created_at ? new Date(evt.created_at).toLocaleTimeString() : 'Just now'}
                    </td>
                    <td className="py-2 px-3">
                      <span className={`inline-block rounded px-2 py-0.5 font-bold text-[10px] ${
                        evt.direction === 'GATE_IN' ? 'bg-emerald-950 text-emerald-300 border border-emerald-800' : 'bg-blue-950 text-blue-300 border border-blue-800'
                      }`}>
                        {evt.direction}
                      </span>
                    </td>
                    <td className="py-2 px-3 font-mono font-medium text-white">
                      {evt.license_plate} <span className="text-neutral-500 text-[11px]">({evt.vehicle_id})</span>
                    </td>
                    <td className="py-2 px-3 text-neutral-300">{evt.vehicle_purpose}</td>
                    <td className="py-2 px-3 font-mono font-semibold text-amber-300">{evt.assigned_bay_slot}</td>
                    <td className="py-2 px-3 font-mono text-neutral-300">{evt.bolt_seal_barcode || '—'}</td>
                    <td className="py-2 px-3">
                      {evt.seal_intact ? (
                        <span className="text-emerald-400 font-medium">✓ Intact</span>
                      ) : (
                        <span className="text-red-400 font-bold">⚠️ Tampered</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
