'use client';

import React, { useState } from 'react';
import type { YardGateCheckEvent, YardBaySlot } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';
import { Truck, Shield, Barcode, CheckCircle2, AlertOctagon, ArrowDownRight, ArrowUpRight, AlertCircle } from 'lucide-react';

interface YardGateCheckScreenProps {
  depotId?: string;
  supplierId?: string;
}

const SAMPLE_PLATES = [
  { plate: '01 A 777 AA', vehicle: 'trk_isuzu_04', purpose: 'INBOUND_FTL' as const },
  { plate: '01 100 UZB', vehicle: 'trk_hyundai_12', purpose: 'OUTBOUND_ROUTE' as const },
  { plate: '01 442 TAS', vehicle: 'trk_fuso_08', purpose: 'CROSS_DOCK' as const },
];

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
        setError('Outbound Security Gate: Outbound gate departure is hard-blocked if bolt seal is broken or tampered');
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
        return 'bg-blue-500/10 text-blue-300 border-blue-500/30';
      case 'CROSS_DOCK':
        return 'bg-amber-500/10 text-amber-300 border-amber-500/30';
      case 'OUTBOUND_LOADING':
        return 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30';
      case 'MECHANICAL_SERVICE':
        return 'bg-rose-500/10 text-rose-300 border-rose-500/30';
      default:
        return 'bg-neutral-800 text-neutral-300 border-neutral-700';
    }
  };

  return (
    <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 shadow-xl space-y-6 backdrop-blur-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-neutral-800">
        <div>
          <div className="flex items-center gap-2">
            <Truck size={20} className="text-blue-400" />
            <h2 className="text-base font-bold text-neutral-100 uppercase tracking-wide">
              Role 9: ALPR Yard Security Gate & Dynamic Bay Slotting
            </h2>
          </div>
          <p className="text-xs text-neutral-400 mt-0.5">
            Automated license plate logging, dynamic parking slot assignment, and outbound tamper-evident bolt seal verification.
          </p>
        </div>

        {/* ALPR Quick Scan Presets */}
        <div className="flex items-center gap-2">
          <span className="text-[11px] font-mono text-neutral-500">ALPR OCR:</span>
          {SAMPLE_PLATES.map((p) => (
            <button
              key={p.plate}
              type="button"
              onClick={() => {
                setLicensePlate(p.plate);
                setVehicleId(p.vehicle);
                setVehiclePurpose(p.purpose);
              }}
              className="px-2 py-0.5 rounded text-[11px] font-mono bg-neutral-800 hover:bg-neutral-750 text-neutral-200 border border-neutral-700"
            >
              {p.plate}
            </button>
          ))}
        </div>
      </div>

      <form onSubmit={handleGateEvent} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Vehicle ID
            </label>
            <input
              type="text"
              required
              value={vehicleId}
              onChange={(e) => setVehicleId(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-blue-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              ALPR License Plate
            </label>
            <input
              type="text"
              required
              value={licensePlate}
              onChange={(e) => setLicensePlate(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-blue-500 uppercase"
            />
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Gate Direction
            </label>
            <select
              value={direction}
              onChange={(e) => setDirection(e.target.value as 'GATE_IN' | 'GATE_OUT')}
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-blue-500"
            >
              <option value="GATE_IN">GATE_IN (Yard Entry)</option>
              <option value="GATE_OUT">GATE_OUT (Yard Departure)</option>
            </select>
          </div>

          <div>
            <label className="block text-[11px] font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Vehicle Purpose
            </label>
            <select
              value={vehiclePurpose}
              onChange={(e) =>
                setVehiclePurpose(
                  e.target.value as 'INBOUND_FTL' | 'CROSS_DOCK' | 'OUTBOUND_ROUTE' | 'WORKSHOP',
                )
              }
              className="w-full bg-neutral-950 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-blue-500"
            >
              <option value="INBOUND_FTL">INBOUND_FTL (Long-haul)</option>
              <option value="CROSS_DOCK">CROSS_DOCK (Express transfer)</option>
              <option value="OUTBOUND_ROUTE">OUTBOUND_ROUTE (Last-mile)</option>
              <option value="WORKSHOP">WORKSHOP (Mechanical Service)</option>
            </select>
          </div>
        </div>

        {/* Bolt Seal Barcode & Integrity */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 bg-neutral-950/70 rounded-xl border border-neutral-800">
          <div>
            <label className="block text-[11px] font-semibold text-neutral-300 uppercase tracking-wider mb-1">
              Tamper-Evident Bolt Seal Barcode {direction === 'GATE_OUT' && <span className="text-rose-400 font-bold">*Required on Exit</span>}
            </label>
            <div className="relative">
              <input
                type="text"
                value={boltSealBarcode}
                onChange={(e) => setBoltSealBarcode(e.target.value)}
                placeholder="e.g. BOLT-UZ-778899"
                className="w-full bg-neutral-900 border border-neutral-750 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-blue-500 pr-9"
              />
              <Barcode size={16} className="absolute right-3 top-2.5 text-neutral-500" />
            </div>
          </div>

          <div className="flex items-center gap-3 pt-5">
            <input
              type="checkbox"
              id="seal_intact"
              checked={sealIntact}
              onChange={(e) => setSealIntact(e.target.checked)}
              className="w-4 h-4 rounded text-emerald-500 focus:ring-emerald-400 bg-neutral-900 border-neutral-700"
            />
            <label htmlFor="seal_intact" className="text-xs font-medium text-neutral-200 cursor-pointer">
              Bolt Seal Physical Integrity Verified (Intact & Undamaged)
            </label>
          </div>
        </div>

        {error && (
          <div className="p-3 bg-rose-950/60 border border-rose-800 rounded-xl text-xs text-rose-200 flex items-center gap-2">
            <AlertCircle size={15} className="shrink-0 text-rose-400" />
            <span>{error}</span>
          </div>
        )}

        <div className="flex justify-end pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="px-6 py-2.5 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-40 transition-all shadow flex items-center gap-2"
          >
            {submitting ? (
              'Recording Event...'
            ) : direction === 'GATE_IN' ? (
              <>
                <ArrowDownRight size={15} />
                <span>Check-In Vehicle & Slot Bay</span>
              </>
            ) : (
              <>
                <ArrowUpRight size={15} />
                <span>Verify Outbound Bolt Seal & Authorize Exit</span>
              </>
            )}
          </button>
        </div>
      </form>

      {/* Dynamic Yard Slotting Live Stream */}
      {recentEvents.length > 0 && (
        <div className="space-y-3 pt-4 border-t border-neutral-800">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
            Recent Yard Gate Events & Bay Slotting
          </h3>

          <div className="divide-y divide-neutral-800 rounded-xl bg-neutral-950/60 border border-neutral-800 overflow-hidden">
            {recentEvents.map((ev, i) => (
              <div key={ev.gate_event_id || i} className="p-3.5 flex flex-wrap items-center justify-between gap-3 text-xs">
                <div className="flex items-center gap-3">
                  <span
                    className={`px-2 py-0.5 rounded font-mono font-bold text-[10px] ${
                      ev.direction === 'GATE_IN'
                        ? 'bg-blue-500/10 text-blue-300 border border-blue-500/30'
                        : 'bg-emerald-500/10 text-emerald-300 border border-emerald-500/30'
                    }`}
                  >
                    {ev.direction}
                  </span>
                  <span className="font-mono font-bold text-neutral-100">{ev.license_plate}</span>
                  <span className="text-neutral-400 font-mono text-[11px]">({ev.vehicle_id})</span>
                </div>

                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-1.5">
                    <span className="text-neutral-500 font-mono text-[11px]">Bay:</span>
                    <span className={`px-2 py-0.5 rounded font-mono font-bold border text-[11px] ${getSlotBadge(ev.assigned_bay_slot)}`}>
                      {ev.assigned_bay_slot}
                    </span>
                  </div>

                  {ev.bolt_seal_barcode ? (
                    <span className="font-mono text-neutral-300 text-[11px]">
                      Seal: {ev.bolt_seal_barcode} {ev.seal_intact ? '✅' : '❌'}
                    </span>
                  ) : (
                    <span className="text-neutral-500 text-[11px] font-mono">No seal recorded</span>
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
