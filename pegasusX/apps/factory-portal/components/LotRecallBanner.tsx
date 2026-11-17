'use client';

import React, { useState } from 'react';
import { foundryFactoryApi } from '@/lib/foundry-api';
import type { RecallBlastRadiusReport } from '@pegasusx/types';

interface LotRecallBannerProps {
  initialLotId?: string;
  cellId?: string;
  supplierId?: string;
  onQuarantineChange?: (isQuarantined: boolean, report: RecallBlastRadiusReport | null) => void;
  className?: string;
}

export default function LotRecallBanner({
  initialLotId = '',
  cellId = 'cell-uz',
  supplierId = 'supp-apex-central',
  onQuarantineChange,
  className = '',
}: LotRecallBannerProps) {
  const [lotId, setLotId] = useState(initialLotId);
  const [checking, setChecking] = useState(false);
  const [report, setReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const [quarantineExecuting, setQuarantineExecuting] = useState(false);
  const [quarantinedConfirmed, setQuarantinedConfirmed] = useState(false);

  const handleCheckRecall = async (targetLot: string) => {
    if (!targetLot.trim()) return;
    setChecking(true);
    setError(null);
    setDismissed(false);
    try {
      const res = await foundryFactoryApi.lotRecall({
        cell_id: cellId.trim(),
        supplier_id: supplierId.trim(),
        raw_material_lot_id: targetLot.trim(),
      });
      setReport(res);
      const isQuarantine =
        res.affected_work_orders.length > 0 ||
        res.affected_finished_lots.length > 0 ||
        res.quarantined_sscc_pallets.length > 0 ||
        res.quarantined_warehouse_totes.length > 0 ||
        res.in_transit_driver_drop_blocks.length > 0 ||
        res.retailer_reverse_pickup_orders.length > 0;
      onQuarantineChange?.(isQuarantine, res);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to query lot recall blast radius';
      setError(msg);
    } finally {
      setChecking(false);
    }
  };

  const handleQuarantineExecution = async () => {
    if (!report) return;
    setQuarantineExecuting(true);
    try {
      await foundryFactoryApi.executeAction({
        cell_id: report.cell_id,
        supplier_id: report.supplier_id,
        action_type: 'QUARANTINE_LOT_BATCH',
        actor_id: 'factory-qc-officer',
        actor_role: 'ROLE_FACTORY',
        idempotency_key: `quarantine-${report.contaminated_raw_material_lot}-${Date.now()}`,
        parameters: {
          marking: 'MARKING:LOT_RECALL_QUARANTINE',
          contaminated_raw_material_lot: report.contaminated_raw_material_lot,
          quarantined_sscc_pallets: report.quarantined_sscc_pallets,
          affected_work_orders: report.affected_work_orders,
        },
      });
      setQuarantinedConfirmed(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to execute quarantine action';
      setError(msg);
    } finally {
      setQuarantineExecuting(false);
    }
  };

  const isQuarantineActive =
    report &&
    (report.affected_work_orders.length > 0 ||
      report.affected_finished_lots.length > 0 ||
      report.quarantined_sscc_pallets.length > 0 ||
      report.quarantined_warehouse_totes.length > 0 ||
      report.in_transit_driver_drop_blocks.length > 0 ||
      report.retailer_reverse_pickup_orders.length > 0);

  if (dismissed && !isQuarantineActive) {
    return null;
  }

  return (
    <div className={`mb-6 rounded-xl border transition-all ${className} ${
      isQuarantineActive
        ? 'border-red-600 bg-red-950/40 text-red-100 shadow-lg shadow-red-950/30'
        : 'border-[var(--color-md-outline-variant,rgba(255,255,255,0.1))] bg-[var(--color-md-surface-container,rgba(30,30,30,0.8))] text-[var(--color-md-on-surface,#f1f5f9)]'
    }`}>
      {/* Header bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-inherit px-5 py-3">
        <div className="flex items-center gap-3">
          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${
            isQuarantineActive ? 'bg-red-600 text-white font-bold animate-pulse' : 'bg-amber-600/30 text-amber-400'
          }`}>
            {isQuarantineActive ? '⚠️' : '🛡️'}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-semibold text-sm">Raw Material Lot Recall Alert & Quarantine Gate</span>
              {isQuarantineActive && (
                <span className="rounded bg-red-700 px-2 py-0.5 text-xs font-mono font-bold tracking-wider text-white">
                  MARKING:LOT_RECALL_QUARANTINE
                </span>
              )}
            </div>
            <p className="text-xs opacity-75">
              Live Palantir Foundry multi-hop blast radius & contaminated batch isolation for factory bay operations.
            </p>
          </div>
        </div>

        {/* Rapid Lot Lookup Form */}
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="Scan / Enter Lot # (e.g. LOT-FLOUR-202610)"
            value={lotId}
            onChange={(e) => setLotId(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void handleCheckRecall(lotId);
            }}
            className="rounded border border-neutral-700 bg-neutral-900 px-3 py-1.5 text-xs text-white placeholder-neutral-500 focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500 w-56 font-mono"
          />
          <button
            type="button"
            onClick={() => void handleCheckRecall(lotId)}
            disabled={checking || !lotId.trim()}
            className="rounded bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50 transition"
          >
            {checking ? 'Scanning...' : 'Verify Recall'}
          </button>
        </div>
      </div>

      {/* Body / Active Alert Details */}
      {error && (
        <div className="px-5 py-3 text-xs bg-red-900/40 text-red-200 border-b border-red-800">
          <strong>Recall Query Error:</strong> {error}
        </div>
      )}

      {isQuarantineActive && report && (
        <div className="p-5 space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h4 className="text-sm font-bold text-red-300">
                CRITICAL CONTAMINATION QUARANTINE: {report.contaminated_raw_material_lot}
              </h4>
              <p className="text-xs text-red-200 mt-0.5">
                Cell: <span className="font-mono">{report.cell_id}</span> • Supplier: <span className="font-mono">{report.supplier_id}</span>
              </p>
            </div>
            <div className="flex items-center gap-2">
              {quarantinedConfirmed ? (
                <span className="rounded bg-emerald-800/80 px-3 py-1 text-xs font-semibold text-emerald-200 border border-emerald-600">
                  ✓ Batch Successfully Quarantined in WMS
                </span>
              ) : (
                <button
                  type="button"
                  onClick={() => void handleQuarantineExecution()}
                  disabled={quarantineExecuting}
                  className="rounded bg-red-600 hover:bg-red-500 px-4 py-1.5 text-xs font-bold text-white shadow transition disabled:opacity-50"
                >
                  {quarantineExecuting ? 'Executing Quarantine...' : 'Lock & Quarantine Pallet Batch'}
                </button>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
            <div className="rounded-lg bg-red-950/60 border border-red-800/60 p-2.5">
              <span className="text-red-400 font-medium">Work Orders</span>
              <p className="text-lg font-bold text-red-100">{report.affected_work_orders.length}</p>
            </div>
            <div className="rounded-lg bg-red-950/60 border border-red-800/60 p-2.5">
              <span className="text-red-400 font-medium">Finished Lots</span>
              <p className="text-lg font-bold text-red-100">{report.affected_finished_lots.length}</p>
            </div>
            <div className="rounded-lg bg-red-950/60 border border-red-800/60 p-2.5">
              <span className="text-red-400 font-medium">SSCC Pallets</span>
              <p className="text-lg font-bold text-red-100">{report.quarantined_sscc_pallets.length}</p>
            </div>
            <div className="rounded-lg bg-red-950/60 border border-red-800/60 p-2.5">
              <span className="text-red-400 font-medium">Transit Drop Blocks</span>
              <p className="text-lg font-bold text-red-100">{report.in_transit_driver_drop_blocks.length}</p>
            </div>
          </div>

          {report.quarantined_sscc_pallets.length > 0 && (
            <div className="rounded-lg bg-neutral-900/80 p-3 text-xs border border-red-900/60">
              <span className="font-semibold text-neutral-300">Quarantined SSCC Pallet Codes:</span>
              <div className="mt-1 flex flex-wrap gap-1.5 max-h-24 overflow-y-auto">
                {report.quarantined_sscc_pallets.map((id: string) => (
                  <span key={id} className="rounded bg-red-950 px-2 py-0.5 font-mono text-red-300 border border-red-800 text-[11px]">
                    {id}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {!isQuarantineActive && report && (
        <div className="flex items-center justify-between px-5 py-3 text-xs bg-emerald-950/40 text-emerald-300 border-t border-emerald-800/40">
          <div className="flex items-center gap-2">
            <span>✓</span>
            <span>
              Lot <strong className="font-mono">{report.contaminated_raw_material_lot}</strong> cleared. No active recall quarantine alerts found.
            </span>
          </div>
          <button
            type="button"
            onClick={() => setDismissed(true)}
            className="text-xs text-neutral-400 hover:text-white"
          >
            Dismiss
          </button>
        </div>
      )}
    </div>
  );
}
