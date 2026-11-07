'use client';

import React, { useState } from 'react';
import type { YardGateCheckEvent, YardBaySlot } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface YardGateCheckScreenProps {
  depotId?: string;
  supplierId?: string;
}

export function YardGateCheckScreen({
  depotId = 'depot_tashkent_central',
  supplierId = 'sup_samarkand_dairy',
}: YardGateCheckScreenProps) {
  const [vehicleId, setVehicleId] = useState('trk_isuzu_04');
  const [licensePlate, setLicensePlate] = useState('01 A 777 AA');
  const [direction, setDirection] = useState<'GATE_IN' | 'GATE_OUT'>('GATE_IN');
  const [vehiclePurpose, setVehiclePurpose] = useState<'INBOUND_FTL' | 'CROSS_DOCK' | 'OUTBOUND_ROUTE' | 'WORKSHOP'>('INBOUND_FTL');
  const [boltSealBarcode, setBoltSealBarcode] = useState('');
  const [sealIntact, setSealIntact] = useState(true);

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [recentEvents, setRecentEvents] = useState<YardGateCheckEvent[]>([]);

  const handleGateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (direction === 'GATE_OUT') {
      if (!boltSealBarcode.trim()) {
        setError('Outbound Security Gate: Cryptographic bolt seal barcode is mandatory for GATE_OUT');
        return;
      }
      if (!sealIntact) {
        setError('Outbound Security Gate: Outbound gate departure is hard-blocked if bolt seal is broken');
        return;
      }
    }

    setSubmitting(true);
    try {
      const event = await foundryWarehouseApi.recordGateEvent({
        depot_id: depotId,
        supplier_id: supplierId,
        vehicle_id: vehicleId.trim(),
        license_plate: licensePlate.trim(),
        direction,
        vehicle_purpose: vehiclePurpose,
        bolt_seal_barcode: boltSealBarcode.trim() || undefined,
        seal_intact: sealIntact,
      });
      setRecentEvents((prev) => [event, ...prev]);
      if (direction === 'GATE_OUT') {
        setBoltSealBarcode('');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gate check-in/out failed');
    } finally {
      setSubmitting(false);
    }
  };

  const getSlotBadge = (slot?: YardBaySlot | string) => {
    switch (slot) {
      case 'INBOUND_STAGING':
        return 'bg-blue-500/20 text-blue-300 border-blue-500/30';
      case 'CROSS_DOCK':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/30';
      case 'OUTBOUND_LOADING':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30';
      case 'MECHANICAL_SERVICE':
        return 'bg-rose-500/20 text-rose-300 border-rose-500/30';
      default:
        return 'bg-slate-700 text-slate-300 border-slate-600';
    }
  };

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-6">
      <div>
        <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
          <span>🚧</span>
          <span>Role 9: ALPR Yard Security Gate & Dynamic Bay Slotting</span>
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Automated license plate logging, dynamic parking slot assignment, and outbound tamper-evident bolt seal verification.
        </p>
      </div>

      <form onSubmit={handleGateEvent} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
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
              ALPR License Plate
            </label>
            <input
              type="text"
              required
              value={licensePlate}
              onChange={(e) => setLicensePlate(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500 uppercase"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Gate Direction
            </label>
            <select
              value={direction}
              onChange={(e) => setDirection(e.target.value as 'GATE_IN' | 'GATE_OUT')}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            >
              <option value="GATE_IN">GATE_IN (Yard Entry)</option>
              <option value="GATE_OUT">GATE_OUT (Yard Departure)</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Vehicle Purpose
            </label>
            <select
              value={vehiclePurpose}
              onChange={(e) => setVehiclePurpose(e.target.value as 'INBOUND_FTL' | 'CROSS_DOCK' | 'OUTBOUND_ROUTE' | 'WORKSHOP')}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            >
              <option value="INBOUND_FTL">INBOUND_FTL (Long-haul)</option>
              <option value="CROSS_DOCK">CROSS_DOCK (Express transfer)</option>
              <option value="OUTBOUND_ROUTE">OUTBOUND_ROUTE (Last-mile)</option>
              <option value="WORKSHOP">WORKSHOP (Mechanical Service)</option>
            </select>
          </div>
        </div>

        {/* Bolt Seal Barcode & Integrity */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 bg-slate-900/50 rounded-xl border border-slate-800">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Tamper-Evident Bolt Seal Barcode {direction === 'GATE_OUT' && <span className="text-rose-400">*Required on Exit</span>}
            </label>
            <input
              type="text"
              value={boltSealBarcode}
              onChange={(e) => setBoltSealBarcode(e.target.value)}
              placeholder="e.g. BOLT-UZ-778899"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-sm font-mono text-slate-100 focus:outline-none focus:border-emerald-500"
            />
          </div>

          <div className="flex items-center gap-3 pt-6">
            <input
              type="checkbox"
              id="seal_intact"
              checked={sealIntact}
              onChange={(e) => setSealIntact(e.target.checked)}
              className="w-4 h-4 rounded text-emerald-500 focus:ring-emerald-400"
            />
            <label htmlFor="seal_intact" className="text-xs font-medium text-slate-200 cursor-pointer">
              Bolt Seal Physical Integrity Verified (Intact & Undamaged)
            </label>
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
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-all shadow-md flex items-center gap-2"
          >
            {submitting ? 'Recording Event...' : direction === 'GATE_IN' ? 'Check-In Vehicle & Slot Bay' : 'Verify Outbound Bolt Seal & Authorize Exit'}
          </button>
        </div>
      </form>

      {/* Dynamic Yard Slotting Live Stream */}
      {recentEvents.length > 0 && (
        <div className="space-y-3 pt-4 border-t border-slate-800">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Recent Yard Gate Events & Bay Slotting
          </h3>

          <div className="divide-y divide-slate-800 rounded-xl bg-slate-900/60 border border-slate-800 overflow-hidden">
            {recentEvents.map((ev, i) => (
              <div key={ev.gate_event_id || i} className="p-3 flex flex-wrap items-center justify-between gap-3 text-xs">
                <div className="flex items-center gap-3">
                  <span
                    className={`px-2 py-0.5 rounded font-mono font-bold ${
                      ev.direction === 'GATE_IN'
                        ? 'bg-blue-500/20 text-blue-300'
                        : 'bg-emerald-500/20 text-emerald-300'
                    }`}
                  >
                    {ev.direction}
                  </span>
                  <span className="font-mono font-bold text-slate-100">{ev.license_plate}</span>
                  <span className="text-slate-400 font-mono">({ev.vehicle_id})</span>
                </div>

                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-1.5">
                    <span className="text-slate-400">Bay:</span>
                    <span className={`px-2 py-0.5 rounded-md font-mono font-bold border text-[11px] ${getSlotBadge(ev.assigned_bay_slot)}`}>
                      {ev.assigned_bay_slot}
                    </span>
                  </div>

                  {ev.bolt_seal_barcode ? (
                    <span className="font-mono text-slate-300 text-[11px]">
                      Seal: {ev.bolt_seal_barcode} {ev.seal_intact ? '✅' : '❌'}
                    </span>
                  ) : (
                    <span className="text-slate-500 text-[11px]">No seal recorded</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
