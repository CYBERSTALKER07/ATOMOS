'use client';

import React, { useState } from 'react';
import type { RecallBlastRadiusReport } from '@pegasusx/types';
import { foundrySupplierApi } from '@/lib/foundry-api';
import {
  AlertTriangle,
  RotateCw,
  Package,
  Factory,
  Truck,
  FileText,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  HelpCircle,
  Radio,
} from 'lucide-react';

interface LotRecallBlastRadiusViewProps {
  supplierId?: string;
}

const PRESET_LOTS = [
  { id: 'rm_milk_batch_889', label: 'Whole Milk Batch #889 (Recall Demo)' },
  { id: 'rm_pasteurized_cream_104', label: 'Pasteurized Cream #104' },
  { id: 'rm_butter_fat_772', label: 'Butterfat Lot #772' },
];

export function LotRecallBlastRadiusView({
  supplierId = 'sup_samarkand_dairy',
}: LotRecallBlastRadiusViewProps) {
  const [cellId, setCellId] = useState('cell-uz');
  const [lotId, setLotId] = useState('rm_milk_batch_889');
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleTriggerRecall = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
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

  return (
    <div className="space-y-6 select-none">
      {/* Contaminated Lot Recall Trigger Card - Pegasus Monochrome with Rose status accents */}
      <div className="rounded-2xl border border-rose-900/60 bg-neutral-900/95 p-6 space-y-5 shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-3 pb-4 border-b border-neutral-800">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-rose-800/80 bg-rose-950/50 text-rose-300">
              <ShieldAlert size={20} />
            </div>
            <div>
              <h3 className="text-sm font-bold text-neutral-100 flex items-center gap-2">
                <span>Contaminated Raw Material Lot Recall Blast-Radius</span>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-rose-950 text-rose-300 border border-rose-800">
                  MARKING:LOT_RECALL_QUARANTINE
                </span>
              </h3>
              <p className="text-xs text-neutral-400">
                1-Click Forward-BFS Lineage Simulation. Freezes DC picking and pushes do-not-deliver stops to active mobile drivers.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="font-mono text-xs text-neutral-400">SUPPLIER:</span>
            <span className="font-mono text-xs px-2.5 py-1 rounded-lg border border-neutral-750 bg-neutral-950 text-neutral-200">
              {supplierId}
            </span>
          </div>
        </div>

        {/* Preset Sample Lots */}
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-neutral-400 font-medium">Sample Contaminated Lots:</span>
          {PRESET_LOTS.map((preset) => (
            <button
              key={preset.id}
              type="button"
              onClick={() => setLotId(preset.id)}
              className={`px-2.5 py-1 rounded-lg border text-xs font-mono transition-colors ${
                lotId === preset.id
                  ? 'border-rose-700 bg-rose-950/60 text-rose-200'
                  : 'border-neutral-800 bg-neutral-950 text-neutral-300 hover:border-neutral-700 hover:bg-neutral-850'
              }`}
            >
              {preset.label}
            </button>
          ))}
        </div>

        <form onSubmit={handleTriggerRecall} className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Cell ID
            </label>
            <input
              type="text"
              required
              value={cellId}
              onChange={(e) => setCellId(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-neutral-600"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-neutral-400 uppercase tracking-wider mb-1">
              Raw Material Lot ID
            </label>
            <input
              type="text"
              required
              value={lotId}
              onChange={(e) => setLotId(e.target.value)}
              placeholder="e.g. rm_milk_batch_889"
              className="w-full bg-neutral-950 border border-neutral-800 rounded-xl px-3 py-2 text-xs font-mono text-neutral-100 focus:outline-none focus:border-neutral-600"
            />
          </div>

          <div className="flex items-end">
            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 px-4 rounded-xl text-xs font-bold bg-rose-700 hover:bg-rose-600 text-white disabled:opacity-50 transition-all shadow active:scale-95 flex items-center justify-center gap-2"
            >
              {loading ? (
                <>
                  <RotateCw size={14} className="animate-spin" />
                  <span>Evaluating Lineage Blast Radius...</span>
                </>
              ) : (
                <>
                  <AlertTriangle size={14} />
                  <span>Execute Lot Recall Blast Radius</span>
                </>
              )}
            </button>
          </div>
        </form>

        {error && (
          <div className="flex items-center gap-2.5 p-3.5 rounded-xl bg-rose-950/40 border border-rose-800 text-xs text-rose-200">
            <XCircle size={16} className="text-rose-400 shrink-0" />
            <span className="flex-1 font-mono">{error}</span>
            <button type="button" onClick={() => setError(null)} className="text-neutral-400 hover:text-neutral-200">
              ✕
            </button>
          </div>
        )}

        {/* 5-Stage Blast-Radius Interactive Visual Pipeline */}
        {report ? (
          <div className="space-y-4 pt-2">
            <div className="flex items-center justify-between pb-2 border-b border-neutral-800">
              <span className="text-xs font-bold text-rose-300 flex items-center gap-2">
                <CheckCircle2 size={14} className="text-emerald-400" />
                <span>Blast Radius Report Generated for {report.contaminated_raw_material_lot}</span>
              </span>
              <span className="font-mono text-xs text-neutral-400">Cell: {report.cell_id}</span>
            </div>

            {/* Stage Pipeline Progress Grid */}
            <div className="grid grid-cols-1 md:grid-cols-5 gap-3">
              {/* Stage 1: Raw Material */}
              <div className="rounded-xl border border-rose-800/80 bg-neutral-950 p-3 space-y-1">
                <div className="flex items-center justify-between text-neutral-400 text-[11px] font-semibold">
                  <span>STAGE 1</span>
                  <span className="text-rose-400 font-mono">SOURCE</span>
                </div>
                <div className="text-xs font-bold text-neutral-100 flex items-center gap-1.5 pt-1">
                  <Package size={14} className="text-rose-400" />
                  <span className="truncate">{report.contaminated_raw_material_lot}</span>
                </div>
                <div className="text-[10px] text-neutral-500 font-mono">Status: QUARANTINED</div>
              </div>

              {/* Stage 2: Factory Production */}
              <div className="rounded-xl border border-neutral-800 bg-neutral-950 p-3 space-y-1">
                <div className="flex items-center justify-between text-neutral-400 text-[11px] font-semibold">
                  <span>STAGE 2</span>
                  <span className="text-amber-400 font-mono">PLANT</span>
                </div>
                <div className="text-xs font-bold text-neutral-100 flex items-center gap-1.5 pt-1">
                  <Factory size={14} className="text-amber-400" />
                  <span>Work Orders: {report.affected_work_orders?.length || 0}</span>
                </div>
                <div className="text-[10px] text-neutral-500 font-mono truncate">
                  {report.affected_work_orders?.join(', ') || 'Zero affected'}
                </div>
              </div>

              {/* Stage 3: Finished Goods */}
              <div className="rounded-xl border border-neutral-800 bg-neutral-950 p-3 space-y-1">
                <div className="flex items-center justify-between text-neutral-400 text-[11px] font-semibold">
                  <span>STAGE 3</span>
                  <span className="text-rose-400 font-mono">DC PALLETS</span>
                </div>
                <div className="text-xs font-bold text-neutral-100 flex items-center gap-1.5 pt-1">
                  <Package size={14} className="text-rose-400" />
                  <span>Pallets: {report.quarantined_sscc_pallets?.length || 0}</span>
                </div>
                <div className="text-[10px] text-neutral-500 font-mono truncate">
                  Totes: {report.quarantined_warehouse_totes?.length || 0} frozen
                </div>
              </div>

              {/* Stage 4: Logistics & Drivers */}
              <div className="rounded-xl border border-neutral-800 bg-neutral-950 p-3 space-y-1">
                <div className="flex items-center justify-between text-neutral-400 text-[11px] font-semibold">
                  <span>STAGE 4</span>
                  <span className="text-rose-400 font-mono">IN-TRANSIT</span>
                </div>
                <div className="text-xs font-bold text-neutral-100 flex items-center gap-1.5 pt-1">
                  <Truck size={14} className="text-rose-400" />
                  <span>Drops Blocked: {report.in_transit_driver_drop_blocks?.length || 0}</span>
                </div>
                <div className="text-[10px] text-neutral-500 font-mono truncate">
                  Push alert sent to drivers
                </div>
              </div>

              {/* Stage 5: Invoices & Retailers */}
              <div className="rounded-xl border border-neutral-800 bg-neutral-950 p-3 space-y-1">
                <div className="flex items-center justify-between text-neutral-400 text-[11px] font-semibold">
                  <span>STAGE 5</span>
                  <span className="text-purple-400 font-mono">RETAIL</span>
                </div>
                <div className="text-xs font-bold text-neutral-100 flex items-center gap-1.5 pt-1">
                  <FileText size={14} className="text-purple-400" />
                  <span>Reverse Pickups: {report.retailer_reverse_pickup_orders?.length || 0}</span>
                </div>
                <div className="text-[10px] text-neutral-500 font-mono truncate">
                  Facturas: {report.impacted_soliq_facturas?.length || 0} flagged
                </div>
              </div>
            </div>

            {/* Active Driver Intercept Alert */}
            <div className="flex items-center gap-3 p-3.5 rounded-xl border border-rose-800/80 bg-rose-950/40 text-rose-200 text-xs">
              <Radio size={16} className="text-rose-400 animate-pulse shrink-0" />
              <div className="flex-1">
                <span className="font-bold">Active Driver Mobile Intercept Active: </span>
                <span>
                  {report.in_transit_driver_drop_blocks?.length || 0} stops have received emergency do-not-deliver push alerts on their Android/iOS devices. Reverse pickups automatically created.
                </span>
              </div>
            </div>
          </div>
        ) : (
          <div className="rounded-xl border border-dashed border-neutral-800 bg-neutral-950/30 py-6 px-4 text-center">
            <HelpCircle size={22} className="mx-auto text-neutral-600 mb-2" />
            <p className="text-xs font-semibold text-neutral-300">Ready to Evaluate Blast Radius</p>
            <p className="text-[11px] text-neutral-500 mt-0.5 max-w-md mx-auto">
              Select a contaminated raw material batch above and click Execute to trace forward through manufacturing work orders, warehouse pallets, in-transit driver stops, and Soliq tax facturas.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
