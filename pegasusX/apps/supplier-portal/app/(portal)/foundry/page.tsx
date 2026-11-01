'use client';

import React, { useState } from 'react';
import { PageChrome } from '@/components/PageChrome';
import { LotRecallBlastRadiusView } from '@/components/foundry/LotRecallBlastRadiusView';
import { foundrySupplierApi } from '@/lib/foundry-api';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';

export default function SupplierFoundryPage() {
  const [tab, setTab] = useState<'recall' | 'sandbox'>('recall');

  // Scenario Sandbox states
  const [scenarioId, setScenarioId] = useState('scenario_wholesale_q4_promo');
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [objectType, setObjectType] = useState('ProductSKU');
  const [objectId, setObjectId] = useState('sku_kefir_1L');
  const [operation, setOperation] = useState<'UPSERT' | 'DELETE'>('UPSERT');
  const [patchJSON, setPatchJSON] = useState('{\n  "wholesale_tier_1_minor": 1450000,\n  "promo_discount_bps": 1200\n}');

  const [stagedPatches, setStagedPatches] = useState<ScenarioPatch[]>([]);
  const [submittingPatch, setSubmittingPatch] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [resolved, setResolved] = useState<OntologyObject | null>(null);

  const [committing, setCommitting] = useState(false);
  const [commitResult, setCommitResult] = useState<ScenarioCommitResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleApplyPatch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    let patchProps: Record<string, unknown> = {};
    try {
      patchProps = JSON.parse(patchJSON);
    } catch {
      setError('Patch properties must be valid JSON');
      return;
    }

    setSubmittingPatch(true);
    try {
      const patch = await foundrySupplierApi.applyScenarioPatch(scenarioId.trim(), {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        object_type: objectType.trim(),
        object_id: objectId.trim(),
        operation,
        patch_properties: patchProps,
      });
      setStagedPatches((prev) => [patch, ...prev]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to apply patch');
    } finally {
      setSubmittingPatch(false);
    }
  };

  const handleResolvePreview = async () => {
    setError(null);
    setResolving(true);
    setResolved(null);
    try {
      const obj = await foundrySupplierApi.resolveScenarioObject(
        scenarioId.trim(),
        cellId.trim(),
        objectType.trim(),
        objectId.trim(),
      );
      setResolved(obj);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Resolve preview failed');
    } finally {
      setResolving(false);
    }
  };

  const handleCommitScenario = async () => {
    setError(null);
    setCommitting(true);
    setCommitResult(null);
    try {
      const res = await foundrySupplierApi.commitScenario(scenarioId.trim());
      setCommitResult(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Commit failed');
    } finally {
      setCommitting(false);
    }
  };

  return (
    <PageChrome
      title="Foundry Scenario Sandbox & Lineage"
      description="Stage Copy-on-Write catalog patches, simulate wholesale price promotions, and analyze contaminated lot recall blast-radius reports."
    >
      <div className="space-y-6">
        <div className="flex items-center gap-2 border-b border-slate-800 pb-3">
          <button
            type="button"
            onClick={() => setTab('recall')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
              tab === 'recall'
                ? 'bg-rose-700 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Lot Recall Blast Radius
          </button>
          <button
            type="button"
            onClick={() => setTab('sandbox')}
            className={`px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
              tab === 'sandbox'
                ? 'bg-purple-700 text-white shadow'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Scenario Sandbox
          </button>
        </div>

        {tab === 'recall' && <LotRecallBlastRadiusView supplierId={supplierId} />}

        {tab === 'sandbox' && (
          <div className="space-y-6">
            <div className="md-card p-6 space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-800 pb-3">
                <div>
                  <h3 className="text-base font-bold text-slate-100">
                    Copy-on-Write Scenario Simulation & Watermark Commit
                  </h3>
                  <p className="text-xs text-slate-400">
                    Draft changes in an isolated sandbox overlay before publishing to wholesale buyers.
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={scenarioId}
                    onChange={(e) => setScenarioId(e.target.value)}
                    className="bg-slate-950 border border-slate-700 rounded-xl px-3 py-1.5 text-xs font-mono text-slate-100"
                    placeholder="Scenario ID"
                  />
                  <button
                    type="button"
                    onClick={handleCommitScenario}
                    disabled={committing}
                    className="px-4 py-2 rounded-xl text-xs font-bold bg-purple-600 hover:bg-purple-500 text-white disabled:opacity-50"
                  >
                    {committing ? 'Verifying...' : 'Commit Scenario'}
                  </button>
                </div>
              </div>

              {error && (
                <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
                  {error}
                </div>
              )}

              {commitResult && (
                <div>
                  {commitResult.conflicts && commitResult.conflicts.length > 0 ? (
                    <div className="p-4 rounded-xl bg-rose-950/50 border border-rose-600 text-rose-200 space-y-2 text-xs">
                      <div className="font-bold flex items-center gap-1.5">
                        <span>🛑</span>
                        <span>Scenario Commit Conflict: BaseVersion Watermark Drift (409)</span>
                      </div>
                      <div className="divide-y divide-rose-800 font-mono text-[11px]">
                        {commitResult.conflicts.map((c, i) => (
                          <div key={i} className="py-1 flex justify-between">
                            <span>{c.object_type}::{c.object_id}</span>
                            <span className="text-amber-300">Base v{c.base_version} vs Live v{c.live_version}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <div className="p-4 rounded-xl bg-emerald-950/50 border border-emerald-500 text-emerald-200 text-xs flex justify-between">
                      <span className="font-bold">Scenario Committed Successfully</span>
                      <span className="font-mono">{commitResult.committed_objects} objects merged</span>
                    </div>
                  )}
                </div>
              )}

              <form onSubmit={handleApplyPatch} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
                  <div>
                    <label className="block text-xs font-semibold text-slate-400 mb-1">Object Type</label>
                    <input
                      type="text"
                      required
                      value={objectType}
                      onChange={(e) => setObjectType(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-400 mb-1">Object ID</label>
                    <input
                      type="text"
                      required
                      value={objectId}
                      onChange={(e) => setObjectId(e.target.value)}
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-slate-400 mb-1">Operation</label>
                    <select
                      value={operation}
                      onChange={(e) => setOperation(e.target.value as 'UPSERT' | 'DELETE')}
                      className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
                    >
                      <option value="UPSERT">UPSERT</option>
                      <option value="DELETE">DELETE</option>
                    </select>
                  </div>

                  <div className="flex items-end gap-2">
                    <button
                      type="submit"
                      disabled={submittingPatch}
                      className="w-full py-2 px-3 rounded-xl text-xs font-bold bg-purple-600 hover:bg-purple-500 text-white disabled:opacity-50"
                    >
                      {submittingPatch ? 'Staging...' : 'Apply Patch'}
                    </button>
                    <button
                      type="button"
                      onClick={handleResolvePreview}
                      disabled={resolving}
                      className="py-2 px-3 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700"
                    >
                      {resolving ? 'Resolving...' : 'Preview'}
                    </button>
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-400 mb-1">
                    Patch Properties (JSON)
                  </label>
                  <textarea
                    rows={3}
                    value={patchJSON}
                    onChange={(e) => setPatchJSON(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-700 rounded-xl p-3 text-xs font-mono text-slate-100"
                  />
                </div>
              </form>

              {resolved && (
                <div className="p-4 rounded-xl bg-slate-950 border border-purple-500/50 space-y-2 text-xs">
                  <div className="flex items-center justify-between border-b border-slate-800 pb-2">
                    <span className="font-bold text-purple-300">Resolved State Preview</span>
                    <span className="font-mono text-slate-400">{resolved.object_type}::{resolved.object_id}</span>
                  </div>
                  <pre className="font-mono text-emerald-400 text-[11px] overflow-x-auto p-2 bg-slate-900 rounded">
                    {JSON.stringify(resolved.properties, null, 2)}
                  </pre>
                </div>
              )}

              {stagedPatches.length > 0 && (
                <div className="space-y-2 pt-2 border-t border-slate-800">
                  <span className="text-xs font-semibold text-slate-400">
                    Staged Patches in {scenarioId} ({stagedPatches.length}):
                  </span>
                  <div className="divide-y divide-slate-800 rounded-xl border border-slate-800 bg-slate-950 text-xs font-mono">
                    {stagedPatches.map((p, idx) => (
                      <div key={idx} className="p-2.5 flex items-center justify-between">
                        <span>[{p.operation}] {p.object_type}::{p.object_id}</span>
                        <span className="text-slate-400 text-[11px] truncate max-w-sm">{JSON.stringify(p.patch_properties)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </PageChrome>
  );
}
