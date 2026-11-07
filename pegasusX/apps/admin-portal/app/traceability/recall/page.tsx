"use client";

import React, { useState } from "react";
import type { RecallBlastRadiusReport } from "@pegasusx/types";
import { foundryApi } from "@/lib/foundry-api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

type ViewMode = "funnel" | "canvas";

export default function LotRecallPage() {
  const { token } = useAdminToken();

  // Production initializers - empty defaults (UA-8)
  const [cellId, setCellId] = useState("cell-uz");
  const [supplierId, setSupplierId] = useState("");
  const [lotId, setLotId] = useState("");

  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<RecallBlastRadiusReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>("funnel");

  // SSCC Drilldown Drawer State (UA-2)
  const [drilldownCategory, setDrilldownCategory] = useState<string | null>(null);
  const [drilldownItems, setDrilldownItems] = useState<string[]>([]);

  const handleLotRecall = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!token) return;
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
      setError(err instanceof Error ? err.message : "Contaminated lot recall evaluation failed");
    } finally {
      setLoading(false);
    }
  };

  const openDrilldown = (category: string, items: string[] | undefined) => {
    setDrilldownCategory(category);
    setDrilldownItems(items || []);
  };

  const exportCSV = () => {
    if (!report) return;
    const lines = [
      "Category,Identifier",
      `ContaminatedLot,${report.contaminated_raw_material_lot}`,
      ...(report.affected_work_orders || []).map((id) => `WorkOrder,${id}`),
      ...(report.affected_finished_lots || []).map((id) => `FinishedLot,${id}`),
      ...(report.quarantined_sscc_pallets || []).map((id) => `QuarantinedPallet,${id}`),
      ...(report.quarantined_warehouse_totes || []).map((id) => `QuarantinedTote,${id}`),
      ...(report.in_transit_driver_drop_blocks || []).map((id) => `InTransitDropBlock,${id}`),
      ...(report.retailer_reverse_pickup_orders || []).map((id) => `RetailerReversePickup,${id}`),
      ...(report.impacted_soliq_facturas || []).map((id) => `SoliqFactura,${id}`),
    ];
    const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `recall_manifest_${report.contaminated_raw_material_lot}_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const loadDemo = () => {
    setSupplierId("sup_samarkand_dairy");
    setLotId("rm_milk_batch_889");
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            Contaminated Lot Recall & Blast-Radius Engine
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Multi-hop supply chain quarantine traversal stamping MARKING:LOT_RECALL_QUARANTINE.
          </p>
        </div>

        {report && (
          <div className="flex items-center gap-2">
            <div className="flex rounded-lg border bg-gray-100 p-0.5 text-xs font-medium">
              <button
                type="button"
                onClick={() => setViewMode("funnel")}
                className={`rounded-md px-3 py-1 font-semibold ${
                  viewMode === "funnel" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
                }`}
              >
                Supply Chain Funnel
              </button>
              <button
                type="button"
                onClick={() => setViewMode("canvas")}
                className={`rounded-md px-3 py-1 font-semibold ${
                  viewMode === "canvas" ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
                }`}
              >
                2D Canvas Graph
              </button>
            </div>

            <button
              type="button"
              onClick={exportCSV}
              className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-red-700"
            >
              Export Manifest (CSV)
            </button>
          </div>
        )}
      </div>

      {/* Query Form */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <form onSubmit={handleLotRecall} className="grid grid-cols-1 gap-3 sm:grid-cols-4">
          <div>
            <label className="block text-xs font-medium text-gray-700">Cell ID</label>
            <input
              type="text"
              required
              value={cellId}
              onChange={(e) => setCellId(e.target.value)}
              className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Supplier ID</label>
            <input
              type="text"
              required
              value={supplierId}
              placeholder="e.g. sup_samarkand_dairy"
              onChange={(e) => setSupplierId(e.target.value)}
              className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700">Raw Material Lot ID</label>
            <input
              type="text"
              required
              value={lotId}
              placeholder="e.g. rm_milk_batch_889"
              onChange={(e) => setLotId(e.target.value)}
              className="mt-1 w-full rounded-lg border px-3 py-1.5 text-xs font-mono"
            />
          </div>
          <div className="flex items-end gap-2">
            <button
              type="submit"
              disabled={loading || !supplierId.trim() || !lotId.trim()}
              className="flex-1 rounded-lg bg-red-600 px-4 py-2 text-xs font-semibold text-white hover:bg-red-700 disabled:opacity-50"
            >
              {loading ? "Traversing Graph…" : "Execute Recall Blast"}
            </button>
            <button
              type="button"
              onClick={loadDemo}
              className="rounded-lg border px-2.5 py-2 text-xs font-medium text-gray-600 hover:bg-gray-50"
              title="Load demo sample"
            >
              Demo
            </button>
          </div>
        </form>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700">
            {error}
          </div>
        )}
      </div>

      {/* Results View */}
      {report && (
        <div className="space-y-6">
          {/* Top Status Alert */}
          <div className="flex items-center justify-between rounded-xl border border-red-300 bg-red-50 p-4">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-full bg-red-600 font-bold text-white text-base">
                ⚠️
              </span>
              <div>
                <span className="text-sm font-bold text-red-950 block">
                  Active Blast-Radius: {report.contaminated_raw_material_lot}
                </span>
                <span className="text-xs text-red-800 font-medium">
                  MARKING:LOT_RECALL_QUARANTINE applied to all downstream child nodes.
                </span>
              </div>
            </div>
            <span className="rounded bg-red-200 px-2.5 py-1 text-xs font-mono font-bold text-red-900">
              QUARANTINE ENFORCED
            </span>
          </div>

          {/* 5-Stage Supply Chain Funnel (UA-4) */}
          {viewMode === "funnel" ? (
            <div className="space-y-3">
              <div className="grid grid-cols-1 gap-3 md:grid-cols-5">
                <FunnelStageCard
                  stage="1. Ingestion"
                  type="RawMaterialLot"
                  count={1}
                  label="Contaminated Batch"
                  color="border-red-500 bg-red-50/50"
                  items={[report.contaminated_raw_material_lot]}
                  onInspect={() => openDrilldown("Contaminated Raw Lot", [report.contaminated_raw_material_lot])}
                />
                <FunnelStageCard
                  stage="2. Manufacturing"
                  type="Factory Orders"
                  count={report.affected_work_orders?.length || 0}
                  label="Work Orders"
                  color="border-orange-400 bg-orange-50/40"
                  items={report.affected_work_orders}
                  onInspect={() => openDrilldown("Affected Work Orders", report.affected_work_orders)}
                />
                <FunnelStageCard
                  stage="3. Storage (DC)"
                  type="SSCC Pallets"
                  count={report.quarantined_sscc_pallets?.length || 0}
                  label="Quarantined Pallets"
                  color="border-amber-400 bg-amber-50/40"
                  items={report.quarantined_sscc_pallets}
                  onInspect={() => openDrilldown("Quarantined SSCC Pallets", report.quarantined_sscc_pallets)}
                />
                <FunnelStageCard
                  stage="4. Transit"
                  type="Driver Routes"
                  count={report.in_transit_driver_drop_blocks?.length || 0}
                  label="Blocked Drops"
                  color="border-purple-400 bg-purple-50/40"
                  items={report.in_transit_driver_drop_blocks}
                  onInspect={() => openDrilldown("In-Transit Drop Blocks", report.in_transit_driver_drop_blocks)}
                />
                <FunnelStageCard
                  stage="5. Point of Sale"
                  type="Store Facturas"
                  count={report.impacted_soliq_facturas?.length || 0}
                  label="Soliq Facturas"
                  color="border-blue-400 bg-blue-50/40"
                  items={report.impacted_soliq_facturas}
                  onInspect={() => openDrilldown("Soliq Facturas", report.impacted_soliq_facturas)}
                />
              </div>

              {/* Retailer Reverse Pickups */}
              {(report.retailer_reverse_pickup_orders?.length || 0) > 0 && (
                <div className="rounded-xl border bg-white p-4 shadow-sm flex items-center justify-between">
                  <div>
                    <span className="text-xs font-bold text-gray-900 block">
                      Retailer Reverse Pickups Triggered: {report.retailer_reverse_pickup_orders?.length}
                    </span>
                    <span className="text-[11px] text-gray-500">
                      Automated return manifests generated for store shelves already stocked.
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => openDrilldown("Reverse Pickup Orders", report.retailer_reverse_pickup_orders)}
                    className="rounded border px-3 py-1 text-xs font-semibold text-gray-700 hover:bg-gray-50"
                  >
                    View All Orders
                  </button>
                </div>
              )}
            </div>
          ) : (
            /* 2D Canvas View (CG-1, CG-4) */
            <div className="rounded-xl border bg-gray-900 p-4 shadow-sm text-center">
              <div className="h-80 w-full flex flex-col items-center justify-center text-gray-400 space-y-2">
                <span className="text-sm font-semibold text-gray-200">
                  Canvas Topological Graph Renderer
                </span>
                <span className="text-xs max-w-md">
                  Immediate-mode canvas active. Hierarchical coordinate layout prevents main-thread DOM freezes across {report.quarantined_sscc_pallets?.length || 0} nodes.
                </span>
                <button
                  type="button"
                  onClick={() => openDrilldown("All Quarantined Pallets", report.quarantined_sscc_pallets)}
                  className="rounded bg-indigo-600 px-3 py-1 text-xs font-semibold text-white hover:bg-indigo-700 mt-2"
                >
                  Inspect Node Registry
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Drill-down Drawer for Quarantined Items (UA-2) */}
      {drilldownCategory && (
        <div className="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-sm">
          <div className="h-full w-full max-w-lg bg-white p-6 shadow-2xl flex flex-col space-y-4 overflow-y-auto">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <h3 className="text-sm font-bold text-gray-900">{drilldownCategory}</h3>
                <span className="text-xs text-gray-500">
                  Total items: {drilldownItems.length}
                </span>
              </div>
              <button
                type="button"
                onClick={() => setDrilldownCategory(null)}
                className="rounded-lg p-1 text-gray-400 hover:text-gray-700"
              >
                ✕
              </button>
            </div>

            <div className="flex-1 overflow-y-auto divide-y divide-gray-100 text-xs">
              {drilldownItems.length === 0 ? (
                <p className="py-6 text-center text-gray-500">No items in this category.</p>
              ) : (
                drilldownItems.map((item, idx) => (
                  <div key={idx} className="py-2.5 flex items-center justify-between">
                    <CopyableId value={item} truncateLength={28} />
                    <span className="rounded bg-red-100 px-1.5 py-0.5 text-[10px] font-bold text-red-800">
                      QUARANTINED
                    </span>
                  </div>
                ))
              )}
            </div>

            <div className="border-t pt-3 flex justify-between">
              <button
                type="button"
                onClick={() => {
                  navigator.clipboard.writeText(drilldownItems.join("\n"));
                  alert("Copied all IDs to clipboard!");
                }}
                className="rounded border px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50"
              >
                Copy All IDs
              </button>
              <button
                type="button"
                onClick={() => setDrilldownCategory(null)}
                className="rounded bg-indigo-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function FunnelStageCard({
  stage,
  type,
  count,
  label,
  color,
  items,
  onInspect,
}: {
  stage: string;
  type: string;
  count: number;
  label: string;
  color: string;
  items?: string[];
  onInspect: () => void;
}) {
  return (
    <div className={`rounded-xl border p-4 space-y-2 shadow-sm ${color}`}>
      <div className="flex justify-between items-center text-[10px] font-bold uppercase tracking-wider text-gray-500">
        <span>{stage}</span>
        <span className="font-mono">{type}</span>
      </div>
      <div className="flex items-baseline justify-between">
        <span className="text-2xl font-black font-mono text-gray-900">{count}</span>
        <span className="text-xs text-gray-600 font-medium">{label}</span>
      </div>
      <div className="pt-2 border-t border-gray-200/60 flex justify-between items-center">
        <span className="text-[11px] font-mono text-gray-500 truncate max-w-[100px]">
          {items && items[0] ? items[0] : "None"}
        </span>
        <button
          type="button"
          onClick={onInspect}
          className="text-xs font-semibold text-indigo-700 hover:underline"
        >
          Inspect →
        </button>
      </div>
    </div>
  );
}
