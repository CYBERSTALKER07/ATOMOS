'use client';

import React, { useState } from 'react';
import type { RecallBlastRadiusReport, OntologyObject } from '@pegasusx/types';
import { foundryApi } from '@/lib/foundry-api';

export default function OntologyBlastRadiusGraph({ token }: { token: string }) {
  const [cellId, setCellId] = useState('cell-uz');
  const [supplierId, setSupplierId] = useState('sup_samarkand_dairy');
  const [lotId, setLotId] = useState('rm_milk_batch_889');

  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Search around state
  const [searchType, setSearchType] = useState('RawMaterialLot');
  const [searchId, setSearchId] = useState('rm_milk_batch_889');
  const [maxHops, setMaxHops] = useState(3);
  const [graphObjects, setGraphObjects] = useState<OntologyObject[]>([]);
  const [searching, setSearching] = useState(false);

  const handleLotRecall = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await foundryApi.lotRecall(token, {
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        raw_material_lot_id: lotId.trim(),
      });
      setReport(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Lot recall failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSearchAround = async () => {
    setError(null);
    setSearching(true);
    try {
      const res = await foundryApi.searchAround(token, {
        cell_id: cellId.trim(),
        start_object_type: searchType.trim(),
        start_object_ids: [searchId.trim()],
        max_hops: maxHops,
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
      <div className="rounded-xl border bg-white p-5 shadow-sm">
        <h3 className="text-base font-semibold text-gray-900">
          Contaminated Lot Recall Blast-Radius Engine
        </h3>
        <p className="text-xs text-gray-500 mt-0.5">
          Triggers multi-hop graph quarantine traversal and stamps MARKING:LOT_RECALL_QUARANTINE.
        </p>

        <form onSubmit={handleLotRecall} className="mt-4 grid grid-cols-1 sm:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-700">Cell ID</label>
            <input
              type="text"
              required
              value={cellId}
              onChange={(e) => setCellId(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Supplier ID</label>
            <input
              type="text"
              required
              value={supplierId}
              onChange={(e) => setSupplierId(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Contaminated Lot ID</label>
            <input
              type="text"
              required
              value={lotId}
              onChange={(e) => setLotId(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div className="flex items-end">
            <button
              type="submit"
              disabled={loading}
              className="w-full rounded bg-red-600 px-4 py-2 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
            >
              {loading ? 'Evaluating...' : 'Simulate Blast Radius'}
            </button>
          </div>
        </form>

        {error && (
          <div className="mt-3 rounded border border-red-200 bg-red-50 p-2 text-xs text-red-700">
            {error}
          </div>
        )}

        {report && (
          <div className="mt-5 space-y-4 rounded-lg border border-red-200 bg-red-50/50 p-4">
            <div className="flex items-center justify-between border-b border-red-200 pb-2">
              <span className="font-semibold text-red-900 text-xs">
                Blast-Radius Report for Lot: {report.contaminated_raw_material_lot}
              </span>
              <span className="rounded bg-red-100 px-2 py-0.5 text-[11px] font-mono font-bold text-red-800">
                MARKING:LOT_RECALL_QUARANTINE
              </span>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 text-xs">
              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Affected Work Orders</span>
                <span className="font-mono text-sm font-bold text-gray-900">
                  {report.affected_work_orders?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-gray-600 truncate">
                  {report.affected_work_orders?.join(', ') || 'None'}
                </div>
              </div>

              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Finished SKU Lots</span>
                <span className="font-mono text-sm font-bold text-gray-900">
                  {report.affected_finished_lots?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-gray-600 truncate">
                  {report.affected_finished_lots?.join(', ') || 'None'}
                </div>
              </div>

              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Quarantined Pallets</span>
                <span className="font-mono text-sm font-bold text-red-600">
                  {report.quarantined_sscc_pallets?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-red-700 truncate">
                  {report.quarantined_sscc_pallets?.join(', ') || 'None'}
                </div>
              </div>

              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Quarantined Totes</span>
                <span className="font-mono text-sm font-bold text-red-600">
                  {report.quarantined_warehouse_totes?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-red-700 truncate">
                  {report.quarantined_warehouse_totes?.join(', ') || 'None'}
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs pt-1">
              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">In-Transit Drop Blocks</span>
                <span className="font-mono text-sm font-bold text-amber-600">
                  {report.in_transit_driver_drop_blocks?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-gray-600 truncate">
                  {report.in_transit_driver_drop_blocks?.join(', ') || 'None'}
                </div>
              </div>

              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Retailer Reverse Pickups</span>
                <span className="font-mono text-sm font-bold text-amber-600">
                  {report.retailer_reverse_pickup_orders?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-gray-600 truncate">
                  {report.retailer_reverse_pickup_orders?.join(', ') || 'None'}
                </div>
              </div>

              <div className="rounded border bg-white p-2.5">
                <span className="text-gray-500 font-medium block">Impacted Soliq Facturas</span>
                <span className="font-mono text-sm font-bold text-purple-600">
                  {report.impacted_soliq_facturas?.length || 0}
                </span>
                <div className="mt-1 text-[11px] font-mono text-gray-600 truncate">
                  {report.impacted_soliq_facturas?.join(', ') || 'None'}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Multi-Hop Search Around Graph Explorer */}
      <div className="rounded-xl border bg-white p-5 shadow-sm">
        <h3 className="text-base font-semibold text-gray-900">
          Multi-Hop SearchAround Graph Traversal
        </h3>
        <p className="text-xs text-gray-500 mt-0.5">
          Execute multi-hop BFS discovery across Ontology entities and edges.
        </p>

        <div className="mt-4 grid grid-cols-1 sm:grid-cols-4 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-700">Start Object Type</label>
            <input
              type="text"
              value={searchType}
              onChange={(e) => setSearchType(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Start Object ID</label>
            <input
              type="text"
              value={searchId}
              onChange={(e) => setSearchId(e.target.value)}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Max Hops (1-12)</label>
            <input
              type="number"
              min="1"
              max="12"
              value={maxHops}
              onChange={(e) => setMaxHops(parseInt(e.target.value || '1', 10))}
              className="mt-1 w-full rounded border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div className="flex items-end">
            <button
              type="button"
              onClick={handleSearchAround}
              disabled={searching}
              className="w-full rounded bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {searching ? 'Traversing...' : 'Traverse Graph'}
            </button>
          </div>
        </div>

        {graphObjects.length > 0 && (
          <div className="mt-4 rounded-lg border divide-y overflow-hidden text-xs font-mono">
            {graphObjects.map((obj) => (
              <div key={`${obj.cell_id}::${obj.object_type}::${obj.object_id}`} className="p-3 flex items-center justify-between">
                <div>
                  <span className="font-bold text-indigo-700 mr-2">[{obj.object_type}]</span>
                  <span>{obj.object_id}</span>
                  <span className="ml-2 text-gray-400">v{obj.version}</span>
                </div>
                {obj.security_markings && obj.security_markings.length > 0 && (
                  <span className="rounded bg-purple-100 px-2 py-0.5 text-[10px] text-purple-800">
                    {obj.security_markings.join(', ')}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
