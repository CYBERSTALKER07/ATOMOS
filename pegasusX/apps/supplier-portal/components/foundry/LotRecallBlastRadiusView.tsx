'use client';

import React, { useState } from 'react';
import type { RecallBlastRadiusReport, OntologyObject } from '@pegasusx/types';
import { foundrySupplierApi } from '@/lib/foundry-api';

interface LotRecallBlastRadiusViewProps {
  supplierId?: string;
}

export function LotRecallBlastRadiusView({
  supplierId = 'sup_samarkand_dairy',
}: LotRecallBlastRadiusViewProps) {
  const [cellId, setCellId] = useState('cell-uz');
  const [lotId, setLotId] = useState('rm_milk_batch_889');
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Search around
  const [searchType, setSearchType] = useState('RawMaterialLot');
  const [searchId, setSearchId] = useState('rm_milk_batch_889');
  const [hops, setHops] = useState(3);
  const [graphObjects, setGraphObjects] = useState<OntologyObject[]>([]);
  const [searching, setSearching] = useState(false);

  const handleTriggerRecall = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await foundrySupplierApi.lotRecall({
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        raw_material_lot_id: lotId.trim(),
      });
      setReport(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Lot recall execution failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSearchAround = async () => {
    setError(null);
    setSearching(true);
    try {
      const res = await foundrySupplierApi.searchAround({
        cell_id: cellId.trim(),
        start_object_type: searchType.trim(),
        start_object_ids: [searchId.trim()],
        max_hops: hops,
      });
      setGraphObjects(res.objects || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search around failed');
    } finally {
      setSearching(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Contaminated Lot Recall Trigger Card */}
      <div className="md-card p-6 border-red-500/30 bg-red-950/10 space-y-4">
        <div className="flex items-center justify-between pb-3 border-b border-red-800/30">
          <div>
            <h3 className="text-lg font-bold text-[var(--color-md-error,#f87171)] flex items-center gap-2">
              <span>⚠️</span>
              <span>Contaminated Raw Material Lot Recall Blast-Radius</span>
            </h3>
            <p className="text-xs text-[var(--color-md-outline,#94a3b8)] mt-0.5">
              Simulate or execute supplier lot recalls. Traces forward through manufacturing work orders, finished lots, SSCC pallets, and invoices.
            </p>
          </div>
          <span className="text-xs px-2.5 py-1 rounded-full font-mono bg-red-950 text-red-300 border border-red-800">
            MARKING:LOT_RECALL_QUARANTINE
          </span>
        </div>

        <form onSubmit={handleTriggerRecall} className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Cell ID
            </label>
            <input
              type="text"
              required
              value={cellId}
              onChange={(e) => setCellId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Raw Material Lot ID
            </label>
            <input
              type="text"
              required
              value={lotId}
              onChange={(e) => setLotId(e.target.value)}
              placeholder="e.g. rm_milk_batch_889"
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>

          <div className="flex items-end">
            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 px-4 rounded-xl text-xs font-bold bg-red-700 hover:bg-red-600 text-white disabled:opacity-50 transition-all shadow"
            >
              {loading ? 'Evaluating Lineage...' : 'Execute Lot Recall Blast Radius'}
            </button>
          </div>
        </form>

        {error && (
          <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
            {error}
          </div>
        )}

        {report && (
          <div className="mt-4 p-4 rounded-xl bg-slate-900 border border-red-500/50 space-y-3">
            <div className="flex items-center justify-between pb-2 border-b border-slate-800">
              <span className="text-xs font-bold text-red-400">
                Recall Blast Radius Report for {report.contaminated_raw_material_lot}
              </span>
              <span className="font-mono text-xs text-slate-400">Cell: {report.cell_id}</span>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Work Orders</span>
                <span className="font-mono text-amber-300 font-bold">{report.affected_work_orders?.length || 0}</span>
                <div className="text-[10px] text-slate-500 truncate">{report.affected_work_orders?.join(', ') || 'None'}</div>
              </div>
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Finished Lots</span>
                <span className="font-mono text-amber-300 font-bold">{report.affected_finished_lots?.length || 0}</span>
                <div className="text-[10px] text-slate-500 truncate">{report.affected_finished_lots?.join(', ') || 'None'}</div>
              </div>
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Quarantined Pallets</span>
                <span className="font-mono text-rose-400 font-bold">{report.quarantined_sscc_pallets?.length || 0}</span>
                <div className="text-[10px] text-rose-500 truncate">{report.quarantined_sscc_pallets?.join(', ') || 'None'}</div>
              </div>
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Totes Quarantined</span>
                <span className="font-mono text-rose-400 font-bold">{report.quarantined_warehouse_totes?.length || 0}</span>
                <div className="text-[10px] text-rose-500 truncate">{report.quarantined_warehouse_totes?.join(', ') || 'None'}</div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs pt-1">
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">In-Transit Drops</span>
                <span className="font-mono text-rose-400 font-bold">{report.in_transit_driver_drop_blocks?.length || 0} blocked</span>
              </div>
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Retailer Returns</span>
                <span className="font-mono text-amber-400 font-bold">{report.retailer_reverse_pickup_orders?.length || 0} initiated</span>
              </div>
              <div className="p-2.5 rounded-lg bg-slate-950 border border-slate-800">
                <span className="text-slate-400 block font-semibold">Impacted Facturas</span>
                <span className="font-mono text-purple-400 font-bold">{report.impacted_soliq_facturas?.length || 0} flagged</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Multi-Hop SearchAround Explorer */}
      <div className="md-card p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-base font-bold text-slate-100">
              Ontology SearchAround Traversal
            </h3>
            <p className="text-xs text-slate-400">
              Traverse multi-hop relationships across wholesale goods, lots, and pallets.
            </p>
          </div>
          <button
            type="button"
            onClick={handleSearchAround}
            disabled={searching}
            className="px-4 py-2 rounded-xl text-xs font-bold bg-indigo-600 hover:bg-indigo-500 text-white disabled:opacity-50"
          >
            {searching ? 'Traversing...' : 'SearchAround'}
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs font-semibold text-slate-400 mb-1">Start Object Type</label>
            <input
              type="text"
              value={searchType}
              onChange={(e) => setSearchType(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>
          <div>
            <label className="block text-xs font-semibold text-slate-400 mb-1">Start Object ID</label>
            <input
              type="text"
              value={searchId}
              onChange={(e) => setSearchId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>
          <div>
            <label className="block text-xs font-semibold text-slate-400 mb-1">Max Hops</label>
            <input
              type="number"
              min="1"
              max="12"
              value={hops}
              onChange={(e) => setHops(parseInt(e.target.value || '1', 10))}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>
        </div>

        {graphObjects.length > 0 && (
          <div className="space-y-1.5 pt-2">
            <span className="text-xs font-semibold text-slate-400">
              Discovered Lineage Nodes ({graphObjects.length}):
            </span>
            <div className="divide-y divide-slate-800 rounded-xl border border-slate-800 bg-slate-950 overflow-hidden text-xs">
              {graphObjects.map((obj) => (
                <div key={`${obj.cell_id}::${obj.object_type}::${obj.object_id}`} className="p-3 flex items-center justify-between">
                  <div className="font-mono">
                    <span className="font-bold text-indigo-400 mr-2">[{obj.object_type}]</span>
                    <span className="text-slate-200">{obj.object_id}</span>
                    <span className="text-slate-500 ml-2">v{obj.version}</span>
                  </div>
                  {obj.security_markings && obj.security_markings.length > 0 && (
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-800">
                      {obj.security_markings.join(', ')}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
