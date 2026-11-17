'use client';

import React, { useState } from 'react';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';

export default function ScenarioSandboxManager({ token }: { token: string }) {
  const [scenarioId, setScenarioId] = useState('scenario_admin_tariff_hike');
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [objectType, setObjectType] = useState('PricingTier');
  const [objectId, setObjectId] = useState('tier_wholesale_vol_100');
  const [operation, setOperation] = useState<'UPSERT' | 'DELETE'>('UPSERT');
  const [propertiesJSON, setPropertiesJSON] = useState('{\n  "discount_bps": 1500,\n  "effective_date": "2026-11-01"\n}');

  const [stagedPatches, setStagedPatches] = useState<ScenarioPatch[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [resolvedObject, setResolvedObject] = useState<OntologyObject | null>(null);

  const [committing, setCommitting] = useState(false);
  const [commitResult, setCommitResult] = useState<ScenarioCommitResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleApplyPatch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    let patchProps: Record<string, unknown> = {};
    try {
      patchProps = JSON.parse(propertiesJSON);
    } catch {
      setError('Patch properties must be valid JSON');
      return;
    }

    setSubmitting(true);
    try {
      const patch = await foundryApi.applyScenarioPatch(token, scenarioId.trim(), {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        object_type: objectType.trim(),
        object_id: objectId.trim(),
        operation,
        patch_properties: patchProps,
      });
      setStagedPatches((prev) => [patch, ...prev]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Apply patch failed');
    } finally {
      setSubmitting(false);
    }
  };

  const handleResolve = async () => {
    setError(null);
    setResolving(true);
    setResolvedObject(null);
    try {
      const obj = await foundryApi.resolveScenarioObject(
        token,
        scenarioId.trim(),
        cellId.trim(),
        objectType.trim(),
        objectId.trim(),
      );
      setResolvedObject(obj);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Resolve preview failed');
    } finally {
      setResolving(false);
    }
  };

  const handleCommit = async () => {
    setError(null);
    setCommitting(true);
    setCommitResult(null);
    try {
      const res = await foundryApi.commitScenario(token, scenarioId.trim());
      setCommitResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Commit failed');
    } finally {
      setCommitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div className="flex items-center justify-between border-b pb-3">
          <div>
            <h3 className="text-base font-semibold text-gray-900">
              Scenario Sandbox Manager & Optimistic Concurrency Commit
            </h3>
            <p className="text-xs text-gray-500 mt-0.5">
              Copy-on-Write sandbox environment. Patches do not mutate live data until BaseVersion watermark verified.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="text"
              value={scenarioId}
              onChange={(e) => setScenarioId(e.target.value)}
              className="rounded border px-3 py-1 text-xs font-mono"
              placeholder="Scenario ID"
            />
            <button
              type="button"
              onClick={handleCommit}
              disabled={committing}
              className="rounded bg-indigo-600 px-3 py-1 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {committing ? 'Verifying...' : 'Commit Scenario'}
            </button>
          </div>
        </div>

        {error && (
          <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </div>
        )}

        {commitResult && (
          <div>
            {commitResult.conflicts && commitResult.conflicts.length > 0 ? (
              <div className="rounded border border-red-300 bg-red-50 p-3 text-xs text-red-800 space-y-2">
                <div className="font-bold flex items-center gap-1.5">
                  <span>🛑</span>
                  <span>Commit Rejected: Watermark Drift Detected (409 Conflict)</span>
                </div>
                <div className="divide-y divide-red-200 font-mono text-[11px]">
                  {commitResult.conflicts.map((c, i) => (
                    <div key={i} className="py-1 flex justify-between">
                      <span>{c.object_type}::{c.object_id}</span>
                      <span>Staged: v{c.base_version} | Live: v{c.live_version}</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="rounded border border-emerald-300 bg-emerald-50 p-3 text-xs text-emerald-800 flex justify-between items-center">
                <span className="font-bold">Scenario Committed Cleanly</span>
                <span className="font-mono">{commitResult.committed_objects} objects merged</span>
              </div>
            )}
          </div>
        )}

        <form onSubmit={handleApplyPatch} className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div>
              <label className="block text-xs font-medium text-gray-700">Object Type</label>
              <input
                type="text"
                required
                value={objectType}
                onChange={(e) => setObjectType(e.target.value)}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Object ID</label>
              <input
                type="text"
                required
                value={objectId}
                onChange={(e) => setObjectId(e.target.value)}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-700">Operation</label>
              <select
                value={operation}
                onChange={(e) => setOperation(e.target.value as 'UPSERT' | 'DELETE')}
                className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
              >
                <option value="UPSERT">UPSERT</option>
                <option value="DELETE">DELETE</option>
              </select>
            </div>
            <div className="flex items-end gap-2">
              <button
                type="submit"
                disabled={submitting}
                className="w-full rounded bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {submitting ? 'Staging...' : 'Apply Patch'}
              </button>
              <button
                type="button"
                onClick={handleResolve}
                disabled={resolving}
                className="rounded border px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
              >
                {resolving ? 'Resolving...' : 'Preview'}
              </button>
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-gray-700">Overlay Properties (JSON)</label>
            <textarea
              rows={3}
              value={propertiesJSON}
              onChange={(e) => setPropertiesJSON(e.target.value)}
              className="mt-1 w-full rounded border p-2 text-xs font-mono"
            />
          </div>
        </form>

        {resolvedObject && (
          <div className="rounded border bg-gray-50 p-3 text-xs space-y-1">
            <span className="font-semibold text-gray-700">Resolved Overlay State:</span>
            <pre className="font-mono text-[11px] text-gray-800 bg-white p-2 rounded border overflow-x-auto">
              {JSON.stringify(resolvedObject.properties, null, 2)}
            </pre>
          </div>
        )}

        {stagedPatches.length > 0 && (
          <div className="space-y-2 pt-2 border-t">
            <span className="text-xs font-semibold text-gray-700">Staged Overlay Patches ({stagedPatches.length}):</span>
            <div className="divide-y rounded border text-xs font-mono">
              {stagedPatches.map((p, idx) => (
                <div key={idx} className="p-2 flex justify-between">
                  <span>[{p.operation}] {p.object_type}::{p.object_id} (Base: v{p.base_version})</span>
                  <span className="text-gray-500 truncate max-w-xs">{JSON.stringify(p.patch_properties)}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
