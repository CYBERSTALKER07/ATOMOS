"use client";

import React, { useState } from "react";
import type { OntologyObject } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";

export default function ScenarioSandboxPage() {
  const { token } = useAdminToken();

  // Production initializers (UA-8)
  const [scenarioId, setScenarioId] = useState("");
  const [cellId, setCellId] = useState("cell-uz");
  const [objectType, setObjectType] = useState("WarehouseCandidate");
  const [objectId, setObjectId] = useState("");
  const [propertyKey, setPropertyKey] = useState("tier_discount_percent");
  const [propertyValue, setPropertyValue] = useState("12.5");

  const [loading, setLoading] = useState(false);
  const [baselineObject, setBaselineObject] = useState<OntologyObject | null>(null);
  const [resolvedObject, setResolvedObject] = useState<OntologyObject | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [commitMessage, setCommitMessage] = useState<string | null>(null);

  // 2-Step Confirmation Modal (UA-7)
  const [showCommitModal, setShowCommitModal] = useState(false);
  const [committing, setCommitting] = useState(false);

  // 3-Way 409 Conflict Rebase State (UA-7)
  const [conflictState, setConflictState] = useState<{
    stagedVersion: number;
    liveVersion: number;
  } | null>(null);

  const handleApplyPatch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token || !scenarioId.trim() || !objectId.trim()) return;
    setError(null);
    setCommitMessage(null);
    setConflictState(null);
    setLoading(true);

    try {
      // 1. Fetch baseline
      const baseObj = await foundryApi.getObject(token, cellId.trim(), objectType.trim(), objectId.trim());
      setBaselineObject(baseObj);

      // Parse value
      let parsedVal: unknown = propertyValue;
      if (!isNaN(Number(propertyValue))) parsedVal = Number(propertyValue);
      else if (propertyValue === "true") parsedVal = true;
      else if (propertyValue === "false") parsedVal = false;

      // 2. Apply patch to sandbox
      await foundryApi.applyScenarioPatch(token, scenarioId.trim(), {
        cell_id: cellId.trim(),
        supplier_id: baseObj.supplier_id || "sup_tashkent_fmcg",
        object_type: objectType.trim(),
        object_id: objectId.trim(),
        operation: "UPSERT",
        patch_properties: {
          [propertyKey.trim()]: parsedVal,
        },
      });

      // 3. Resolve overlaid state
      const resObj = await foundryApi.resolveScenarioObject(
        token,
        scenarioId.trim(),
        cellId.trim(),
        objectType.trim(),
        objectId.trim(),
      );
      setResolvedObject(resObj);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to apply scenario patch");
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteCommit = async () => {
    if (!token || !scenarioId.trim()) return;
    setCommitting(true);
    setError(null);
    try {
      const res = await foundryApi.commitScenario(token, scenarioId.trim());
      setCommitMessage(`Scenario ${res.scenario_id} committed successfully! Committed objects: ${res.committed_objects}`);
      setShowCommitModal(false);
      setResolvedObject(null);
      setBaselineObject(null);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Commit failed";
      if (msg.includes("409") || msg.includes("conflict") || msg.includes("drift")) {
        setConflictState({
          stagedVersion: baselineObject?.version || 1,
          liveVersion: (baselineObject?.version || 1) + 1,
        });
      }
      setError(msg);
      setShowCommitModal(false);
    } finally {
      setCommitting(false);
    }
  };

  const loadDemo = () => {
    setScenarioId("sim_q4_express_rebate");
    setObjectId("wh_central_tashkent");
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Copy-on-Write Scenario Sandbox & Diff Engine
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Isolated simulation branches with optimistic concurrency watermarks and kinetic writeback guards.
          </p>
        </div>

        {resolvedObject && (
          <button
            type="button"
            onClick={() => setShowCommitModal(true)}
            className="rounded-lg bg-emerald-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-emerald-700"
          >
            Review & Commit Scenario →
          </button>
        )}
      </div>

      {commitMessage && (
        <div className="rounded-xl border border-emerald-300 bg-emerald-50 p-4 text-xs text-emerald-800 font-semibold">
          ✓ {commitMessage}
        </div>
      )}

      {/* Input Workbench */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <form onSubmit={handleApplyPatch} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
            <div>
              <label className="block text-xs font-medium text-gray-700">Scenario Branch ID</label>
              <input
                type="text"
                required
                value={scenarioId}
                placeholder="e.g. sim_q4_express"
                onChange={(e) => setScenarioId(e.target.value)}
                className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Target Object Type</label>
              <select
                value={objectType}
                onChange={(e) => setObjectType(e.target.value)}
                className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
              >
                <option value="WarehouseCandidate">WarehouseCandidate</option>
                <option value="SupplierInventoryV2">SupplierInventoryV2</option>
                <option value="DeliveryRoutePlan">DeliveryRoutePlan</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Object ID</label>
              <input
                type="text"
                required
                value={objectId}
                placeholder="e.g. wh_central_tashkent"
                onChange={(e) => setObjectId(e.target.value)}
                className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div className="flex items-end gap-2">
              <button
                type="submit"
                disabled={loading || !scenarioId.trim() || !objectId.trim()}
                className="flex-1 rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {loading ? "Staging Patch…" : "Stage Simulation Patch"}
              </button>
              <button
                type="button"
                onClick={loadDemo}
                className="rounded-lg border px-2.5 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50"
              >
                Demo
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 pt-2 border-t">
            <div>
              <label className="block text-xs font-medium text-gray-700">Patch Property Key</label>
              <input
                type="text"
                required
                value={propertyKey}
                onChange={(e) => setPropertyKey(e.target.value)}
                className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Simulated Value</label>
              <input
                type="text"
                required
                value={propertyValue}
                onChange={(e) => setPropertyValue(e.target.value)}
                className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
              />
            </div>
          </div>
        </form>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700 font-medium">
            {error}
          </div>
        )}
      </div>

      {/* 3-Way 409 Conflict Rebase Guidance (UA-7) */}
      {conflictState && (
        <div className="rounded-xl border border-amber-300 bg-amber-50 p-5 shadow-sm space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-amber-700 font-bold text-base">⚠️</span>
            <h3 className="text-sm font-bold text-amber-900">
              409 Version Conflict (Watermark Drift Detected)
            </h3>
          </div>
          <p className="text-xs text-amber-800">
            The live record was modified by another transaction while this simulation was staged (Base Watermark: v
            {conflictState.stagedVersion} vs Live Head: v{conflictState.liveVersion}).
          </p>
          <div className="flex gap-3 pt-1">
            <button
              type="button"
              onClick={() => {
                setConflictState(null);
                void handleApplyPatch({ preventDefault: () => {} } as React.FormEvent);
              }}
              className="rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-amber-700"
            >
              Rebase Scenario onto Live Head
            </button>
            <button
              type="button"
              onClick={() => {
                setConflictState(null);
                setResolvedObject(null);
              }}
              className="rounded-lg border border-amber-400 bg-white px-3 py-1.5 text-xs font-semibold text-amber-900 hover:bg-amber-50"
            >
              Discard Conflicting Branch
            </button>
          </div>
        </div>
      )}

      {/* Side-by-Side Diff Viewer (UA-7) */}
      {baselineObject && resolvedObject && (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          {/* Baseline Left */}
          <div className="rounded-xl border bg-white p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between border-b pb-2">
              <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
                Live Head State (Base Watermark)
              </span>
              <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-mono font-bold text-gray-700">
                Version {baselineObject.version}
              </span>
            </div>
            <pre className="rounded-lg bg-gray-50 p-4 font-mono text-xs overflow-x-auto max-h-96 border text-gray-800">
              {JSON.stringify(baselineObject.properties, null, 2)}
            </pre>
          </div>

          {/* Overlaid Scenario Right */}
          <div className="rounded-xl border border-indigo-200 bg-indigo-50/20 p-5 shadow-sm space-y-3">
            <div className="flex items-center justify-between border-b border-indigo-200 pb-2">
              <span className="text-xs font-bold uppercase tracking-wider text-indigo-700">
                Sandboxed Scenario Branch State
              </span>
              <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs font-mono font-bold text-indigo-800">
                Overlaid Simulation
              </span>
            </div>
            <pre className="rounded-lg bg-white p-4 font-mono text-xs overflow-x-auto max-h-96 border border-indigo-200 text-indigo-950 font-semibold">
              {JSON.stringify(resolvedObject.properties, null, 2)}
            </pre>
          </div>
        </div>
      )}

      {/* 2-Step Confirmation Modal (UA-7) */}
      {showCommitModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-2xl space-y-4">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 text-amber-700 text-lg">
                ⚠️
              </span>
              <div>
                <h3 className="text-base font-bold text-gray-900">Confirm Scenario Writeback</h3>
                <p className="text-xs text-gray-500">This will commit sandbox mutations to live production.</p>
              </div>
            </div>

            <div className="rounded-lg bg-gray-50 p-3 text-xs space-y-2 border">
              <div className="flex justify-between">
                <span className="text-gray-500">Scenario Branch:</span>
                <span className="font-mono font-bold text-gray-900">{scenarioId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Target Object:</span>
                <span className="font-mono font-bold text-gray-900">{objectId}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-500">Mutated Property:</span>
                <span className="font-mono font-bold text-indigo-700">{propertyKey} = {propertyValue}</span>
              </div>
            </div>

            <p className="text-xs text-gray-600">
              Kinetic actions will execute atomic Spanner write closures and buffer an Outbox event for cross-pod broadcast.
            </p>

            <div className="flex justify-end gap-3 pt-2 border-t">
              <button
                type="button"
                onClick={() => setShowCommitModal(false)}
                className="rounded-lg border px-4 py-2 text-xs font-semibold text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={committing}
                onClick={() => void handleExecuteCommit()}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-xs font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
              >
                {committing ? "Committing Spanner Closure…" : "Confirm Production Commit"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
