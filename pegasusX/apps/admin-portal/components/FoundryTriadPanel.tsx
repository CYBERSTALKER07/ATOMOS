'use client';

import React, { useState } from 'react';
import type { ActionExecutionRecord } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';
import OntologyBlastRadiusGraph from './OntologyBlastRadiusGraph';
import ScenarioSandboxManager from './ScenarioSandboxManager';

export default function FoundryTriadPanel({ token }: { token: string }) {
  const [subTab, setSubTab] = useState<'blast' | 'sandbox' | 'kinetic'>('blast');

  // Kinetic Action execution states
  const [actionType, setActionType] = useState('ACTION_RELEASE_PICK_WAVE');
  const [actorId, setActorId] = useState('admin_platform');
  const [actorRole, setActorRole] = useState('ROLE_ADMIN');
  const [scenarioId, setScenarioId] = useState('');
  const [parametersJSON, setParametersJSON] = useState(
    '{\n  "pkcs12_cert_hours_remaining": 96,\n  "ikpu_code_valid": true,\n  "fx_corridor_breached": false\n}',
  );

  const [executing, setExecuting] = useState(false);
  const [executionRecord, setExecutionRecord] = useState<ActionExecutionRecord | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleExecuteAction = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setExecutionRecord(null);

    let params: Record<string, unknown> = {};
    try {
      params = JSON.parse(parametersJSON);
    } catch {
      setError('Parameters must be valid JSON');
      return;
    }

    setExecuting(true);
    try {
      const rec = await foundryApi.executeAction(token, {
        cell_id: 'cell-uz',
        supplier_id: 'sup_samarkand_dairy',
        action_type: actionType,
        actor_id: actorId,
        actor_role: actorRole,
        scenario_id: scenarioId.trim() || undefined,
        idempotency_key: `action_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
        parameters: params,
      });
      setExecutionRecord(rec);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Action execution failed');
    } finally {
      setExecuting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between border-b pb-4">
        <div>
          <h2 className="text-lg font-bold text-gray-900">
            Palantir Foundry Semantic-Kinetic-Dynamic Triad Console
          </h2>
          <p className="text-xs text-gray-500 mt-0.5">
            Cross-role semantic graph, Copy-on-Write scenario simulations, and pre-execution kinetic action guards.
          </p>
        </div>

        <div className="flex gap-1 rounded-lg border bg-gray-100 p-1">
          <button
            type="button"
            onClick={() => setSubTab('blast')}
            className={`rounded px-3 py-1 text-xs font-semibold ${
              subTab === 'blast' ? 'bg-white shadow text-gray-900' : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            Ontology Blast-Radius
          </button>
          <button
            type="button"
            onClick={() => setSubTab('sandbox')}
            className={`rounded px-3 py-1 text-xs font-semibold ${
              subTab === 'sandbox' ? 'bg-white shadow text-gray-900' : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            Scenario Sandbox
          </button>
          <button
            type="button"
            onClick={() => setSubTab('kinetic')}
            className={`rounded px-3 py-1 text-xs font-semibold ${
              subTab === 'kinetic' ? 'bg-white shadow text-gray-900' : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            Kinetic Action Guard
          </button>
        </div>
      </div>

      {subTab === 'blast' && <OntologyBlastRadiusGraph token={token} />}
      {subTab === 'sandbox' && <ScenarioSandboxManager token={token} />}

      {subTab === 'kinetic' && (
        <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
          <div>
            <h3 className="text-base font-semibold text-gray-900">
              Kinetic Action Execution & Precondition Rubric Inspector
            </h3>
            <p className="text-xs text-gray-500 mt-0.5">
              Evaluates registered deterministic pre-condition rubrics before state mutation.
            </p>
          </div>

          <form onSubmit={handleExecuteAction} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="block text-xs font-medium text-gray-700">Action Type</label>
                <select
                  value={actionType}
                  onChange={(e) => setActionType(e.target.value)}
                  className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
                >
                  <option value="ACTION_RELEASE_PICK_WAVE">ACTION_RELEASE_PICK_WAVE (Step 5)</option>
                  <option value="ACTION_SEAL_VEHICLE_MANIFEST">ACTION_SEAL_VEHICLE_MANIFEST (Step 6)</option>
                  <option value="ACTION_START_DRIVER_ROUTE">ACTION_START_DRIVER_ROUTE (Step 7)</option>
                  <option value="ACTION_COMPLETE_DOORSTEP_EPOD">ACTION_COMPLETE_DOORSTEP_EPOD (Step 8)</option>
                  <option value="ACTION_BATCH_SUPPLIER_PAYOUT">ACTION_BATCH_SUPPLIER_PAYOUT (Step 10)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-700">Actor Identity</label>
                <input
                  type="text"
                  required
                  value={actorId}
                  onChange={(e) => setActorId(e.target.value)}
                  className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-700">Actor Role</label>
                <input
                  type="text"
                  required
                  value={actorRole}
                  onChange={(e) => setActorRole(e.target.value)}
                  className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-700">Action Parameters (JSON)</label>
              <textarea
                rows={4}
                value={parametersJSON}
                onChange={(e) => setParametersJSON(e.target.value)}
                className="mt-1 w-full rounded border p-2 text-xs font-mono"
              />
            </div>

            {error && (
              <div className="rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
                {error}
              </div>
            )}

            <div className="flex justify-end">
              <button
                type="submit"
                disabled={executing}
                className="rounded bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {executing ? 'Evaluating Preconditions...' : 'Execute Kinetic Action'}
              </button>
            </div>
          </form>

          {executionRecord && (
            <div className="space-y-3 pt-3 border-t">
              <div className="flex items-center justify-between text-xs">
                <span className="font-semibold text-gray-900">
                  Execution ID: <span className="font-mono">{executionRecord.action_execution_id}</span>
                </span>
                <span
                  className={`rounded px-2 py-0.5 font-bold ${
                    executionRecord.status === 'COMMITTED'
                      ? 'bg-emerald-100 text-emerald-800'
                      : 'bg-red-100 text-red-800'
                  }`}
                >
                  {executionRecord.status}
                </span>
              </div>

              <div className="rounded border divide-y overflow-hidden text-xs">
                <div className="bg-gray-50 p-2 font-semibold text-gray-700">
                  Precondition Rubric Report ({executionRecord.precondition_report?.length || 0} checks)
                </div>
                {executionRecord.precondition_report?.map((r, i) => (
                  <div key={i} className="p-2 flex items-center justify-between">
                    <span className="font-mono">{r.rule_name}</span>
                    <span
                      className={`rounded px-2 py-0.5 text-[11px] font-bold ${
                        r.passed ? 'bg-emerald-100 text-emerald-800' : 'bg-red-100 text-red-800'
                      }`}
                    >
                      {r.passed ? 'PASS' : `FAIL: ${r.reason || 'Check failed'}`}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
