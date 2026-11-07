'use client';

import React, { useState } from 'react';
import type { RecallBlastRadiusReport, OntologyObject } from '@pegasusx/types';
import { foundryWarehouseApi } from '@/lib/foundry-api';

interface OntologyBlastRadiusViewerProps {
  defaultCellId?: string;
  defaultSupplierId?: string;
}

export function OntologyBlastRadiusViewer({
  defaultCellId = 'cell-uz',
  defaultSupplierId = 'sup_samarkand_dairy',
}: OntologyBlastRadiusViewerProps) {
  const [cellId, setCellId] = useState(defaultCellId);
  const [supplierId, setSupplierId] = useState(defaultSupplierId);
  const [rawMaterialLotId, setRawMaterialLotId] = useState('rm_milk_batch_889');

  // Search Around states
  const [startObjectType, setStartObjectType] = useState('RawMaterialLot');
  const [startObjectId, setStartObjectId] = useState('rm_milk_batch_889');
  const [maxHops, setMaxHops] = useState<number>(3);
  const [searchObjects, setSearchObjects] = useState<OntologyObject[]>([]);
  const [searching, setSearching] = useState(false);

  // Recall states
  const [loadingRecall, setLoadingRecall] = useState(false);
  const [recallReport, setRecallReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Trigger 1-Click Lot Recall
  const handleTriggerRecall = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoadingRecall(true);
    try {
      const report = await foundryWarehouseApi.lotRecall({
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        raw_material_lot_id: rawMaterialLotId.trim(),
      });
      setRecallReport(report);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Lot recall execution failed');
    } finally {
      setLoadingRecall(false);
    }
  };

  // Perform Multi-Hop Search Around
  const handleSearchAround = async () => {
    setError(null);
    setSearching(true);
    try {
      const res = await foundryWarehouseApi.searchAround({
        cell_id: cellId.trim(),
        start_object_type: startObjectType.trim(),
        start_object_ids: [startObjectId.trim()],
        max_hops: maxHops,
      });
      setSearchObjects(res.objects || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'SearchAround failed');
    } finally {
      setSearching(false);
    }
  };

  return (
    <div className="bg-[var(--color-md-surface-card,#1e293b)] p-6 rounded-2xl border border-slate-700/60 shadow-xl space-y-6">
      <div className="pb-4 border-b border-slate-700/60">
        <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
          <span>🧬</span>
          <span>Ontology Semantic Graph & Contaminated Lot Recall Blast-Radius</span>
        </h2>
        <p className="text-xs text-slate-400 mt-1">
          Multi-hop SearchAround graph traversal and 1-click contaminated raw material lot recall propagating MARKING:LOT_RECALL_QUARANTINE.
        </p>
      </div>

      {/* Lot Recall Trigger Form */}
      <div className="p-5 bg-rose-950/20 border border-rose-800/40 rounded-2xl space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xl">⚠️</span>
            <h3 className="text-sm font-bold text-rose-200 uppercase tracking-wider">
              1-Click Contaminated Lot Recall Initiation
            </h3>
          </div>
          <span className="text-[11px] font-mono text-rose-300">
            Automated Blast-Radius Quarantine
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
              Supplier ID
            </label>
            <input
              type="text"
              required
              value={supplierId}
              onChange={(e) => setSupplierId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">
              Contaminated Raw Material Lot ID
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                required
                value={rawMaterialLotId}
                onChange={(e) => setRawMaterialLotId(e.target.value)}
                placeholder="e.g. rm_milk_batch_889"
                className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
              />
              <button
                type="submit"
                disabled={loadingRecall}
                className="shrink-0 px-4 py-2 rounded-xl text-xs font-bold bg-rose-700 hover:bg-rose-600 text-white disabled:opacity-50 transition-all shadow"
              >
                {loadingRecall ? 'Quarantining...' : 'Recall'}
              </button>
            </div>
          </div>
        </form>
      </div>

      {error && (
        <div className="p-3 bg-red-950/50 border border-red-700 rounded-xl text-xs text-red-200">
          {error}
        </div>
      )}

      {/* Recall Blast-Radius Report */}
      {recallReport && (
        <div className="p-5 rounded-2xl bg-slate-900/90 border border-rose-500/60 space-y-4 shadow-xl">
          <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-slate-800">
            <div>
              <span className="text-xs font-bold text-rose-400 flex items-center gap-1.5">
                <span>🛑</span>
                <span>Active Lot Recall Quarantine Report</span>
              </span>
              <div className="font-mono text-sm font-bold text-slate-100 mt-0.5">
                Lot: {recallReport.contaminated_raw_material_lot}
              </div>
            </div>
            <span className="px-3 py-1 rounded-full text-xs font-mono font-bold bg-rose-950 text-rose-200 border border-rose-800">
              MARKING:LOT_RECALL_QUARANTINE
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
            {/* Work Orders */}
            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Factory Work Orders ({recallReport.affected_work_orders?.length || 0})
              </span>
              <div className="font-mono text-amber-300 max-h-24 overflow-y-auto">
                {recallReport.affected_work_orders?.join(', ') || 'None'}
              </div>
            </div>

            {/* Finished Lots */}
            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Finished SKU Lots ({recallReport.affected_finished_lots?.length || 0})
              </span>
              <div className="font-mono text-amber-300 max-h-24 overflow-y-auto">
                {recallReport.affected_finished_lots?.join(', ') || 'None'}
              </div>
            </div>

            {/* Quarantined SSCC Pallets */}
            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Quarantined Pallets ({recallReport.quarantined_sscc_pallets?.length || 0})
              </span>
              <div className="font-mono text-rose-300 max-h-24 overflow-y-auto">
                {recallReport.quarantined_sscc_pallets?.join(', ') || 'None'}
              </div>
            </div>

            {/* Quarantined Warehouse Totes */}
            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Warehouse Pick Totes ({recallReport.quarantined_warehouse_totes?.length || 0})
              </span>
              <div className="font-mono text-rose-300 max-h-24 overflow-y-auto">
                {recallReport.quarantined_warehouse_totes?.join(', ') || 'None'}
              </div>
            </div>
          </div>

          {/* Delivery & Fiscal Impact */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs pt-2">
            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                In-Transit Drop Blocks ({recallReport.in_transit_driver_drop_blocks?.length || 0})
              </span>
              <div className="font-mono text-rose-400 font-bold max-h-20 overflow-y-auto">
                {recallReport.in_transit_driver_drop_blocks?.join(', ') || 'None'}
              </div>
            </div>

            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Retailer Reverse Pickups ({recallReport.retailer_reverse_pickup_orders?.length || 0})
              </span>
              <div className="font-mono text-amber-300 max-h-20 overflow-y-auto">
                {recallReport.retailer_reverse_pickup_orders?.join(', ') || 'None'}
              </div>
            </div>

            <div className="p-3 bg-slate-950 rounded-xl border border-slate-800 space-y-1">
              <span className="text-slate-400 block font-semibold">
                Impacted Soliq Facturas ({recallReport.impacted_soliq_facturas?.length || 0})
              </span>
              <div className="font-mono text-slate-300 max-h-20 overflow-y-auto">
                {recallReport.impacted_soliq_facturas?.join(', ') || 'None'}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Multi-Hop SearchAround Explorer */}
      <div className="p-5 bg-slate-900/50 border border-slate-800 rounded-2xl space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-bold text-slate-200 uppercase tracking-wider">
              Multi-Hop SearchAround BFS Traversal
            </h3>
            <p className="text-xs text-slate-400">
              Traverse Ontology links dynamically up to 12 hops.
            </p>
          </div>
          <button
            type="button"
            onClick={handleSearchAround}
            disabled={searching}
            className="px-4 py-2 rounded-xl text-xs font-bold bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50 transition-all shadow"
          >
            {searching ? 'Traversing Graph...' : 'Execute SearchAround'}
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Root Object Type
            </label>
            <input
              type="text"
              value={startObjectType}
              onChange={(e) => setStartObjectType(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Root Object ID
            </label>
            <input
              type="text"
              value={startObjectId}
              onChange={(e) => setStartObjectId(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
              Max Hops (1-12)
            </label>
            <input
              type="number"
              min="1"
              max="12"
              value={maxHops}
              onChange={(e) => setMaxHops(Math.min(12, Math.max(1, parseInt(e.target.value || '1', 10))))}
              className="w-full bg-slate-950 border border-slate-700 rounded-xl px-3 py-2 text-xs font-mono text-slate-100"
            />
          </div>
        </div>

        {searchObjects.length > 0 && (
          <div className="space-y-2 pt-2">
            <span className="text-xs font-semibold text-slate-400">
              Discovered Graph Entities ({searchObjects.length}):
            </span>
            <div className="divide-y divide-slate-800 rounded-xl border border-slate-800 bg-slate-950 overflow-hidden max-h-60 overflow-y-auto">
              {searchObjects.map((obj) => (
                <div key={`${obj.cell_id}::${obj.object_type}::${obj.object_id}`} className="p-3 text-xs flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 rounded bg-blue-950 text-blue-300 font-mono font-bold">
                      {obj.object_type}
                    </span>
                    <span className="font-mono text-slate-200 font-semibold">{obj.object_id}</span>
                    <span className="text-slate-500 font-mono">v{obj.version}</span>
                  </div>

                  {obj.security_markings && obj.security_markings.length > 0 && (
                    <div className="flex items-center gap-1">
                      {obj.security_markings.map((m) => (
                        <span key={m} className="px-2 py-0.5 rounded text-[10px] font-mono bg-purple-950 text-purple-300 border border-purple-800">
                          {m}
                        </span>
                      ))}
                    </div>
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
