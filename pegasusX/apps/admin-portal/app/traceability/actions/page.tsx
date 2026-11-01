"use client";

import React, { useState } from "react";
import type { ActionExecutionRecord } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

type ActionTypeKey =
  | "ACTION_RELEASE_PICK_WAVE"
  | "ACTION_SEAL_VEHICLE_MANIFEST"
  | "ACTION_START_DRIVER_ROUTE"
  | "ACTION_COMPLETE_DOORSTEP_EPOD"
  | "ACTION_BATCH_SUPPLIER_PAYOUT";

interface KineticActionSchema {
  label: string;
  description: string;
  stepName: string;
  defaultParams: Record<string, any>;
}

const ACTION_SCHEMAS: Record<ActionTypeKey, KineticActionSchema> = {
  ACTION_RELEASE_PICK_WAVE: {
    label: "Release Pick Wave (Warehouse)",
    description: "Evaluates fiscal readiness (Guard #3) and FX corridor volatility (Guard #9) before releasing warehouse picking waves.",
    stepName: "Step 5: Pick Wave Release",
    defaultParams: {
      pkcs12_cert_hours_remaining: 120,
      all_skus_ikpu_valid: true,
      fx_policy: "FLOAT_WITH_CORRIDOR_CAP",
      fx_delta_bps: 45,
      fx_corridor_cap_bps: 100,
    },
  },
  ACTION_SEAL_VEHICLE_MANIFEST: {
    label: "Seal Vehicle Manifest (Yard & Fleet)",
    description: "Evaluates 3D crushability inversion (Guard #6) and DVIR pre-trip authorization (Guard #12) before dispatching sealed trailers.",
    stepName: "Step 6: Manifest Sealing",
    defaultParams: {
      vehicle_id: "veh_isuzu_01",
      has_crushability_inversion: false,
    },
  },
  ACTION_START_DRIVER_ROUTE: {
    label: "Start Driver Route (Logistics)",
    description: "Evaluates driver pre-trip DVIR signoff (Guard #11) before allowing route departure from depot yard.",
    stepName: "Step 7: Route Departure",
    defaultParams: {
      driver_id: "drv_tashkent_404",
      route_id: "route_daily_center_01",
      pre_trip_completed: true,
    },
  },
  ACTION_COMPLETE_DOORSTEP_EPOD: {
    label: "Complete Doorstep EPoD (Delivery)",
    description: "Evaluates 50m geofence fence radius (Guard #4) and tender proof reconciliation before marking order delivered.",
    stepName: "Step 8: Doorstep Handover",
    defaultParams: {
      order_id: "ord_retailer_9912",
      geofence_verified: true,
      cash_collected_minor: 25000000,
    },
  },
  ACTION_BATCH_SUPPLIER_PAYOUT: {
    label: "Batch Supplier Payout (Treasury)",
    description: "Evaluates fiscal receipt issuance (Guard #10) and escrow verification before releasing settlement funds to supplier accounts.",
    stepName: "Step 10: Financial Settlement",
    defaultParams: {
      batch_id: "payout_batch_2026_w44",
      all_slices_fiscalized: true,
    },
  },
};

export default function KineticActionsPage() {
  const { token } = useAdminToken();

  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("sup_tashkent_fmcg");
  const [actionType, setActionType] = useState<ActionTypeKey>("ACTION_RELEASE_PICK_WAVE");
  const [actorId, setActorId] = useState("admin_ops_cockpit");
  const [actorRole, setActorRole] = useState("PLATFORM_ADMIN");
  const [scenarioId, setScenarioId] = useState("");
  const [idempotencyKey, setIdempotencyKey] = useState(`idem_${Date.now()}`);

  // Dynamic parameters state based on actionType
  const [params, setParams] = useState<Record<string, any>>(
    ACTION_SCHEMAS.ACTION_RELEASE_PICK_WAVE.defaultParams
  );

  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ActionExecutionRecord | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleActionTypeChange = (newType: ActionTypeKey) => {
    setActionType(newType);
    setParams(ACTION_SCHEMAS[newType].defaultParams);
    setResult(null);
    setError(null);
  };

  const handleGenerateKey = () => {
    setIdempotencyKey(`idem_${Date.now()}_${Math.random().toString(36).substring(2, 7)}`);
  };

  const updateParam = (key: string, val: any) => {
    setParams((prev) => ({ ...prev, [key]: val }));
  };

  const handleExecute = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
    setError(null);
    setLoading(true);

    try {
      const res = await foundryApi.executeAction(token, {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        action_type: actionType,
        actor_id: actorId.trim(),
        actor_role: actorRole.trim(),
        scenario_id: scenarioId.trim() || undefined,
        idempotency_key: idempotencyKey.trim(),
        parameters: params,
      });
      setResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Kinetic action execution failed");
    } finally {
      setLoading(false);
    }
  };

  // Plain-language remediation advice mapper (UA-10)
  const getRubricRemediation = (ruleName: string, reason?: string) => {
    switch (ruleName) {
      case "Guard3_PreWavePKCS12CertWindow":
        return "Action required: Renew the organization's Soliq PKCS#12 e-signature certificate immediately. Minimum 72 hours validity is mandatory before releasing pick waves to prevent invoice fiscalization lockout.";
      case "Guard3_PreWaveIKPUValidation":
        return "Action required: Inspect catalog items for missing or unverified 17-digit MXIK/IKPU tax codes. Update master catalog taxonomy before wave dispatch.";
      case "Guard9_FXVolatilityCorridor":
        return "Action required: Central Bank FX rate fluctuation exceeds the corridor cap. Prompt retailer for reconfirmation of pricing terms before stock allocation.";
      case "Guard12_VehicleDVIRAuthorization":
        return "Action required: Assigned vehicle has failed pre-trip safety inspection or is red-tagged. Reassign trailer to a certified road-worthy tractor in Yard Ops.";
      case "Guard6_PalletCrushability":
        return "Action required: High-density heavy pallets are staged above crushable carton goods. Re-run 3D load optimizer before sealing door manifest.";
      case "Guard11_DriverPreTripCompleted":
        return "Action required: Driver has not submitted mandatory 8-point DVIR inspection. Route start is legally blocked until pre-trip is recorded.";
      case "Guard4_EPoDGeofenceCheck":
        return "Action required: Device GPS position is outside the 50-meter retail geofence perimeter. Driver must be physically on-site to record doorstep handover.";
      case "Guard10_SliceFiscalReadiness":
        return "Action required: Settlement slices contain unfiscalized deliveries. Complete Soliq e-factura issuance prior to batch disbursement.";
      default:
        return reason || "Pre-execution rubric check failed. Review parameter constraints.";
    }
  };

  const activeSchema = ACTION_SCHEMAS[actionType];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="border-b pb-4">
        <h1 className="text-xl font-bold tracking-tight text-gray-900">
          Kinetic Action Pre-Execution Guard Cockpit
        </h1>
        <p className="text-xs text-gray-500 mt-0.5">
          Evaluate immutable pre-execution rubrics across the 10-Step Master Operational Cascade before applying physical or financial mutations.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Form Panel */}
        <div className="lg:col-span-6 space-y-4">
          <form onSubmit={handleExecute} className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
                Operational Action Contract
              </span>
              <span className="rounded bg-indigo-50 px-2 py-0.5 text-[10px] font-semibold text-indigo-700">
                {activeSchema.stepName}
              </span>
            </div>

            {/* Action Type Selector */}
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">
                Kinetic Action Contract
              </label>
              <select
                value={actionType}
                onChange={(e) => handleActionTypeChange(e.target.value as ActionTypeKey)}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              >
                {(Object.keys(ACTION_SCHEMAS) as ActionTypeKey[]).map((key) => (
                  <option key={key} value={key}>
                    {ACTION_SCHEMAS[key].label}
                  </option>
                ))}
              </select>
              <p className="text-[11px] text-gray-500 mt-1">{activeSchema.description}</p>
            </div>

            {/* Core Identifiers */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Cell ID</label>
                <input
                  type="text"
                  value={cellId}
                  onChange={(e) => setCellId(e.target.value)}
                  required
                  className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Supplier ID</label>
                <input
                  type="text"
                  value={supplierId}
                  onChange={(e) => setSupplierId(e.target.value)}
                  required
                  className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Actor ID</label>
                <input
                  type="text"
                  value={actorId}
                  onChange={(e) => setActorId(e.target.value)}
                  required
                  className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-gray-600 mb-1">Actor Role</label>
                <select
                  value={actorRole}
                  onChange={(e) => setActorRole(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-medium focus:border-indigo-500 focus:outline-none"
                >
                  <option value="PLATFORM_ADMIN">PLATFORM_ADMIN</option>
                  <option value="WAREHOUSE_DISPATCHER">WAREHOUSE_DISPATCHER</option>
                  <option value="YARD_CONTROLLER">YARD_CONTROLLER</option>
                  <option value="TREASURY_OFFICER">TREASURY_OFFICER</option>
                </select>
              </div>
            </div>

            {/* Scenario ID (Optional CoW branch) */}
            <div>
              <label className="block text-[11px] font-medium text-gray-600 mb-1">
                Scenario ID (Optional CoW Simulation Branch)
              </label>
              <input
                type="text"
                placeholder="Leave blank for Live Production execution"
                value={scenarioId}
                onChange={(e) => setScenarioId(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono placeholder:text-gray-400 focus:border-indigo-500 focus:outline-none"
              />
            </div>

            {/* Idempotency Key */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-[11px] font-medium text-gray-600">Idempotency Key</label>
                <button
                  type="button"
                  onClick={handleGenerateKey}
                  className="text-[10px] font-semibold text-indigo-600 hover:text-indigo-800"
                >
                  Regenerate
                </button>
              </div>
              <input
                type="text"
                value={idempotencyKey}
                onChange={(e) => setIdempotencyKey(e.target.value)}
                required
                className="w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono focus:border-indigo-500 focus:outline-none"
              />
            </div>

            {/* Schema-Guided Parameters Section (UA-1) */}
            <div className="rounded-lg border bg-gray-50 p-4 space-y-3">
              <span className="text-xs font-bold text-gray-800">
                Action Parameters ({actionType})
              </span>

              {actionType === "ACTION_RELEASE_PICK_WAVE" && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">
                      PKCS#12 Cert Remaining Hours
                    </label>
                    <div className="flex items-center gap-2">
                      <input
                        type="number"
                        min="0"
                        max="8760"
                        value={params.pkcs12_cert_hours_remaining ?? 120}
                        onChange={(e) => updateParam("pkcs12_cert_hours_remaining", Number(e.target.value))}
                        className="w-32 rounded border border-gray-300 px-2.5 py-1.5 text-xs font-semibold focus:outline-none"
                      />
                      <span className={`text-[11px] font-semibold ${
                        (params.pkcs12_cert_hours_remaining ?? 0) >= 72 ? "text-emerald-600" : "text-rose-600"
                      }`}>
                        {(params.pkcs12_cert_hours_remaining ?? 0) >= 72
                          ? "✓ Exceeds 72h minimum window"
                          : "⚠ Warning: < 72h will trigger Guard #3 rejection"}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between py-1 border-t">
                    <div>
                      <div className="text-xs font-medium text-gray-800">All SKUs IKPU Valid</div>
                      <div className="text-[11px] text-gray-500">17-digit Soliq commodity taxonomy code verified</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => updateParam("all_skus_ikpu_valid", !params.all_skus_ikpu_valid)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                        params.all_skus_ikpu_valid ? "bg-indigo-600" : "bg-gray-200"
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                          params.all_skus_ikpu_valid ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>

                  <div className="grid grid-cols-2 gap-3 pt-2 border-t">
                    <div>
                      <label className="block text-[11px] font-medium text-gray-700 mb-1">
                        FX Delta (bps)
                      </label>
                      <input
                        type="number"
                        value={params.fx_delta_bps ?? 45}
                        onChange={(e) => updateParam("fx_delta_bps", Number(e.target.value))}
                        className="w-full rounded border border-gray-300 px-2.5 py-1 text-xs"
                      />
                    </div>
                    <div>
                      <label className="block text-[11px] font-medium text-gray-700 mb-1">
                        FX Corridor Cap (bps)
                      </label>
                      <input
                        type="number"
                        value={params.fx_corridor_cap_bps ?? 100}
                        onChange={(e) => updateParam("fx_corridor_cap_bps", Number(e.target.value))}
                        className="w-full rounded border border-gray-300 px-2.5 py-1 text-xs"
                      />
                    </div>
                  </div>
                </>
              )}

              {actionType === "ACTION_SEAL_VEHICLE_MANIFEST" && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">Vehicle ID</label>
                    <input
                      type="text"
                      value={params.vehicle_id ?? ""}
                      onChange={(e) => updateParam("vehicle_id", e.target.value)}
                      className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                    />
                  </div>

                  <div className="flex items-center justify-between py-1 border-t">
                    <div>
                      <div className="text-xs font-medium text-gray-800">Crushability Inversion</div>
                      <div className="text-[11px] text-gray-500">Heavy pallet staged on top of crushable cartons</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => updateParam("has_crushability_inversion", !params.has_crushability_inversion)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                        params.has_crushability_inversion ? "bg-rose-600" : "bg-gray-200"
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                          params.has_crushability_inversion ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>
                </>
              )}

              {actionType === "ACTION_START_DRIVER_ROUTE" && (
                <>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] font-medium text-gray-700 mb-1">Driver ID</label>
                      <input
                        type="text"
                        value={params.driver_id ?? ""}
                        onChange={(e) => updateParam("driver_id", e.target.value)}
                        className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-[11px] font-medium text-gray-700 mb-1">Route ID</label>
                      <input
                        type="text"
                        value={params.route_id ?? ""}
                        onChange={(e) => updateParam("route_id", e.target.value)}
                        className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                      />
                    </div>
                  </div>

                  <div className="flex items-center justify-between py-1 border-t">
                    <div>
                      <div className="text-xs font-medium text-gray-800">Pre-Trip DVIR Completed</div>
                      <div className="text-[11px] text-gray-500">Driver submitted 8-point walkaround check</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => updateParam("pre_trip_completed", !params.pre_trip_completed)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                        params.pre_trip_completed ? "bg-indigo-600" : "bg-gray-200"
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                          params.pre_trip_completed ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>
                </>
              )}

              {actionType === "ACTION_COMPLETE_DOORSTEP_EPOD" && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">Order ID</label>
                    <input
                      type="text"
                      value={params.order_id ?? ""}
                      onChange={(e) => updateParam("order_id", e.target.value)}
                      className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                    />
                  </div>

                  <div className="flex items-center justify-between py-1 border-t">
                    <div>
                      <div className="text-xs font-medium text-gray-800">GPS Geofence Verified</div>
                      <div className="text-[11px] text-gray-500">Within 50m of retailer shop delivery pin</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => updateParam("geofence_verified", !params.geofence_verified)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                        params.geofence_verified ? "bg-indigo-600" : "bg-gray-200"
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                          params.geofence_verified ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>

                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">Cash Collected (tiyins)</label>
                    <input
                      type="number"
                      value={params.cash_collected_minor ?? 0}
                      onChange={(e) => updateParam("cash_collected_minor", Number(e.target.value))}
                      className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                    />
                  </div>
                </>
              )}

              {actionType === "ACTION_BATCH_SUPPLIER_PAYOUT" && (
                <>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1">Payout Batch ID</label>
                    <input
                      type="text"
                      value={params.batch_id ?? ""}
                      onChange={(e) => updateParam("batch_id", e.target.value)}
                      className="w-full rounded border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
                    />
                  </div>

                  <div className="flex items-center justify-between py-1 border-t">
                    <div>
                      <div className="text-xs font-medium text-gray-800">All Delivery Slices Fiscalized</div>
                      <div className="text-[11px] text-gray-500">Soliq receipt registered for every delivery line</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => updateParam("all_slices_fiscalized", !params.all_slices_fiscalized)}
                      className={`relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                        params.all_slices_fiscalized ? "bg-indigo-600" : "bg-gray-200"
                      }`}
                    >
                      <span
                        className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                          params.all_slices_fiscalized ? "translate-x-4" : "translate-x-0"
                        }`}
                      />
                    </button>
                  </div>
                </>
              )}
            </div>

            <button
              type="submit"
              disabled={loading || !token}
              className="w-full rounded-lg bg-indigo-600 py-2.5 text-xs font-bold text-white shadow hover:bg-indigo-700 disabled:opacity-50"
            >
              {loading ? "Evaluating Rubrics & Executing..." : "Execute Kinetic Action"}
            </button>
          </form>
        </div>

        {/* Audit & Rubric Response Panel */}
        <div className="lg:col-span-6 space-y-4">
          {error && (
            <div className="rounded-xl border border-rose-200 bg-rose-50 p-4">
              <div className="flex items-start gap-2">
                <span className="text-rose-600 text-lg">⚠</span>
                <div>
                  <h3 className="text-xs font-bold text-rose-900">Pre-Execution Rubric Rejection</h3>
                  <p className="text-xs text-rose-700 mt-0.5">{error}</p>
                </div>
              </div>
            </div>
          )}

          {!result && !error && (
            <div className="rounded-xl border border-dashed bg-gray-50/50 p-8 text-center">
              <div className="text-sm font-semibold text-gray-500">
                Awaiting Kinetic Action Execution
              </div>
              <p className="text-xs text-gray-400 mt-1 max-w-sm mx-auto">
                Select an operational contract, tune parameters, and execute to inspect the pre-condition evaluation report and immutable state mutation audit log.
              </p>
            </div>
          )}

          {result && (
            <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
              <div className="flex items-center justify-between border-b pb-3">
                <div>
                  <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
                    Execution Outcome
                  </span>
                  <div className="flex items-center gap-2 mt-1">
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-bold ${
                        result.status === "COMMITTED"
                          ? "bg-emerald-100 text-emerald-800"
                          : result.status === "SIMULATED_IN_SCENARIO"
                          ? "bg-blue-100 text-blue-800"
                          : "bg-rose-100 text-rose-800"
                      }`}
                    >
                      {result.status}
                    </span>
                    <span className="text-xs text-gray-500">
                      {new Date(result.executed_at).toLocaleTimeString()}
                    </span>
                  </div>
                </div>

                <CopyableId id={result.action_execution_id} label="Execution ID" />
              </div>

              {/* Rubric Precondition Report */}
              <div>
                <h4 className="text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">
                  Pre-Execution Rubric Evaluation
                </h4>
                {result.precondition_report && result.precondition_report.length > 0 ? (
                  <div className="space-y-2">
                    {result.precondition_report.map((item, idx) => (
                      <div
                        key={idx}
                        className={`rounded-lg border p-3 text-xs ${
                          item.passed ? "border-emerald-200 bg-emerald-50/50" : "border-rose-200 bg-rose-50"
                        }`}
                      >
                        <div className="flex items-center justify-between font-semibold">
                          <span className={item.passed ? "text-emerald-900" : "text-rose-900"}>
                            {item.rule_name}
                          </span>
                          <span
                            className={`rounded px-1.5 py-0.2 font-bold text-[10px] ${
                              item.passed ? "bg-emerald-200 text-emerald-800" : "bg-rose-200 text-rose-800"
                            }`}
                          >
                            {item.passed ? "PASSED" : "FAILED"}
                          </span>
                        </div>
                        {!item.passed && (
                          <div className="mt-2 text-rose-800 space-y-1">
                            <div><strong>Cause:</strong> {item.reason}</div>
                            <div className="rounded bg-rose-100/70 p-2 text-[11px] text-rose-900">
                              <strong>Remediation:</strong> {getRubricRemediation(item.rule_name, item.reason)}
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-gray-500 italic">No pre-conditions registered for this action.</div>
                )}
              </div>

              {/* Diff / Audit Payload */}
              <div>
                <h4 className="text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">
                  Immutable Audit Diff Payload
                </h4>
                <pre className="rounded-lg bg-gray-900 p-3 text-xs text-emerald-400 font-mono overflow-x-auto max-h-60">
                  {JSON.stringify(result.diff_payload || {}, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
