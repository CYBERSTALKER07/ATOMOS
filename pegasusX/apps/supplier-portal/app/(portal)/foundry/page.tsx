'use client';

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { PageChrome } from '@/components/PageChrome';
import { LotRecallBlastRadiusView } from '@/components/foundry/LotRecallBlastRadiusView';
import { foundrySupplierApi } from '@/lib/foundry-api';
import type { ScenarioPatch, ScenarioCommitResult, OntologyObject } from '@pegasusx/types';
import {
  Layers,
  AlertTriangle,
  GitBranch,
  Sparkles,
  CheckCircle2,
  XCircle,
  RotateCw,
  Search,
  Trash2,
  Eye,
  FileCode,
  ArrowRight,
  ShieldCheck,
} from 'lucide-react';

type FoundryTab = 'recall' | 'sandbox' | 'searcharound';

export default function SupplierFoundryPage() {
  const [tab, setTab] = useState<FoundryTab>('sandbox');

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

  // SearchAround State
  const [saType, setSaType] = useState('RawMaterialLot');
  const [saId, setSaId] = useState('rm_milk_batch_889');
  const [saHops, setSaHops] = useState(3);
  const [saSearching, setSaSearching] = useState(false);
  const [saObjects, setSaObjects] = useState<OntologyObject[]>([]);
  const [saSearched, setSaSearched] = useState(false);

  // Real-time JSON validation
  const jsonStatus = useMemo(() => {
    try {
      const parsed = JSON.parse(patchJSON);
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        return { valid: false, message: 'JSON root must be an object' };
      }
      return { valid: true, size: new Blob([patchJSON]).size };
    } catch (e) {
      return { valid: false, message: e instanceof Error ? e.message : 'Invalid JSON syntax' };
    }
  }, [patchJSON]);

  // Quick template loader
  const loadTemplate = (name: 'promo' | 'pack_snap' | 'discontinue') => {
    setError(null);
    if (name === 'promo') {
      setObjectType('ProductSKU');
      setObjectId('sku_kefir_1L');
      setOperation('UPSERT');
      setPatchJSON(JSON.stringify({ wholesale_tier_1_minor: 1450000, promo_discount_bps: 1200, min_order_cases: 10 }, null, 2));
    } else if (name === 'pack_snap') {
      setObjectType('WholesaleProduct');
      setObjectId('prod_cheese_suluguni');
      setOperation('UPSERT');
      setPatchJSON(JSON.stringify({ case_pack_quantity: 12, split_case_allowed: false, unit_weight_grams: 450 }, null, 2));
    } else if (name === 'discontinue') {
      setObjectType('ProductSKU');
      setObjectId('sku_summer_lemonade_500ml');
      setOperation('DELETE');
      setPatchJSON(JSON.stringify({ discontinue_reason: 'SEASONAL_ROTATION', replacement_sku: 'sku_winter_tea_500ml' }, null, 2));
    }
  };

  const handleApplyPatch = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    setError(null);
    if (!jsonStatus.valid) {
      setError(jsonStatus.message || 'Patch properties must be valid JSON');
      return;
    }

    setSubmittingPatch(true);
    try {
      const patchProps = JSON.parse(patchJSON) as Record<string, unknown>;
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
      if (!res.conflicts || res.conflicts.length === 0) {
        setStagedPatches([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Commit failed');
    } finally {
      setCommitting(false);
    }
  };

  const handleDiscardPatch = (indexToRemove: number) => {
    setStagedPatches((prev) => prev.filter((_, idx) => idx !== indexToRemove));
  };

  const handleRebaseConflict = (conflictObjectId: string) => {
    // Clear resolved conflict by removing the conflicted patch
    setCommitResult((prev) => {
      if (!prev || !prev.conflicts) return prev;
      const remaining = prev.conflicts.filter((c) => c.object_id !== conflictObjectId);
      return { ...prev, conflicts: remaining };
    });
  };

  const handleExecuteSearchAround = async () => {
    setError(null);
    setSaSearching(true);
    setSaSearched(true);
    try {
      const res = await foundrySupplierApi.searchAround({
        cell_id: cellId.trim(),
        start_object_type: saType.trim(),
        start_object_ids: [saId.trim()],
        max_hops: saHops,
      });
      setSaObjects(res.objects || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'SearchAround failed');
    } finally {
      setSaSearching(false);
    }
  };

  // Keyboard shortcut listener for Cmd+Enter / Ctrl+Enter (Stage Patch) and Cmd+S (Commit)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault();
        handleApplyPatch();
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 's') {
        e.preventDefault();
        if (stagedPatches.length > 0 && !committing) {
          handleCommitScenario();
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [patchJSON, stagedPatches, committing, jsonStatus.valid]);

  return (
    <PageChrome
      title="Foundry Semantic & Dynamic Sandbox"
      description="Stage Copy-on-Write catalog patches, simulate wholesale price promotions, and analyze contaminated lot recall blast-radius reports."
    >
      <div className="space-y-6 select-none">
        {/* Navigation Tabs - Pegasus Monochrome styling */}
        <div className="flex flex-wrap items-center gap-2 border-b border-neutral-800 pb-3">
          <button
            type="button"
            onClick={() => setTab('sandbox')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              tab === 'sandbox'
                ? 'bg-neutral-100 text-neutral-950 shadow-sm'
                : 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-900'
            }`}
          >
            <GitBranch size={15} />
            <span>Scenario Sandbox (CoW)</span>
            {stagedPatches.length > 0 && (
              <span className={`px-1.5 py-0.2 rounded-full text-[10px] font-mono ${
                tab === 'sandbox' ? 'bg-neutral-900 text-neutral-100' : 'bg-neutral-800 text-neutral-300'
              }`}>
                {stagedPatches.length}
              </span>
            )}
          </button>

          <button
            type="button"
            onClick={() => setTab('recall')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              tab === 'recall'
                ? 'bg-rose-950/80 text-rose-300 border border-rose-800 shadow-sm'
                : 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-900'
            }`}
          >
            <AlertTriangle size={15} className="text-rose-400" />
            <span>Lot Recall Blast Radius</span>
          </button>

          <button
            type="button"
            onClick={() => setTab('searcharound')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold transition-all ${
              tab === 'searcharound'
                ? 'bg-neutral-100 text-neutral-950 shadow-sm'
                : 'text-neutral-400 hover:text-neutral-200 hover:bg-neutral-900'
            }`}
          >
            <Search size={15} />
            <span>Ontology SearchAround</span>
          </button>
        </div>

        {/* Tab 1: Scenario Sandbox (CoW Overlay) */}
        {tab === 'sandbox' && (
          <div className="space-y-6">
            <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 space-y-5 shadow-xl">
              {/* Sandbox Header Strip */}
              <div className="flex flex-wrap items-center justify-between gap-4 border-b border-neutral-800 pb-4">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-neutral-750 bg-neutral-950 text-neutral-300">
                    <Layers size={20} />
                  </div>
                  <div>
                    <h3 className="text-sm font-bold text-neutral-100 flex items-center gap-2">
                      <span>Copy-on-Write Scenario Sandbox</span>
                      <span className="rounded border border-neutral-750 bg-neutral-800 px-2 py-0.5 font-mono text-[10px] text-neutral-400">
                        BaseVersion Watermarked
                      </span>
                    </h3>
                    <p className="text-xs text-neutral-400">
                      Modifications are isolated in an overlay layer until committed with optimistic concurrency checks.
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2 font-mono text-xs">
                  <div className="flex items-center rounded-xl border border-neutral-750 bg-neutral-950 px-3 py-1.5">
                    <span className="text-neutral-500 mr-2">SCENARIO:</span>
                    <input
                      type="text"
                      value={scenarioId}
                      onChange={(e) => setScenarioId(e.target.value)}
                      className="bg-transparent font-mono text-xs text-neutral-200 focus:outline-none"
                      placeholder="Scenario ID"
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleCommitScenario}
                    disabled={committing || stagedPatches.length === 0}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold bg-neutral-100 hover:bg-white text-neutral-950 disabled:opacity-40 transition-all shadow active:scale-95"
                    title="Commit Scenario (⌘S)"
                  >
                    {committing ? (
                      <>
                        <RotateCw size={14} className="animate-spin" />
                        <span>Verifying...</span>
                      </>
                    ) : (
                      <>
                        <ShieldCheck size={14} />
                        <span>Commit Scenario ({stagedPatches.length})</span>
                      </>
                    )}
                  </button>
                </div>
              </div>

              {/* Error Message */}
              {error && (
                <div className="flex items-center gap-2.5 p-3.5 rounded-xl bg-rose-950/40 border border-rose-800/80 text-xs text-rose-200">
                  <XCircle size={16} className="text-rose-400 shrink-0" />
                  <span className="flex-1 font-mono">{error}</span>
                  <button
                    type="button"
                    onClick={() => setError(null)}
                    className="text-neutral-400 hover:text-neutral-200"
                  >
                    ✕
                  </button>
                </div>
              )}

              {/* Commit Result: Conflicts (409) vs Success */}
              {commitResult && (
                <div>
                  {commitResult.conflicts && commitResult.conflicts.length > 0 ? (
                    <div className="p-4 rounded-xl bg-rose-950/40 border border-rose-700/80 text-rose-200 space-y-3 text-xs">
                      <div className="flex items-center justify-between">
                        <div className="font-bold flex items-center gap-2 text-rose-300">
                          <AlertTriangle size={16} />
                          <span>Guard #7: Stale-State Commit Conflict Detected (409 Conflict)</span>
                        </div>
                        <span className="font-mono text-[11px] text-rose-400">
                          {commitResult.conflicts.length} object(s) drifted in production
                        </span>
                      </div>
                      <p className="text-[11px] text-neutral-400">
                        The live Spanner table was updated by another session while this scenario was staged. Rebase the scenario or discard the stale patch:
                      </p>
                      <div className="divide-y divide-rose-900/60 rounded-xl border border-rose-800/50 bg-neutral-950/80 font-mono text-xs overflow-hidden">
                        {commitResult.conflicts.map((c, i) => (
                          <div key={i} className="p-3 flex flex-wrap items-center justify-between gap-3">
                            <div>
                              <span className="font-bold text-neutral-200">{c.object_type}::{c.object_id}</span>
                              <div className="text-[11px] text-neutral-400 mt-0.5">
                                Staged BaseVersion: <span className="text-amber-400">v{c.base_version}</span> ➔ Live Version: <span className="text-rose-400 font-bold">v{c.live_version}</span>
                              </div>
                            </div>
                            <div className="flex items-center gap-2">
                              <button
                                type="button"
                                onClick={() => handleRebaseConflict(c.object_id)}
                                className="px-2.5 py-1 rounded-lg border border-neutral-700 bg-neutral-800 text-neutral-200 hover:bg-neutral-700 text-[11px]"
                              >
                                Rebase to v{c.live_version}
                              </button>
                              <button
                                type="button"
                                onClick={() => {
                                  setStagedPatches((prev) => prev.filter((p) => p.object_id !== c.object_id));
                                  handleRebaseConflict(c.object_id);
                                }}
                                className="px-2.5 py-1 rounded-lg border border-rose-800 bg-rose-950 text-rose-300 hover:bg-rose-900 text-[11px]"
                              >
                                Discard Patch
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <div className="p-4 rounded-xl bg-emerald-950/40 border border-emerald-700/80 text-emerald-200 text-xs flex items-center justify-between">
                      <div className="flex items-center gap-2 font-bold text-emerald-300">
                        <CheckCircle2 size={16} />
                        <span>Scenario Committed Successfully</span>
                      </div>
                      <span className="font-mono text-emerald-400">
                        {commitResult.committed_objects} objects merged into production Spanner
                      </span>
                    </div>
                  )}
                </div>
              )}

              {/* Quick Template Presets */}
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs text-neutral-400 font-medium">Quick Templates:</span>
                <button
                  type="button"
                  onClick={() => loadTemplate('promo')}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 hover:bg-neutral-850 text-xs transition-colors"
                >
                  <Sparkles size={12} className="text-amber-400" />
                  <span>Wholesale Tier 1 Promo (-12%)</span>
                </button>
                <button
                  type="button"
                  onClick={() => loadTemplate('pack_snap')}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 hover:bg-neutral-850 text-xs transition-colors"
                >
                  <Sparkles size={12} className="text-sky-400" />
                  <span>Pack Multiple 12-case Snap</span>
                </button>
                <button
                  type="button"
                  onClick={() => loadTemplate('discontinue')}
                  className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 hover:bg-neutral-850 text-xs transition-colors"
                >
                  <Trash2 size={12} className="text-rose-400" />
                  <span>Discontinue Stale SKU</span>
                </button>
              </div>

              {/* Patch Authoring Form */}
              <form onSubmit={handleApplyPatch} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
                  <div>
                    <label className="block text-xs font-semibold text-neutral-400 mb-1">Object Type</label>
                    <input
                      type="text"
                      required
                      value={objectType}
                      onChange={(e) => setObjectType(e.target.value)}
                      className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:border-neutral-600 focus:outline-none"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-neutral-400 mb-1">Object ID</label>
                    <input
                      type="text"
                      required
                      value={objectId}
                      onChange={(e) => setObjectId(e.target.value)}
                      className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:border-neutral-600 focus:outline-none"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold text-neutral-400 mb-1">Operation</label>
                    <select
                      value={operation}
                      onChange={(e) => setOperation(e.target.value as 'UPSERT' | 'DELETE')}
                      className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:border-neutral-600 focus:outline-none"
                    >
                      <option value="UPSERT">UPSERT (Apply Delta)</option>
                      <option value="DELETE">DELETE (Tombstone)</option>
                    </select>
                  </div>

                  <div className="flex items-end gap-2">
                    <button
                      type="submit"
                      disabled={submittingPatch || !jsonStatus.valid}
                      className="flex-1 py-2 px-3 rounded-xl text-xs font-bold bg-neutral-100 hover:bg-white text-neutral-950 disabled:opacity-40 transition-all shadow active:scale-95 flex items-center justify-center gap-1.5"
                      title="Apply Patch (⌘↵)"
                    >
                      {submittingPatch ? (
                        <>
                          <RotateCw size={13} className="animate-spin" />
                          <span>Staging...</span>
                        </>
                      ) : (
                        <>
                          <Plus size={13} />
                          <span>Stage Patch</span>
                        </>
                      )}
                    </button>
                    <button
                      type="button"
                      onClick={handleResolvePreview}
                      disabled={resolving}
                      className="py-2 px-3 rounded-xl text-xs font-semibold bg-neutral-800 hover:bg-neutral-750 text-neutral-200 border border-neutral-750 flex items-center gap-1.5 transition-colors"
                      title="Preview Overlay Resolution"
                    >
                      {resolving ? <RotateCw size={13} className="animate-spin" /> : <Eye size={13} />}
                      <span>Preview</span>
                    </button>
                  </div>
                </div>

                {/* JSON Editor with Real-Time Syntax Verification */}
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label className="text-xs font-semibold text-neutral-400 flex items-center gap-2">
                      <FileCode size={14} />
                      <span>Patch Properties (JSON in Uzbek Tiyin / Integer Math)</span>
                    </label>
                    <div className="font-mono text-[11px] flex items-center gap-1.5">
                      {jsonStatus.valid ? (
                        <span className="flex items-center gap-1 text-emerald-400">
                          <CheckCircle2 size={12} />
                          <span>Valid JSON ({jsonStatus.size} B)</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-rose-400">
                          <XCircle size={12} />
                          <span>{jsonStatus.message}</span>
                        </span>
                      )}
                    </div>
                  </div>
                  <textarea
                    rows={4}
                    value={patchJSON}
                    onChange={(e) => setPatchJSON(e.target.value)}
                    className={`w-full bg-neutral-950 rounded-xl p-3 text-xs font-mono text-neutral-100 focus:outline-none border transition-colors ${
                      jsonStatus.valid ? 'border-neutral-800 focus:border-neutral-600' : 'border-rose-700/80 focus:border-rose-500'
                    }`}
                  />
                  <p className="mt-1 text-[11px] font-mono text-neutral-500">
                    Shortcut: Press <kbd className="rounded border border-neutral-800 bg-neutral-900 px-1 text-neutral-300">⌘ + ↵</kbd> to stage patch
                  </p>
                </div>
              </form>

              {/* Resolved Preview Panel */}
              {resolved && (
                <div className="p-4 rounded-xl bg-neutral-950 border border-neutral-750 space-y-2 text-xs">
                  <div className="flex items-center justify-between border-b border-neutral-800 pb-2">
                    <span className="font-bold text-neutral-200 flex items-center gap-2">
                      <Eye size={14} className="text-sky-400" />
                      <span>Live + Scenario Overlay Resolved State</span>
                    </span>
                    <span className="font-mono text-neutral-400">{resolved.object_type}::{resolved.object_id}</span>
                  </div>
                  <pre className="font-mono text-emerald-400 text-[11px] overflow-x-auto p-3 bg-neutral-900/80 rounded-lg border border-neutral-850">
                    {JSON.stringify(resolved.properties, null, 2)}
                  </pre>
                </div>
              )}

              {/* Staged Patches List & Empty State */}
              <div className="space-y-3 pt-2 border-t border-neutral-800">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-semibold text-neutral-400 uppercase tracking-wider">
                    Staged Patches in {scenarioId} ({stagedPatches.length}):
                  </span>
                  {stagedPatches.length > 0 && (
                    <button
                      type="button"
                      onClick={() => setStagedPatches([])}
                      className="text-xs text-neutral-500 hover:text-rose-400 transition-colors"
                    >
                      Clear All Patches
                    </button>
                  )}
                </div>

                {stagedPatches.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-neutral-800 bg-neutral-950/40 py-8 px-4 text-center">
                    <GitBranch size={24} className="mx-auto text-neutral-600 mb-2" />
                    <p className="text-xs font-semibold text-neutral-300">No Patches Staged in Current Scenario</p>
                    <p className="text-[11px] text-neutral-500 mt-1 max-w-sm mx-auto">
                      Use the patch form above or choose a quick template to stage Copy-on-Write modifications.
                    </p>
                  </div>
                ) : (
                  <div className="divide-y divide-neutral-800 rounded-xl border border-neutral-800 bg-neutral-950 text-xs font-mono overflow-hidden">
                    {stagedPatches.map((p, idx) => (
                      <div key={idx} className="p-3 flex items-center justify-between hover:bg-neutral-900/60 transition-colors">
                        <div className="flex items-center gap-2.5">
                          <span className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                            p.operation === 'DELETE' ? 'bg-rose-950 text-rose-300 border border-rose-800' : 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                          }`}>
                            {p.operation}
                          </span>
                          <span className="text-neutral-200 font-bold">{p.object_type}::{p.object_id}</span>
                          <span className="text-neutral-500 text-[11px] truncate max-w-md">
                            {JSON.stringify(p.patch_properties)}
                          </span>
                        </div>
                        <button
                          type="button"
                          onClick={() => handleDiscardPatch(idx)}
                          className="flex h-6 w-6 items-center justify-center rounded text-neutral-500 hover:text-rose-400 hover:bg-neutral-800 transition-colors"
                          title="Discard Patch"
                        >
                          ✕
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Contaminated Lot Recall Blast Radius */}
        {tab === 'recall' && <LotRecallBlastRadiusView supplierId={supplierId} />}

        {/* Tab 3: Ontology SearchAround Explorer */}
        {tab === 'searcharound' && (
          <div className="rounded-2xl border border-neutral-800 bg-neutral-900/90 p-6 space-y-5 shadow-xl">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-800 pb-4">
              <div>
                <h3 className="text-sm font-bold text-neutral-100 flex items-center gap-2">
                  <Search size={16} />
                  <span>Ontology Graph Multi-Hop SearchAround</span>
                </h3>
                <p className="text-xs text-neutral-400">
                  Execute millisecond breadth-first traversals across wholesale items, pallets, and manufacturing work orders.
                </p>
              </div>
              <button
                type="button"
                onClick={handleExecuteSearchAround}
                disabled={saSearching}
                className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-bold bg-neutral-100 hover:bg-white text-neutral-950 disabled:opacity-40 transition-all shadow"
              >
                {saSearching ? <RotateCw size={13} className="animate-spin" /> : <Search size={13} />}
                <span>Execute SearchAround</span>
              </button>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="block text-xs font-semibold text-neutral-400 mb-1">Start Object Type</label>
                <input
                  type="text"
                  value={saType}
                  onChange={(e) => setSaType(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-neutral-600"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-neutral-400 mb-1">Start Object ID</label>
                <input
                  type="text"
                  value={saId}
                  onChange={(e) => setSaId(e.target.value)}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-neutral-600"
                />
              </div>
              <div>
                <label className="block text-xs font-semibold text-neutral-400 mb-1">Max Hops (1..12)</label>
                <input
                  type="number"
                  min="1"
                  max="12"
                  value={saHops}
                  onChange={(e) => setSaHops(parseInt(e.target.value || '1', 10))}
                  className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-neutral-600"
                />
              </div>
            </div>

            {/* Quick Demo Node Pickers */}
            <div className="flex flex-wrap items-center gap-2 pt-1">
              <span className="text-xs text-neutral-400">Sample Nodes:</span>
              <button
                type="button"
                onClick={() => { setSaType('RawMaterialLot'); setSaId('rm_milk_batch_889'); }}
                className="px-2 py-0.5 rounded border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 text-xs font-mono"
              >
                RawMaterialLot::rm_milk_batch_889
              </button>
              <button
                type="button"
                onClick={() => { setSaType('WorkOrder'); setSaId('WO-MILK-001'); }}
                className="px-2 py-0.5 rounded border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 text-xs font-mono"
              >
                WorkOrder::WO-MILK-001
              </button>
              <button
                type="button"
                onClick={() => { setSaType('SSCCPallet'); setSaId('00040123451234567891'); }}
                className="px-2 py-0.5 rounded border border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 text-xs font-mono"
              >
                SSCCPallet::00040123451234567891
              </button>
            </div>

            {/* SearchAround Results or Empty State */}
            {saSearched && (
              <div className="space-y-3 pt-3 border-t border-neutral-800">
                <span className="text-xs font-semibold text-neutral-400 uppercase tracking-wider">
                  Discovered Lineage Graph Nodes ({saObjects.length}):
                </span>

                {saObjects.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-neutral-800 bg-neutral-950/40 py-8 px-4 text-center">
                    <Search size={22} className="mx-auto text-neutral-600 mb-2" />
                    <p className="text-xs font-semibold text-neutral-300">No Lineage Nodes Found</p>
                    <p className="text-[11px] text-neutral-500 mt-1 max-w-sm mx-auto">
                      Zero nodes connected to [{saType}] {saId} within {saHops} hops. Try expanding the hop limit or selecting a known entity.
                    </p>
                  </div>
                ) : (
                  <div className="divide-y divide-neutral-800 rounded-xl border border-neutral-800 bg-neutral-950 text-xs overflow-hidden">
                    {saObjects.map((obj) => (
                      <div
                        key={`${obj.cell_id}::${obj.object_type}::${obj.object_id}`}
                        className="p-3 flex items-center justify-between hover:bg-neutral-900/60 transition-colors"
                      >
                        <div className="font-mono flex items-center gap-2">
                          <span className="font-bold text-sky-400">[{obj.object_type}]</span>
                          <span className="text-neutral-200 font-semibold">{obj.object_id}</span>
                          <span className="text-neutral-500 text-[11px]">v{obj.version}</span>
                        </div>
                        {obj.security_markings && obj.security_markings.length > 0 && (
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-rose-950 text-rose-300 border border-rose-800">
                            {obj.security_markings.join(', ')}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </PageChrome>
  );
}

function Plus(props: React.SVGProps<SVGSVGElement> & { size?: number }) {
  const s = props.size || 16;
  return (
    <svg width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <line x1="12" y1="5" x2="12" y2="19"></line>
      <line x1="5" y1="12" x2="19" y2="12"></line>
    </svg>
  );
}
