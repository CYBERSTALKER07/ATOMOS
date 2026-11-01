"use client";

import React, { useState } from "react";
import type { FleetDVIRInspection, DispatchCheckResponse } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

export default function DVIRFleetPage() {
  const { token } = useAdminToken();

  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [vehicleId, setVehicleId] = useState("veh_isuzu_01");
  const [driverId, setDriverId] = useState("drv_tashkent_404");
  const [odometerKm, setOdometerKm] = useState(145200);
  const [requiresColdChain, setRequiresColdChain] = useState(true);
  const [reeferTemp, setReeferTemp] = useState(3.5);

  // 8-Point Checklist State
  const [checklist, setChecklist] = useState({
    steering_and_horn_ok: true,
    service_and_parking_brakes_ok: true,
    tires_tread_and_pressure_ok: true,
    lights_and_hazards_ok: true,
    wipers_and_fluids_ok: true,
    emergency_kit_and_extinguisher_ok: true,
    cargo_box_sanitation_ok: true,
    reefer_temp_calibrated_ok: true,
  });

  const [loading, setLoading] = useState(false);
  const [inspectionResult, setInspectionResult] = useState<FleetDVIRInspection | null>(null);
  const [dispatchStatus, setDispatchStatus] = useState<DispatchCheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const togglePoint = (key: keyof typeof checklist) => {
    setChecklist((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const handleSubmitInspection = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setSuccessMsg(null);
    setLoading(true);

    try {
      const res = await foundryApi.submitDVIR(token, {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        vehicle_id: vehicleId.trim(),
        driver_id: driverId.trim(),
        inspection_type: "PRE_TRIP",
        checklist,
        odometer_km: odometerKm,
        reefer_temp_celsius: requiresColdChain ? reeferTemp : 0,
        requires_cold_chain: requiresColdChain,
      });
      setInspectionResult(res);
      if (res.status === "RED_TAG_QUARANTINE") {
        setError(`CRITICAL DEFECT DETECTED: Vehicle grounded under RED_TAG_QUARANTINE. Work Order: ${res.maintenance_work_order_id || "WO-PENDING"}`);
      } else {
        setSuccessMsg(`Inspection complete. Vehicle ${res.vehicle_id} is DISPATCH_AUTHORIZED.`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "DVIR inspection failed");
    } finally {
      setLoading(false);
    }
  };

  const handleRunDispatchCheck = async () => {
    if (!token || !vehicleId.trim()) return;
    setError(null);
    setLoading(true);

    try {
      const res = await foundryApi.dispatchCheck(token, vehicleId.trim());
      setDispatchStatus(res);
    } catch (err: any) {
      setError(err instanceof Error ? err.message : "Dispatch check failed");
    } finally {
      setLoading(false);
    }
  };

  const CHECKLIST_ITEMS: { key: keyof typeof checklist; label: string; desc: string; critical: boolean }[] = [
    { key: "service_and_parking_brakes_ok", label: "Service & Parking Brakes", desc: "No brake air pressure leaks, brake pads >= 6mm", critical: true },
    { key: "steering_and_horn_ok", label: "Steering & Electric Horn", desc: "Steering play <= 10 degrees, audible warning horn functional", critical: true },
    { key: "tires_tread_and_pressure_ok", label: "Tires Tread & Cold Pressure", desc: "Tread depth >= 3mm, cold psi within manufacturer spec", critical: true },
    { key: "lights_and_hazards_ok", label: "Headlights, Signals & Hazards", desc: "High/low beams, turn signals, and hazard flashers operational", critical: false },
    { key: "wipers_and_fluids_ok", label: "Wipers, Blades & Washer Fluid", desc: "Clear wipe pattern, windshield fluid topped up", critical: false },
    { key: "emergency_kit_and_extinguisher_ok", label: "Emergency Kit & Fire Extinguisher", desc: "Fire extinguisher charged, high-vis vests and triangle present", critical: true },
    { key: "cargo_box_sanitation_ok", label: "Cargo Box Sanitation & Odor", desc: "Clean cargo interior, free from chemicals or biological contamination", critical: false },
    { key: "reefer_temp_calibrated_ok", label: "Reefer Thermal Calibration (Cold-Chain)", desc: "Reefer unit holds setpoint within +/- 0.5C tolerance", critical: true },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Fleet Safety & DVIR Pre-Trip Walkaround Cockpit
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Role 9 Fleet & Yard Management: 8-point digital walkaround, red-tag quarantine enforcement, and dispatch authorization.
          </p>
        </div>

        <button
          type="button"
          onClick={handleRunDispatchCheck}
          disabled={loading || !token}
          className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-sm hover:bg-gray-50 disabled:opacity-50"
        >
          {loading ? "Checking..." : "Inspect Dispatch Status"}
        </button>
      </div>

      {/* Notifications */}
      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 flex items-start gap-2">
          <span className="text-base text-rose-600 font-bold">⛔</span>
          <div>{error}</div>
        </div>
      )}
      {successMsg && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800 flex items-start gap-2">
          <span className="text-base text-emerald-600 font-bold">✓</span>
          <div>{successMsg}</div>
        </div>
      )}

      {/* Dispatch Check Result Banner */}
      {dispatchStatus && (
        <div className={`rounded-xl border p-4 shadow-sm flex items-center justify-between ${
          dispatchStatus.dispatch_authorized ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50"
        }`}>
          <div>
            <div className="text-xs font-bold uppercase tracking-wider text-gray-700">
              Vehicle Dispatch Clearance: <span className="font-mono">{dispatchStatus.vehicle_id}</span>
            </div>
            <div className={`text-sm font-extrabold mt-0.5 ${
              dispatchStatus.dispatch_authorized ? "text-emerald-800" : "text-rose-800"
            }`}>
              {dispatchStatus.dispatch_authorized ? "DISPATCH_AUTHORIZED" : "RED_TAG_GROUNDED (PROHIBITED FROM EXIT)"}
            </div>
          </div>

          <div className="text-right">
            <span className="text-xs text-gray-500">Status Check</span>
            <div className="text-xs font-semibold text-gray-800">
              {dispatchStatus.dispatch_authorized ? "CLEAR FOR HIGHWAY" : "IMMEDIATE REPAIRS MANDATORY"}
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Inspection Form */}
        <div className="lg:col-span-8 rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <form onSubmit={handleSubmitInspection} className="space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
                Pre-Trip Digital Inspection Submission
              </span>
              <span className="rounded bg-indigo-50 px-2 py-0.5 text-[10px] font-bold text-indigo-700">
                8-Point Safety Protocol
              </span>
            </div>

            {/* Vehicle & Driver Details */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Vehicle ID</label>
                <input
                  type="text"
                  value={vehicleId}
                  onChange={(e) => setVehicleId(e.target.value)}
                  required
                  className="w-full rounded border px-2.5 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Driver ID</label>
                <input
                  type="text"
                  value={driverId}
                  onChange={(e) => setDriverId(e.target.value)}
                  required
                  className="w-full rounded border px-2.5 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Odometer (km)</label>
                <input
                  type="number"
                  value={odometerKm}
                  onChange={(e) => setOdometerKm(Number(e.target.value))}
                  required
                  className="w-full rounded border px-2.5 py-1.5 text-xs font-mono"
                />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Reefer Temp (°C)</label>
                <input
                  type="number"
                  step="0.1"
                  value={reeferTemp}
                  onChange={(e) => setReeferTemp(Number(e.target.value))}
                  className="w-full rounded border px-2.5 py-1.5 text-xs font-mono"
                />
              </div>
            </div>

            {/* 8-Point Checklist Items */}
            <div className="space-y-2 pt-2">
              <span className="text-xs font-bold text-gray-800">Safety Walkaround Checklist</span>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                {CHECKLIST_ITEMS.map((item) => {
                  const isChecked = checklist[item.key];
                  return (
                    <div
                      key={item.key}
                      onClick={() => togglePoint(item.key)}
                      className={`cursor-pointer rounded-lg border p-3 transition-colors ${
                        isChecked
                          ? "border-emerald-200 bg-emerald-50/40 hover:bg-emerald-50"
                          : "border-rose-200 bg-rose-50/40 hover:bg-rose-50"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            checked={isChecked}
                            onChange={() => {}}
                            className="h-4 w-4 rounded text-indigo-600 focus:ring-indigo-500"
                          />
                          <span className={`text-xs font-bold ${isChecked ? "text-gray-900" : "text-rose-900"}`}>
                            {item.label}
                          </span>
                        </div>
                        {item.critical && (
                          <span className="rounded bg-rose-100 px-1.5 py-0.2 text-[9px] font-bold text-rose-700">
                            CRITICAL
                          </span>
                        )}
                      </div>
                      <p className="text-[11px] text-gray-500 mt-1 pl-6">{item.desc}</p>
                    </div>
                  );
                })}
              </div>
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Recording DVIR..." : "Submit Pre-Trip DVIR Inspection"}
            </button>
          </form>
        </div>

        {/* Sidebar Info & Red Tag Policy */}
        <div className="lg:col-span-4 space-y-4">
          <div className="rounded-xl border bg-gray-50 p-5 space-y-3">
            <h3 className="text-xs font-bold text-gray-900 uppercase tracking-wider">
              DVIR Red-Tag Mandate
            </h3>
            <p className="text-xs text-gray-600">
              If any critical defect (brakes, steering, tires, emergency kit, or cold-chain calibration) fails inspection:
            </p>
            <ul className="text-xs text-gray-600 space-y-1.5 list-disc pl-4">
              <li>Vehicle is stamped with <span className="font-semibold text-rose-600">RED_TAG_QUARANTINE</span>.</li>
              <li>Tractor/trailer is automatically assigned to Yard Bay <span className="font-mono font-semibold">MECHANICAL_SERVICE</span>.</li>
              <li>Outbound yard gate barrier will physically refuse to lift on ALPR scan (Guard #12).</li>
              <li>Work order must be signed off by a certified fleet mechanic before re-authorization.</li>
            </ul>
          </div>

          {inspectionResult && (
            <div className="rounded-xl border bg-white p-4 shadow-sm space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-gray-500 uppercase">Inspection Record</span>
                <CopyableId id={inspectionResult.inspection_id} label="Inspection" />
              </div>
              <div className="text-xs text-gray-700 space-y-1">
                <div>Vehicle: <span className="font-mono font-semibold">{inspectionResult.vehicle_id}</span></div>
                <div>Status: <span className="font-bold">{inspectionResult.status}</span></div>
                {inspectionResult.maintenance_work_order_id && (
                  <div>Work Order: <span className="font-mono font-bold text-rose-600">{inspectionResult.maintenance_work_order_id}</span></div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
