"use client";

import React, { useState } from "react";
import type { YardGateCheckEvent } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

export default function YardGatePage() {
  const { token } = useAdminToken();

  const [depotId, setDepotId] = useState("depot_tashkent_central");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [vehicleId, setVehicleId] = useState("veh_isuzu_01");
  const [licensePlate, setLicensePlate] = useState("01 A 777 AA");
  const [direction, setDirection] = useState<"GATE_IN" | "GATE_OUT">("GATE_OUT");
  const [boltSealBarcode, setBoltSealBarcode] = useState(`bolt_seal_${Date.now()}`);
  const [sealIntact, setSealIntact] = useState(true);
  const [baySlot, setBaySlot] = useState("OUTBOUND_LOADING");

  const [loading, setLoading] = useState(false);
  const [eventHistory, setEventHistory] = useState<YardGateCheckEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [groundedAlert, setGroundedAlert] = useState<{
    vehicleId: string;
    criticalDefects?: string[];
    message: string;
  } | null>(null);

  const handleRecordGateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setGroundedAlert(null);
    setLoading(true);

    try {
      const res = await foundryApi.recordGateEvent(token, {
        depot_id: depotId.trim(),
        supplier_id: supplierId.trim(),
        vehicle_id: vehicleId.trim(),
        license_plate: licensePlate.trim(),
        direction,
        vehicle_purpose: direction === "GATE_OUT" ? "OUTBOUND_ROUTE" : "INBOUND_FTL",
        bolt_seal_barcode: boltSealBarcode.trim() || undefined,
        seal_intact: sealIntact,
        assigned_bay_slot: baySlot,
      });
      setEventHistory((prev) => [res, ...prev]);
    } catch (err: any) {
      const msg = err instanceof Error ? err.message : "Gate event failed";
      // Inspect for red-tag gate grounding (UA-8)
      if (msg.includes("grounded") || msg.includes("RED_TAG") || msg.includes("red-tagged")) {
        setGroundedAlert({
          vehicleId,
          criticalDefects: ["Brakes below tolerance", "Steering play excursion", "Reefer calibration failure"],
          message: msg,
        });
      } else {
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b pb-4">
        <h1 className="text-xl font-bold tracking-tight text-gray-900">
          Yard Gate Control Tower & ALPR Clearance Simulator
        </h1>
        <p className="text-xs text-gray-500 mt-0.5">
          Role 9 Yard Security: Automated license plate recognition (ALPR), bolt seal integrity verification, and red-tag outbound gate intercept.
        </p>
      </div>

      {/* Red-Tag Intercept Alert Modal / Banner (UA-8) */}
      {groundedAlert && (
        <div className="rounded-xl border-2 border-rose-600 bg-rose-50 p-5 shadow-lg space-y-3">
          <div className="flex items-start gap-3">
            <span className="text-3xl text-rose-600 font-bold">⛔</span>
            <div>
              <div className="inline-flex items-center rounded bg-rose-600 px-2.5 py-0.5 text-xs font-bold text-white uppercase tracking-wider">
                Outbound Gate Intercept Triggered
              </div>
              <h2 className="text-base font-extrabold text-rose-950 mt-1">
                Vehicle Grounded Under Red-Tag Quarantine
              </h2>
              <p className="text-xs text-rose-800 mt-0.5">
                The gate barrier is locked down. Vehicle <span className="font-mono font-bold">{groundedAlert.vehicleId}</span> failed pre-trip safety checks and is prohibited from departing the yard.
              </p>
            </div>
          </div>

          <div className="rounded-lg border border-rose-200 bg-white p-3 text-xs text-rose-900 space-y-1">
            <div className="font-bold">Detected Critical Safety Violations:</div>
            <ul className="list-disc pl-4 space-y-0.5 text-gray-700">
              {groundedAlert.criticalDefects?.map((d, i) => (
                <li key={i}>{d}</li>
              ))}
            </ul>
          </div>

          <div className="text-xs text-rose-800">
            <strong>Mandatory Action:</strong> Direct vehicle immediately to Bay <strong>MECHANICAL_SERVICE</strong>. Clearance requires certified mechanic badge sign-off in DVIR station.
          </div>
        </div>
      )}

      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
          <span className="text-base text-rose-600 font-bold">⚠</span>
          <div>
            <strong>Gate Clearance Denied:</strong> {error}
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Gate Event Simulator Form */}
        <div className="lg:col-span-5 rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <form onSubmit={handleRecordGateEvent} className="space-y-4">
            <h2 className="text-xs font-bold uppercase tracking-wider text-gray-500 border-b pb-2">
              ALPR Gate Simulator
            </h2>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Gate Direction</label>
                <select
                  value={direction}
                  onChange={(e) => setDirection(e.target.value as "GATE_IN" | "GATE_OUT")}
                  className="w-full rounded border px-3 py-1.5 text-xs font-semibold"
                >
                  <option value="GATE_OUT">GATE_OUT (Outbound Departure)</option>
                  <option value="GATE_IN">GATE_IN (Inbound Arrival)</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Assigned Bay Slot</label>
                <select
                  value={baySlot}
                  onChange={(e) => setBaySlot(e.target.value)}
                  className="w-full rounded border px-3 py-1.5 text-xs font-semibold"
                >
                  <option value="OUTBOUND_LOADING">OUTBOUND_LOADING</option>
                  <option value="INBOUND_STAGING">INBOUND_STAGING</option>
                  <option value="CROSS_DOCK">CROSS_DOCK</option>
                  <option value="MECHANICAL_SERVICE">MECHANICAL_SERVICE</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">Vehicle ID</label>
                <input
                  type="text"
                  value={vehicleId}
                  onChange={(e) => setVehicleId(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-700 mb-1">License Plate (ALPR)</label>
                <input
                  type="text"
                  value={licensePlate}
                  onChange={(e) => setLicensePlate(e.target.value)}
                  required
                  className="w-full rounded border px-3 py-1.5 text-xs font-mono font-bold uppercase"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Tamper Bolt Seal Barcode
              </label>
              <input
                type="text"
                value={boltSealBarcode}
                onChange={(e) => setBoltSealBarcode(e.target.value)}
                className="w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>

            <div className="flex items-center justify-between rounded-lg border bg-gray-50 p-3">
              <div>
                <div className="text-xs font-bold text-gray-800">Physical Bolt Seal Intact</div>
                <div className="text-[11px] text-gray-500">Seal is unbroken and matches manifest barcode</div>
              </div>
              <button
                type="button"
                onClick={() => setSealIntact(!sealIntact)}
                className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                  sealIntact ? "bg-emerald-600" : "bg-rose-600"
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    sealIntact ? "translate-x-4" : "translate-x-0"
                  }`}
                />
              </button>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Scanning ALPR & Evaluating..." : "Simulate Gate Sensor Trigger"}
            </button>
          </form>
        </div>

        {/* Live Gate Event Feed */}
        <div className="lg:col-span-7 rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <div className="flex items-center justify-between border-b pb-3">
            <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
              Yard Gate Movement Audit Log
            </span>
            <span className="text-xs text-gray-500">
              {eventHistory.length} events recorded
            </span>
          </div>

          {eventHistory.length === 0 ? (
            <div className="rounded-lg border border-dashed p-8 text-center text-xs text-gray-400">
              No recent gate passages recorded. Run the simulator to record gate check-in or outbound departure.
            </div>
          ) : (
            <div className="space-y-2">
              {eventHistory.map((ev, idx) => (
                <div key={idx} className="rounded-lg border p-3 text-xs flex items-center justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span
                        className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${
                          ev.direction === "GATE_OUT"
                            ? "bg-blue-100 text-blue-800"
                            : "bg-emerald-100 text-emerald-800"
                        }`}
                      >
                        {ev.direction}
                      </span>
                      <span className="font-mono font-bold text-gray-900">{ev.license_plate}</span>
                      <span className="text-gray-400">({ev.vehicle_id})</span>
                    </div>
                    <div className="text-gray-500">
                      Bay: <span className="font-semibold text-gray-700">{ev.assigned_bay_slot}</span> | Seal:{" "}
                      <span className={ev.seal_intact ? "text-emerald-700 font-semibold" : "text-rose-700 font-bold"}>
                        {ev.seal_intact ? "INTACT" : "COMPROMISED"}
                      </span>
                    </div>
                  </div>

                  <div className="text-right">
                    <CopyableId id={ev.gate_event_id} label="Event" />
                    <div className="text-[10px] text-gray-400 mt-1">
                      {ev.created_at ? new Date(ev.created_at).toLocaleTimeString() : "Just now"}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
