"use client";

import React, { useEffect, useState, useMemo } from "react";
import { api, type BillingInvoice, type BillingFeeSchedule } from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";

type PerspectiveMode = "OVERALL" | "SUPPLIER" | "WAREHOUSE" | "RETAILER";

const DEMO_SUPPLIERS = [
  { id: "sup_tashkent_fmcg", name: "Tashkent FMCG Distribution LLC", tier: "ENTERPRISE", gmvUZS: 820000000, feeBps: 450, unpaidMinor: 2450000000 },
  { id: "sup_samarkand_dairy", name: "Samarkand Dairy Products JV", tier: "VOLUME", gmvUZS: 410000000, feeBps: 550, unpaidMinor: 1120000000 },
  { id: "sup_fergana_agro", name: "Fergana Agro Logistics FE", tier: "STANDARD", gmvUZS: 190000000, feeBps: 650, unpaidMinor: 480000000 },
];

const DEMO_WAREHOUSES = [
  { id: "depot_tashkent_central", name: "Tashkent Central Super-Depot", capacityDm3: 1200000, usedDm3: 940000, vaultCashUZS: 45200000, varianceScore: 0.02 },
  { id: "depot_samarkand_hub", name: "Samarkand Regional Hub", capacityDm3: 650000, usedDm3: 480000, vaultCashUZS: 21800000, varianceScore: 0.04 },
  { id: "depot_fergana_depot", name: "Fergana Valley Cross-Dock", capacityDm3: 400000, usedDm3: 210000, vaultCashUZS: 12400000, varianceScore: 0.01 },
];

const DEMO_RETAILERS = [
  { id: "ret_korzinka_01", tin: "301234567", name: "Korzinka Supermarket #01 (Chilanzar)", creditLimitUZS: 100000000, creditUsedUZS: 62000000, amlCashTodayUZS: 14500000, rtiCrateDepositUZS: 3200000 },
  { id: "ret_makro_44", tin: "308765432", name: "Makro Express #44 (Yunusabad)", creditLimitUZS: 80000000, creditUsedUZS: 41000000, amlCashTodayUZS: 19800000, rtiCrateDepositUZS: 2100000 },
  { id: "ret_baraka_market", tin: "305544332", name: "Baraka Grocery Corner (Mirzo Ulugbek)", creditLimitUZS: 30000000, creditUsedUZS: 18000000, amlCashTodayUZS: 24200000, rtiCrateDepositUZS: 850000 },
];

export default function FinOpsBillingPage() {
  const { token } = useAdminToken();

  const [mode, setMode] = useState<PerspectiveMode>("OVERALL");
  const [selectedSupplierId, setSelectedSupplierId] = useState(DEMO_SUPPLIERS[0].id);
  const [selectedWarehouseId, setSelectedWarehouseId] = useState(DEMO_WAREHOUSES[0].id);
  const [selectedRetailerId, setSelectedRetailerId] = useState(DEMO_RETAILERS[0].id);

  const [invoices, setInvoices] = useState<BillingInvoice[]>([]);
  const [schedules, setSchedules] = useState<BillingFeeSchedule[]>([]);
  const [loading, setLoading] = useState(false);
  const [runningBilling, setRunningBilling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const fetchBillingData = async () => {
    if (!token) return;
    setError(null);
    setLoading(true);
    try {
      const [invRes, schRes] = await Promise.all([
        api.listBillingInvoices(token),
        api.listBillingFeeSchedules(token),
      ]);
      setInvoices(invRes.invoices || []);
      setSchedules(schRes.fee_schedules || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load billing data");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBillingData();
  }, [token]);

  const handleRunMonthly = async () => {
    if (!token) return;
    setRunningBilling(true);
    setError(null);
    setSuccessMsg(null);
    try {
      const res = await api.runMonthlyBilling(token);
      setSuccessMsg(`Monthly billing cycle executed successfully! Billed ${res.billed} invoices for period ${res.month}.`);
      await fetchBillingData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Monthly billing run failed");
    } finally {
      setRunningBilling(false);
    }
  };

  // Macro Metrics (Derived & Aggregate)
  const totalGMVUZS = useMemo(() => {
    return DEMO_SUPPLIERS.reduce((acc, s) => acc + s.gmvUZS, 0);
  }, []);

  const platformTakeRateUZS = useMemo(() => {
    if (schedules.length > 0 && schedules[0].GmvBps) {
      return (totalGMVUZS * schedules[0].GmvBps) / 10000;
    }
    return (totalGMVUZS * 0.05); // 500 bps default dynamic fee
  }, [totalGMVUZS, schedules]);

  const soliqVatUZS = (totalGMVUZS * 0.12); // Dynamic 12% Soliq VAT
  const logisticsAllocationUZS = (totalGMVUZS * 0.025);
  const netSupplierPayoutUZS = totalGMVUZS - platformTakeRateUZS - soliqVatUZS - logisticsAllocationUZS;
  const cloudInfraCostUZS = 24500000; // Cloud Spanner, K8s, Redis, Kafka compute cost

  const activeSupplier = DEMO_SUPPLIERS.find((s) => s.id === selectedSupplierId) || DEMO_SUPPLIERS[0];
  const activeWarehouse = DEMO_WAREHOUSES.find((w) => w.id === selectedWarehouseId) || DEMO_WAREHOUSES[0];
  const activeRetailer = DEMO_RETAILERS.find((r) => r.id === selectedRetailerId) || DEMO_RETAILERS[0];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-gray-900">
            FinOps, Multi-Participant Billing & Unit Economics
          </h1>
          <p className="text-xs text-gray-500 mt-0.5">
            Enterprise multi-tier financial alignment inspired by SAP S/4HANA TM &amp; o9 Solutions Digital Brain.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={handleRunMonthly}
            disabled={runningBilling || !token}
            className="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-bold text-white shadow hover:bg-emerald-700 disabled:opacity-50"
          >
            {runningBilling ? "Running Cycle..." : "Run Monthly Billing Cycle"}
          </button>
          <button
            type="button"
            onClick={fetchBillingData}
            disabled={loading}
            className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 shadow-sm hover:bg-gray-50"
          >
            {loading ? "Refreshing..." : "Refresh"}
          </button>
        </div>
      </div>

      {/* Notifications */}
      {error && (
        <div className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800">
          <strong>FinOps Error:</strong> {error}
        </div>
      )}
      {successMsg && (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800">
          ✓ {successMsg}
        </div>
      )}

      {/* Perspective Dimension Selector (Inspired by o9 Solutions Multi-Tier EKG) */}
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border bg-white p-3 shadow-sm">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold uppercase tracking-wider text-gray-500">
            Perspective:
          </span>
          <div className="flex rounded-lg border bg-gray-100 p-0.5 text-xs font-semibold">
            {(["OVERALL", "SUPPLIER", "WAREHOUSE", "RETAILER"] as PerspectiveMode[]).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setMode(p)}
                className={`rounded-md px-3 py-1 transition-colors ${
                  mode === p ? "bg-white text-gray-900 shadow-sm" : "text-gray-600 hover:text-gray-900"
                }`}
              >
                {p === "OVERALL" ? "Overall Ecosystem" : p.charAt(0) + p.slice(1).toLowerCase()}
              </button>
            ))}
          </div>
        </div>

        {/* Entity Selectors */}
        {mode === "SUPPLIER" && (
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-gray-600">Select Supplier:</label>
            <select
              value={selectedSupplierId}
              onChange={(e) => setSelectedSupplierId(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-semibold"
            >
              {DEMO_SUPPLIERS.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} ({s.tier})
                </option>
              ))}
            </select>
          </div>
        )}

        {mode === "WAREHOUSE" && (
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-gray-600">Select Depot:</label>
            <select
              value={selectedWarehouseId}
              onChange={(e) => setSelectedWarehouseId(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-semibold"
            >
              {DEMO_WAREHOUSES.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </div>
        )}

        {mode === "RETAILER" && (
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-gray-600">Select Retailer:</label>
            <select
              value={selectedRetailerId}
              onChange={(e) => setSelectedRetailerId(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-semibold"
            >
              {DEMO_RETAILERS.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name} (TIN: {r.tin})
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* OVERALL ECOSYSTEM VIEW */}
      {mode === "OVERALL" && (
        <div className="space-y-6">
          {/* Key Metric Pulse Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Gross Invoiced GMV</span>
              <div className="text-xl font-extrabold text-gray-900 mt-1">
                {(totalGMVUZS / 1000000).toFixed(1)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-emerald-600 mt-1 font-semibold">
                ↑ 14.8% vs last 30 days
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Platform Take Rate</span>
              <div className="text-xl font-extrabold text-indigo-700 mt-1">
                {(platformTakeRateUZS / 1000000).toFixed(2)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-gray-500 mt-1">
                Effective blended: <span className="font-semibold text-gray-700">5.20% (520 bps)</span>
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Soliq 12% VAT Accrued</span>
              <div className="text-xl font-extrabold text-gray-900 mt-1">
                {(soliqVatUZS / 1000000).toFixed(1)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-gray-500 mt-1">
                100% electronic factura registered
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Cloud FinOps Cost</span>
              <div className="text-xl font-extrabold text-emerald-700 mt-1">
                {(cloudInfraCostUZS / 1000000).toFixed(2)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-emerald-700 font-semibold mt-1">
                Cost-to-Serve: 1.72% of GMV
              </div>
            </div>
          </div>

          {/* Tender Waterfall Visualization (Sankey-Style Flow) */}
          <div className="rounded-xl border bg-white p-6 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <h2 className="text-sm font-bold text-gray-900">
                  Ecosystem Value &amp; Settlement Waterfall
                </h2>
                <p className="text-xs text-gray-500">
                  Deterministic breakdown of 100% captured GMV across settlement slices, tax obligations, and infrastructure.
                </p>
              </div>
              <span className="rounded bg-indigo-50 px-2.5 py-1 text-xs font-bold text-indigo-700">
                Gross GMV: {totalGMVUZS.toLocaleString()} UZS
              </span>
            </div>

            {/* Visual Waterfall Stack */}
            <div className="space-y-3">
              <div className="h-6 w-full rounded-lg bg-gray-100 overflow-hidden flex shadow-inner">
                <div style={{ width: `${Math.max(5, (netSupplierPayoutUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%` }} className="bg-indigo-600 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Supplier Net Settlement: ${((netSupplierPayoutUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%`}>
                  Supplier ({((netSupplierPayoutUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%)
                </div>
                <div style={{ width: `${Math.max(5, (soliqVatUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%` }} className="bg-amber-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Soliq VAT: ${((soliqVatUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%`}>
                  VAT ({((soliqVatUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%)
                </div>
                <div style={{ width: `${Math.max(5, (platformTakeRateUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%` }} className="bg-emerald-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Platform Take Rate: ${((platformTakeRateUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%`}>
                  Take ({((platformTakeRateUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%)
                </div>
                <div style={{ width: `${Math.max(5, (logisticsAllocationUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%` }} className="bg-purple-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Logistics & Fleet: ${((logisticsAllocationUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%`}>
                  Fleet ({((logisticsAllocationUZS / (totalGMVUZS || 1)) * 100).toFixed(1)}%)
                </div>
              </div>

              {/* Waterfall Legend & Row Details */}
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 pt-2">
                <div className="rounded-lg border border-indigo-100 bg-indigo-50/40 p-3">
                  <div className="flex items-center gap-1.5 text-xs font-bold text-indigo-900">
                    <span className="h-2 w-2 rounded-full bg-indigo-600" />
                    Supplier Remittance
                  </div>
                  <div className="text-base font-extrabold text-gray-900 mt-1">
                    {netSupplierPayoutUZS.toLocaleString()} UZS
                  </div>
                  <div className="text-[11px] text-gray-500">Batched post-e-factura confirmation</div>
                </div>

                <div className="rounded-lg border border-amber-100 bg-amber-50/40 p-3">
                  <div className="flex items-center gap-1.5 text-xs font-bold text-amber-900">
                    <span className="h-2 w-2 rounded-full bg-amber-500" />
                    State Tax (Soliq VAT)
                  </div>
                  <div className="text-base font-extrabold text-gray-900 mt-1">
                    {soliqVatUZS.toLocaleString()} UZS
                  </div>
                  <div className="text-[11px] text-gray-500">12% statutory corporate tax sweep</div>
                </div>

                <div className="rounded-lg border border-emerald-100 bg-emerald-50/40 p-3">
                  <div className="flex items-center gap-1.5 text-xs font-bold text-emerald-900">
                    <span className="h-2 w-2 rounded-full bg-emerald-500" />
                    Platform Operating Revenue
                  </div>
                  <div className="text-base font-extrabold text-gray-900 mt-1">
                    {platformTakeRateUZS.toLocaleString()} UZS
                  </div>
                  <div className="text-[11px] text-gray-500">GMV bps + order metering fees</div>
                </div>

                <div className="rounded-lg border border-purple-100 bg-purple-50/40 p-3">
                  <div className="flex items-center gap-1.5 text-xs font-bold text-purple-900">
                    <span className="h-2 w-2 rounded-full bg-purple-500" />
                    Fleet Logistics Allocation
                  </div>
                  <div className="text-base font-extrabold text-gray-900 mt-1">
                    {(totalGMVUZS * 0.025).toLocaleString()} UZS
                  </div>
                  <div className="text-[11px] text-gray-500">Carrier per-drop & fuel surcharge</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* SUPPLIER PERSPECTIVE */}
      {mode === "SUPPLIER" && (
        <div className="space-y-6">
          <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-indigo-600">Supplier Ledger</span>
                <h2 className="text-lg font-bold text-gray-900">{activeSupplier.name}</h2>
                <div className="text-xs text-gray-500">ID: <CopyableId id={activeSupplier.id} label="Supplier" /></div>
              </div>

              <div className="text-right">
                <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs font-bold text-indigo-800">
                  Tier: {activeSupplier.tier}
                </span>
                <div className="text-xs text-gray-500 mt-1">Platform Fee: {activeSupplier.feeBps} bps ({(activeSupplier.feeBps / 100).toFixed(2)}%)</div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Monthly GMV</div>
                <div className="text-lg font-extrabold text-gray-900 mt-1">
                  {activeSupplier.gmvUZS.toLocaleString()} UZS
                </div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Platform Invoiced Fees</div>
                <div className="text-lg font-extrabold text-indigo-700 mt-1">
                  {((activeSupplier.gmvUZS * activeSupplier.feeBps) / 10000).toLocaleString()} UZS
                </div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Unsettled Escrow Balance</div>
                <div className="text-lg font-extrabold text-amber-700 mt-1">
                  {(activeSupplier.unpaidMinor / 100).toLocaleString()} UZS
                </div>
                <div className="text-[10px] text-gray-400 mt-0.5">Holding in rolling damage reserve</div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* WAREHOUSE / DEPOT PERSPECTIVE */}
      {mode === "WAREHOUSE" && (
        <div className="space-y-6">
          <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-indigo-600">Depot Operations &amp; Safe Vault</span>
                <h2 className="text-lg font-bold text-gray-900">{activeWarehouse.name}</h2>
                <div className="text-xs text-gray-500">ID: <CopyableId id={activeWarehouse.id} label="Depot" /></div>
              </div>

              <div className="text-right">
                <div className="text-xs font-bold text-gray-500 uppercase">Cash Turn-In Variance Score</div>
                <div className={`text-base font-extrabold ${activeWarehouse.varianceScore < 0.03 ? "text-emerald-700" : "text-amber-700"}`}>
                  {(activeWarehouse.varianceScore * 100).toFixed(2)}% (Low Shrinkage)
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Physical Cash in Safe Vault</div>
                <div className="text-xl font-extrabold text-emerald-800 mt-1">
                  {activeWarehouse.vaultCashUZS.toLocaleString()} UZS
                </div>
                <div className="text-[11px] text-gray-500 mt-0.5">Cleared from completed blind counts</div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Storage Capacity Utilization</div>
                <div className="text-lg font-extrabold text-gray-900 mt-1">
                  {((activeWarehouse.usedDm3 / activeWarehouse.capacityDm3) * 100).toFixed(1)}%
                </div>
                <div className="text-[11px] text-gray-500 mt-0.5">
                  {(activeWarehouse.usedDm3 / 1000).toLocaleString()} / {(activeWarehouse.capacityDm3 / 1000).toLocaleString()} m³
                </div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Daily CIT Handover Readiness</div>
                <div className="text-sm font-bold text-indigo-700 mt-1">
                  READY FOR ARMORED CAR
                </div>
                <div className="text-[11px] text-gray-500 mt-0.5">Safe balance exceeds 10M UZS pickup threshold</div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* RETAILER PERSPECTIVE */}
      {mode === "RETAILER" && (
        <div className="space-y-6">
          <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <span className="text-[10px] font-bold uppercase tracking-wider text-indigo-600">Retailer Commercial Account</span>
                <h2 className="text-lg font-bold text-gray-900">{activeRetailer.name}</h2>
                <div className="text-xs text-gray-500 flex items-center gap-2 mt-0.5">
                  <span>TIN: {activeRetailer.tin}</span>
                  <span>•</span>
                  <CopyableId id={activeRetailer.id} label="Retailer" />
                </div>
              </div>

              <div className="text-right">
                <span className="rounded bg-blue-100 px-2.5 py-1 text-xs font-bold text-blue-800">
                  Credit Health: EXCELLENT
                </span>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Trade Credit Balance (Uncapped)</div>
                <div className="text-lg font-extrabold text-gray-900 mt-1">
                  {activeRetailer.creditUsedUZS.toLocaleString()} UZS
                </div>
                <div className="text-[11px] text-emerald-600 font-semibold mt-1">
                  Uncapped Open Credit Terms
                </div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Daily Cash Settlements (Uncapped)</div>
                <div className="text-lg font-extrabold text-gray-900 mt-1">
                  {activeRetailer.amlCashTodayUZS.toLocaleString()} UZS
                </div>
                <div className="text-[11px] text-emerald-600 font-semibold mt-1">
                  Uncapped Multi-Tender Policy Active
                </div>
              </div>

              <div className="rounded-lg border bg-gray-50 p-4">
                <div className="text-xs font-medium text-gray-500">Returnable RTI Crate Deposit</div>
                <div className="text-lg font-extrabold text-indigo-700 mt-1">
                  {activeRetailer.rtiCrateDepositUZS.toLocaleString()} UZS
                </div>
                <div className="text-[11px] text-gray-500 mt-0.5">Refunded on empty container pickup</div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Invoices & Fee Schedules Section */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-4">
        <div className="flex items-center justify-between border-b pb-3">
          <h3 className="text-xs font-bold uppercase tracking-wider text-gray-500">
            Platform Invoices &amp; Settlement Batches
          </h3>
          <span className="text-xs text-gray-500 font-medium">{invoices.length} Invoices</span>
        </div>

        {invoices.length === 0 ? (
          <div className="rounded-lg border border-dashed p-8 text-center text-xs text-gray-400">
            {loading ? "Loading invoices from Cloud Spanner..." : "No invoices found for current billing window."}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-left text-xs">
              <thead className="border-b bg-gray-50 uppercase tracking-wider text-gray-500 font-bold">
                <tr>
                  <th className="px-4 py-2.5">Invoice ID</th>
                  <th className="px-4 py-2.5">Supplier</th>
                  <th className="px-4 py-2.5">Order Ref</th>
                  <th className="px-4 py-2.5">Status</th>
                  <th className="px-4 py-2.5">Principal</th>
                  <th className="px-4 py-2.5">Balance Due</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {invoices.map((inv) => (
                  <tr key={inv.invoice_id} className="hover:bg-gray-50">
                    <td className="px-4 py-2 font-mono">
                      <CopyableId id={inv.invoice_id} label="Invoice" />
                    </td>
                    <td className="px-4 py-2 font-mono text-gray-700">{inv.billed_supplier_id}</td>
                    <td className="px-4 py-2 font-mono text-gray-500">{inv.order_id || "-"}</td>
                    <td className="px-4 py-2">
                      <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                        inv.status === "PAID" ? "bg-emerald-100 text-emerald-800" : "bg-amber-100 text-amber-800"
                      }`}>
                        {inv.status}
                      </span>
                    </td>
                    <td className="px-4 py-2 font-mono font-semibold">{(inv.principal_minor / 100).toLocaleString()} UZS</td>
                    <td className="px-4 py-2 font-mono text-rose-600 font-bold">{(inv.balance_minor / 100).toLocaleString()} UZS</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
