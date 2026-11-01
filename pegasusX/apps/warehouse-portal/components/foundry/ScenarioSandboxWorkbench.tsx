'use client';

import React, { useState } from 'react';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface ScenarioSandboxWorkbenchProps {
  defaultScenarioId?: string;
  defaultCellId?: string;
  defaultSupplierId?: string;
}

export function ScenarioSandboxWorkbench({
  defaultScenarioId = 'scenario_sim_q4_surge',
  defaultCellId = 'cell-uz',
  defaultSupplierId = 'sup_samarkand_dairy',
}: ScenarioSandboxWorkbenchProps) {
  const [scenarioId, setScenarioId] = useState(defaultScenarioId);
  const [cellId, setCellId] = useState(defaultCellId);
  const [supplierId, setSupplierId] = useState(defaultSupplierId);

  // Patch form
  const [objectType, setObjectType] = useState('ProductSKU');
  const [objectId, setObjectId] = useState('sku_kefir_1L');
  const [operation, setOperation] = useState<'UPSERT' | 'DELETE'>('UPSERT');
  const [propertiesJSON, setPropertiesJSON] = useState('{\n  "base_price_minor": 1800000,\n  "surge_multiplier": 1.15\n}');

  const [stagedPatches, setStagedPatches] = useState<ScenarioPatch[]>([]);
  const [staging, setStaging] = useState(false);

  // Resolve preview
  const [resolving, setResolving] = useState(false);
  const [resolvedObject, setResolvedObject] = useState<OntologyObject | null>(null);

  // Commit
  const [committing, setCommitting] = useState(false);
  const [commitResult, setCommitResult] = useState<ScenarioCommitResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Stage Copy-on-Write Patch
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

    setStaging(true);
    try {
      const patch = await foundryWarehouseApi.applyScenarioPatch(scenarioId.trim(), {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        object_type: objectType.trim(),
        object_id: objectId.trim(),
        operation,
        patch_properties: patchProps,
      });
      setStagedPatches((prev) => [patch, ...prev]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply scenario patch');
    } finally {
      setStaging(false);
    }
  };

  // Resolve preview overlay state
  const handleResolvePreview = async () => {
    setError(null);
    setResolving(true);
    setResolvedObject(null);
    try {
      const obj = await foundryWarehouseApi.resolveScenarioObject(
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

  // Commit Scenario
  const handleCommit = async () => {
    setError(null);
    setCommitting(true);
    setCommitResult(null);
    try {
      const result = await foundryWarehouseApi.commitScenario(scenarioId.trim());
      setCommitResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Commit failed');
    } finally {
      setCommitting(false);
    }
  };

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4 pb-4 border-b border-slate-700/60">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
            <span>🧪</span>
            <span>Foundry Copy-on-Write Scenario Sandbox Workbench</span>
          </h2>
          <p className="text-xs text-slate-400 mt-1">
            Simulate and stage mutations in an isolated digital twin overlay without side-effects on live production state.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <input
            type="text"
            value={scenarioId}
            onChange={(e) => setScenarioId(e.target.value)}
            placeholder="Scenario ID"
            className="bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono text-slate-100 focus:outline-none focus:border-purple-500"
          />
          <button
            type="button"
            onClick={handleCommit}
            disabled={committing}
            className="px-4 py-2 rounded-xl text-xs font-bold bg-purple-600 hover:bg-purple-500 text-white disabled:opacity-50 transition-all shadow flex items-center gap-1.5"
          >
            {committing ? 'Verifying Watermark...' : 'Commit Scenario to Live'}
          </button>
        </div>
      </div>

      {error && (
        <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
          {error}
        </div>
      )}

      {/* Commit Result / Watermark Conflict Display */}
      {commitResult && (
        <div>
          {commitResult.conflicts && commitResult.conflicts.length > 0 ? (
            <div className="p-4 rounded-xl bg-rose-950/50 border border-rose-600 text-rose-200 space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-bold text-sm flex items-center gap-2">
                  <span>🛑</span>
                  <span>Scenario Commit Aborted: BaseVersion Watermark Drift (409 Conflict)</span>
                </span>
                <span className="text-xs font-mono bg-rose-900 px-2 py-0.5 rounded">
                  {commitResult.conflicts.length} Object Conflicts
                </span>
              </div>
              <p className="text-xs text-rose-300">
                Live production state mutated while scenario was staged. The optimistic concurrency watermark check prevented a silent overwrite.
              </p>
              <div className="divide-y divide-rose-800/60 rounded-lg bg-slate-950 p-2 font-mono text-xs">
                {commitResult.conflicts.map((c, i) => (
                  <div key={i} className="py-1.5 flex items-center justify-between">
                    <span>
                      {c.object_type}::{c.object_id}
                    </span>
                    <span className="text-amber-300">
                      Staged BaseVersion: v{c.base_version} → Current LiveVersion: v{c.live_version}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="p-4 rounded-xl bg-emerald-950/50 border border-emerald-500 text-emerald-200 flex items-center justify-between text-xs">
              <div className="flex items-center gap-2 font-bold text-sm">
                <span>✅</span>
                <span>Scenario {commitResult.scenario_id} Successfully Committed</span>
              </div>
              <span className="font-mono">{commitResult.committed_objects} objects merged to live ontology</span>
            </div>
          )}
        </div>
      )}

      {/* Patch Form */}
      <div className="p-5 bg-slate-900/50 rounded-2xl border border-slate-800 space-y-4">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-300">
          Stage Copy-on-Write Mutation Patch
        </h3>

        <form onSubmit={handleApplyPatch} className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
                Object Type
              </label>
              <input
                type="text"
                required
                value={objectType}
                onChange={(e) => setObjectType(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
                Object ID
              </label>
              <input
                type="text"
                required
                value={objectId}
                onChange={(e) => setObjectId(e.target.value)}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
                Operation
              </label>
              <select
                value={operation}
                onChange={(e) => setOperation(e.target.value as 'UPSERT' | 'DELETE')}
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
              >
                <option value="UPSERT">UPSERT (Overlay Properties)</option>
                <option value="DELETE">DELETE (Overlay Deletion)</option>
              </select>
            </div>

            <div className="flex items-end gap-2">
              <button
                type="submit"
                disabled={staging}
                className="w-full py-2 px-3 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-all shadow"
              >
                {staging ? 'Staging...' : 'Apply Patch'}
              </button>
              <button
                type="button"
                onClick={handleResolvePreview}
                disabled={resolving}
                className="shrink-0 py-2 px-3 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-all shadow"
              >
                {resolving ? 'Resolving...' : 'Preview'}
              </button>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Patch Properties (JSON Overlay)
            </label>
            <textarea
              rows={3}
              value={propertiesJSON}
              onChange={(e) => setPropertiesJSON(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl p-3 text-xs font-mono text-slate-100 focus:outline-none focus:border-purple-500"
            />
          </div>
        </form>
      </div>

      {/* Resolved State Preview */}
      {resolvedObject && (
        <div className="p-4 rounded-xl bg-slate-950 border border-purple-500/50 space-y-2 text-xs">
          <div className="flex items-center justify-between pb-2 border-b border-slate-800">
            <span className="font-bold text-purple-300">
              Resolved State: BaseState ⊕ ScenarioPatches
            </span>
            <span className="font-mono text-slate-400">
              {resolvedObject.object_type}::{resolvedObject.object_id} (Base v{resolvedObject.version})
            </span>
          </div>
          <pre className="p-3 bg-slate-900 rounded-lg text-emerald-400 font-mono text-xs overflow-x-auto">
            {JSON.stringify(resolvedObject.properties, null, 2)}
          </pre>
        </div>
      )}

      {/* Staged Patches List */}
      {stagedPatches.length > 0 && (
        <div className="space-y-3">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400">
            Staged Scenario Patches ({stagedPatches.length})
          </h4>
          <div className="divide-y divide-slate-800 rounded-xl bg-slate-900/60 border border-slate-800 overflow-hidden text-xs">
            {stagedPatches.map((p, i) => (
              <div key={i} className="p-3 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="px-2 py-0.5 rounded font-mono font-bold bg-purple-950 text-purple-300">
                    {p.operation}
                  </span>
                  <span className="font-mono text-slate-200">
                    {p.object_type}::{p.object_id}
                  </span>
                  <span className="text-slate-500 font-mono text-[11px]">Anchored v{p.base_version}</span>
                </div>
                <pre className="font-mono text-[11px] text-slate-400">
                  {JSON.stringify(p.patch_properties)}
                </pre>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
