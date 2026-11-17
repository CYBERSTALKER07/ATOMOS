"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  type BillingFeeSchedule,
  type BillingInvoice,
  type FinanceAccountLedgerRow,
  type FinanceReconciliationBreak,
  type FinanceReconciliationRun,
  type FinanceTrialBalanceReport,
  type ReconDimension,
  type Tenant,
} from "@/lib/api";
import { useAdminToken } from "@/lib/session";
import { CopyableId } from "@/components/ui/CopyableId";
import {
  applyBpsFee,
  formatBpsPercent,
  formatMinorUnits,
  formatRatioPercentTenths,
} from "@/lib/money";

const RECON_DIMENSIONS: readonly ReconDimension[] = [
  "ORDERS_AR",
  "STOCK_GL",
  "CASH_PSP",
  "TAX_FISCAL",
];

function currentTashkentPeriod(): string {
  try {
    const parts = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Tashkent",
      year: "numeric",
      month: "2-digit",
    }).formatToParts(new Date());
    const y = parts.find((p) => p.type === "year")?.value;
    const m = parts.find((p) => p.type === "month")?.value;
    if (y && m) return `${y}-${m}`;
  } catch {
    // fallback below
  }
  return new Date().toISOString().slice(0, 7);
}

function formatFinOpsMinor(minor: number | bigint | null | undefined, currency = "UZS", signDisplay: "auto" | "always" = "auto"): string {
  const safe = typeof minor === "bigint" ? minor : BigInt(Math.trunc(Number.isFinite(minor ?? 0) ? (minor ?? 0) : 0));
  const abs = safe < BigInt(0) ? -safe : safe;
  const minFrac: 0 | 2 = abs % BigInt(100) === BigInt(0) ? 0 : 2;
  return formatMinorUnits(safe, {
    currency: (currency || "UZS").toUpperCase(),
    minimumFractionDigits: minFrac,
    maximumFractionDigits: 2,
    signDisplay,
  });
}

function formatMillionsUzs(amountUZS: number, fractionDigits: 1 | 2): string {
  if (amountUZS <= 0) return "0.0";
  const hundredthsOfMillion = BigInt(Math.trunc(amountUZS)) / BigInt(10000);
  return formatMinorUnits(hundredthsOfMillion, {
    groupSeparator: "",
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  });
}

type PerspectiveMode = "OVERALL" | "SUPPLIER" | "WAREHOUSE" | "RETAILER";

interface DerivedSupplier {
  id: string;
  name: string;
  tier: string;
  gmvUZS: number;
  feeBps: number;
  unpaidMinor: number;
}

export default function FinOpsBillingPage() {
  const { token } = useAdminToken();

  const [mode, setMode] = useState<PerspectiveMode>("OVERALL");
  const [selectedSupplierId, setSelectedSupplierId] = useState<string>("");

  const [invoices, setInvoices] = useState<BillingInvoice[]>([]);
  const [schedules, setSchedules] = useState<BillingFeeSchedule[]>([]);
  const [supplierTenants, setSupplierTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [runningBilling, setRunningBilling] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  // M0.2 GL Platform Fee & Trial Balance + M0.3 4-Way Reconciliation Health state
  const [glEntityId, setGlEntityId] = useState<string>("");
  const [glPeriod, setGlPeriod] = useState<string>(() => currentTashkentPeriod());
  const [trialBalance, setTrialBalance] = useState<FinanceTrialBalanceReport | null>(null);
  const [platformFeeLedger, setPlatformFeeLedger] = useState<FinanceAccountLedgerRow[]>([]);
  const [reconRuns, setReconRuns] = useState<FinanceReconciliationRun[]>([]);
  const [reconBreaks, setReconBreaks] = useState<FinanceReconciliationBreak[]>([]);
  const [glLoading, setGlLoading] = useState(false);
  const [glError, setGlError] = useState<string | null>(null);
  const [runningRecon, setRunningRecon] = useState(false);
  const [busyBreakId, setBusyBreakId] = useState<string | null>(null);

  const fetchBillingData = useCallback(async () => {
    if (!token) return;
    setError(null);
    setLoading(true);
    try {
      const [invRes, schRes, tenRes] = await Promise.allSettled([
        api.listBillingInvoices(token),
        api.listBillingFeeSchedules(token),
        api.listTenants(token, "", 100),
      ]);
      if (invRes.status === "fulfilled") {
        setInvoices(invRes.value.invoices || []);
      } else {
        throw invRes.reason;
      }
      if (schRes.status === "fulfilled") {
        setSchedules(schRes.value.fee_schedules || []);
      } else {
        throw schRes.reason;
      }
      if (tenRes.status === "fulfilled") {
        setSupplierTenants(
          (tenRes.value.tenants || []).filter(
            (t) => (t.TenantType || "").toUpperCase() === "SUPPLIER",
          ),
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load billing data");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void fetchBillingData();
  }, [fetchBillingData]);

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

  // Derive suppliers dynamically from schedules and invoices
  const derivedSuppliers = useMemo<DerivedSupplier[]>(() => {
    const map = new Map<string, { gmvUZS: number; unpaidMinor: number; tier: string; feeBps: number }>();

    for (const sch of schedules) {
      if (!sch.supplier_id) continue;
      map.set(sch.supplier_id, {
        gmvUZS: 0,
        unpaidMinor: 0,
        tier: sch.tier || "STANDARD",
        feeBps: sch.gmv_bps ?? 500,
      });
    }

    for (const inv of invoices) {
      const sid = inv.billed_supplier_id;
      if (!sid) continue;
      const current = map.get(sid) || {
        gmvUZS: 0,
        unpaidMinor: 0,
        tier: "STANDARD",
        feeBps: 500,
      };
      const invGmvUZS = Math.round((inv.principal_minor || 0) / 100);
      current.gmvUZS += invGmvUZS;
      current.unpaidMinor += inv.balance_minor || 0;
      map.set(sid, current);
    }

    return Array.from(map.entries()).map(([id, info]) => ({
      id,
      name: `Supplier ${id}`,
      tier: info.tier,
      gmvUZS: info.gmvUZS,
      feeBps: info.feeBps,
      unpaidMinor: info.unpaidMinor,
    }));
  }, [invoices, schedules]);

  useEffect(() => {
    if (derivedSuppliers.length > 0 && (!selectedSupplierId || !derivedSuppliers.some(s => s.id === selectedSupplierId))) {
      setSelectedSupplierId(derivedSuppliers[0].id);
    }
  }, [derivedSuppliers, selectedSupplierId]);

  const entityChoices = useMemo(() => {
    const map = new Map<string, string>();
    for (const s of derivedSuppliers) {
      map.set(s.id, s.name);
    }
    for (const t of supplierTenants) {
      if (t.TenantID && !map.has(t.TenantID)) {
        map.set(t.TenantID, t.DisplayName || t.TenantID);
      }
    }
    return Array.from(map.entries()).map(([id, label]) => ({ id, label }));
  }, [derivedSuppliers, supplierTenants]);

  useEffect(() => {
    if (!glEntityId && entityChoices.length > 0) {
      setGlEntityId(entityChoices[0].id);
    }
  }, [entityChoices, glEntityId]);

  const fetchFinanceGlAndRecon = useCallback(async () => {
    if (!token || !glEntityId.trim()) return;
    setGlLoading(true);
    setGlError(null);
    try {
      const targetPeriod = /^\d{4}-(0[1-9]|1[0-2])$/.test(glPeriod.trim()) ? glPeriod.trim() : undefined;
      const [tbRes, feeLedgerRes, runsRes, breaksRes] = await Promise.all([
        api.getFinanceTrialBalance(token, glEntityId, targetPeriod),
        api.getFinanceAccountLedger(token, glEntityId, "REV:PLATFORM:FEE", targetPeriod, 15),
        api.listFinanceReconRuns(token, glEntityId, 5),
        api.listFinanceReconBreaks(token, glEntityId, "OPEN", 50),
      ]);
      setTrialBalance(tbRes);
      setPlatformFeeLedger(feeLedgerRes.items || []);
      setReconRuns(runsRes.runs || []);
      setReconBreaks(breaksRes.breaks || []);
    } catch (err) {
      setGlError(err instanceof Error ? err.message : "Failed to load GL & reconciliation telemetry");
      setTrialBalance(null);
      setPlatformFeeLedger([]);
      setReconRuns([]);
      setReconBreaks([]);
    } finally {
      setGlLoading(false);
    }
  }, [glEntityId, glPeriod, token]);

  useEffect(() => {
    void fetchFinanceGlAndRecon();
  }, [fetchFinanceGlAndRecon]);

  const handleRunRecon = async () => {
    if (!token || !glEntityId.trim()) return;
    setRunningRecon(true);
    setGlError(null);
    try {
      const targetPeriod = /^\d{4}-(0[1-9]|1[0-2])$/.test(glPeriod.trim()) ? glPeriod.trim() : undefined;
      await api.triggerFinanceReconRun(token, glEntityId, targetPeriod);
      await fetchFinanceGlAndRecon();
    } catch (err) {
      setGlError(err instanceof Error ? err.message : "Reconciliation run failed");
    } finally {
      setRunningRecon(false);
    }
  };

  const handleBreakAction = async (breakId: string, action: "ACKNOWLEDGE" | "RESOLVE") => {
    if (!token || !glEntityId.trim()) return;
    setBusyBreakId(breakId);
    setGlError(null);
    try {
      await api.updateFinanceReconBreak(token, glEntityId, breakId, { action });
      await fetchFinanceGlAndRecon();
    } catch (err) {
      setGlError(err instanceof Error ? err.message : "Break update failed");
    } finally {
      setBusyBreakId(null);
    }
  };

  // Macro Metrics (Derived & Aggregate from Invoices)
  const totalGMVUZS = useMemo(() => {
    return invoices.reduce((acc, inv) => acc + Math.round((inv.principal_minor || 0) / 100), 0);
  }, [invoices]);

  const platformTakeRateUZS = useMemo(() => {
    if (totalGMVUZS === 0) return 0;
    const firstSchedule = schedules[0] as (BillingFeeSchedule & { GmvBps?: number }) | undefined;
    const bps = firstSchedule?.gmv_bps ?? firstSchedule?.GmvBps ?? 500;
    return applyBpsFee(totalGMVUZS, bps);
  }, [totalGMVUZS, schedules]);

  const soliqVatUZS = totalGMVUZS > 0 ? applyBpsFee(totalGMVUZS, 1200) : 0; // Dynamic 12% Soliq VAT (1200 bps)
  const logisticsAllocationUZS = totalGMVUZS > 0 ? applyBpsFee(totalGMVUZS, 250) : 0;
  const netSupplierPayoutUZS = totalGMVUZS > 0 ? Math.max(0, totalGMVUZS - platformTakeRateUZS - soliqVatUZS - logisticsAllocationUZS) : 0;
  const cloudInfraCostUZS = totalGMVUZS > 0 ? applyBpsFee(totalGMVUZS, 172) : 0; // Baseline compute allocation
  const supplierSharePct = totalGMVUZS > 0 ? formatRatioPercentTenths(netSupplierPayoutUZS, totalGMVUZS) : "0.0";
  const vatSharePct = totalGMVUZS > 0 ? formatRatioPercentTenths(soliqVatUZS, totalGMVUZS) : "0.0";
  const takeSharePct = totalGMVUZS > 0 ? formatRatioPercentTenths(platformTakeRateUZS, totalGMVUZS) : "0.0";
  const fleetSharePct = totalGMVUZS > 0 ? formatRatioPercentTenths(logisticsAllocationUZS, totalGMVUZS) : "0.0";

  const effectiveBps = totalGMVUZS > 0 ? Math.round((platformTakeRateUZS / totalGMVUZS) * 10000) : 500;

  const activeSupplier = derivedSuppliers.find((s) => s.id === selectedSupplierId) || derivedSuppliers[0];

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

      {/* Perspective Dimension Selector */}
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
        {mode === "SUPPLIER" && derivedSuppliers.length > 0 && (
          <div className="flex items-center gap-2">
            <label className="text-xs font-medium text-gray-600">Select Supplier:</label>
            <select
              value={selectedSupplierId}
              onChange={(e) => setSelectedSupplierId(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1 text-xs font-semibold"
            >
              {derivedSuppliers.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} ({s.tier})
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
                {formatMillionsUzs(totalGMVUZS, 1)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-emerald-600 mt-1 font-semibold">
                {invoices.length} active invoices reconciled
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Platform Take Rate</span>
              <div className="text-xl font-extrabold text-indigo-700 mt-1">
                {formatMillionsUzs(platformTakeRateUZS, 2)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-gray-500 mt-1">
                Effective blended: <span className="font-semibold text-gray-700">{formatBpsPercent(effectiveBps, 2)}% ({effectiveBps} bps)</span>
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Soliq 12% VAT Accrued</span>
              <div className="text-xl font-extrabold text-gray-900 mt-1">
                {formatMillionsUzs(soliqVatUZS, 1)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-gray-500 mt-1">
                Electronic factura statutory sweep
              </div>
            </div>

            <div className="rounded-xl border bg-white p-4 shadow-sm">
              <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">Cloud FinOps Cost</span>
              <div className="text-xl font-extrabold text-emerald-700 mt-1">
                {formatMillionsUzs(cloudInfraCostUZS, 2)}M <span className="text-xs font-medium text-gray-500">UZS</span>
              </div>
              <div className="text-[11px] text-emerald-700 font-semibold mt-1">
                {totalGMVUZS > 0 ? "Cost-to-Serve: 1.72% of GMV" : "Infrastructure idle baseline"}
              </div>
            </div>
          </div>

          {/* Tender Waterfall Visualization */}
          <div className="rounded-xl border bg-white p-6 shadow-sm space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div>
                <h2 className="text-sm font-bold text-gray-900">
                  Ecosystem Value &amp; Settlement Waterfall
                </h2>
                <p className="text-xs text-gray-500">
                  Deterministic breakdown of captured GMV across settlement slices, tax obligations, and infrastructure.
                </p>
              </div>
              <span className="rounded bg-indigo-50 px-2.5 py-1 text-xs font-bold text-indigo-700">
                Gross GMV: {totalGMVUZS.toLocaleString()} UZS
              </span>
            </div>

            {/* Visual Waterfall Stack */}
            {totalGMVUZS === 0 ? (
              <div className="rounded-lg border border-dashed p-6 text-center text-xs text-gray-400">
                No GMV volume recorded in the current billing window. Run monthly billing or wait for cleared invoices.
              </div>
            ) : (
              <div className="space-y-3">
                <div className="h-6 w-full rounded-lg bg-gray-100 overflow-hidden flex shadow-inner">
                  <div style={{ width: `${supplierSharePct}%` }} className="bg-indigo-600 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Supplier Net Settlement: ${supplierSharePct}%`}>
                    Supplier ({supplierSharePct}%)
                  </div>
                  <div style={{ width: `${vatSharePct}%` }} className="bg-amber-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Soliq VAT: ${vatSharePct}%`}>
                    VAT ({vatSharePct}%)
                  </div>
                  <div style={{ width: `${takeSharePct}%` }} className="bg-emerald-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Platform Take Rate: ${takeSharePct}%`}>
                    Take ({takeSharePct}%)
                  </div>
                  <div style={{ width: `${fleetSharePct}%` }} className="bg-purple-500 h-full flex items-center justify-center text-[10px] font-bold text-white tracking-wider" title={`Logistics & Fleet: ${fleetSharePct}%`}>
                    Fleet ({fleetSharePct}%)
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
                      {logisticsAllocationUZS.toLocaleString()} UZS
                    </div>
                    <div className="text-[11px] text-gray-500">Carrier per-drop & fuel surcharge</div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* SUPPLIER PERSPECTIVE */}
      {mode === "SUPPLIER" && (
        <div className="space-y-6">
          {!activeSupplier ? (
            <div className="rounded-xl border border-dashed bg-white p-8 text-center text-xs text-gray-500">
              {loading ? "Loading supplier ledgers from Cloud Spanner..." : "No supplier fee schedules or invoices found for the current billing cycle."}
            </div>
          ) : (
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
                  <div className="text-xs text-gray-500 mt-1">Platform Fee: {activeSupplier.feeBps} bps ({formatBpsPercent(activeSupplier.feeBps, 2)}%)</div>
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
                    {applyBpsFee(activeSupplier.gmvUZS, activeSupplier.feeBps).toLocaleString()} UZS
                  </div>
                </div>

                <div className="rounded-lg border bg-gray-50 p-4">
                  <div className="text-xs font-medium text-gray-500">Unsettled Escrow Balance</div>
                  <div className="text-lg font-extrabold text-amber-700 mt-1">
                    {formatMinorUnits(activeSupplier.unpaidMinor, { minimumFractionDigits: 0, maximumFractionDigits: 2 })} UZS
                  </div>
                  <div className="text-[10px] text-gray-400 mt-0.5">Holding in rolling damage reserve</div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* WAREHOUSE / DEPOT PERSPECTIVE */}
      {mode === "WAREHOUSE" && (
        <div className="space-y-6">
          <div className="rounded-xl border border-dashed bg-white p-8 text-center text-xs text-gray-500">
            No depot warehouse operations active in the current billing window.
          </div>
        </div>
      )}

      {/* RETAILER PERSPECTIVE */}
      {mode === "RETAILER" && (
        <div className="space-y-6">
          <div className="rounded-xl border border-dashed bg-white p-8 text-center text-xs text-gray-500">
            No retailer commercial credit accounts active in the current billing window.
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
                    <td className="px-4 py-2 font-mono font-semibold">{formatMinorUnits(inv.principal_minor, { minimumFractionDigits: 0, maximumFractionDigits: 2 })} UZS</td>
                    <td className="px-4 py-2 font-mono text-rose-600 font-bold">{formatMinorUnits(inv.balance_minor, { minimumFractionDigits: 0, maximumFractionDigits: 2 })} UZS</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* GL Platform Fee, Trial Balance & 4-Way Reconciliation Health */}
      <div className="rounded-xl border bg-white p-5 shadow-sm space-y-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b pb-3">
          <div>
            <h3 className="text-sm font-bold text-gray-900">
              General Ledger Platform Fees, Trial Balance &amp; 4-Way Reconciliation Health
            </h3>
            <p className="text-xs text-gray-500">
              Live Cloud Spanner GL balances (REV:PLATFORM:FEE, AR:SUPPLIER:FEES) and 4-way reconciliation checks scoped by entity.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {entityChoices.length > 0 ? (
              <select
                aria-label="GL Entity"
                value={glEntityId}
                onChange={(e) => setGlEntityId(e.target.value)}
                className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
              >
                {entityChoices.map((ent) => (
                  <option key={ent.id} value={ent.id}>
                    {ent.label} ({ent.id})
                  </option>
                ))}
              </select>
            ) : (
              <input
                type="text"
                aria-label="GL Entity ID"
                placeholder="Supplier entity ID..."
                value={glEntityId}
                onChange={(e) => setGlEntityId(e.target.value)}
                className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
              />
            )}
            <input
              type="month"
              aria-label="GL Period"
              value={glPeriod}
              onChange={(e) => setGlPeriod(e.target.value)}
              className="rounded-lg border border-gray-300 px-2.5 py-1.5 text-xs font-mono"
            />
            <button
              type="button"
              disabled={!token || !glEntityId.trim() || runningRecon}
              onClick={() => void handleRunRecon()}
              className="rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-semibold text-white hover:bg-gray-800 disabled:opacity-50"
            >
              {runningRecon ? "Running Recon..." : "Run 4-Way Recon"}
            </button>
            <button
              type="button"
              disabled={!token || !glEntityId.trim() || glLoading}
              onClick={() => void fetchFinanceGlAndRecon()}
              className="rounded-lg border border-gray-300 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              {glLoading ? "Loading..." : "Reload GL"}
            </button>
          </div>
        </div>

        {glError && (
          <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800" role="alert">
            {glError}
          </div>
        )}

        {!glEntityId.trim() ? (
          <div className="rounded-lg border border-dashed p-6 text-center text-xs text-gray-500">
            Select or enter a supplier entity ID to inspect GL trial balance, platform fee postings, and 4-way reconciliation runs.
          </div>
        ) : (
          <>
            {(() => {
              const currency = trialBalance?.currency || reconRuns[0]?.currency || "UZS";
              const platformFeeLine = trialBalance?.accounts.find((a) => a.account_code === "REV:PLATFORM:FEE");
              const feeReceivableLine = trialBalance?.accounts.find((a) => a.account_code === "AR:SUPPLIER:FEES");
              const latestRun = reconRuns[0] ?? null;

              return (
                <>
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div className="rounded-lg border bg-gray-50 p-4">
                      <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
                        GL Platform Fee Revenue
                      </span>
                      <div className="text-lg font-extrabold text-gray-900 mt-1 font-mono">
                        {formatFinOpsMinor(platformFeeLine?.net_minor ?? 0, currency)}
                      </div>
                      <div className="text-[11px] text-gray-500 mt-1 font-mono">REV:PLATFORM:FEE</div>
                    </div>

                    <div className="rounded-lg border bg-gray-50 p-4">
                      <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
                        GL Supplier Fee Receivable
                      </span>
                      <div className="text-lg font-extrabold text-gray-900 mt-1 font-mono">
                        {formatFinOpsMinor(feeReceivableLine?.net_minor ?? 0, currency)}
                      </div>
                      <div className="text-[11px] text-gray-500 mt-1 font-mono">AR:SUPPLIER:FEES</div>
                    </div>

                    <div className="rounded-lg border bg-gray-50 p-4">
                      <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
                        GL Trial Balance ({trialBalance?.period_id || glPeriod})
                      </span>
                      <div className="text-lg font-extrabold text-gray-900 mt-1">
                        {trialBalance ? (trialBalance.balanced ? "BALANCED" : "UNBALANCED") : "—"}
                      </div>
                      <div className="text-[11px] text-gray-500 mt-1 font-mono">
                        DR {formatFinOpsMinor(trialBalance?.total_debits_minor ?? 0, currency)} / CR {formatFinOpsMinor(trialBalance?.total_credits_minor ?? 0, currency)}
                      </div>
                    </div>

                    <div className="rounded-lg border bg-gray-50 p-4">
                      <span className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
                        4-Way Recon Health
                      </span>
                      <div className="text-lg font-extrabold text-gray-900 mt-1">
                        {latestRun ? latestRun.status : "NO RUNS"}
                      </div>
                      <div className="text-[11px] text-gray-500 mt-1 font-mono">
                        {reconBreaks.length} open · Exp {formatFinOpsMinor(latestRun?.exposure_minor ?? 0, currency)}
                      </div>
                    </div>
                  </div>

                  {/* 4-Way Dimension Strip */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                    {RECON_DIMENSIONS.map((dim) => {
                      const check = latestRun?.checks?.[dim];
                      return (
                        <div key={dim} className="rounded-lg border p-3 bg-white">
                          <div className="flex items-center justify-between text-xs font-bold text-gray-800">
                            <span className="font-mono">{dim}</span>
                            <span className="rounded-full border px-2 py-0.5 text-[10px]">
                              {check?.status ?? "PENDING"}
                            </span>
                          </div>
                          {check ? (
                            <div className="mt-2 text-[11px] font-mono text-gray-600 space-y-0.5">
                              <div>Delta: {formatFinOpsMinor(check.delta_minor, check.currency || currency, "always")}</div>
                              <div>Breaks: {check.break_count}</div>
                            </div>
                          ) : (
                            <div className="mt-2 text-[11px] text-gray-400">No run snapshot</div>
                          )}
                        </div>
                      );
                    })}
                  </div>

                  {/* Open Recon Breaks */}
                  {reconBreaks.length > 0 && (
                    <div className="overflow-x-auto border rounded-lg">
                      <table className="w-full border-collapse text-left text-xs">
                        <thead className="border-b bg-gray-50 uppercase text-gray-500 font-bold">
                          <tr>
                            <th className="px-3 py-2">Dimension</th>
                            <th className="px-3 py-2">Object</th>
                            <th className="px-3 py-2 text-right">Delta</th>
                            <th className="px-3 py-2">Severity</th>
                            <th className="px-3 py-2">Status</th>
                            <th className="px-3 py-2 text-right">Actions</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100">
                          {reconBreaks.map((brk) => (
                            <tr key={brk.break_id}>
                              <td className="px-3 py-2 font-mono font-semibold">{brk.dimension}</td>
                              <td className="px-3 py-2 font-mono">{brk.object_type}:{brk.object_id}</td>
                              <td className="px-3 py-2 font-mono text-right">
                                {formatFinOpsMinor(brk.delta_minor, brk.currency || currency, "always")}
                              </td>
                              <td className="px-3 py-2">{brk.severity}</td>
                              <td className="px-3 py-2">{brk.status}</td>
                              <td className="px-3 py-2 text-right space-x-1">
                                {brk.status === "OPEN" && (
                                  <button
                                    type="button"
                                    disabled={busyBreakId === brk.break_id}
                                    onClick={() => void handleBreakAction(brk.break_id, "ACKNOWLEDGE")}
                                    className="rounded border px-2 py-0.5 text-[11px] font-semibold hover:bg-gray-50"
                                  >
                                    Ack
                                  </button>
                                )}
                                {(brk.status === "OPEN" || brk.status === "ACKNOWLEDGED") && (
                                  <button
                                    type="button"
                                    disabled={busyBreakId === brk.break_id}
                                    onClick={() => void handleBreakAction(brk.break_id, "RESOLVE")}
                                    className="rounded border px-2 py-0.5 text-[11px] font-semibold hover:bg-gray-50"
                                  >
                                    Resolve
                                  </button>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Platform Fee Postings & Trial Balance COA */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <div className="rounded-lg border p-3">
                      <h4 className="text-xs font-bold uppercase tracking-wider text-gray-500 mb-2">
                        Recent REV:PLATFORM:FEE GL Postings ({platformFeeLedger.length})
                      </h4>
                      {platformFeeLedger.length === 0 ? (
                        <p className="text-xs text-gray-400">No platform fee GL postings recorded for {glEntityId}.</p>
                      ) : (
                        <table className="w-full text-left text-xs border-collapse">
                          <thead>
                            <tr className="border-b text-gray-500">
                              <th className="py-1.5 pr-2">Entry</th>
                              <th className="py-1.5 pr-2">Source</th>
                              <th className="py-1.5 pr-2">Dir</th>
                              <th className="py-1.5 text-right">Amount</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-100">
                            {platformFeeLedger.map((row) => (
                              <tr key={`${row.entry_id}:${row.line}`}>
                                <td className="py-1.5 pr-2 font-mono">{row.entry_id}</td>
                                <td className="py-1.5 pr-2 font-mono text-gray-600">{row.source_type}:{row.source_id}</td>
                                <td className="py-1.5 pr-2 font-mono">{row.direction}</td>
                                <td className="py-1.5 font-mono text-right font-semibold">
                                  {formatFinOpsMinor(row.amount_minor, row.currency || currency)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>

                    <div className="rounded-lg border p-3">
                      <h4 className="text-xs font-bold uppercase tracking-wider text-gray-500 mb-2">
                        Trial Balance Summary ({trialBalance?.accounts.length ?? 0} Accounts)
                      </h4>
                      {!trialBalance || trialBalance.accounts.length === 0 ? (
                        <p className="text-xs text-gray-400">No GL balances posted for {glEntityId} in {glPeriod}.</p>
                      ) : (
                        <table className="w-full text-left text-xs border-collapse">
                          <thead>
                            <tr className="border-b text-gray-500">
                              <th className="py-1.5 pr-2">Account</th>
                              <th className="py-1.5 pr-2 text-right">Debit</th>
                              <th className="py-1.5 pr-2 text-right">Credit</th>
                              <th className="py-1.5 text-right">Net</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-100">
                            {trialBalance.accounts.map((acc) => (
                              <tr key={acc.account_code}>
                                <td className="py-1.5 pr-2 font-mono">{acc.account_code}</td>
                                <td className="py-1.5 pr-2 font-mono text-right">
                                  {formatFinOpsMinor(acc.debit_minor, acc.currency || currency)}
                                </td>
                                <td className="py-1.5 pr-2 font-mono text-right">
                                  {formatFinOpsMinor(acc.credit_minor, acc.currency || currency)}
                                </td>
                                <td className="py-1.5 font-mono text-right font-semibold">
                                  {formatFinOpsMinor(acc.net_minor, acc.currency || currency)}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </div>
                  </div>
                </>
              );
            })()}
          </>
        )}
      </div>
    </div>
  );
}
